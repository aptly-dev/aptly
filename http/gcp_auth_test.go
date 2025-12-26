package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"golang.org/x/oauth2"
)

func TestGCPAuthTransport_RoundTrip(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Error("Expected Authorization header, got none")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	transport := NewGCPRoundTripper(http.DefaultTransport)

	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		t.Skip("Skipping test: GOOGLE_APPLICATION_CREDENTIALS not set")
	}

	client := &http.Client{Transport: transport}
	resp, err := client.Get(ts.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestGCPAuthTransport_RoundTrip_with_dummy_tokenSource(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer dummy-token" {
			t.Errorf("Expected Authorization header 'Bearer dummy-token', got '%s'", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Use a dummy token source for testing
	transport := &gcpRoundTripper{
		base: http.DefaultTransport,
		tokenSrc: oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: "dummy-token",
		}),
	}
	transport.initOnce.Do(func() {}) // Mark as initialized for testing

	client := &http.Client{Transport: transport}
	resp, err := client.Get(ts.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestGCPAuthTransport_RoundTrip_with_InvalidCredentials(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer ts.Close()

	// Create a temporary invalid credentials file
	tmpFile, err := os.CreateTemp("", "invalid_credentials.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %s", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(`{"invalid": "data"}`); err != nil {
		t.Fatalf("Failed to write to temp file: %s", err)
	}
	tmpFile.Close()

	defaultEnv := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", tmpFile.Name())
	defer os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", defaultEnv)

	transport := &gcpRoundTripper{
		base: http.DefaultTransport,
	}

	client := &http.Client{Transport: transport}
	resp, err := client.Get(ts.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", resp.StatusCode)
	}

	if transport.initErr == nil {
		t.Error("Expected init error due to invalid credentials, got none")
	}
}
