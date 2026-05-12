// Package webscrapingai is the official Go client for the
// WebScraping.AI API (https://webscraping.ai).
//
// A minimal example:
//
//	client, err := webscrapingai.NewClient(&webscrapingai.Config{
//	    APIKey: "YOUR_API_KEY",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	html, err := client.HTML(ctx, &webscrapingai.HTMLOptions{
//	    URL: "https://example.com",
//	})
//
// All methods take a context and a typed options struct. Non-2xx
// responses are returned as typed *APIError variants (BadRequestError,
// RateLimitError, …); transport failures as *TimeoutError /
// *ConnectionError. Branch with errors.As.
//
// The API key is also read from the WEBSCRAPING_AI_API_KEY environment
// variable if Config.APIKey is empty.
package webscrapingai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/webscraping-ai/webscraping-ai-go/internal/query"
)

// DefaultBaseURL is the production API base URL.
const DefaultBaseURL = "https://api.webscraping.ai"

// DefaultTimeout is the default per-request timeout when no
// context deadline is set.
const DefaultTimeout = 60 * time.Second

// Config configures a Client. All fields are optional except APIKey
// (which may also be supplied via WEBSCRAPING_AI_API_KEY).
type Config struct {
	// APIKey is the WebScraping.AI API key. If empty, the constructor
	// falls back to the WEBSCRAPING_AI_API_KEY environment variable.
	APIKey string
	// BaseURL overrides the API base URL. Defaults to DefaultBaseURL.
	BaseURL string
	// Timeout sets a fallback per-request timeout used when the caller
	// passes a context without a deadline. Defaults to DefaultTimeout.
	// Use 0 to disable the implicit timeout entirely.
	Timeout time.Duration
	// HTTPClient lets callers inject their own *http.Client (for custom
	// transports, retries, metrics, etc.). Defaults to a fresh
	// *http.Client with Timeout=0 — the SDK manages timeouts via
	// context so the http.Client should not have its own Timeout set.
	HTTPClient *http.Client
}

// Client is the WebScraping.AI API client.
type Client struct {
	apiKey         string
	baseURL        string
	httpClient     *http.Client
	defaultTimeout time.Duration
	userAgent      string
}

// NewClient builds a Client from cfg. Returns an error if no API key is
// available (neither cfg.APIKey nor WEBSCRAPING_AI_API_KEY is set).
//
// cfg may be nil — equivalent to &Config{} (env var must be set).
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = &Config{}
	}
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("WEBSCRAPING_AI_API_KEY")
	}
	if apiKey == "" {
		return nil, errors.New("webscrapingai: APIKey is required (set Config.APIKey or WEBSCRAPING_AI_API_KEY)")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	// cfg.Timeout: 0 → use DefaultTimeout; negative → disable implicit
	// timeout (caller manages it via context); positive → use as-is.
	var timeout time.Duration
	switch {
	case cfg.Timeout == 0:
		timeout = DefaultTimeout
	case cfg.Timeout < 0:
		timeout = 0
	default:
		timeout = cfg.Timeout
	}

	return &Client{
		apiKey:         apiKey,
		baseURL:        baseURL,
		httpClient:     httpClient,
		defaultTimeout: timeout,
		userAgent:      "webscraping-ai-go/" + Version,
	}, nil
}

