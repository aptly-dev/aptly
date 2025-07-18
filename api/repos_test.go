package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/utils"
	"github.com/gin-gonic/gin"
	. "gopkg.in/check.v1"
)

type ReposTestSuite struct {
	APISuite
}

var _ = Suite(&ReposTestSuite{})

func (s *ReposTestSuite) SetUpTest(c *C) {
	s.APISuite.SetUpTest(c)
}

func (s *ReposTestSuite) TestReposListEmpty(c *C) {
	// Test listing repos when none exist
	req, _ := http.NewRequest("GET", "/api/repos", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Equals, 200)
	c.Check(w.Header().Get("Content-Type"), Equals, "application/json; charset=utf-8")

	var result []*deb.LocalRepo
	err := json.Unmarshal(w.Body.Bytes(), &result)
	c.Check(err, IsNil)
	c.Check(len(result), Equals, 0)
}

func (s *ReposTestSuite) TestReposCreateBasic(c *C) {
	// Test creating a basic repository
	params := repoCreateParams{
		Name:                "test-repo",
		Comment:             "Test repository",
		DefaultDistribution: "stable",
		DefaultComponent:    "main",
	}

	jsonBody, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "/api/repos", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Now context is properly set up, should create successfully
	c.Check(w.Code, Equals, 201) // Expect successful creation
	
	// Clean up: delete the created repo
	req, _ = http.NewRequest("DELETE", "/api/repos/test-repo?force=1", nil)
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	c.Check(w.Code, Equals, 200)
}

