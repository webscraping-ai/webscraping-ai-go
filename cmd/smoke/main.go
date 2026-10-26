// Hand-run smoke test against the live API. Not part of `go test ./...`
// — uses real credits (see https://webscraping.ai/docs for pricing): page
// tools (html/text/selected/selected_multiple, question/fields) run with
// js=false and the datacenter proxy to keep the cost down, plus one SERP
// search, a two-page located SERP search (billed per page returned) and
// one /data call on a page with a dedicated parser. /serp/locations is
// free. The /data any-site check on example.com is not charged if it
// returns 503, 422 or a 500 timeout; a success envelope is charged.
//
// Each case asserts on the result shape, not just the absence of an
// error, and a panic in one case is reported as a FAIL without stopping
// the sweep. Exits non-zero if any case failed.
//
// Usage:
//
//	WEBSCRAPING_AI_API_KEY=... go run ./cmd/smoke
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	webscrapingai "github.com/webscraping-ai/webscraping-ai-go/v4"
)

func main() {
	apiKey := os.Getenv("WEBSCRAPING_AI_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "WEBSCRAPING_AI_API_KEY is required")
		os.Exit(2)
	}

	client, err := webscrapingai.NewClient(&webscrapingai.Config{
		APIKey:  apiKey,
		Timeout: 90 * time.Second,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx := context.Background()
	target := "https://example.com"
	// js=false + datacenter keeps every page-tool call at the
	// documented minimum cost.
	common := webscrapingai.CommonOptions{JS: &jsOff, Proxy: "datacenter"}
	nonEmpty := func(s string, err error) (string, error) {
		if err == nil && strings.TrimSpace(s) == "" {
			return "", errors.New("empty result")
		}
		return s, err
	}

	cases := []struct {
		name string
		run  func() (string, error)
	}{
		{"account", func() (string, error) {
			info, err := client.Account(ctx)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("email=%s remaining=%d resets_at=%d concurrency=%d",
				info.Email, info.RemainingAPICalls, info.ResetsAt, info.RemainingConcurrency), nil
		}},
		{"html", func() (string, error) {
			return nonEmpty(client.HTML(ctx, &webscrapingai.HTMLOptions{CommonOptions: common, URL: target}))
		}},
		{"text", func() (string, error) {
			return nonEmpty(client.Text(ctx, &webscrapingai.TextOptions{CommonOptions: common, URL: target}))
		}},
		{"selected", func() (string, error) {
			return nonEmpty(client.Selected(ctx, &webscrapingai.SelectedOptions{CommonOptions: common, URL: target, Selector: "h1"}))
		}},
		{"selected_multiple", func() (string, error) {
			out, err := client.SelectedMultiple(ctx, &webscrapingai.SelectedMultipleOptions{
				CommonOptions: common,
				URL:           target,
				Selectors:     []string{"h1", "p"},
			})
			if err != nil {
				return "", err
			}
			// The API answers 200 [[]] when selectors are mis-encoded.
			for _, inner := range out {
				if len(inner) > 0 {
					return fmt.Sprintf("%v", out), nil
				}
			}
			return "", fmt.Errorf("no selector matched anything: %v", out)
		}},
		{"question", func() (string, error) {
			return nonEmpty(client.Question(ctx, &webscrapingai.QuestionOptions{
				CommonOptions: common,
				URL:           target,
				Question:      "What is this page about? Answer in one sentence.",
			}))
		}},
		{"fields", func() (string, error) {
			out, err := client.Fields(ctx, &webscrapingai.FieldsOptions{
				CommonOptions: common,
				URL:           target,
				Fields: map[string]string{
					"title":       "Page title",
					"description": "Short description",
				},
			})
			if err != nil {
				return "", err
			}
			if out == nil || out.Result == nil {
				return "", errors.New("response has no result")
			}
			return fmt.Sprintf("%v", out.Result), nil
		}},
		{"serp", func() (string, error) {
			out, err := client.Serp(ctx, &webscrapingai.SerpOptions{Q: "coffee machines"})
			if err != nil {
				return "", err
			}
			if len(out.OrganicResults) == 0 {
				return "", fmt.Errorf("no organic_results (state=%q)", out.SearchInformation.OrganicResultsState)
			}
			if out.SearchParameters.Q != "coffee machines" {
				return "", fmt.Errorf("search_parameters.q = %q, want %q", out.SearchParameters.Q, "coffee machines")
			}
			top := out.OrganicResults[0].Link
			return fmt.Sprintf("state=%q results=%d top=%s",
				out.SearchInformation.OrganicResultsState, len(out.OrganicResults), top), nil
		}},
		{"serp_location", func() (string, error) {
			out, err := client.Serp(ctx, &webscrapingai.SerpOptions{
				Q:        "coffee machines",
				Location: "Austin, TX",
				Pages:    &twoPages,
			})
			if err != nil {
				return "", err
			}
			if len(out.OrganicResults) == 0 {
				return "", fmt.Errorf("no organic_results (state=%q)", out.SearchInformation.OrganicResultsState)
			}
			used := out.SearchParameters.LocationUsed
			if used == nil || !strings.Contains(*used, "Austin") {
				return "", fmt.Errorf("search_parameters.location_used = %v, want it to contain Austin", used)
			}
			return fmt.Sprintf("location_used=%q results=%d", *used, len(out.OrganicResults)), nil
		}},
		{"serp_locations", func() (string, error) {
			out, err := client.SerpLocations(ctx, &webscrapingai.SerpLocationsOptions{Q: "austin", Limit: &threeLocations})
			if err != nil {
				return "", err
			}
			if len(out) == 0 {
				return "", errors.New("no locations")
			}
			if !strings.Contains(out[0].CanonicalName, "Austin") {
				return "", fmt.Errorf("first canonical_name = %q, want it to contain Austin", out[0].CanonicalName)
			}
			return fmt.Sprintf("count=%d first=%q id=%s", len(out), out[0].CanonicalName, out[0].ID), nil
		}},
		{"data", func() (string, error) {
			out, err := client.Data(ctx, &webscrapingai.DataOptions{URL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ"})
			if err != nil {
				return "", err
			}
			if out.ParseStatus != "ok" {
				return "", fmt.Errorf("parse_status = %q, want ok", out.ParseStatus)
			}
			if out.RequestParameters.Provider != "youtube" {
				return "", fmt.Errorf("request_parameters.provider = %q, want youtube", out.RequestParameters.Provider)
			}
			if out.Data == nil {
				return "", errors.New("data is null")
			}
			var data map[string]any
			if err := json.Unmarshal(out.Data, &data); err != nil {
				return "", fmt.Errorf("data is not an object: %w", err)
			}
			title, _ := data["title"].(string)
			if strings.TrimSpace(title) == "" {
				return "", fmt.Errorf("data.title is empty: %v", data["title"])
			}
			return fmt.Sprintf("provider=%s type=%s parse_status=%s title=%q",
				out.RequestParameters.Provider, out.RequestParameters.Type, out.ParseStatus, title), nil
		}},
		{"data_any_site", func() (string, error) {
			// /data accepts any public page. A page without a dedicated
			// parser may come back as a success envelope, a 503
			// structure_processing (learning outlasted the request), a
			// 422 structure_unavailable, both as a plain *APIError, or a
			// 500 timeout (extracting the data or loading the page ran out
			// of time) as a *ServerError wrapping one. None of the
			// non-success outcomes is charged.
			out, err := client.Data(ctx, &webscrapingai.DataOptions{URL: "https://example.com/"})
			if err != nil {
				want := map[int]string{503: "structure_processing", 422: "structure_unavailable", 500: "timeout"}
				var apiErr *webscrapingai.APIError
				if !errors.As(err, &apiErr) || want[apiErr.HTTPStatus] == "" {
					return "", fmt.Errorf("expected success, 503, 422 or 500, got %w", err)
				}
				var body struct {
					ErrorCode string `json:"error_code"`
				}
				if jerr := json.Unmarshal([]byte(apiErr.ResponseBody), &body); jerr != nil {
					return "", fmt.Errorf("%d body is not JSON: %w", apiErr.HTTPStatus, jerr)
				}
				if body.ErrorCode != want[apiErr.HTTPStatus] {
					return "", fmt.Errorf("%d error_code = %q, want %s", apiErr.HTTPStatus, body.ErrorCode, want[apiErr.HTTPStatus])
				}
				return fmt.Sprintf("HTTP %d %s", apiErr.HTTPStatus, body.ErrorCode), nil
			}
			if out.RequestParameters.URL == "" {
				return "", errors.New("success envelope lacks request_parameters")
			}
			return fmt.Sprintf("provider=%s type=%s parse_status=%s",
				out.RequestParameters.Provider, out.RequestParameters.Type, out.ParseStatus), nil
		}},
	}

	failures := 0
	for _, c := range cases {
		preview, err := runCase(c.run)
		if err != nil {
			failures++
			var apiErr *webscrapingai.APIError
			if errors.As(err, &apiErr) {
				fmt.Printf("  FAIL %-18s  APIError(HTTP %d): %s\n", c.name, apiErr.HTTPStatus, redact(apiErr.Message, apiKey))
			} else {
				fmt.Printf("  FAIL %-18s  %T: %s\n", c.name, err, redact(err.Error(), apiKey))
			}
			continue
		}
		if len(preview) > 120 {
			preview = preview[:120]
		}
		fmt.Printf("  ok   %-18s  %s\n", c.name, redact(preview, apiKey))
	}

	if failures > 0 {
		os.Exit(1)
	}
}

var (
	jsOff          = false
	twoPages       = 2
	threeLocations = 3
)

// runCase runs one case, turning a panic into an error so the sweep
// continues.
func runCase(run func() (string, error)) (preview string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return run()
}

var apiKeyParam = regexp.MustCompile(`api_key=[^&\s"']*`)

// redact scrubs the API key (and any api_key=... pattern) from output.
func redact(s, apiKey string) string {
	s = apiKeyParam.ReplaceAllString(s, "api_key=REDACTED")
	if apiKey != "" {
		s = strings.ReplaceAll(s, apiKey, "REDACTED")
	}
	return s
}
