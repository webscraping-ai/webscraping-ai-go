package webscrapingai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/webscraping-ai/webscraping-ai-go/internal/query"
)

// do executes a GET request for path with the encoded params and
// returns the response body bytes plus the parsed Content-Type media
// type. Non-2xx responses are translated to typed *APIError variants;
// transport failures become *TimeoutError / *ConnectionError.
func (c *Client) do(ctx context.Context, path string, params query.Params) ([]byte, string, error) {
	// API key always travels in the query string per the OpenAPI
	// security scheme. Prepend so it appears first in URLs (cosmetic,
	// matches the other SDKs).
	full := query.Params{{Key: "api_key", Value: c.apiKey}}
	full = append(full, params...)

	endpoint := c.baseURL + path
	if encoded := query.EncodeToString(full); encoded != "" {
		endpoint += "?" + encoded
	}

	// Sanity-check the URL once assembled. The encoder produces a valid
	// query string, but a bad baseURL could still slip through.
	if _, err := url.Parse(endpoint); err != nil {
		return nil, "", &ConnectionError{Cause: err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", &ConnectionError{Cause: err}
	}
	req.Header.Set("Accept", "application/json, text/plain, text/html, */*")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", classifyTransportError(ctx, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", &ConnectionError{Cause: err}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", parseAPIError(resp.StatusCode, resp.Status, body)
	}

	return body, resp.Header.Get("Content-Type"), nil
}

// classifyTransportError translates a net/http transport error into a
// TimeoutError / ConnectionError. Context cancellation always maps to
// TimeoutError so users get a stable category regardless of which side
// triggered the abort.
func classifyTransportError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return &TimeoutError{Cause: err}
	}
	// net/url wraps the underlying error; unwrap for a clearer Cause.
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return &TimeoutError{Cause: urlErr.Err}
		}
		return &ConnectionError{Cause: urlErr.Err}
	}
	return &ConnectionError{Cause: err}
}

// parseAPIError builds the typed error returned to callers when the
// API responds with a non-2xx status.
func parseAPIError(status int, statusText string, body []byte) error {
	base := &APIError{
		HTTPStatus:   status,
		ResponseBody: string(body),
	}

	// The API's error envelope is JSON: {"message", "status_code",
	// "status_message", "body"}. If the body isn't JSON we fall back to
	// the raw body / status text.
	var payload struct {
		Message       string `json:"message"`
		StatusCode    int    `json:"status_code"`
		StatusMessage string `json:"status_message"`
		Body          string `json:"body"`
	}
	if len(body) > 0 && json.Unmarshal(body, &payload) == nil {
		base.Message = payload.Message
		base.StatusCode = payload.StatusCode
		base.StatusMessage = payload.StatusMessage
		base.Body = payload.Body
	}
	if base.Message == "" {
		base.Message = strings.TrimSpace(string(body))
	}
	if base.Message == "" {
		base.Message = statusText
	}

	return errorForStatus(base)
}

// decodeJSON unmarshals body into out. If the body isn't valid JSON the
// caller gets a ConnectionError wrapping the parse failure — there's no
// better category for "response was 2xx but unparseable".
func decodeJSON(body []byte, out any) error {
	if err := json.Unmarshal(body, out); err != nil {
		return &ConnectionError{Cause: err}
	}
	return nil
}
