package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGCPAuthTransport_RoundTrip(t *testing.T) {
	// Create a test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if Authorization header is present
		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Error("Expected Authorization header, got none")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Skip this test if no GCP credentials are available
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		t.Skip("Skipping test: GOOGLE_APPLICATION_CREDENTIALS not set")
	}

	ctx := context.Background()
	transport, err := NewGCPAuthTransport(ctx, http.DefaultTransport)
	if err != nil {
		t.Fatalf("Failed to create GCP auth transport: %v", err)
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
