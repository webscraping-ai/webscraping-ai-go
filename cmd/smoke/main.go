// Hand-run smoke test against the live API. Not part of `go test ./...`
// — costs ~46 credits per full sweep: page tools run with js=false and
// the datacenter proxy (html/text/selected/selected_multiple 4 × 1,
// question/fields 2 × 6), plus 15 for the SERP search and 15 for one
// /data call. The /data unsupported-URL check is a free 400.
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
	// cost the header claims: 1 credit per page call, 6 per AI call.
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
		{"data_unsupported", func() (string, error) {
			// No client-side site filter: the server must answer with a
			// free 400 for a site /data doesn't support.
			_, err := client.Data(ctx, &webscrapingai.DataOptions{URL: "https://example.com/"})
			var br *webscrapingai.BadRequestError
			if !errors.As(err, &br) {
				if err == nil {
					return "", errors.New("expected a 400 from the server, got success")
				}
				return "", fmt.Errorf("expected *BadRequestError, got %w", err)
			}
			if !strings.Contains(br.Message, "Unsupported URL") {
				return "", fmt.Errorf("400 message lacks %q: %s", "Unsupported URL", br.Message)
			}
			return fmt.Sprintf("HTTP %d: %s", br.HTTPStatus, br.Message), nil
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

var jsOff = false

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
