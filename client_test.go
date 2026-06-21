package webscrapingai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestServer returns a server whose handler captures the most recent
// request's URL.RawQuery and serves the response defined by handler.
func newTestServer(handler http.HandlerFunc) (*httptest.Server, *capturedRequest) {
	cap := &capturedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.Method = r.Method
		cap.Path = r.URL.Path
		cap.RawQuery = r.URL.RawQuery
		cap.UserAgent = r.Header.Get("User-Agent")
		handler(w, r)
	}))
	return srv, cap
}

type capturedRequest struct {
	Method    string
	Path      string
	RawQuery  string
	UserAgent string
}

func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c, err := NewClient(&Config{
		APIKey:  "test-key",
		BaseURL: srv.URL,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

// --- NewClient ---------------------------------------------------------

func TestNewClient_RequiresAPIKey(t *testing.T) {
	t.Setenv("WEBSCRAPING_AI_API_KEY", "")
	if _, err := NewClient(nil); err == nil {
		t.Fatal("expected error without API key")
	}
}

func TestNewClient_EnvFallback(t *testing.T) {
	t.Setenv("WEBSCRAPING_AI_API_KEY", "env-key")
	c, err := NewClient(nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.apiKey != "env-key" {
		t.Fatalf("apiKey = %q, want env-key", c.apiKey)
	}
}

func TestNewClient_ExplicitKeyWinsOverEnv(t *testing.T) {
	t.Setenv("WEBSCRAPING_AI_API_KEY", "env-key")
	c, err := NewClient(&Config{APIKey: "explicit"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.apiKey != "explicit" {
		t.Fatalf("apiKey = %q, want explicit", c.apiKey)
	}
}

func TestNewClient_BaseURLTrimsTrailingSlashes(t *testing.T) {
	c, err := NewClient(&Config{APIKey: "k", BaseURL: "https://example.com//"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.baseURL != "https://example.com" {
		t.Fatalf("baseURL = %q", c.baseURL)
	}
}

// --- Account ----------------------------------------------------------

func TestClient_Account(t *testing.T) {
	srv, cap := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"email":"vlad@example.com","remaining_api_calls":12345,"resets_at":1780358400,"remaining_concurrency":7}`))
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	info, err := c.Account(context.Background())
	if err != nil {
		t.Fatalf("Account: %v", err)
	}
	if info.Email != "vlad@example.com" || info.RemainingAPICalls != 12345 ||
		info.ResetsAt != 1780358400 || info.RemainingConcurrency != 7 {
		t.Fatalf("unexpected AccountInfo: %+v", info)
	}
	if cap.Path != "/account" {
		t.Fatalf("path = %q", cap.Path)
	}
	if cap.RawQuery != "api_key=test-key" {
		t.Fatalf("RawQuery = %q", cap.RawQuery)
	}
	if !strings.HasPrefix(cap.UserAgent, "webscraping-ai-go/") {
		t.Fatalf("User-Agent = %q", cap.UserAgent)
	}
}

// --- HTML -------------------------------------------------------------

func TestClient_HTML_BasicURL(t *testing.T) {
	srv, cap := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>ok</html>"))
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	out, err := c.HTML(context.Background(), &HTMLOptions{URL: "https://example.com"})
	if err != nil {
		t.Fatalf("HTML: %v", err)
	}
	if out != "<html>ok</html>" {
		t.Fatalf("got %q", out)
	}
	if cap.Path != "/html" {
		t.Fatalf("path = %q", cap.Path)
	}
	if !strings.Contains(cap.RawQuery, "url=https%3A%2F%2Fexample.com") {
		t.Fatalf("RawQuery missing url: %q", cap.RawQuery)
	}
}

func TestClient_HTML_PropagatesCommonOptions(t *testing.T) {
	srv, cap := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	_, err := c.HTML(context.Background(), &HTMLOptions{
		URL: "https://example.com",
		CommonOptions: CommonOptions{
			Headers:    map[string]string{"X-Custom": "yes"},
			Timeout:    intPtr(15000),
			JS:         boolPtr(false),
			Proxy:      "residential",
			Country:    "us",
			ErrorOn404: boolPtr(true),
		},
	})
	if err != nil {
		t.Fatalf("HTML: %v", err)
	}
	q := cap.RawQuery
	for _, want := range []string{
		"headers%5BX-Custom%5D=yes",
		"timeout=15000",
		"js=false",
		"proxy=residential",
		"country=us",
		"error_on_404=true",
	} {
		if !strings.Contains(q, want) {
			t.Errorf("query missing %q in %q", want, q)
		}
	}
}

// --- Text -------------------------------------------------------------

func TestClient_Text(t *testing.T) {
	srv, cap := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	out, err := c.Text(context.Background(), &TextOptions{
		URL:         "https://example.com",
		TextFormat:  "json",
		ReturnLinks: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("Text: %v", err)
	}
	if out != "hello world" {
		t.Fatalf("got %q", out)
	}
	if !strings.Contains(cap.RawQuery, "text_format=json") || !strings.Contains(cap.RawQuery, "return_links=true") {
		t.Fatalf("RawQuery = %q", cap.RawQuery)
	}
}

// --- Selected ---------------------------------------------------------

func TestClient_Selected(t *testing.T) {
	srv, cap := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<h1>Title</h1>"))
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	out, err := c.Selected(context.Background(), &SelectedOptions{
		URL:      "https://example.com",
		Selector: "h1",
	})
	if err != nil {
		t.Fatalf("Selected: %v", err)
	}
	if out != "<h1>Title</h1>" {
		t.Fatalf("got %q", out)
	}
	if !strings.Contains(cap.RawQuery, "selector=h1") {
		t.Fatalf("RawQuery = %q", cap.RawQuery)
	}
}

func TestClient_Selected_OptionalSelector(t *testing.T) {
	srv, cap := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>ok</html>"))
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	out, err := c.Selected(context.Background(), &SelectedOptions{URL: "https://example.com"})
	if err != nil {
		t.Fatalf("Selected: %v", err)
	}
	if out != "<html>ok</html>" {
		t.Fatalf("got %q", out)
	}
	// With no selector, the param must be omitted entirely.
	if strings.Contains(cap.RawQuery, "selector=") {
		t.Fatalf("selector should be omitted: %q", cap.RawQuery)
	}
}

// --- SelectedMultiple -------------------------------------------------

func TestClient_SelectedMultiple(t *testing.T) {
	srv, cap := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[["Title","First para"]]`))
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	out, err := c.SelectedMultiple(context.Background(), &SelectedMultipleOptions{
		URL:       "https://example.com",
		Selectors: []string{"h1", "p"},
	})
	if err != nil {
		t.Fatalf("SelectedMultiple: %v", err)
	}
	if len(out) != 1 || len(out[0]) != 2 || out[0][0] != "Title" {
		t.Fatalf("unexpected shape: %+v", out)
	}
	// Selectors must repeat the same key without brackets.
	if !strings.Contains(cap.RawQuery, "selectors=h1&selectors=p") {
		t.Fatalf("RawQuery = %q", cap.RawQuery)
	}
}

func TestClient_SelectedMultiple_OptionalSelectors(t *testing.T) {
	srv, cap := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[["whole page"]]`))
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	out, err := c.SelectedMultiple(context.Background(), &SelectedMultipleOptions{URL: "https://example.com"})
	if err != nil {
		t.Fatalf("SelectedMultiple: %v", err)
	}
	if len(out) != 1 || len(out[0]) != 1 || out[0][0] != "whole page" {
		t.Fatalf("unexpected shape: %+v", out)
	}
	// With no selectors, the param must be omitted entirely.
	if strings.Contains(cap.RawQuery, "selectors=") {
		t.Fatalf("selectors should be omitted: %q", cap.RawQuery)
	}
}

// --- Question ---------------------------------------------------------

func TestClient_Question_UnwrapsJSONString(t *testing.T) {
	srv, _ := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`"This page is about an example domain."`))
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	out, err := c.Question(context.Background(), &QuestionOptions{
		URL:      "https://example.com",
		Question: "What is this page?",
	})
	if err != nil {
		t.Fatalf("Question: %v", err)
	}
	if out != "This page is about an example domain." {
		t.Fatalf("got %q", out)
	}
}

// --- Fields -----------------------------------------------------------

func TestClient_Fields(t *testing.T) {
	srv, cap := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"title":"Example Domain","description":null}}`))
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	out, err := c.Fields(context.Background(), &FieldsOptions{
		URL:    "https://example.com",
		Fields: map[string]string{"title": "Page title", "description": "Short description"},
	})
	if err != nil {
		t.Fatalf("Fields: %v", err)
	}
	if out.Result["title"] != "Example Domain" {
		t.Fatalf("title = %q", out.Result["title"])
	}
	// Fields use deepObject brackets.
	if !strings.Contains(cap.RawQuery, "fields%5Btitle%5D=Page%20title") {
		t.Fatalf("RawQuery missing deepObject fields: %q", cap.RawQuery)
	}
}

// --- Required-arg validation -----------------------------------------

func TestClient_MissingURL(t *testing.T) {
	c, _ := NewClient(&Config{APIKey: "k"})
	if _, err := c.HTML(context.Background(), &HTMLOptions{}); err == nil {
		t.Error("HTML should require URL")
	}
	if _, err := c.Text(context.Background(), &TextOptions{}); err == nil {
		t.Error("Text should require URL")
	}
	if _, err := c.Selected(context.Background(), &SelectedOptions{}); err == nil {
		t.Error("Selected should require URL")
	}
	if _, err := c.SelectedMultiple(context.Background(), &SelectedMultipleOptions{}); err == nil {
		t.Error("SelectedMultiple should require URL")
	}
	if _, err := c.Question(context.Background(), &QuestionOptions{URL: "x"}); err == nil {
		t.Error("Question should require Question")
	}
	if _, err := c.Fields(context.Background(), &FieldsOptions{URL: "x"}); err == nil {
		t.Error("Fields should require Fields")
	}
}

// --- Error mapping ----------------------------------------------------

func TestClient_ErrorMapping(t *testing.T) {
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
	for _, tc := range cases {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			srv, _ := newTestServer(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = fmt.Fprintf(w, `{"message":"boom","status_code":%d,"status_message":"bad upstream","body":"target body"}`, tc.status)
			})
			defer srv.Close()
			c := newTestClient(t, srv)
			_, err := c.HTML(context.Background(), &HTMLOptions{URL: "x"})
			if err == nil {
				t.Fatalf("expected error")
			}
			if !tc.check(err) {
				t.Fatalf("typed-check failed: %v", err)
			}
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("should unwrap to *APIError")
			}
			if apiErr.HTTPStatus != tc.status {
				t.Fatalf("HTTPStatus = %d, want %d", apiErr.HTTPStatus, tc.status)
			}
			if apiErr.Message != "boom" || apiErr.StatusCode != tc.status || apiErr.Body != "target body" {
				t.Fatalf("APIError envelope not parsed: %+v", apiErr)
			}
		})
	}
}

