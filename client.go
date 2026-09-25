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
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/webscraping-ai/webscraping-ai-go/v4/internal/query"
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
	// passes a context without a deadline. Zero (the default value)
	// selects DefaultTimeout (60s). A negative value disables the
	// implicit timeout entirely (the caller manages deadlines via
	// context). A positive value sets the timeout directly.
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
	// Validate up front, before any api_key is attached to a URL, so a
	// malformed BaseURL fails here instead of in an error that embeds
	// the full request URL.
	if u, err := url.Parse(baseURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, errors.New("webscrapingai: BaseURL must be an absolute http(s) URL")
	}

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
//
// opts.Selector is optional; when empty the API returns the whole-page
// HTML (subject to the other options).
func (c *Client) Selected(ctx context.Context, opts *SelectedOptions) (string, error) {
	if opts == nil || opts.URL == "" {
		return "", errors.New("webscrapingai: Selected requires opts.URL")
	}
	ctx, cancel := c.contextWithTimeout(ctx)
	defer cancel()
	params := commonParams(opts.CommonOptions)
	params.Set("url", opts.URL)
	setIfNotEmpty(&params, "selector", opts.Selector)
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
//
// opts.Selectors is optional; when empty the API returns the whole-page
// HTML (subject to the other options).
func (c *Client) SelectedMultiple(ctx context.Context, opts *SelectedMultipleOptions) (SelectedMultipleResult, error) {
	if opts == nil || opts.URL == "" {
		return nil, errors.New("webscrapingai: SelectedMultiple requires opts.URL")
	}
	ctx, cancel := c.contextWithTimeout(ctx)
	defer cancel()
	params := commonParams(opts.CommonOptions)
	params.Set("url", opts.URL)
	if len(opts.Selectors) > 0 {
		params.Set("selectors", opts.Selectors)
	}
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

// Serp calls GET /serp and returns the parsed search engine results for
// opts.Q. Flat 15 credits per search; failed searches are not charged.
//
// opts.Q must not be blank and opts.Page, when set, must be >= 1; both
// are checked before any request (the server also rejects an invalid
// page with a 400, not billed; checking client-side saves the round
// trip). Pages are 1-100: the server rejects a Page above 100 with a
// 400. Q is sent exactly as given.
//
// Unlike the page endpoints /serp is query-shaped: none of the scraping
// options (JS, proxy, country, …) apply, so SerpOptions does not embed
// CommonOptions.
func (c *Client) Serp(ctx context.Context, opts *SerpOptions) (*SerpResult, error) {
	if opts == nil || strings.TrimSpace(opts.Q) == "" {
		return nil, errors.New("webscrapingai: Serp requires a non-blank opts.Q")
	}
	if opts.Page != nil && *opts.Page < 1 {
		return nil, errors.New("webscrapingai: Serp opts.Page must be >= 1")
	}
	ctx, cancel := c.contextWithTimeout(ctx)
	defer cancel()
	params := query.Params{}
	params.Set("q", opts.Q)
	setIfNotEmpty(&params, "engine", opts.Engine)
	setIfNotEmpty(&params, "gl", opts.GL)
	setIfNotEmpty(&params, "hl", opts.HL)
	setIntPtr(&params, "page", opts.Page)
	body, _, err := c.do(ctx, "/serp", params)
	if err != nil {
		return nil, err
	}
	var out SerpResult
	if err := decodeJSON(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Data calls GET /data and returns structured JSON for a page on a
// supported site (e.g. a YouTube video, TikTok profile, X post, LinkedIn
// company, Instagram reel or Reddit thread). 15 credits per
// request, including parse_failed and not_found results; failed fetches
// are not charged.
//
// opts.URL must not be blank; nothing else about it is checked
// client-side. More sites are added on the server. An unsupported URL
// or page type returns a 400 that is not charged (*BadRequestError). Its
// message lists what is supported.
//
// Like /serp, /data ignores the scraping options, so DataOptions does
// not embed CommonOptions. Use opts.Params for provider-specific
// parameters this client doesn't have a field for.
func (c *Client) Data(ctx context.Context, opts *DataOptions) (*DataResult, error) {
	if opts == nil || strings.TrimSpace(opts.URL) == "" {
		return nil, errors.New("webscrapingai: Data requires a non-blank opts.URL")
	}
	params := query.Params{}
	params.Set("url", opts.URL)
	setIfNotEmpty(&params, "country", opts.Country)
	setBoolPtr(&params, "transcript", opts.Transcript)
	setIfNotEmpty(&params, "transcript_language", opts.TranscriptLanguage)
	if err := addExtraParams(&params, opts.Params); err != nil {
		return nil, err
	}
	ctx, cancel := c.contextWithTimeout(ctx)
	defer cancel()
	body, _, err := c.do(ctx, "/data", params)
	if err != nil {
		return nil, err
	}
	var out DataResult
	if err := decodeJSON(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// dataTypedParams maps each DataOptions typed query param to its field.
var dataTypedParams = map[string]string{
	"country":             "Country",
	"transcript":          "Transcript",
	"transcript_language": "TranscriptLanguage",
}

// addExtraParams appends extra (sorted by key) to p. It refuses keys that
// would override api_key or url, and keys naming a typed option.
// Keys are compared on their name before any "[" and case-insensitively,
// so "API_KEY" or "url[]" cannot smuggle a second value past the check.
func addExtraParams(p *query.Params, extra map[string]string) error {
	if len(extra) == 0 {
		return nil
	}
	keys := make([]string, 0, len(extra))
	for k := range extra {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		base := k
		if i := strings.IndexByte(base, '['); i >= 0 {
			base = base[:i]
		}
		base = strings.TrimSpace(base)
		if base == "" {
			return errors.New("webscrapingai: extra param keys must not be blank")
		}
		if strings.EqualFold(base, "api_key") || strings.EqualFold(base, "url") {
			return fmt.Errorf("webscrapingai: extra param %q is not allowed (use Config.APIKey / opts.URL)", k)
		}
		if field, ok := dataTypedParams[strings.ToLower(base)]; ok {
			return fmt.Errorf("webscrapingai: extra param %q is not allowed (use DataOptions.%s)", k, field)
		}
		p.Set(k, extra[k])
	}
	return nil
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
