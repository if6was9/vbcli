package vestaboard

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClientMissingAPIKey(t *testing.T) {
	t.Parallel()

	if _, err := NewClient(""); err == nil {
		t.Fatal("expected missing key error")
	}
}

func TestSendText(t *testing.T) {
	t.Parallel()

	var gotHeader string
	var gotBody map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get(headerName)
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		apiKey:     "abc123",
		httpClient: server.Client(),
	}

	if err := client.SendText(context.Background(), "Hello World"); err != nil {
		t.Fatalf("send text: %v", err)
	}
	if gotHeader != "abc123" {
		t.Fatalf("header = %q, want %q", gotHeader, "abc123")
	}
	if gotBody["text"] != "Hello World" {
		t.Fatalf("text = %q", gotBody["text"])
	}
}

func TestSendCharacters(t *testing.T) {
	t.Parallel()

	var gotBody map[string][][]int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		apiKey:     "abc123",
		httpClient: server.Client(),
	}

	payload := [][]int{{68, 69}, {70, 71}}
	if err := client.SendCharacters(context.Background(), payload); err != nil {
		t.Fatalf("send characters: %v", err)
	}
	if len(gotBody["characters"]) != 2 || gotBody["characters"][1][1] != 71 {
		t.Fatalf("unexpected characters payload: %#v", gotBody["characters"])
	}
}

func TestSendTextConflictIsNotError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"message":"already current"}`))
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		apiKey:     "abc123",
		httpClient: server.Client(),
	}

	if err := client.SendText(context.Background(), "Hello World"); err != nil {
		t.Fatalf("expected 409 to be non-error, got: %v", err)
	}
}

func TestVerboseLogsRequestURL(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var logs bytes.Buffer
	client, err := NewClient("abc123", WithVerboseLogging(true, &logs))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	client.baseURL = server.URL
	client.httpClient = server.Client()

	if err := client.SendText(context.Background(), "Hello World"); err != nil {
		t.Fatalf("send text: %v", err)
	}

	if !strings.Contains(logs.String(), "request URL: "+server.URL+"/") {
		t.Fatalf("expected request URL in logs, got %q", logs.String())
	}
	if !strings.Contains(logs.String(), "request payload:\n{\n  \"text\": \"Hello World\"\n}") {
		t.Fatalf("expected formatted request payload in logs, got %q", logs.String())
	}
	if !strings.Contains(logs.String(), "response status: 200") {
		t.Fatalf("expected response status in logs, got %q", logs.String())
	}
	if !strings.Contains(logs.String(), "response payload:\n<empty>") {
		t.Fatalf("expected empty response payload marker in logs, got %q", logs.String())
	}
}

func TestFormatMessage(t *testing.T) {
	t.Parallel()

	type style struct {
		Height int `json:"height"`
		Width  int `json:"width"`
	}
	type componentStyle struct {
		Align   string `json:"align"`
		Justify string `json:"justify"`
	}
	type component struct {
		Template string         `json:"template"`
		Style    componentStyle `json:"style"`
	}
	type request struct {
		Components []component `json:"components"`
		Style      *style      `json:"style,omitempty"`
	}

	var gotBody request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"characters":[[1,2,3],[4,5,6]]}`))
	}))
	defer server.Close()

	client, err := NewClient("abc123")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	client.vbmlURL = server.URL
	client.httpClient = server.Client()

	characters, err := client.FormatMessage(context.Background(), "hello", "flagship", "center", "center")
	if err != nil {
		t.Fatalf("format message: %v", err)
	}
	if len(gotBody.Components) != 1 || gotBody.Components[0].Template != "hello" {
		t.Fatalf("unexpected components payload: %#v", gotBody.Components)
	}
	if gotBody.Components[0].Style.Align != "center" || gotBody.Components[0].Style.Justify != "center" {
		t.Fatalf("unexpected component style: %#v", gotBody.Components[0].Style)
	}
	if gotBody.Style != nil {
		t.Fatalf("did not expect style for flagship model, got %#v", gotBody.Style)
	}
	if len(characters) != 2 || characters[1][2] != 6 {
		t.Fatalf("unexpected characters: %#v", characters)
	}
}

func TestFormatMessageVerboseLogsURL(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"characters":[[65]]}`))
	}))
	defer server.Close()

	var logs bytes.Buffer
	client, err := NewClient("abc123", WithVerboseLogging(true, &logs))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	client.vbmlURL = server.URL
	client.httpClient = server.Client()

	if _, err := client.FormatMessage(context.Background(), "hello", "flagship", "center", "center"); err != nil {
		t.Fatalf("format message: %v", err)
	}
	if !strings.Contains(logs.String(), "request URL: "+server.URL+"/compose") {
		t.Fatalf("expected VBML URL in logs, got %q", logs.String())
	}
	if !strings.Contains(logs.String(), "response status: 200") {
		t.Fatalf("expected VBML status in logs, got %q", logs.String())
	}
	if !strings.Contains(logs.String(), "\"characters\": [") && !strings.Contains(logs.String(), "[\n  [\n    65\n  ]\n]") {
		t.Fatalf("expected VBML response payload in logs, got %q", logs.String())
	}
}

func TestFormatMessageArrayResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[[11,12],[13,14]]`))
	}))
	defer server.Close()

	client, err := NewClient("abc123")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	client.vbmlURL = server.URL
	client.httpClient = server.Client()

	characters, err := client.FormatMessage(context.Background(), "hello", "flagship", "center", "center")
	if err != nil {
		t.Fatalf("format message: %v", err)
	}
	if len(characters) != 2 || characters[1][1] != 14 {
		t.Fatalf("unexpected characters: %#v", characters)
	}
}

func TestFormatMessageNoteIncludesStyle(t *testing.T) {
	t.Parallel()

	type style struct {
		Height int `json:"height"`
		Width  int `json:"width"`
	}
	type componentStyle struct {
		Align   string `json:"align"`
		Justify string `json:"justify"`
	}
	type component struct {
		Template string         `json:"template"`
		Style    componentStyle `json:"style"`
	}
	type request struct {
		Components []component `json:"components"`
		Style      *style      `json:"style,omitempty"`
	}

	var gotBody request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"characters":[[1,2,3]]}`))
	}))
	defer server.Close()

	client, err := NewClient("abc123")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	client.vbmlURL = server.URL
	client.httpClient = server.Client()

	if _, err := client.FormatMessage(context.Background(), "hello", "note", "top", "left"); err != nil {
		t.Fatalf("format message: %v", err)
	}
	if gotBody.Style == nil {
		t.Fatal("expected style in request for note model")
	}
	if gotBody.Style.Height != 3 || gotBody.Style.Width != 15 {
		t.Fatalf("unexpected note dimensions: %#v", gotBody.Style)
	}
	if len(gotBody.Components) != 1 || gotBody.Components[0].Style.Align != "top" || gotBody.Components[0].Style.Justify != "left" {
		t.Fatalf("unexpected component style: %#v", gotBody.Components)
	}
}
