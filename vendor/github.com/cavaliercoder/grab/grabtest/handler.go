package grabtest

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var (
	DefaultHandlerContentLength       = 1 << 20
	DefaultHandlerMD5Checksum         = "c35cc7d8d91728a0cb052831bc4ef372"
	DefaultHandlerMD5ChecksumBytes    = MustHexDecodeString(DefaultHandlerMD5Checksum)
	DefaultHandlerSHA256Checksum      = "fbbab289f7f94b25736c58be46a994c441fd02552cc6022352e3d86d2fab7c83"
	DefaultHandlerSHA256ChecksumBytes = MustHexDecodeString(DefaultHandlerSHA256Checksum)
)

type StatusCodeFunc func(req *http.Request) int

type handler struct {
	statusCodeFunc     StatusCodeFunc
	methodWhitelist    []string
	headerBlacklist    []string
	contentLength      int
	acceptRanges       bool
	attachmentFilename string
	lastModified       time.Time
	ttfb               time.Duration
	rateLimiter        *time.Ticker
}

func NewHandler(options ...HandlerOption) (http.Handler, error) {
	h := &handler{
		statusCodeFunc:  func(req *http.Request) int { return http.StatusOK },
		methodWhitelist: []string{"GET", "HEAD"},
		contentLength:   DefaultHandlerContentLength,
		acceptRanges:    true,
	}
	for _, option := range options {
		if err := option(h); err != nil {
			return nil, err
		}
	}
	return h, nil
}

func WithTestServer(t *testing.T, f func(url string), options ...HandlerOption) {
	h, err := NewHandler(options...)
	if err != nil {
		t.Fatalf("unable to create test server handler: %v", err)
		return
	}
	s := httptest.NewServer(h)
	defer func() {
		h.(*handler).close()
		s.Close()
	}()
	f(s.URL)
}

func (h *handler) close() {
	if h.rateLimiter != nil {
		h.rateLimiter.Stop()
	}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// delay response
	if h.ttfb > 0 {
		time.Sleep(h.ttfb)
	}

	// validate request method
	allowed := false
	for _, m := range h.methodWhitelist {
		if r.Method == m {
			allowed = true
			break
		}
	}
	if !allowed {
		httpError(w, http.StatusMethodNotAllowed)
		return
	}

	// set server options
	if h.acceptRanges {
		w.Header().Set("Accept-Ranges", "bytes")
	}

	// set attachment filename
	if h.attachmentFilename != "" {
		w.Header().Set(
			"Content-Disposition",
			fmt.Sprintf("attachment;filename=\"%s\"", h.attachmentFilename),
		)
	}

	// set last modified timestamp
	lastMod := time.Now()
	if !h.lastModified.IsZero() {
		lastMod = h.lastModified
	}
	w.Header().Set("Last-Modified", lastMod.Format(http.TimeFormat))

	// set content-length
	offset := 0
	if h.acceptRanges {
		if reqRange := r.Header.Get("Range"); reqRange != "" {
			if _, err := fmt.Sscanf(reqRange, "bytes=%d-", &offset); err != nil {
				httpError(w, http.StatusBadRequest)
				return
			}
			if offset >= h.contentLength {
				httpError(w, http.StatusRequestedRangeNotSatisfiable)
				return
			}
		}
	}
	w.Header().Set("Content-Length", fmt.Sprintf("%d", h.contentLength-offset))

	// apply header blacklist
	for _, key := range h.headerBlacklist {
		w.Header().Del(key)
	}

	// send header and status code
	w.WriteHeader(h.statusCodeFunc(r))

	// send body
	if r.Method == "GET" {
		// use buffered io to reduce overhead on the reader
		bw := bufio.NewWriterSize(w, 4096)
		for i := offset; !isRequestClosed(r) && i < h.contentLength; i++ {
			bw.Write([]byte{byte(i)})
			if h.rateLimiter != nil {
				bw.Flush()
				w.(http.Flusher).Flush() // force the server to send the data to the client
				select {
				case <-h.rateLimiter.C:
				case <-r.Context().Done():
				}
			}
		}
		if !isRequestClosed(r) {
			bw.Flush()
		}
	}
}

// isRequestClosed returns true if the client request has been canceled.
func isRequestClosed(r *http.Request) bool {
	return r.Context().Err() != nil
}

func httpError(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
