# webscraping-ai-go

[![CI](https://github.com/webscraping-ai/webscraping-ai-go/actions/workflows/ci.yml/badge.svg)](https://github.com/webscraping-ai/webscraping-ai-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/webscraping-ai/webscraping-ai-go.svg)](https://pkg.go.dev/github.com/webscraping-ai/webscraping-ai-go)

Official Go client for the [WebScraping.AI](https://webscraping.ai) API.

## Install

```bash
go get github.com/webscraping-ai/webscraping-ai-go@latest
```

Requires Go 1.22 or newer. Zero runtime dependencies — uses the standard
library's `net/http`.

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"

    webscrapingai "github.com/webscraping-ai/webscraping-ai-go"
)

func main() {
    client, err := webscrapingai.NewClient(&webscrapingai.Config{
        APIKey: "YOUR_API_KEY",
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Full HTML
    html, err := client.HTML(ctx, &webscrapingai.HTMLOptions{
        URL: "https://example.com",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(html)

    // Visible text
    text, _ := client.Text(ctx, &webscrapingai.TextOptions{
        URL:        "https://example.com",
        TextFormat: "json",
    })
    fmt.Println(text)

    // CSS-selected HTML
    heading, _ := client.Selected(ctx, &webscrapingai.SelectedOptions{
        URL:      "https://example.com",
        Selector: "h1",
    })
    fmt.Println(heading)

    // Multiple selectors at once
    parts, _ := client.SelectedMultiple(ctx, &webscrapingai.SelectedMultipleOptions{
        URL:       "https://example.com",
        Selectors: []string{"h1", "p"},
    })
    fmt.Println(parts)

    // LLM-powered helpers
    answer, _ := client.Question(ctx, &webscrapingai.QuestionOptions{
        URL:      "https://example.com",
        Question: "What is this page about?",
    })
    fmt.Println(answer)

    fields, _ := client.Fields(ctx, &webscrapingai.FieldsOptions{
        URL: "https://example.com",
        Fields: map[string]string{
            "title": "Main product title",
            "price": "Current product price",
        },
    })
    fmt.Println(fields.Result)

    // Account quota
    info, _ := client.Account(ctx)
    fmt.Printf("%s — %d remaining\n", info.Email, info.RemainingAPICalls)
}
```

`Config.APIKey` is read from the `WEBSCRAPING_AI_API_KEY` environment
variable as a fallback:

```go
// WEBSCRAPING_AI_API_KEY="..." in the environment
client, err := webscrapingai.NewClient(nil)
```

## Configuration

```go
client, err := webscrapingai.NewClient(&webscrapingai.Config{
    APIKey:     "YOUR_API_KEY",
    BaseURL:    "https://api.webscraping.ai",  // default
    Timeout:    60 * time.Second,              // default per-request timeout
    HTTPClient: &http.Client{},                // optional: inject your own
})
```

`Config.Timeout` is the *fallback* per-request deadline used when the
caller passes a `context.Context` without its own deadline. Set it to a
negative value to disable the implicit timeout entirely (caller manages
it via `context.WithTimeout`).

## Error handling

Every non-2xx response is mapped to a typed error. Branch on what
matters with `errors.As`:

```go
import "errors"

_, err := client.HTML(ctx, &webscrapingai.HTMLOptions{URL: "https://example.com"})

var (
    apiErr  *webscrapingai.APIError
    authErr *webscrapingai.AuthenticationError
    rlErr   *webscrapingai.RateLimitError
    timeout *webscrapingai.TimeoutError
)
switch {
case errors.As(err, &authErr):
    // 403 — wrong or missing API key
case errors.As(err, &rlErr):
    // 429 — too many concurrent requests
case errors.As(err, &timeout):
    // request did not complete in time
case errors.As(err, &apiErr):
    // any other HTTP response error
}
```

Full error hierarchy:

- `webscrapingai.Error` (interface — base marker for everything from this SDK)
  - `*APIError` (HTTP response received, non-2xx)
    - `*BadRequestError` — HTTP 400
    - `*PaymentRequiredError` — HTTP 402
    - `*AuthenticationError` — HTTP 403
    - `*RateLimitError` — HTTP 429
    - `*ServerError` — HTTP 500
    - `*GatewayTimeoutError` — HTTP 504
  - `*TimeoutError` — no response, context deadline elapsed
  - `*ConnectionError` — no response, transport-level error

Each typed wrapper embeds `*APIError`, so `errors.As(err, &apiErr)` on
`*APIError` also matches any of the per-status wrappers.

`APIError.StatusCode`, `StatusMessage`, `Body`, and `ResponseBody` are
populated when the API surfaces target-page errors as a 5xx.

## Response shapes

Two endpoints return shapes that differ from the OpenAPI spec — they're
upstream drift, reproduced by every official SDK:

- **`Fields`** wraps the extracted fields under `Result`:
  `{Result: {"title": "...", "price": "..."}}`. Access via
  `fields.Result["title"]`.
- **`SelectedMultiple`** returns `[][]string`, not the flat `[]string`
  the spec implies — one outer wrapper containing all matches.

## Development

```bash
go test ./...           # all tests
go vet ./...
gofmt -l .              # any output → unformatted files

# Live smoke (hits production, costs ~17 credits):
WEBSCRAPING_AI_API_KEY=... go run ./cmd/smoke
```

## License

MIT — see [LICENSE](LICENSE).
