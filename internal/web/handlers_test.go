package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseAndGetURL(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/shorten-url",
		strings.NewReader("url=  https://example.com  "),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	got, err := ParseAndGetURL(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "https://example.com"

	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParseAndGetURLMissing(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/shorten-url",
		strings.NewReader("url="),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, err := ParseAndGetURL(req)

	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	if err.Error() != "URL required" {
		t.Errorf("got error %q, want %q", err.Error(), "URL required")
	}
}

func TestWriteShortURL(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Host = "localhost:8080"
	req.Header.Set("X-Forwarded-Proto", "https")

	rec := httptest.NewRecorder()

	writeShortURL(rec, req, "abc123")

	if rec.Code != http.StatusCreated {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusCreated)
	}

	if !strings.Contains(rec.Body.String(), "https://localhost:8080/abc123") {
		t.Errorf("response does not contain expected short URL")
	}
}