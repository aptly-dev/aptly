package cmd

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	check "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

func TestMetricsOnlyHandler(t *testing.T) {
	tests := []struct {
		name             string
		method           string
		target           string
		wantStatus       int
		wantForwarded    bool
		wantQuery        string
		wantRequestURI   string
		wantAllowMethods string
	}{
		{
			name:           "get",
			method:         http.MethodGet,
			target:         "/metrics?_async=true&value=a%2Bb",
			wantStatus:     http.StatusNoContent,
			wantForwarded:  true,
			wantQuery:      "_async=true&value=a%2Bb",
			wantRequestURI: "/api/metrics?_async=true&value=a%2Bb",
		},
		{
			name:           "head",
			method:         http.MethodHead,
			target:         "/metrics",
			wantStatus:     http.StatusNoContent,
			wantForwarded:  true,
			wantRequestURI: "/api/metrics",
		},
		{
			name:       "api metrics path",
			method:     http.MethodGet,
			target:     "/api/metrics",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "trailing slash",
			method:     http.MethodGet,
			target:     "/metrics/",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "encoded alias",
			method:     http.MethodGet,
			target:     "/met%72ics",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "case variant",
			method:     http.MethodGet,
			target:     "/METRICS",
			wantStatus: http.StatusNotFound,
		},
		{
			name:             "unsupported method",
			method:           http.MethodPost,
			target:           "/metrics",
			wantStatus:       http.StatusMethodNotAllowed,
			wantAllowMethods: "GET, HEAD",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			forwarded := false
			var forwardedMethod, forwardedPath, forwardedQuery, forwardedRequestURI string
			apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				forwarded = true
				forwardedMethod = r.Method
				forwardedPath = r.URL.Path
				forwardedQuery = r.URL.RawQuery
				forwardedRequestURI = r.RequestURI
				w.WriteHeader(http.StatusNoContent)
			})

			request := httptest.NewRequest(test.method, test.target, nil)
			originalPath := request.URL.Path
			originalRawPath := request.URL.RawPath
			originalRequestURI := request.RequestURI
			response := httptest.NewRecorder()

			metricsOnlyHandler(apiHandler).ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if forwarded != test.wantForwarded {
				t.Fatalf("forwarded = %t, want %t", forwarded, test.wantForwarded)
			}
			if got := response.Header().Get("Allow"); got != test.wantAllowMethods {
				t.Errorf("Allow = %q, want %q", got, test.wantAllowMethods)
			}
			if request.URL.Path != originalPath || request.URL.RawPath != originalRawPath || request.RequestURI != originalRequestURI {
				t.Errorf("original request was mutated")
			}

			if test.wantForwarded {
				if forwardedMethod != test.method {
					t.Errorf("forwarded method = %q, want %q", forwardedMethod, test.method)
				}
				if forwardedPath != "/api/metrics" {
					t.Errorf("forwarded path = %q, want /api/metrics", forwardedPath)
				}
				if forwardedQuery != test.wantQuery {
					t.Errorf("forwarded query = %q, want %q", forwardedQuery, test.wantQuery)
				}
				if forwardedRequestURI != test.wantRequestURI {
					t.Errorf("forwarded request URI = %q, want %q", forwardedRequestURI, test.wantRequestURI)
				}
			}
		})
	}
}

func TestValidateMetricsListener(t *testing.T) {
	for _, test := range []struct {
		name           string
		address        string
		metricsEnabled bool
		wantError      bool
	}{
		{name: "disabled listener", metricsEnabled: false},
		{name: "existing API metrics only", metricsEnabled: true},
		{name: "enabled listener", address: "127.0.0.1:9090", metricsEnabled: true},
		{name: "listener requires metrics", address: "127.0.0.1:9090", metricsEnabled: false, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateMetricsListener(test.address, test.metricsEnabled)
			if (err != nil) != test.wantError {
				t.Fatalf("error = %v, wantError %t", err, test.wantError)
			}
		})
	}
}

func TestSelectActivatedAPIListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	selected, err := selectActivatedAPIListener(nil)
	if err != nil || selected != nil {
		t.Fatalf("no listeners: selected = %v, error = %v", selected, err)
	}

	selected, err = selectActivatedAPIListener([]net.Listener{listener})
	if err != nil || selected != listener {
		t.Fatalf("one listener: selected = %v, error = %v", selected, err)
	}

	if _, err = selectActivatedAPIListener([]net.Listener{nil}); err == nil {
		t.Fatal("nil listener did not return an error")
	}
	if _, err = selectActivatedAPIListener([]net.Listener{listener, listener}); err == nil {
		t.Fatal("multiple listeners did not return an error")
	}
}

func TestServeAPIWithMetricsClosesAPIListenerOnMetricsBindFailure(t *testing.T) {
	apiListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = apiListener.Close() })
	apiAddress := apiListener.Addr().String()

	occupiedMetricsListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = apiListener.Close()
		t.Fatal(err)
	}
	defer func() { _ = occupiedMetricsListener.Close() }()

	err = serveAPIWithMetrics(apiListener, false, "", occupiedMetricsListener.Addr().String())
	if err == nil || !strings.Contains(err.Error(), "failed to listen on metrics address") {
		t.Fatalf("error = %v, want metrics bind failure", err)
	}

	connection, dialErr := net.DialTimeout("tcp", apiAddress, 100*time.Millisecond)
	if dialErr == nil {
		_ = connection.Close()
		t.Fatal("API listener still accepts connections after metrics bind failure")
	}
}

