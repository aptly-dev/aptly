package api

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/aptly-dev/aptly/aptly"
	ctx "github.com/aptly-dev/aptly/context"
	"github.com/gin-gonic/gin"
)

// pprofTestServer starts a real httptest.Server with the aptly router configured
// with enablePprofEndpoint: true. The caller is responsible for calling
// server.Close() and aptlyCtx.Shutdown().
func pprofTestServer(t *testing.T) (*httptest.Server, *ctx.AptlyContext) {
	t.Helper()

	aptly.Version = "testVersion"

	file, err := os.CreateTemp("", "aptly-pprof-integration")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(file.Name()) })

	jsonString, err := json.Marshal(gin.H{
		"architectures":       []string{},
		"enablePprofEndpoint": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.Write(jsonString); err != nil {
		t.Fatal(err)
	}
	_ = file.Close()

	aptlyCtx, router, err := newPprofContext(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	return httptest.NewServer(router), aptlyCtx
}

// runPprof runs "go tool pprof -top <url>" and returns the combined output.
func runPprof(t *testing.T, url string) string {
	t.Helper()
	cmd := exec.Command("go", "tool", "pprof", "-top", url)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go tool pprof -top %s failed: %v\noutput:\n%s", url, err, out)
	}
	return string(out)
}

// TestPprofToolGoroutine verifies that go tool pprof can fetch and parse the
// goroutine profile from the /api/debug/pprof/goroutine endpoint.
func TestPprofToolGoroutine(t *testing.T) {
	server, aptlyCtx := pprofTestServer(t)
	defer server.Close()
	defer aptlyCtx.Shutdown()

	url := fmt.Sprintf("%s/api/debug/pprof/goroutine", server.URL)
	out := runPprof(t, url)
	t.Logf("go tool pprof goroutine output:\n%s", out)

	if !strings.Contains(out, "Type: goroutine") {
		t.Errorf("expected 'Type: goroutine' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "flat") {
		t.Errorf("expected 'flat' column header in output, got:\n%s", out)
	}
}

// TestPprofToolHeap verifies that go tool pprof can fetch and parse the heap
// profile from the /api/debug/pprof/heap endpoint.
func TestPprofToolHeap(t *testing.T) {
	server, aptlyCtx := pprofTestServer(t)
	defer server.Close()
	defer aptlyCtx.Shutdown()

	url := fmt.Sprintf("%s/api/debug/pprof/heap", server.URL)
	out := runPprof(t, url)
	t.Logf("go tool pprof heap output:\n%s", out)

	// Heap profiles are typed as "inuse_space" by default.
	if !strings.Contains(out, "Type: inuse_space") {
		t.Errorf("expected 'Type: inuse_space' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "flat") {
		t.Errorf("expected 'flat' column header in output, got:\n%s", out)
	}
}

// TestPprofToolCmdline verifies that the cmdline endpoint is reachable.
// cmdline returns plain text (the process argv), not a pprof binary profile,
// so we confirm the HTTP fetch succeeds rather than parsing it as a profile.
func TestPprofToolCmdline(t *testing.T) {
	server, aptlyCtx := pprofTestServer(t)
	defer server.Close()
	defer aptlyCtx.Shutdown()

	url := fmt.Sprintf("%s/api/debug/pprof/cmdline", server.URL)

	resp, err := server.Client().Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	t.Logf("cmdline endpoint status: %d", resp.StatusCode)
}
