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
	// (Google's gl parameter). Defaults to the country of Location,
	// Lat/Lon or UULE, otherwise "us".
	GL string
	// HL is the two-letter language code for the results (Google's hl
	// parameter). API default: "en".
	HL string
	// Page is the results page number, starting at 1 (10 results per
	// page). API default: 1. Values below 1 are rejected client-side;
	// the server rejects a page above 100 with a 400 (not billed).
	Page *int
	// Pages is the number of consecutive results pages to fetch in one
	// call, starting at Page (1 to 5, API default 1). Duplicates across
	// pages are removed, Position counts through the combined list and
	// each result gets a Page. Billed per results page returned. Values
	// below 1 are rejected client-side; the server rejects more than 5
	// (or Page+Pages-1 above 100) with a 400 (not billed).
	Pages *int
	// Location is where the search runs from: a location name such as
	// "Austin,Texas,United States" or "Austin, TX", or an id from
	// Client.SerpLocations. Large cities also match local names, e.g.
	// "Москва". An unknown name returns a 400 (not billed). Cannot be
	// combined with UULE, Lat or Lon. Sent as given.
	Location string
	// UULE is a Google-encoded location ("w+CAIQICI…" or "a+…"). Cannot
	// be combined with Location, Lat or Lon.
	UULE string
	// Lat is the latitude of the search origin in decimal degrees (-90
	// to 90). Requires Lon; cannot be combined with Location or UULE.
	// Must be finite (NaN and ±Inf are rejected client-side). Sent as a
	// plain decimal with at most 7 fractional digits.
	Lat *float64
	// Lon is the longitude of the search origin in decimal degrees (-180
	// to 180). Requires Lat; cannot be combined with Location or UULE.
	// Must be finite. Sent like Lat.
	Lon *float64
	// Radius is the radius in meters around Location or Lat/Lon (1 to
	// 1000). Values below 1 are rejected client-side.
	Radius *int
}

// SerpLocationsOptions are the options for Client.SerpLocations. A nil
// *SerpLocationsOptions is the same as an empty one.
type SerpLocationsOptions struct {
	// Q is part of a location name, e.g. "austin". Matches English
	// names. Empty returns the most populous locations. Sent as given.
	Q string
	// Limit is the maximum number of locations to return (1 to 10, API
	// default 10). Values below 1 are rejected client-side; the server
	// rejects more than 10 with a 400.
	Limit *int
}

// DataOptions are the options for Client.Data.
//
// /data returns structured JSON for any public web page. This struct
// does not embed CommonOptions; for pages without a dedicated parser,
// the scraping options the API accepts (timeout, js_timeout, wait_for,
// device) can be passed through Params. There are no js or proxy options:
// /data renders pages with JavaScript where the site needs it and always
// uses proxy=auto.
type DataOptions struct {
	// URL is the page's normal URL, e.g.
	// "https://books.toscrape.com/catalogue/a-light-in-the-attic_1000/index.html".
	// Required; a blank (whitespace-only) URL is rejected client-side.
	// Sent as given; the client does not check the site or page type.
	URL string
	// Country is the two-letter country code of the proxy used to fetch
	// the page, "us" by default. The server rejects unknown codes with a
	// 400.
	Country string
	// Transcript: video pages on sites with a dedicated parser only.
	// Also fetch the video's transcript into data.transcript. It's null
	// when no matching captions are available. If the transcript fetch
	// itself fails, the whole request fails with a 500 and is not charged.
	Transcript *bool
	// TranscriptLanguage: with Transcript set to true, the caption
	// language to pick, e.g. "en" or "de". Without it, English is
	// preferred, then the first available track. If the video has no
	// captions in that language, data.transcript is null.
	TranscriptLanguage string
	// Params are extra query parameters sent as-is, for options this
	// version of the client doesn't know about yet. Keys are sent in
	// sorted order after the typed fields. Keys naming
	// api_key or url are rejected, and so are country, transcript and
	// transcript_language (use the named fields), whether or not those
	// fields are set.
	Params map[string]string
}
