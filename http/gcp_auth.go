package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GCPAuthTransport wraps http.RoundTripper to add Google Cloud authentication
type GCPAuthTransport struct {
	base     http.RoundTripper
	tokenSrc oauth2.TokenSource
}

// NewGCPAuthTransport creates a new GCPAuthTransport with default Google Cloud credentials
// It uses Application Default Credentials (ADC) which checks:
// 1. GOOGLE_APPLICATION_CREDENTIALS environment variable
// 2. gcloud auth application-default credentials
// 3. GCE/GKE metadata server
// See https://cloud.google.com/docs/authentication/application-default-credentials for usage details.
func NewGCPAuthTransport(ctx context.Context, base http.RoundTripper) (*GCPAuthTransport, error) {
	creds, err := google.FindDefaultCredentials(ctx,
		"https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("failed to find default credentials: %w", err)
	}

	return &GCPAuthTransport{
		base:     base,
		tokenSrc: creds.TokenSource,
	}, nil
}

func (t *GCPAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqCopy := req.Clone(req.Context())

	if strings.HasPrefix(reqCopy.URL.Scheme, "ar+") {
		reqCopy.URL.Scheme = strings.TrimPrefix(reqCopy.URL.Scheme, "ar+")
	}

	token, err := t.tokenSrc.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth2 token: %w", err)
	}
	token.SetAuthHeader(reqCopy)

	return t.base.RoundTrip(reqCopy)
}

func NewGCPRoundTripper(ctx context.Context, base http.RoundTripper) http.RoundTripper {
	gcpTransport, err := NewGCPAuthTransport(ctx, base)
	if err != nil {
		return base
	}
	return gcpTransport
}
