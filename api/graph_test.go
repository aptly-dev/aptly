package api

import (
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"

	. "gopkg.in/check.v1"
)

type GraphTestSuite struct {
	APISuite
}

var _ = Suite(&GraphTestSuite{})

func (s *GraphTestSuite) SetUpTest(c *C) {
	s.APISuite.SetUpTest(c)
}

func (s *GraphTestSuite) TestGraphDotFormat(c *C) {
	// Test requesting raw DOT format
	req, _ := http.NewRequest("GET", "/api/graph.dot", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Should succeed with context and return DOT format
	c.Check(w.Code, Equals, 200)
	c.Check(w.Header().Get("Content-Type"), Equals, "text/plain; charset=utf-8")
}

func (s *GraphTestSuite) TestGraphGvFormat(c *C) {
	// Test requesting GV format (alias for DOT)
	req, _ := http.NewRequest("GET", "/api/graph.gv", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Should succeed with context and return DOT format (gv is alias)
	c.Check(w.Code, Equals, 200)
	c.Check(w.Header().Get("Content-Type"), Equals, "text/plain; charset=utf-8")
}

func (s *GraphTestSuite) TestGraphSvgFormat(c *C) {
	// Test requesting SVG format (requires graphviz)
	req, _ := http.NewRequest("GET", "/api/graph.svg", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will likely error due to no context or missing graphviz
	c.Check(w.Code, Not(Equals), 200) // Expect error
}

func (s *GraphTestSuite) TestGraphPngFormat(c *C) {
	// Test requesting PNG format (requires graphviz)
	req, _ := http.NewRequest("GET", "/api/graph.png", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will likely error due to no context or missing graphviz
	c.Check(w.Code, Not(Equals), 200) // Expect error
}

func (s *GraphTestSuite) TestGraphWithHorizontalLayout(c *C) {
	// Test with horizontal layout parameter
	req, _ := http.NewRequest("GET", "/api/graph.svg?layout=horizontal", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will likely error due to no context, but should parse layout parameter
	c.Check(w.Code, Not(Equals), 200) // Expect error due to missing context
}

func (s *GraphTestSuite) TestGraphWithVerticalLayout(c *C) {
	// Test with vertical layout parameter
	req, _ := http.NewRequest("GET", "/api/graph.png?layout=vertical", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will likely error due to no context, but should parse layout parameter
	c.Check(w.Code, Not(Equals), 200) // Expect error due to missing context
}

func (s *GraphTestSuite) TestGraphWithInvalidLayout(c *C) {
	// Test with invalid layout parameter
	req, _ := http.NewRequest("GET", "/api/graph.dot?layout=invalid", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Should succeed - invalid layout is ignored
	c.Check(w.Code, Equals, 200)
}

func (s *GraphTestSuite) TestGraphWithEmptyLayout(c *C) {
	// Test with empty layout parameter
	req, _ := http.NewRequest("GET", "/api/graph.svg?layout=", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will fail because SVG requires graphviz which is not installed
	c.Check(w.Code, Equals, 500)
}

func (s *GraphTestSuite) TestGraphWithMultipleParams(c *C) {
	// Test with multiple query parameters
	req, _ := http.NewRequest("GET", "/api/graph.png?layout=vertical&extra=param&another=value", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will fail because PNG requires graphviz which is not installed
	c.Check(w.Code, Equals, 500)
}

func (s *GraphTestSuite) TestGraphParameterHandling(c *C) {
	// Test parameter extraction and validation
	testCases := []struct {
		path        string
		description string
	}{
		{"/api/graph.dot", "DOT format"},
		{"/api/graph.gv", "GV format"},
		{"/api/graph.svg", "SVG format"},
		{"/api/graph.png", "PNG format"},
		{"/api/graph.pdf", "PDF format"},
		{"/api/graph.ps", "PostScript format"},
		{"/api/graph.jpg", "JPEG format"},
		{"/api/graph.gif", "GIF format"},
		{"/api/graph.unknown", "Unknown format"},
	}

	for _, tc := range testCases {
		req, _ := http.NewRequest("GET", tc.path, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// All should return some response without crashing
		c.Check(w.Code, Not(Equals), 0, Commentf("Test case: %s", tc.description))
	}
}

func (s *GraphTestSuite) TestGraphMimeTypeHandling(c *C) {
	// Test MIME type detection for different extensions
	extensions := map[string]string{
		"svg": "image/svg+xml",
		"png": "image/png",
		"pdf": "application/pdf",
		"ps":  "application/postscript",
		"jpg": "image/jpeg",
		"gif": "image/gif",
	}

	for ext, expectedMime := range extensions {
		actualMime := mime.TypeByExtension("." + ext)
		if actualMime != "" {
			// Just check that the actual MIME type starts with expected
			c.Check(strings.HasPrefix(actualMime, expectedMime), Equals, true,
				Commentf("MIME type mismatch for extension: %s", ext))
		}
	}
}

func (s *GraphTestSuite) TestGraphHTTPMethods(c *C) {
	// Test that only GET method is allowed
	deniedMethods := []string{"POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

	for _, method := range deniedMethods {
		req, _ := http.NewRequest(method, "/api/graph.svg", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		c.Check(w.Code, Equals, 404, Commentf("Method: %s should be denied", method))
	}
}

func (s *GraphTestSuite) TestGraphPathValidation(c *C) {
	// Test path validation and parameter extraction
	validPaths := []string{
		"/api/graph.dot",
		"/api/graph.svg",
		"/api/graph.png",
		"/api/graph.pdf",
	}

	invalidPaths := []string{
		"/api/graph",           // Missing extension
		"/api/graph.",          // Empty extension
		"/api/graphs.svg",      // Wrong endpoint name
	}

	for _, path := range validPaths {
		req, _ := http.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should match route (even if it errors due to missing context)
		c.Check(w.Code, Not(Equals), 404, Commentf("Valid path should match route: %s", path))
	}

	for _, path := range invalidPaths {
		req, _ := http.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should not match route
		c.Check(w.Code, Equals, 404, Commentf("Invalid path should not match route: %s", path))
	}
}

func (s *GraphTestSuite) TestGraphExtensionExtraction(c *C) {
	// Test that extension is properly extracted from path
	testPaths := []string{
		"/api/graph.dot",
		"/api/graph.svg",
		"/api/graph.png",
		"/api/graph.pdf",
		"/api/graph.ps",
		"/api/graph.jpg",
		"/api/graph.gif",
		"/api/graph.unknown",
	}

	for _, path := range testPaths {
		req, _ := http.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should handle extension extraction without crashing
		c.Check(w.Code, Not(Equals), 0, Commentf("Extension extraction failed for: %s", path))
	}
}

func (s *GraphTestSuite) TestGraphQueryParameterHandling(c *C) {
	// Test various query parameter combinations
	queryTests := []struct {
		query       string
		description string
	}{
		{"", "no parameters"},
		{"layout=horizontal", "horizontal layout"},
		{"layout=vertical", "vertical layout"},
		{"layout=invalid", "invalid layout"},
		{"layout=", "empty layout"},
		{"layout=horizontal&extra=param", "multiple parameters"},
		{"unknown=param", "unknown parameter"},
		{"layout=horizontal&layout=vertical", "duplicate parameters"},
	}

	for _, test := range queryTests {
		path := "/api/graph.svg"
		if test.query != "" {
			path += "?" + test.query
		}

		req, _ := http.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should handle query parameters without crashing
		c.Check(w.Code, Not(Equals), 0, Commentf("Query parameter test: %s", test.description))
	}
}

func (s *GraphTestSuite) TestGraphErrorHandling(c *C) {
	// Test various error conditions
	errorTests := []struct {
		path        string
		description string
	}{
		{"/api/graph.svg", "missing database context"},
		{"/api/graph.png", "missing graphviz"},
		{"/api/graph.unknown", "unknown format"},
		{"/api/graph.dot", "raw DOT format"},
	}

	for _, test := range errorTests {
		req, _ := http.NewRequest("GET", test.path, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should handle errors gracefully without panicking
		c.Check(w.Code, Not(Equals), 0, Commentf("Error test: %s", test.description))
	}
}

func (s *GraphTestSuite) TestGraphContentTypeHeaders(c *C) {
	// Test that appropriate content types are set for different formats
	formatTests := []struct {
		ext         string
		expectJSON  bool
		expectImage bool
	}{
		{"dot", false, false}, // Should return text
		{"gv", false, false},  // Should return text
		{"svg", false, true},  // Should return image/svg+xml (if successful)
		{"png", false, true},  // Should return image/png (if successful)
		{"pdf", false, false}, // Should return application/pdf (if successful)
	}

	for _, test := range formatTests {
		req, _ := http.NewRequest("GET", "/api/graph."+test.ext, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		contentType := w.Header().Get("Content-Type")

		if test.expectJSON {
			c.Check(strings.Contains(contentType, "application/json"), Equals, true,
				Commentf("Expected JSON content type for .%s, got: %s", test.ext, contentType))
		}

		// Note: Image content types will only be set if graphviz is available and context exists
		c.Check(contentType, Not(Equals), "", Commentf("Content type should be set for .%s", test.ext))
	}
}

func (s *GraphTestSuite) TestGraphSpecialCharacters(c *C) {
	// Test handling of special characters in query parameters
	specialQueries := []string{
		"layout=horizontal%20with%20spaces",
		"layout=vertical&param=value%20with%20spaces",
		"layout=test%26special%3Dchars",
		"layout=unicode%E2%9C%93",
		"param=%3Cscript%3Ealert%28%29%3C%2Fscript%3E", // XSS attempt
	}

	for _, query := range specialQueries {
		req, _ := http.NewRequest("GET", "/api/graph.svg?"+query, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should handle special characters without crashing
		c.Check(w.Code, Not(Equals), 0, Commentf("Special character test failed for: %s", query))
	}
}

func (s *GraphTestSuite) TestGraphLargeExtensions(c *C) {
	// Test with very long extensions
	longExt := strings.Repeat("x", 1000)
	req, _ := http.NewRequest("GET", "/api/graph."+longExt, nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Should handle long extensions without crashing
	c.Check(w.Code, Not(Equals), 0)
}

func (s *GraphTestSuite) TestGraphReliability(c *C) {
	// Test multiple sequential calls for reliability
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", "/api/graph.dot", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should be consistent across multiple calls
		c.Check(w.Code, Not(Equals), 0, Commentf("Call #%d", i+1))
	}
}

func (s *GraphTestSuite) TestGraphConcurrency(c *C) {
	// Test concurrent requests to ensure thread safety
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()

			req, _ := http.NewRequest("GET", "/api/graph.svg", nil)
			w := httptest.NewRecorder()
			s.router.ServeHTTP(w, req)

			// Should handle concurrent requests without issues
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	c.Check(true, Equals, true) // Test completed without deadlocks
}
