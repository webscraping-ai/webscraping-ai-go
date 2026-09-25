# webscraping-ai-go

[![CI](https://github.com/webscraping-ai/webscraping-ai-go/actions/workflows/ci.yml/badge.svg)](https://github.com/webscraping-ai/webscraping-ai-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/webscraping-ai/webscraping-ai-go/v4.svg)](https://pkg.go.dev/github.com/webscraping-ai/webscraping-ai-go/v4)

Official Go client for the [WebScraping.AI](https://webscraping.ai) API —
web scraping with Chromium JavaScript rendering, rotating
datacenter/residential/stealth proxies, and AI-powered question answering and
structured field extraction on any page. See the
[API documentation](https://webscraping.ai/docs) for the full parameter reference.

## Install

```bash
go get github.com/webscraping-ai/webscraping-ai-go/v4@latest
```

The `/v4` suffix in the import path is Go's
[semantic import versioning](https://go.dev/ref/mod#major-version-suffixes)
convention for modules at major version ≥ 2. The version line is kept
in lockstep with the other WebScraping.AI SDKs (Ruby, Python, PHP,
JavaScript — all at 4.0.x).

Requires Go 1.22 or newer. Zero runtime dependencies — uses the standard
library's `net/http`.

## Quick start

[Sign up](https://webscraping.ai/auth/sign_up) to get an API key — the free
trial includes 2,000 credits, no credit card required. Your key lives in the
[dashboard](https://webscraping.ai/dashboard).

```go
package main

import (
    "context"
    "fmt"
    "log"

    webscrapingai "github.com/webscraping-ai/webscraping-ai-go/v4"
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

    fields, err := client.Fields(ctx, &webscrapingai.FieldsOptions{
        URL: "https://example.com",
        Fields: map[string]string{
            "title": "Main product title",
            "price": "Current product price",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(fields.Result)

    // Google search results (flat 15 credits per search)
    serp, err := client.Serp(ctx, &webscrapingai.SerpOptions{
        Q: "coffee machines",
    })
    if err != nil {
        log.Fatal(err)
    }
    for _, r := range serp.OrganicResults {
        fmt.Println(r.Position, r.Title, r.Link)
    }

    // Structured data for a page on a supported site (flat 15 credits)
    data, err := client.Data(ctx, &webscrapingai.DataOptions{
        URL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(data.RequestParameters.Provider, data.ParseStatus, string(data.Data))

    // Account quota
    info, err := client.Account(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("%s — %d remaining\n", info.Email, info.RemainingAPICalls)
}
```

`Config.APIKey` is read from the `WEBSCRAPING_AI_API_KEY` environment
variable as a fallback:

```go
// WEBSCRAPING_AI_API_KEY="..." in the environment
client, err := webscrapingai.NewClient(nil)
```

## Search engine results (SERP)

`Serp` calls `GET /serp` and returns parsed Google results as a typed
`*SerpResult`. It is query-shaped — pass the search query in `Q` instead
of a URL. None of the page-scraping options (JS, proxy, country, …)
apply. Flat 15 credits per search; failed searches are not charged.

`Serp` rejects a blank (empty or whitespace-only) `Q` and a `Page` below 1
before sending anything. The server also rejects an invalid page with a
400 (not billed); checking client-side saves the round trip. Pages are
1–100: the server rejects a `Page` above 100 with a 400.

```go
page := 2
serp, err := client.Serp(ctx, &webscrapingai.SerpOptions{
    Q:      "coffee machines", // required
    Engine: "google",          // optional, default "google" (only engine today)
    GL:     "de",              // optional two-letter country, default "us"
    HL:     "de",              // optional two-letter language, default "en"
    Page:   &page,             // optional, 1-based, 10 results per page (server rejects > 100 with a 400)
})
if err != nil {
    log.Fatal(err)
}

fmt.Println(serp.SearchInformation.OrganicResultsState) // "Results for exact spelling"
for _, r := range serp.OrganicResults {
    // Position restarts at 1 on every page; (page-1)*10 + Position is
    // the absolute rank.
    fmt.Println(r.Position, r.Title, r.Link, r.Domain)
    if r.Snippet != nil {
        fmt.Println("  ", *r.Snippet)
    }
}
for _, rs := range serp.RelatedSearches {
    fmt.Println("related:", rs.Query)
}
if serp.Pagination.Next != nil {
    fmt.Println("next page:", *serp.Pagination.Next)
}
```

Optional response fields (`Snippet`, `Date`, `ShowingResultsFor`,
`TotalResults`, `Pagination.Next`) are pointers and are `nil` when the
API omits them; `RelatedSearches` is `nil` when the page shows none.

## Structured data (/data)

`Data` calls `GET /data` and returns structured JSON for a public page on a
supported site. Pass the page's normal URL; the site (`Provider`) and page
kind (`Type`) are detected on the server. Supported sites today include
YouTube, TikTok, X/Twitter, LinkedIn, Instagram and Reddit (for example a
YouTube video, a TikTok profile or a Reddit thread). More sites are added on
the server without a client release, so the client never checks the URL
beyond rejecting a blank one. An unsupported URL or page type returns a
400 that is not charged (`*BadRequestError`). Its message lists what is
supported. For other sites, use `Fields`.

15 credits per request, including results with `ParseStatus`
`parse_failed` or `not_found`; failed fetches are not charged. None of the
page-scraping options (JS, proxy, headers, …) apply.

```go
transcript := true
res, err := client.Data(ctx, &webscrapingai.DataOptions{
    URL:                "https://www.youtube.com/watch?v=dQw4w9WgXcQ", // required
    Country:            "us",         // optional two-letter proxy country, "us" by default
    Transcript:         &transcript, // optional, YouTube videos only
    TranscriptLanguage: "en",         // optional caption language, with Transcript
    // Params: map[string]string{"some_new_param": "value"}, // extra params, sent as-is
})
if err != nil {
    log.Fatal(err)
}

fmt.Println(res.RequestParameters.Provider, res.RequestParameters.Type, res.ParseStatus) // youtube video ok

// Data is raw JSON (json.RawMessage) whose shape depends on Provider and
// Type; it is nil when the API returned null. Decode it into a map or your
// own struct.
var video struct {
    Title     string `json:"title"`
    ViewCount int64  `json:"view_count"`
}
if res.Data != nil {
    if err := json.Unmarshal(res.Data, &video); err != nil {
        log.Fatal(err)
    }
}
fmt.Println(video.Title)
```

- `Country`: two-letter country code of the proxy used to fetch the page,
  `us` by default. The server rejects unknown codes with a 400.
- `Transcript`: YouTube videos only. Also fetch the video's transcript into
  `data.transcript`. It's null when no matching captions are available. If
  the transcript fetch itself fails, the whole request fails with a 500 and
  is not charged.
- `TranscriptLanguage`: caption language to pick, e.g. `en` or `de`.
  Without it, English is preferred, then the first available track. If the
  video has no captions in that language, `data.transcript` is null.

`Provider`, `Type` and `ParseStatus` are plain strings: new values appear
as the server adds sites. `ParseStatus` is `ok`, `parse_failed` (fetched
but not parsed; `Data` may be nil or partial) or `not_found`.

`Params` sends extra query parameters as-is, so provider-specific options
added on the server later work without upgrading the client. It rejects
`api_key` and `url` keys, and the typed option names `country`,
`transcript` and `transcript_language` (use `Country`, `Transcript` and
`TranscriptLanguage`), whether or not those fields are set.

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

# Live smoke (hits production, costs ~46 credits):
WEBSCRAPING_AI_API_KEY=... go run ./cmd/smoke
```

## Links

- [WebScraping.AI](https://webscraping.ai) — features, pricing, signup
- [API documentation](https://webscraping.ai/docs)
- [Dashboard](https://webscraping.ai/dashboard) — API key, usage, request builder
- Other official clients: [Python](https://github.com/webscraping-ai/webscraping-ai-python) · [JavaScript](https://github.com/webscraping-ai/webscraping-ai-js) · [Ruby](https://github.com/webscraping-ai/webscraping-ai-ruby) · [PHP](https://github.com/webscraping-ai/webscraping-ai-php) · [Java](https://github.com/webscraping-ai/webscraping-ai-java) · [.NET](https://github.com/webscraping-ai/webscraping-ai-dotnet) · [CLI](https://github.com/webscraping-ai/webscraping-ai-cli) · [MCP server](https://github.com/webscraping-ai/webscraping-ai-mcp-server) · [n8n node](https://github.com/webscraping-ai/webscraping-ai-n8n)
- Support: [support@webscraping.ai](mailto:support@webscraping.ai)

## License

MIT — see [LICENSE](LICENSE).