func TestClient_ErrorBodyNotJSON(t *testing.T) {
	srv, _ := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		_, _ = w.Write([]byte("text-only body"))
	})
	defer srv.Close()
	c := newTestClient(t, srv)
	_, err := c.HTML(context.Background(), &HTMLOptions{URL: "x"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %v", err)
	}
	if apiErr.Message != "text-only body" {
		t.Fatalf("Message = %q", apiErr.Message)
	}
}

// --- Transport errors -------------------------------------------------

func TestClient_TimeoutOnContextDeadline(t *testing.T) {
	srv, _ := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	})
	defer srv.Close()
	c := newTestClient(t, srv)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := c.HTML(ctx, &HTMLOptions{URL: "x"})
	var te *TimeoutError
	if !errors.As(err, &te) {
		t.Fatalf("expected *TimeoutError, got %v", err)
	}
}

func TestClient_ConnectionErrorOnRefused(t *testing.T) {
	c, _ := NewClient(&Config{
		APIKey:  "k",
		BaseURL: "http://127.0.0.1:1", // unlikely to be listening
	})
	_, err := c.HTML(context.Background(), &HTMLOptions{URL: "x"})
	var ce *ConnectionError
	if !errors.As(err, &ce) {
		t.Fatalf("expected *ConnectionError, got %v", err)
	}
}

func TestClient_DefaultTimeoutApplied(t *testing.T) {
	// 200ms server pause; client has 30ms default timeout, no
	// context deadline.
	srv, _ := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	})
	defer srv.Close()
	c, _ := NewClient(&Config{
		APIKey:  "k",
		BaseURL: srv.URL,
		Timeout: 30 * time.Millisecond,
	})
	_, err := c.HTML(context.Background(), &HTMLOptions{URL: "x"})
	var te *TimeoutError
	if !errors.As(err, &te) {
		t.Fatalf("expected *TimeoutError, got %v", err)
	}
}

// --- Marker interface -------------------------------------------------

func TestClient_ErrorsImplementMarker(t *testing.T) {
	srv, _ := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"message":"rl"}`))
	})
	defer srv.Close()
	c := newTestClient(t, srv)
	_, err := c.HTML(context.Background(), &HTMLOptions{URL: "x"})
	var marker Error
	if !errors.As(err, &marker) {
		t.Fatalf("expected Error marker, got %v", err)
	}
}