// HTML calls GET /html and returns the page HTML. When opts.Format is
// "json" the returned string is the raw JSON envelope ({"html": "..."}).
func (c *Client) HTML(ctx context.Context, opts *HTMLOptions) (string, error) {
	if opts == nil || opts.URL == "" {
		return "", errors.New("webscrapingai: HTML requires opts.URL")
	}
	ctx, cancel := c.contextWithTimeout(ctx)
	defer cancel()
	params := commonParams(opts.CommonOptions)
	params.Set("url", opts.URL)
	setIfNotEmpty(&params, "format", opts.Format)
	setBoolPtr(&params, "return_script_result", opts.ReturnScriptResult)
	body, _, err := c.do(ctx, "/html", params)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// Text calls GET /text and returns the page's visible text. When
// opts.TextFormat is "json" the response is a JSON envelope; the client
// returns the raw body for the caller to parse.
func (c *Client) Text(ctx context.Context, opts *TextOptions) (string, error) {
	if opts == nil || opts.URL == "" {
		return "", errors.New("webscrapingai: Text requires opts.URL")
	}
	ctx, cancel := c.contextWithTimeout(ctx)
	defer cancel()
	params := commonParams(opts.CommonOptions)
	params.Set("url", opts.URL)
	setIfNotEmpty(&params, "text_format", opts.TextFormat)
	setBoolPtr(&params, "return_links", opts.ReturnLinks)
	body, _, err := c.do(ctx, "/text", params)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// Selected calls GET /selected and returns the matched element's HTML.
func (c *Client) Selected(ctx context.Context, opts *SelectedOptions) (string, error) {
	if opts == nil || opts.URL == "" || opts.Selector == "" {
		return "", errors.New("webscrapingai: Selected requires opts.URL and opts.Selector")
	}
	ctx, cancel := c.contextWithTimeout(ctx)
	defer cancel()
	params := commonParams(opts.CommonOptions)
	params.Set("url", opts.URL)
	params.Set("selector", opts.Selector)
	setIfNotEmpty(&params, "format", opts.Format)
	body, _, err := c.do(ctx, "/selected", params)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// SelectedMultiple calls GET /selected-multiple and returns the matched
// elements as [][]string. See SelectedMultipleResult for the response
// shape — the API returns one outer wrapper, not a flat list.
func (c *Client) SelectedMultiple(ctx context.Context, opts *SelectedMultipleOptions) (SelectedMultipleResult, error) {
	if opts == nil || opts.URL == "" || len(opts.Selectors) == 0 {
		return nil, errors.New("webscrapingai: SelectedMultiple requires opts.URL and at least one selector")
	}
	ctx, cancel := c.contextWithTimeout(ctx)
	defer cancel()
	params := commonParams(opts.CommonOptions)
	params.Set("url", opts.URL)
	params.Set("selectors", opts.Selectors)
	body, _, err := c.do(ctx, "/selected-multiple", params)
	if err != nil {
		return nil, err
	}
	var out SelectedMultipleResult
	if err := decodeJSON(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Question calls GET /ai/question and returns the LLM's answer about
// the page. When opts.Format is "json" the returned string is the raw
// JSON envelope.
func (c *Client) Question(ctx context.Context, opts *QuestionOptions) (string, error) {
	if opts == nil || opts.URL == "" || opts.Question == "" {
		return "", errors.New("webscrapingai: Question requires opts.URL and opts.Question")
	}
	ctx, cancel := c.contextWithTimeout(ctx)
	defer cancel()
	params := commonParams(opts.CommonOptions)
	params.Set("url", opts.URL)
	params.Set("question", opts.Question)
	setIfNotEmpty(&params, "format", opts.Format)
	body, _, err := c.do(ctx, "/ai/question", params)
	if err != nil {
		return "", err
	}
	// The API may return a JSON-quoted string ("answer") even without
	// format=json. Strip the wrapping quotes if present so callers get
	// the answer text directly.
	return unwrapJSONString(body), nil
}

// Fields calls GET /ai/fields and returns the extracted fields.
//
// The API currently wraps the response under a "result" key — see
// FieldsResult.
func (c *Client) Fields(ctx context.Context, opts *FieldsOptions) (*FieldsResult, error) {
	if opts == nil || opts.URL == "" || len(opts.Fields) == 0 {
		return nil, errors.New("webscrapingai: Fields requires opts.URL and at least one field")
	}
	ctx, cancel := c.contextWithTimeout(ctx)
	defer cancel()
	params := commonParams(opts.CommonOptions)
	params.Set("url", opts.URL)
	params.Set("fields", opts.Fields)
	body, _, err := c.do(ctx, "/ai/fields", params)
	if err != nil {
		return nil, err
	}
	var out FieldsResult
	if err := decodeJSON(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Account calls GET /account and returns the account quota info.
func (c *Client) Account(ctx context.Context) (*AccountInfo, error) {
	ctx, cancel := c.contextWithTimeout(ctx)
	defer cancel()
	body, _, err := c.do(ctx, "/account", nil)
	if err != nil {
		return nil, err
	}
	var out AccountInfo
	if err := decodeJSON(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// contextWithTimeout applies the client's default timeout to ctx if the
// caller did not already set a deadline. Callers who want unlimited
// duration can construct a client with Config.Timeout = -1.
func (c *Client) contextWithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.defaultTimeout <= 0 {
		return ctx, func() {}
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.defaultTimeout)
}

// commonParams flattens the embedded CommonOptions into the param list.
func commonParams(c CommonOptions) query.Params {
	p := query.Params{}
	if len(c.Headers) > 0 {
		p.Set("headers", c.Headers)
	}
	setIntPtr(&p, "timeout", c.Timeout)
	setBoolPtr(&p, "js", c.JS)
	setIntPtr(&p, "js_timeout", c.JSTimeout)
	setIfNotEmpty(&p, "wait_for", c.WaitFor)
	setIfNotEmpty(&p, "proxy", c.Proxy)
	setIfNotEmpty(&p, "country", c.Country)
	setIfNotEmpty(&p, "custom_proxy", c.CustomProxy)
	setIfNotEmpty(&p, "device", c.Device)
	setBoolPtr(&p, "error_on_404", c.ErrorOn404)
	setBoolPtr(&p, "error_on_redirect", c.ErrorOnRedirect)
	setIfNotEmpty(&p, "js_script", c.JSScript)
	return p
}

func setIfNotEmpty(p *query.Params, key, value string) {
	if value != "" {
		p.Set(key, value)
	}
}

func setBoolPtr(p *query.Params, key string, value *bool) {
	if value != nil {
		p.Set(key, *value)
	}
}

func setIntPtr(p *query.Params, key string, value *int) {
	if value != nil {
		p.Set(key, *value)
	}
}

// unwrapJSONString returns the inner string when body is a JSON-encoded
// string (the API returns answers double-quoted by default); otherwise
// returns the body verbatim.
func unwrapJSONString(body []byte) string {
	if len(body) >= 2 && body[0] == '"' && body[len(body)-1] == '"' {
		var s string
		if json.Unmarshal(body, &s) == nil {
			return s
		}
	}
	return string(body)
}
