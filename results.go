package webscrapingai

import "encoding/json"

// AccountInfo is the parsed response from Client.Account.
type AccountInfo struct {
	// Email is the account email.
	Email string `json:"email"`
	// RemainingAPICalls is the remaining credit count for the period.
	RemainingAPICalls int `json:"remaining_api_calls"`
	// ResetsAt is the UNIX timestamp of the next billing cycle start, at
	// which the remaining API calls quota resets.
	ResetsAt int `json:"resets_at"`
	// RemainingConcurrency is the remaining number of concurrent requests
	// allowed for the account.
	RemainingConcurrency int `json:"remaining_concurrency"`
}

// FieldsResult is the parsed response from Client.Fields.
//
// The API currently wraps the extracted values under a "result" key, so
// callers reach the data via .Result["title"] / .Result["price"] / etc.
// This shape is upstream drift from the OpenAPI spec — see README.
type FieldsResult struct {
	Result map[string]string `json:"result"`
}

// SelectedMultipleResult is the parsed response from
// Client.SelectedMultiple.
//
// The API currently returns [][]string (one outer wrapper containing
// the concatenated matches for all selectors), not the flat []string
// the spec implies. The client passes the shape through verbatim — see
// README.
type SelectedMultipleResult [][]string

// SerpResult is the parsed response from Client.Serp. Field names follow
// the common SERP API naming (SerpApi family).
type SerpResult struct {
	// SearchParameters are the normalized parameters the search ran with.
	SearchParameters SerpSearchParameters `json:"search_parameters"`
	// SearchInformation is what the engine reported about the search.
	SearchInformation SerpSearchInformation `json:"search_information"`
	// OrganicResults are the organic (non-ad) results, in rank order.
	OrganicResults []SerpOrganicResult `json:"organic_results"`
	// RelatedSearches are the engine's "Related searches" suggestions.
	// Nil when the page shows none.
	RelatedSearches []SerpRelatedSearch `json:"related_searches,omitempty"`
	// Pagination describes the current and next results page.
	Pagination SerpPagination `json:"pagination"`
}

// SerpSearchParameters mirrors SerpResult.search_parameters.
type SerpSearchParameters struct {
	Engine string `json:"engine"`
	Q      string `json:"q"`
	GL     string `json:"gl"`
	HL     string `json:"hl"`
	Page   int    `json:"page"`
}

// SerpSearchInformation mirrors SerpResult.search_information.
type SerpSearchInformation struct {
	// QueryDisplayed is the query the results are for. Equals Q unless
	// the engine applied a spelling fix.
	QueryDisplayed string `json:"query_displayed"`
	// OrganicResultsState is one of "Results for exact spelling",
	// "Empty showing fixed spelling results" (see ShowingResultsFor), or
	// "Fully empty" (no organic results; still a successful, billed
	// search).
	OrganicResultsState string `json:"organic_results_state"`
	// ShowingResultsFor is the auto-corrected query, set only when the
	// engine applied a spelling fix.
	ShowingResultsFor *string `json:"showing_results_for,omitempty"`
	// TotalResults is the engine's estimated total result count, set
	// only when the upstream page reports it.
	TotalResults *int64 `json:"total_results,omitempty"`
}

// SerpOrganicResult is one entry of SerpResult.OrganicResults.
type SerpOrganicResult struct {
	// Position is the 1-based rank within this page; it restarts at 1 on
	// every page. Compute (page-1)*10 + Position for an absolute rank.
	Position int    `json:"position"`
	Title    string `json:"title"`
	Link     string `json:"link"`
	// Domain is the hostname of Link without a leading "www.".
	Domain string `json:"domain"`
	// DisplayedLink is the breadcrumb-style URL shown under the title
	// (falls back to Domain when none is shown).
	DisplayedLink string `json:"displayed_link"`
	// Snippet is the result description, when one is shown.
	Snippet *string `json:"snippet,omitempty"`
	// Date is the date shown next to the snippet, verbatim (absolute or
	// relative), when present.
	Date *string `json:"date,omitempty"`
}

// SerpRelatedSearch is one entry of SerpResult.RelatedSearches.
type SerpRelatedSearch struct {
	Query string `json:"query"`
}

// SerpPagination mirrors SerpResult.pagination.
type SerpPagination struct {
	Current int `json:"current"`
	// Next is the next page number; nil when no further page is offered.
	Next *int `json:"next,omitempty"`
}

// DataResult is the parsed response from Client.Data.
//
// Provider, Type and ParseStatus are plain strings, not enums: new sites,
// page types and statuses are added on the server over time.
type DataResult struct {
	// RequestParameters echoes the URL and how it was classified.
	RequestParameters DataRequestParameters `json:"request_parameters"`
	// ParseStatus is "ok" when the page was parsed, "parse_failed" when
	// it was fetched but couldn't be parsed (Data may be nil or partial),
	// or "not_found" when the page doesn't exist. All are successful,
	// charged requests.
	ParseStatus string `json:"parse_status"`
	// Data is the page's fields as raw JSON; its shape depends on
	// Provider and Type. Decode it with json.Unmarshal into a map or your
	// own struct. Nil when the API returned null.
	Data json.RawMessage `json:"data"`
}

// UnmarshalJSON decodes a DataResult, turning a JSON null "data" into a
// nil Data so callers can test out.Data == nil however they decoded it.
func (r *DataResult) UnmarshalJSON(b []byte) error {
	type plain DataResult
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	if string(p.Data) == "null" {
		p.Data = nil
	}
	*r = DataResult(p)
	return nil
}

// DataRequestParameters mirrors DataResult.request_parameters.
type DataRequestParameters struct {
	URL string `json:"url"`
	// Provider is the detected site, e.g. "youtube", "tiktok", "twitter",
	// "linkedin", "instagram" or "reddit". An open set.
	Provider string `json:"provider"`
	// Type is the detected page kind, e.g. "video", "channel", "profile",
	// "post", "company" or "job". An open set.
	Type string `json:"type"`
}
