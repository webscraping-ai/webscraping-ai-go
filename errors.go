package webscrapingai

import "fmt"

// Error is the marker interface implemented by every error this package
// returns. Use errors.As to branch:
//
//	var apiErr *webscrapingai.APIError
//	if errors.As(err, &apiErr) { ... }   // any non-2xx HTTP response
//
//	var rl *webscrapingai.RateLimitError
//	if errors.As(err, &rl) { ... }       // specifically HTTP 429
type Error interface {
	error
	webscrapingAI() // unexported marker
}

// APIError represents a non-2xx HTTP response from the API.
//
// The four "target-page" fields (StatusCode, StatusMessage, Body,
// ResponseBody) are populated when the API surfaced an upstream
// (target-page) failure as a 5xx — see README "Error handling".
type APIError struct {
	// HTTPStatus is the HTTP response status (e.g. 429).
	HTTPStatus int
	// Message is the human-readable error from the API's JSON body, or
	// the response body verbatim if it wasn't JSON, or the status text.
	Message string
	// StatusCode mirrors the "status_code" field the API includes when
	// the target page returned a non-2xx status of its own.
	StatusCode int
	// StatusMessage mirrors the "status_message" field — see StatusCode.
	StatusMessage string
	// Body is the target page body the API forwarded back when the page
	// returned an error and error_on_404 / error_on_redirect were set.
	Body string
	// ResponseBody is the raw HTTP response body, useful for debugging.
	ResponseBody string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("webscraping.ai: %s (HTTP %d)", e.Message, e.HTTPStatus)
	}
	return fmt.Sprintf("webscraping.ai: HTTP %d", e.HTTPStatus)
}

func (e *APIError) webscrapingAI() {}

// BadRequestError — HTTP 400. Usually a malformed URL or invalid param
// combination.
type BadRequestError struct{ *APIError }

// PaymentRequiredError — HTTP 402. Out of credits.
type PaymentRequiredError struct{ *APIError }

// AuthenticationError — HTTP 403. Wrong, missing, or revoked api_key.
type AuthenticationError struct{ *APIError }

// RateLimitError — HTTP 429. Too many concurrent requests for the plan.
type RateLimitError struct{ *APIError }

// ServerError — HTTP 500. Either an API-side problem, or the API
// forwarding a target-page failure (check Body / StatusCode).
type ServerError struct{ *APIError }

// GatewayTimeoutError — HTTP 504. Target page took too long to respond.
type GatewayTimeoutError struct{ *APIError }

// Unwrap returns the embedded *APIError so errors.As against *APIError
// works on the typed wrappers.
func (e *BadRequestError) Unwrap() error      { return e.APIError }
func (e *PaymentRequiredError) Unwrap() error { return e.APIError }
func (e *AuthenticationError) Unwrap() error  { return e.APIError }
func (e *RateLimitError) Unwrap() error       { return e.APIError }
func (e *ServerError) Unwrap() error          { return e.APIError }
func (e *GatewayTimeoutError) Unwrap() error  { return e.APIError }

// TimeoutError indicates the request was aborted because the context
// deadline / configured timeout elapsed before a response was received.
type TimeoutError struct {
	Cause error
}

func (e *TimeoutError) Error() string {
	if e.Cause != nil {
		return "webscraping.ai: request timed out: " + e.Cause.Error()
	}
	return "webscraping.ai: request timed out"
}

func (e *TimeoutError) Unwrap() error  { return e.Cause }
func (e *TimeoutError) webscrapingAI() {}

// ConnectionError indicates a transport-level failure (DNS, TLS,
// connection refused, etc.) before any HTTP response was received.
type ConnectionError struct {
	Cause error
}

func (e *ConnectionError) Error() string {
	if e.Cause != nil {
		return "webscraping.ai: connection failed: " + e.Cause.Error()
	}
	return "webscraping.ai: connection failed"
}

func (e *ConnectionError) Unwrap() error  { return e.Cause }
func (e *ConnectionError) webscrapingAI() {}

// errorForStatus wraps base in the most specific typed error available
// for the HTTP status, or returns &base if there isn't one.
func errorForStatus(base *APIError) error {
	switch base.HTTPStatus {
	case 400:
		return &BadRequestError{base}
	case 402:
		return &PaymentRequiredError{base}
	case 403:
		return &AuthenticationError{base}
	case 429:
		return &RateLimitError{base}
	case 500:
		return &ServerError{base}
	case 504:
		return &GatewayTimeoutError{base}
	default:
		return base
	}
}
