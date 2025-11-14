package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGCPAuthTransport_RoundTrip(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if Authorization header is present
		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Error("Expected Authorization header, got none")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		t.Skip("Skipping test: GOOGLE_APPLICATION_CREDENTIALS not set")
	}

	transport := NewGCPRoundTripper(http.DefaultTransport)

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
