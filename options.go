package webscrapingai

// CommonOptions are options accepted by every endpoint. Each endpoint's
// own option struct embeds this.
//
// Pointer fields (*bool, *int) distinguish "not set" from the zero
// value — leave them nil to defer to the API's documented default.
type CommonOptions struct {
	// Headers are custom HTTP headers forwarded to the target page.
	Headers map[string]string
	// Timeout is the maximum target-page-retrieval time in milliseconds
	// (API default: 10000, max 30000).
	Timeout *int
	// JS toggles execution of on-page JavaScript via the headless
	// browser. API default: true.
	JS *bool
	// JSTimeout is the maximum JS rendering time in milliseconds after
	// page load (API default: 2000).
	JSTimeout *int
	// WaitFor is a CSS selector to wait for before returning content.
	WaitFor string
	// Proxy selects the proxy pool: "datacenter", "residential", or
	// "stealth". Empty string defers to the API's default.
	Proxy string
	// Country is the ISO country code for the proxy IP (e.g. "us").
	Country string
	// CustomProxy is a custom proxy URL ("http://user:pass@host:port").
	CustomProxy string
	// Device selects device emulation: "desktop", "mobile", "tablet".
	Device string
	// ErrorOn404 causes the API to return a 5xx for 404 target pages.
	ErrorOn404 *bool
	// ErrorOnRedirect causes the API to return a 5xx on redirects.
	ErrorOnRedirect *bool
	// JSScript is a custom JavaScript snippet to evaluate on the page.
	JSScript string
}

// HTMLOptions are the options for Client.HTML.
type HTMLOptions struct {
	CommonOptions
	// URL is the target page URL. Required.
	URL string
	// ReturnScriptResult returns the value of JSScript instead of HTML.
	ReturnScriptResult *bool
	// Format wraps the response: "json" returns {"html": "..."}, "text"
	// (default) returns the HTML body verbatim.
	Format string
}

// TextOptions are the options for Client.Text.
type TextOptions struct {
	CommonOptions
	// URL is the target page URL. Required.
	URL string
	// TextFormat selects the text serialisation: "plain" (default),
	// "xml", or "json".
	TextFormat string
	// ReturnLinks toggles inclusion of link metadata.
	ReturnLinks *bool
}

// SelectedOptions are the options for Client.Selected.
type SelectedOptions struct {
	CommonOptions
	// URL is the target page URL. Required.
	URL string
	// Selector is the CSS selector to extract. Required.
	Selector string
	// Format selects the response shape: "json" or "text" (default).
	Format string
}

// SelectedMultipleOptions are the options for Client.SelectedMultiple.
type SelectedMultipleOptions struct {
	CommonOptions
	// URL is the target page URL. Required.
	URL string
	// Selectors is the list of CSS selectors to extract. Required.
	Selectors []string
}

// QuestionOptions are the options for Client.Question.
type QuestionOptions struct {
	CommonOptions
	// URL is the target page URL. Required.
	URL string
	// Question is the natural-language question. Required.
	Question string
	// Format selects the response shape: "json" or "text" (default).
	Format string
}

// FieldsOptions are the options for Client.Fields.
type FieldsOptions struct {
	CommonOptions
	// URL is the target page URL. Required.
	URL string
	// Fields maps the field name to its natural-language description.
	// Required and must be non-empty.
	Fields map[string]string
}

// SerpOptions are the options for Client.Serp.
//
// /serp is query-shaped, not URL-shaped: the scraping options in
// CommonOptions do not apply, so this struct does not embed it.
type SerpOptions struct {
	// Q is the search query. Required; blank (whitespace-only) queries
	// are rejected client-side. Sent as given, without trimming.
	Q string
	// Engine is the search engine to query. Only "google" (the API
	// default) is supported today. Empty string defers to the default.
	Engine string
	// GL is the two-letter country code for geolocation of the search
	// (Google's gl parameter). API default: "us".
	GL string
	// HL is the two-letter language code for the results (Google's hl
	// parameter). API default: "en".
	HL string
	// Page is the results page number, starting at 1 (10 results per
	// page). API default: 1. Values below 1 are rejected client-side;
	// the server rejects a page above 100 with a 400 (not billed).
	Page *int
}
