package webscrapingai

// AccountInfo is the parsed response from Client.Account.
type AccountInfo struct {
	// Email is the account email.
	Email string `json:"email"`
	// RemainingAPICalls is the remaining credit count for the period.
	RemainingAPICalls int `json:"remaining_api_calls"`
	// ResumesAt is an ISO timestamp at which the credit quota resets.
	ResumesAt string `json:"resumes_at"`
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
