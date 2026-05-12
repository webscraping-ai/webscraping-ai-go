// Hand-run smoke test against the live API. Not part of `go test ./...`
// — costs ~17 credits per full sweep.
//
// Usage:
//
//	WEBSCRAPING_AI_API_KEY=... go run ./cmd/smoke
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
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

	cases := []struct {
		name string
		run  func() (string, error)
	}{
		{"account", func() (string, error) {
			info, err := client.Account(ctx)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("email=%s remaining=%d", info.Email, info.RemainingAPICalls), nil
		}},
		{"html", func() (string, error) {
			return client.HTML(ctx, &webscrapingai.HTMLOptions{URL: target})
		}},
		{"text", func() (string, error) {
			return client.Text(ctx, &webscrapingai.TextOptions{URL: target})
		}},
		{"selected", func() (string, error) {
			return client.Selected(ctx, &webscrapingai.SelectedOptions{URL: target, Selector: "h1"})
		}},
		{"selected_multiple", func() (string, error) {
			out, err := client.SelectedMultiple(ctx, &webscrapingai.SelectedMultipleOptions{
				URL:       target,
				Selectors: []string{"h1", "p"},
			})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%v", out), nil
		}},
		{"question", func() (string, error) {
			return client.Question(ctx, &webscrapingai.QuestionOptions{
				URL:      target,
				Question: "What is this page about? Answer in one sentence.",
			})
		}},
		{"fields", func() (string, error) {
			out, err := client.Fields(ctx, &webscrapingai.FieldsOptions{
				URL: target,
				Fields: map[string]string{
					"title":       "Page title",
					"description": "Short description",
				},
			})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%v", out.Result), nil
		}},
	}

	failures := 0
	for _, c := range cases {
		preview, err := c.run()
		if err != nil {
			failures++
			var apiErr *webscrapingai.APIError
			if errors.As(err, &apiErr) {
				fmt.Printf("  FAIL %-18s  APIError(HTTP %d): %s\n", c.name, apiErr.HTTPStatus, apiErr.Message)
			} else {
				fmt.Printf("  FAIL %-18s  %T: %s\n", c.name, err, err)
			}
			continue
		}
		if len(preview) > 120 {
			preview = preview[:120]
		}
		fmt.Printf("  ok   %-18s  %s\n", c.name, preview)
	}

	if failures > 0 {
		os.Exit(1)
	}
}
