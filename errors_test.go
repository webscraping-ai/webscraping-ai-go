package webscrapingai

import (
	"errors"
	"testing"
)

func TestAPIError_ErrorIncludesStatus(t *testing.T) {
	e := &APIError{HTTPStatus: 429, Message: "Too many requests"}
	want := "webscraping.ai: Too many requests (HTTP 429)"
	if got := e.Error(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAPIError_ErrorWithoutMessage(t *testing.T) {
	e := &APIError{HTTPStatus: 500}
	want := "webscraping.ai: HTTP 500"
	if got := e.Error(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestErrorForStatus_KnownStatuses(t *testing.T) {
	cases := []struct {
		status int
		check  func(error) bool
	}{
		{400, func(e error) bool { var t *BadRequestError; return errors.As(e, &t) }},
		{402, func(e error) bool { var t *PaymentRequiredError; return errors.As(e, &t) }},
		{403, func(e error) bool { var t *AuthenticationError; return errors.As(e, &t) }},
		{429, func(e error) bool { var t *RateLimitError; return errors.As(e, &t) }},
		{500, func(e error) bool { var t *ServerError; return errors.As(e, &t) }},
		{504, func(e error) bool { var t *GatewayTimeoutError; return errors.As(e, &t) }},
	}
	for _, c := range cases {
		err := errorForStatus(&APIError{HTTPStatus: c.status})
		if !c.check(err) {
			t.Errorf("status %d: typed-check failed", c.status)
		}
	}
}

func TestErrorForStatus_UnwrapsToAPIError(t *testing.T) {
	err := errorForStatus(&APIError{HTTPStatus: 429, Message: "rate limited"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("errors.As should unwrap to *APIError")
	}
	if apiErr.HTTPStatus != 429 {
		t.Fatalf("HTTPStatus = %d, want 429", apiErr.HTTPStatus)
	}
}

func TestErrorForStatus_UnknownReturnsAPIError(t *testing.T) {
	err := errorForStatus(&APIError{HTTPStatus: 418, Message: "teapot"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("should still be *APIError")
	}
	var br *BadRequestError
	if errors.As(err, &br) {
		t.Fatal("418 should not match *BadRequestError")
	}
}

func TestTimeoutError_UnwrapsCause(t *testing.T) {
	cause := errors.New("ctx deadline")
	e := &TimeoutError{Cause: cause}
	if !errors.Is(e, cause) {
		t.Fatal("errors.Is should match the wrapped cause")
	}
	if e.Error() != "webscraping.ai: request timed out: ctx deadline" {
		t.Fatalf("unexpected message: %q", e.Error())
	}
}

func TestConnectionError_UnwrapsCause(t *testing.T) {
	cause := errors.New("no route to host")
	e := &ConnectionError{Cause: cause}
	if !errors.Is(e, cause) {
		t.Fatal("errors.Is should match the wrapped cause")
	}
}

func TestErrors_AllSatisfyMarker(t *testing.T) {
	errs := []error{
		&APIError{HTTPStatus: 500},
		&BadRequestError{&APIError{HTTPStatus: 400}},
		&PaymentRequiredError{&APIError{HTTPStatus: 402}},
		&AuthenticationError{&APIError{HTTPStatus: 403}},
		&RateLimitError{&APIError{HTTPStatus: 429}},
		&ServerError{&APIError{HTTPStatus: 500}},
		&GatewayTimeoutError{&APIError{HTTPStatus: 504}},
		&TimeoutError{},
		&ConnectionError{},
	}
	for _, e := range errs {
		var marker Error
		if !errors.As(e, &marker) {
			t.Errorf("%T should satisfy Error interface", e)
		}
	}
}
