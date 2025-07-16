package api

import (
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "gopkg.in/check.v1"
)

type StorageTestSuite struct {
	router *gin.Engine
}

var _ = Suite(&StorageTestSuite{})

func (s *StorageTestSuite) SetUpTest(c *C) {
	s.router = gin.New()
	s.router.GET("/api/storage", apiDiskFree)

	gin.SetMode(gin.TestMode)
}

func (s *StorageTestSuite) TestStorageListStructure(c *C) {
	// Test storage list endpoint structure
	req, _ := http.NewRequest("GET", "/api/storage", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Equals, 200)
	c.Check(w.Header().Get("Content-Type"), Equals, "application/json; charset=utf-8")

	// Should return some storage information without error
}

func (s *StorageTestSuite) TestStorageHTTPMethods(c *C) {
	// Test that only GET method is allowed
	deniedMethods := []string{"POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

	for _, method := range deniedMethods {
		req, _ := http.NewRequest(method, "/api/storage", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		c.Check(w.Code, Equals, 404, Commentf("Method: %s should be denied", method))
	}
}

func (s *StorageTestSuite) TestStorageEndpointReliability(c *C) {
	// Test multiple calls to ensure endpoint is reliable
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", "/api/storage", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		c.Check(w.Code, Equals, 200, Commentf("Call #%d", i+1))
		c.Check(w.Header().Get("Content-Type"), Equals, "application/json; charset=utf-8")
	}
}

func (s *StorageTestSuite) TestStorageResponseStructure(c *C) {
	// Test that response structure is consistent
	req, _ := http.NewRequest("GET", "/api/storage", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Equals, 200)

	// Should have valid JSON response
	body := w.Body.String()
	c.Check(len(body), Not(Equals), 0)

	// Should start with valid JSON structure
	c.Check(body[0], Equals, byte('{'), Commentf("Response should be JSON object"))
}