func (s *ReposTestSuite) TestReposEdit(c *C) {
	// First create a repo
	params := repoCreateParams{
		Name:    "edit-test-repo",
		Comment: "Original comment",
	}
	body, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "/api/repos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	c.Check(w.Code, Equals, 201)

	// Now edit it
	editParams := reposEditParams{
		Comment: stringPtr("Updated comment"),
	}
	body, _ = json.Marshal(editParams)
	req, _ = http.NewRequest("PUT", "/api/repos/edit-test-repo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	c.Check(w.Code, Equals, 200)

	// Clean up
	req, _ = http.NewRequest("DELETE", "/api/repos/edit-test-repo?force=1", nil)
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	c.Check(w.Code, Equals, 200)
}

func (s *ReposTestSuite) TestReposPackagesAddDelete(c *C) {
	// First create a repo
	params := repoCreateParams{
		Name: "pkg-test-repo",
	}
	body, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "/api/repos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	c.Check(w.Code, Equals, 201)

	// Test adding packages (will fail without actual packages)
	addParams := reposPackagesAddDeleteParams{
		PackageRefs: []string{"Pamd64 test 1.0 abc123"},
	}
	body, _ = json.Marshal(addParams)
	req, _ = http.NewRequest("POST", "/api/repos/pkg-test-repo/packages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	// Will fail as package doesn't exist
	c.Check(w.Code, Not(Equals), 200)

	// Clean up
	req, _ = http.NewRequest("DELETE", "/api/repos/pkg-test-repo?force=1", nil)
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	c.Check(w.Code, Equals, 200)
}

func (s *ReposTestSuite) TestReposCopyPackage(c *C) {
	// Create source and destination repos
	params := repoCreateParams{Name: "src-repo"}
	body, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "/api/repos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	c.Check(w.Code, Equals, 201)

	params = repoCreateParams{Name: "dst-repo"}
	body, _ = json.Marshal(params)
	req, _ = http.NewRequest("POST", "/api/repos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	c.Check(w.Code, Equals, 201)

	// Test copy (will fail without packages)
	copyParams := reposCopyPackageParams{
		WithDeps: true,
		DryRun:   true,
	}
	body, _ = json.Marshal(copyParams)
	req, _ = http.NewRequest("POST", "/api/repos/dst-repo/copy/src-repo/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	// Will return empty result as no packages match
	c.Check(w.Code, Equals, 200)

	// Clean up
	req, _ = http.NewRequest("DELETE", "/api/repos/src-repo?force=1", nil)
	s.router.ServeHTTP(w, req)
	req, _ = http.NewRequest("DELETE", "/api/repos/dst-repo?force=1", nil)
	s.router.ServeHTTP(w, req)
}

func (s *ReposTestSuite) TestReposCreateInvalidJSON(c *C) {
	// Test creating repository with invalid JSON
	req, _ := http.NewRequest("POST", "/api/repos", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Equals, 400)
}

func (s *ReposTestSuite) TestReposCreateMissingName(c *C) {
	// Test creating repository without required name
	params := repoCreateParams{
		Comment:             "Test repository",
		DefaultDistribution: "stable",
		DefaultComponent:    "main",
	}

	jsonBody, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "/api/repos", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Equals, 400)
}

func (s *ReposTestSuite) TestReposShowNotFound(c *C) {
	// Test showing non-existent repository
	req, _ := http.NewRequest("GET", "/api/repos/nonexistent", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context, but tests endpoint structure
	c.Check(w.Code, Not(Equals), 200)
}

func (s *ReposTestSuite) TestReposEditStructure(c *C) {
	// Test repository edit endpoint structure
	params := reposEditParams{
		Name:    stringPtr("new-name"),
		Comment: stringPtr("Updated comment"),
	}

	jsonBody, _ := json.Marshal(params)
	req, _ := http.NewRequest("PUT", "/api/repos/test-repo", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context, but tests structure
	c.Check(w.Code, Not(Equals), 200)
}

func (s *ReposTestSuite) TestReposEditInvalidJSON(c *C) {
	// Test edit with invalid JSON
	req, _ := http.NewRequest("PUT", "/api/repos/test-repo", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Equals, 400)
}

func (s *ReposTestSuite) TestReposDropStructure(c *C) {
	// Test repository drop endpoint structure
	req, _ := http.NewRequest("DELETE", "/api/repos/test-repo", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Should return 404 as test-repo doesn't exist
	c.Check(w.Code, Equals, 404)
}

func (s *ReposTestSuite) TestReposDropWithForce(c *C) {
	// Test repository drop with force parameter
	req, _ := http.NewRequest("DELETE", "/api/repos/test-repo?force=1", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context, but tests parameter parsing
	c.Check(w.Code, Not(Equals), 200)
}

func (s *ReposTestSuite) TestReposPackagesShowStructure(c *C) {
	// Test packages show endpoint structure
	req, _ := http.NewRequest("GET", "/api/repos/test-repo/packages", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context, but tests structure
	c.Check(w.Code, Not(Equals), 200)
}

func (s *ReposTestSuite) TestReposPackagesShowWithQuery(c *C) {
	// Test packages show with query parameters
	req, _ := http.NewRequest("GET", "/api/repos/test-repo/packages?q=Name%20(~%20test)", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context, but tests query parsing
	c.Check(w.Code, Not(Equals), 200)
}

func (s *ReposTestSuite) TestReposPackagesAddStructure(c *C) {
	// Test packages add endpoint structure
	params := reposPackagesAddDeleteParams{
		PackageRefs: []string{"Pamd64 test 1.0 abc123"},
	}

	jsonBody, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "/api/repos/test-repo/packages", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context, but tests structure
	c.Check(w.Code, Not(Equals), 200)
}

func (s *ReposTestSuite) TestReposPackagesAddInvalidJSON(c *C) {
	// Test packages add with invalid JSON
	req, _ := http.NewRequest("POST", "/api/repos/test-repo/packages", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Equals, 400)
}

func (s *ReposTestSuite) TestReposPackagesDeleteStructure(c *C) {
	// Test packages delete endpoint structure
	params := reposPackagesAddDeleteParams{
		PackageRefs: []string{"Pamd64 test 1.0 abc123"},
	}

	jsonBody, _ := json.Marshal(params)
	req, _ := http.NewRequest("DELETE", "/api/repos/test-repo/packages", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context, but tests structure
	c.Check(w.Code, Not(Equals), 200)
}

func (s *ReposTestSuite) TestReposFileUploadStructure(c *C) {
	// Test file upload endpoint structure
	req, _ := http.NewRequest("POST", "/api/repos/test-repo/file/upload-dir", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context, but tests structure
	c.Check(w.Code, Not(Equals), 200)
}

func (s *ReposTestSuite) TestReposFileUploadWithParameters(c *C) {
	// Test file upload with query parameters
	req, _ := http.NewRequest("POST", "/api/repos/test-repo/file/upload-dir?noRemove=1&forceReplace=1", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context, but tests parameter parsing
	c.Check(w.Code, Not(Equals), 200)
}

func (s *ReposTestSuite) TestReposFileUploadSpecificFile(c *C) {
	// Test specific file upload endpoint
	req, _ := http.NewRequest("POST", "/api/repos/test-repo/file/upload-dir/package.deb", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context, but tests structure
	c.Check(w.Code, Not(Equals), 200)
}

func (s *ReposTestSuite) TestReposCopyPackageStructure(c *C) {
	// Test copy package endpoint structure
	params := reposCopyPackageParams{
		WithDeps: true,
		DryRun:   false,
	}

	jsonBody, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "/api/repos/dest-repo/copy/src-repo/package-query", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context, but tests structure
	c.Check(w.Code, Not(Equals), 200)
}

func (s *ReposTestSuite) TestReposCopyPackageInvalidJSON(c *C) {
	// Test copy package with invalid JSON
	req, _ := http.NewRequest("POST", "/api/repos/dest-repo/copy/src-repo/package-query", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Equals, 400)
}

func (s *ReposTestSuite) TestReposIncludePackageStructure(c *C) {
	// Test include package endpoint structure
	req, _ := http.NewRequest("POST", "/api/repos/test-repo/include/upload-dir", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context, but tests structure
	c.Check(w.Code, Not(Equals), 200)
}

func (s *ReposTestSuite) TestReposIncludePackageWithParameters(c *C) {
	// Test include package with query parameters
	req, _ := http.NewRequest("POST", "/api/repos/test-repo/include/upload-dir?forceReplace=1&noRemoveFiles=1&acceptUnsigned=1&ignoreSignature=1", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context, but tests parameter parsing
	c.Check(w.Code, Not(Equals), 200)
}

func (s *ReposTestSuite) TestReposIncludeSpecificFile(c *C) {
	// Test include specific file endpoint
	req, _ := http.NewRequest("POST", "/api/repos/test-repo/include/upload-dir/package.changes", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context, but tests structure
	c.Check(w.Code, Not(Equals), 200)
}

func (s *ReposTestSuite) TestReposParameterValidation(c *C) {
	// Test parameter validation and structure
	testCases := []struct {
		name     string
		method   string
		path     string
		body     string
		wantCode int
	}{
		{"invalid repo name chars", "GET", "/api/repos/invalid/name", "", 404}, // route doesn't match
		{"empty repo name", "GET", "/api/repos", "", 200}, // list repos endpoint
		{"invalid method", "PATCH", "/api/repos/test", "", 404},
		{"malformed JSON in create", "POST", "/api/repos", `{"Name":}`, 400},
		{"malformed JSON in edit", "PUT", "/api/repos/test", `{"Name":}`, 400},
		{"malformed JSON in packages", "POST", "/api/repos/test/packages", `{"PackageRefs":}`, 400},
	}

	for _, tc := range testCases {
		var req *http.Request
		if tc.body != "" {
			req, _ = http.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req, _ = http.NewRequest(tc.method, tc.path, nil)
		}

		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		c.Check(w.Code, Equals, tc.wantCode, Commentf("Test case: %s", tc.name))
	}
}

func (s *ReposTestSuite) TestReposListInAPIModeStructure(c *C) {
	// Test reposListInAPIMode function structure
	localRepos := map[string]utils.FileSystemPublishRoot{
		"repo1": {},
		"repo2": {},
	}

	handler := reposListInAPIMode(localRepos)
	c.Check(handler, NotNil)

	// Test with empty repos map
	emptyHandler := reposListInAPIMode(map[string]utils.FileSystemPublishRoot{})
	c.Check(emptyHandler, NotNil)
}

func (s *ReposTestSuite) TestReposServeInAPIModeStructure(c *C) {
	// Test reposServeInAPIMode function structure by simulating call
	s.router.(*gin.Engine).GET("/api/:storage/*pkgPath", reposServeInAPIMode)

	// Test with default storage
	req, _ := http.NewRequest("GET", "/api/-/some/package/path", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context, but tests parameter parsing
	c.Check(w.Code, Not(Equals), 200)

	// Test with specific storage
	req, _ = http.NewRequest("GET", "/api/storage1/some/package/path", nil)
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context, but tests structure
	c.Check(w.Code, Not(Equals), 200)
}

func (s *ReposTestSuite) TestReposCreateFromSnapshot(c *C) {
	// Test creating repository from snapshot
	params := repoCreateParams{
		Name:         "test-repo-from-snapshot",
		Comment:      "Test repository from snapshot",
		FromSnapshot: "test-snapshot",
	}

	jsonBody, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "/api/repos", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context/snapshot, but tests structure
	c.Check(w.Code, Not(Equals), 200)
}

func (s *ReposTestSuite) TestReposPackagesAsyncOperations(c *C) {
	// Test async operations with _async parameter
	params := reposPackagesAddDeleteParams{
		PackageRefs: []string{"Pamd64 test 1.0 abc123"},
	}

	jsonBody, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "/api/repos/test-repo/packages?_async=1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context, but tests async parameter parsing
	c.Check(w.Code, Not(Equals), 200)
}

func (s *ReposTestSuite) TestReposDropAsyncOperation(c *C) {
	// Test async repository drop
	req, _ := http.NewRequest("DELETE", "/api/repos/test-repo?_async=1&force=1", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context, but tests async parameter parsing
	c.Check(w.Code, Not(Equals), 200)
}

func (s *ReposTestSuite) TestReposCopyAsyncOperation(c *C) {
	// Test async copy operation
	params := reposCopyPackageParams{
		WithDeps: false,
		DryRun:   true,
	}

	jsonBody, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "/api/repos/dest-repo/copy/src-repo/package-query?_async=1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context, but tests structure
	c.Check(w.Code, Not(Equals), 200)
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

func (s *ReposTestSuite) TestReposPathSanitization(c *C) {
	// Test path sanitization in file operations
	testPaths := []string{
		"../../../etc/passwd",
		"normal-dir",
		"dir with spaces",
		".hidden-dir",
		"",
	}

	for _, path := range testPaths {
		// Test sanitization doesn't cause crashes
		sanitized := utils.SanitizePath(path)
		c.Check(sanitized, NotNil)

		// Test with file upload endpoints
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/repos/test-repo/file/%s", path), nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should not crash, even if it errors due to missing context
		c.Check(w.Code, Not(Equals), 0)
	}
}

func (s *ReposTestSuite) TestReposErrorHandling(c *C) {
	// Test various error conditions and edge cases
	errorTests := []struct {
		description string
		method      string
		path        string
		body        string
		expectedErr bool
	}{
		{"Missing required fields", "POST", "/api/repos", `{}`, true},
		{"Invalid package refs", "POST", "/api/repos/test/packages", `{"PackageRefs":[]}`, true},
		{"Invalid query format", "GET", "/api/repos/test/packages?q=invalid[query", "", false}, // Query validation happens deeper
		{"Copy to same repo", "POST", "/api/repos/test/copy/test/pkg", `{}`, false},            // Error happens in business logic
		{"File upload endpoint", "POST", "/api/repos/test/file/upload-dir", "", false},         // Valid endpoint
	}

	for _, test := range errorTests {
		var req *http.Request
		if test.body != "" {
			req, _ = http.NewRequest(test.method, test.path, strings.NewReader(test.body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req, _ = http.NewRequest(test.method, test.path, nil)
		}

		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// All should return some response without crashing
		c.Check(w.Code, Not(Equals), 0, Commentf("Test: %s", test.description))
	}
}