func TestShutdownHTTPServersForcesCloseAfterTimeout(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(releaseRequest) })
	})

	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(requestStarted)
		<-releaseRequest
		w.WriteHeader(http.StatusOK)
	})}
	serveResult := make(chan error, 1)
	go func() { serveResult <- server.Serve(listener) }()

	clientResult := make(chan error, 1)
	go func() {
		response, requestErr := http.Get("http://" + listener.Addr().String())
		if response != nil {
			_ = response.Body.Close()
		}
		clientResult <- requestErr
	}()

	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("request did not reach server")
	}

	started := time.Now()
	shutdownHTTPServers([]apiHTTPServer{{name: "test", server: server, listener: listener}}, 50*time.Millisecond)
	if elapsed := time.Since(started); elapsed < 40*time.Millisecond {
		t.Fatalf("shutdown closed an active request immediately after %s", elapsed)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("forced shutdown took %s", elapsed)
	}

	select {
	case requestErr := <-clientResult:
		if requestErr == nil {
			t.Error("client request unexpectedly completed during forced shutdown")
		}
	case <-time.After(time.Second):
		t.Fatal("client connection was not closed")
	}

	select {
	case serveErr := <-serveResult:
		if serveErr != http.ErrServerClosed {
			t.Errorf("Serve error = %v, want %v", serveErr, http.ErrServerClosed)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve did not return")
	}

	releaseOnce.Do(func() { close(releaseRequest) })
}

func TestServeHTTPServersStopsSiblingOnFailure(t *testing.T) {
	apiListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	metricsListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = apiListener.Close()
		t.Fatal(err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	servers := []apiHTTPServer{
		{name: "API", server: &http.Server{Handler: handler}, listener: apiListener},
		{name: "metrics", server: &http.Server{Handler: handler}, listener: metricsListener},
	}
	for _, server := range servers {
		server := server
		t.Cleanup(func() {
			_ = server.server.Close()
			_ = server.listener.Close()
		})
	}

	sigchan := make(chan os.Signal)
	signalsRestored := make(chan struct{})
	tasksWaited := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		result <- serveHTTPServers(
			servers,
			sigchan,
			func() { close(signalsRestored) },
			func() { close(tasksWaited) },
			time.Second,
		)
	}()

	waitForHTTP(t, apiListener.Addr().String())
	waitForHTTP(t, metricsListener.Addr().String())
	if err = apiListener.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case err = <-result:
		if err == nil || !strings.Contains(err.Error(), "API server stopped unexpectedly") {
			t.Fatalf("error = %v, want unexpected API server failure", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server coordinator did not return")
	}

	select {
	case <-signalsRestored:
	default:
		t.Error("signal handling was not restored")
	}
	select {
	case <-tasksWaited:
	default:
		t.Error("background tasks were not awaited")
	}

	connection, dialErr := net.DialTimeout("tcp", metricsListener.Addr().String(), 100*time.Millisecond)
	if dialErr == nil {
		_ = connection.Close()
		t.Error("metrics sibling still accepts connections")
	}
}

func TestServeHTTPServersShutsDownOnSignal(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})}
	t.Cleanup(func() {
		_ = server.Close()
		_ = listener.Close()
	})

	sigchan := make(chan os.Signal, 1)
	signalsRestored := make(chan struct{})
	tasksWaitedAfterShutdown := make(chan bool, 1)
	result := make(chan error, 1)
	go func() {
		result <- serveHTTPServers(
			[]apiHTTPServer{{name: "API", server: server, listener: listener}},
			sigchan,
			func() { close(signalsRestored) },
			func() {
				connection, dialErr := net.DialTimeout("tcp", listener.Addr().String(), 100*time.Millisecond)
				if dialErr == nil {
					_ = connection.Close()
				}
				tasksWaitedAfterShutdown <- dialErr != nil
			},
			time.Second,
		)
	}()

	waitForHTTP(t, listener.Addr().String())
	sigchan <- syscall.SIGTERM

	select {
	case err = <-result:
		if err != nil {
			t.Fatalf("signal shutdown returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server coordinator did not return")
	}

	select {
	case <-signalsRestored:
	default:
		t.Error("signal handling was not restored")
	}
	select {
	case listenerClosed := <-tasksWaitedAfterShutdown:
		if !listenerClosed {
			t.Error("background tasks were awaited before HTTP shutdown")
		}
	case <-time.After(time.Second):
		t.Error("background tasks were not awaited")
	}
}

func waitForHTTP(t *testing.T, address string) {
	t.Helper()
	client := &http.Client{Timeout: 100 * time.Millisecond}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get("http://" + address)
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("HTTP server at %s did not become ready", address)
}
