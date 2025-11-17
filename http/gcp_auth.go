package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// gcpRoundTripper wraps http.RoundTripper to add Google Cloud authentication.
// It delays GCP authentication initialization until the first actual request is made.
// This avoids unnecessary credential loading when ar+https protocol is not actually used.
//
// It uses Application Default Credentials (ADC) which checks:
// 1. GOOGLE_APPLICATION_CREDENTIALS environment variable
// 2. gcloud auth application-default credentials
// 3. GCE/GKE metadata server
// See https://cloud.google.com/docs/authentication/application-default-credentials for usage details.
type gcpRoundTripper struct {
	base     http.RoundTripper
	initOnce sync.Once
	tokenSrc oauth2.TokenSource
	initErr  error
}

func (t *gcpRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Lazy initialization: only initialize GCP credentials on first request
	t.initOnce.Do(func() {
		creds, err := google.FindDefaultCredentials(context.Background(),
			"https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			t.initErr = fmt.Errorf("failed to find default credentials: %w", err)
			return
		}
		t.tokenSrc = creds.TokenSource
	})

	reqCopy := req.Clone(req.Context())
	reqCopy.URL.Scheme = strings.TrimPrefix(reqCopy.URL.Scheme, "ar+")

	// Fall back to base transport if GCP auth initialization failed
	if t.initErr != nil {
		return t.base.RoundTrip(reqCopy)
	}

	token, err := t.tokenSrc.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth2 token: %w", err)
	}
	token.SetAuthHeader(reqCopy)

	return t.base.RoundTrip(reqCopy)
}

// NewGCPRoundTripper creates a new RoundTripper that handles GCP authentication for ar+https protocol.
func NewGCPRoundTripper(base http.RoundTripper) http.RoundTripper {
	return &gcpRoundTripper{
		base: base,
	}
}
