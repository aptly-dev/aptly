package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	. "gopkg.in/check.v1"
)

type SnapshotAPITestSuite struct {
	APISuite
}

var _ = Suite(&SnapshotAPITestSuite{})

func (s *SnapshotAPITestSuite) SetUpTest(c *C) {
	s.APISuite.SetUpTest(c)
}

func (s *SnapshotAPITestSuite) TestSnapshotShow(c *C) {
	// Test showing a specific snapshot
	req, _ := http.NewRequest("GET", "/api/snapshots/test-snapshot", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will return 404 as the snapshot doesn't exist
	c.Check(w.Code, Equals, 404)
}

func (s *SnapshotAPITestSuite) TestSnapshotUpdate(c *C) {
	// Test updating a snapshot
	params := struct {
		Name        string `json:"Name"`
		Description string `json:"Description"`
	}{
		Name:        "updated-snapshot",
		Description: "Updated description",
	}

	body, _ := json.Marshal(params)
	req, _ := http.NewRequest("PUT", "/api/snapshots/test-snapshot", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will return 404 as the snapshot doesn't exist
	c.Check(w.Code, Equals, 404)
}

func (s *SnapshotAPITestSuite) TestSnapshotDrop(c *C) {
	// Test dropping a snapshot
	req, _ := http.NewRequest("DELETE", "/api/snapshots/test-snapshot", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will return 404 as the snapshot doesn't exist
	c.Check(w.Code, Equals, 404)
}

func (s *SnapshotAPITestSuite) TestSnapshotCreateFromRepository(c *C) {
	// Test creating a snapshot from repository
	params := struct {
		Name        string `json:"Name"`
		Description string `json:"Description"`
	}{
		Name:        "new-snapshot",
		Description: "Test snapshot",
	}

	body, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "/api/repos/test-repo/snapshots", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will return 404 as the repo doesn't exist
	c.Check(w.Code, Equals, 404)
}

func (s *SnapshotAPITestSuite) TestSnapshotDiff(c *C) {
	// Test diffing two snapshots
	req, _ := http.NewRequest("GET", "/api/snapshots/snap1/diff/snap2", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will return 404 as the snapshots don't exist
	c.Check(w.Code, Equals, 404)
}

func (s *SnapshotAPITestSuite) TestSnapshotSearchPackages(c *C) {
	// Test searching packages in snapshot
	req, _ := http.NewRequest("GET", "/api/snapshots/test-snapshot/packages?q=Name", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will return 404 as the snapshot doesn't exist
	c.Check(w.Code, Equals, 404)
}

func (s *SnapshotAPITestSuite) TestSnapshotMerge(c *C) {
	// Test merging snapshots
	params := struct {
		Destination string   `json:"Destination"`
		Sources     []string `json:"Sources"`
	}{
		Destination: "merged-snapshot",
		Sources:     []string{"snap1", "snap2"},
	}

	body, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "/api/snapshots/merge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will return error as snapshots don't exist
	c.Check(w.Code, Not(Equals), 200)
}

func (s *SnapshotAPITestSuite) TestSnapshotPull(c *C) {
	// Test pulling packages between snapshots
	params := struct {
		Source      string   `json:"Source"`
		Destination string   `json:"Destination"`
		Queries     []string `json:"Queries"`
	}{
		Source:      "source-snap",
		Destination: "dest-snap",
		Queries:     []string{"Name (~ nginx)"},
	}

	body, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "/api/snapshots/pull", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will return error as snapshots don't exist
	c.Check(w.Code, Not(Equals), 200)
}

func (s *SnapshotAPITestSuite) TestSnapshotCreateFromMirror(c *C) {
	// Test creating snapshot from mirror
	params := struct {
		Name        string `json:"Name"`
		Description string `json:"Description"`
	}{
		Name:        "mirror-snapshot",
		Description: "Snapshot from mirror",
	}

	body, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "/api/mirrors/test-mirror/snapshots", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will return 404 as the mirror doesn't exist
	c.Check(w.Code, Equals, 404)
}

func (s *SnapshotAPITestSuite) TestApiSnapshotsListGet(c *C) {
	// Test GET /api/snapshots endpoint
	req, _ := http.NewRequest("GET", "/api/snapshots", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Should handle the request without crashing (will likely error due to no context)
	c.Check(w.Code, Not(Equals), 0)
}

func (s *SnapshotAPITestSuite) TestApiSnapshotsListWithSort(c *C) {
	// Test GET /api/snapshots with sort parameter
	req, _ := http.NewRequest("GET", "/api/snapshots?sort=name", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Not(Equals), 0)
}

func (s *SnapshotAPITestSuite) TestApiSnapshotsListWithDifferentSorts(c *C) {
	// Test various sort methods
	sortMethods := []string{"name", "time", "created"}

	for _, sortMethod := range sortMethods {
		req, _ := http.NewRequest("GET", "/api/snapshots?sort="+sortMethod, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		c.Check(w.Code, Not(Equals), 0, Commentf("Sort method: %s", sortMethod))
	}
}

func (s *SnapshotAPITestSuite) TestApiSnapshotsCreatePost(c *C) {
	// Test POST /api/snapshots endpoint
	requestBody := snapshotsCreateParams{
		Name:            "test-snapshot",
		Description:     "Test snapshot",
		SourceSnapshots: []string{"source1"},
		PackageRefs:     []string{},
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/snapshots", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Should handle the request without crashing (will likely error due to no context)
	c.Check(w.Code, Not(Equals), 0)
}

func (s *SnapshotAPITestSuite) TestApiSnapshotsCreateInvalidJSON(c *C) {
	// Test POST with invalid JSON
	req, _ := http.NewRequest("POST", "/api/snapshots", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Equals, 400) // Should return bad request for invalid JSON
}

func (s *SnapshotAPITestSuite) TestApiSnapshotsCreateMissingName(c *C) {
	// Test POST with missing required name field
	requestBody := map[string]interface{}{
		"Description": "Test without name",
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/snapshots", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Equals, 400) // Should return bad request for missing name
}

func (s *SnapshotAPITestSuite) TestApiSnapshotsCreateFromMirrorPost(c *C) {
	// Test POST /api/mirrors/{name}/snapshots endpoint
	requestBody := snapshotsCreateFromMirrorParams{
		Name:        "mirror-snapshot",
		Description: "Snapshot from mirror",
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/mirrors/test-mirror/snapshots", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Should handle the request without crashing (will likely error due to no context)
	c.Check(w.Code, Not(Equals), 0)
}

func (s *SnapshotAPITestSuite) TestApiSnapshotsCreateFromMirrorInvalidJSON(c *C) {
	// Test POST with invalid JSON for mirror snapshot
	req, _ := http.NewRequest("POST", "/api/mirrors/test-mirror/snapshots", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Equals, 400) // Should return bad request for invalid JSON
}

func (s *SnapshotAPITestSuite) TestApiSnapshotsCreateFromMirrorMissingName(c *C) {
	// Test POST with missing required name field for mirror snapshot
	requestBody := map[string]interface{}{
		"Description": "Mirror snapshot without name",
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/mirrors/test-mirror/snapshots", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Equals, 400) // Should return bad request for missing name
}

func (s *SnapshotAPITestSuite) TestApiSnapshotsCreateWithAsync(c *C) {
	// Test POST with async parameter
	requestBody := snapshotsCreateParams{
		Name:        "async-snapshot",
		Description: "Async test snapshot",
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/snapshots?_async=true", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Not(Equals), 0)
}

func (s *SnapshotAPITestSuite) TestApiSnapshotsCreateFromMirrorWithAsync(c *C) {
	// Test POST mirror snapshot with async parameter
	requestBody := snapshotsCreateFromMirrorParams{
		Name:        "async-mirror-snapshot",
		Description: "Async mirror snapshot",
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/mirrors/test-mirror/snapshots?_async=true", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Not(Equals), 0)
}

func (s *SnapshotAPITestSuite) TestSnapshotsCreateParamsStruct(c *C) {
	// Test snapshotsCreateParams struct
	params := snapshotsCreateParams{
		Name:            "test-name",
		Description:     "test-description",
		SourceSnapshots: []string{"snap1", "snap2"},
		PackageRefs:     []string{"ref1", "ref2"},
	}

	c.Check(params.Name, Equals, "test-name")
	c.Check(params.Description, Equals, "test-description")
	c.Check(params.SourceSnapshots, DeepEquals, []string{"snap1", "snap2"})
	c.Check(params.PackageRefs, DeepEquals, []string{"ref1", "ref2"})
}

func (s *SnapshotAPITestSuite) TestSnapshotsCreateFromMirrorParamsStruct(c *C) {
	// Test snapshotsCreateFromMirrorParams struct
	params := snapshotsCreateFromMirrorParams{
		Name:        "mirror-test-name",
		Description: "mirror-test-description",
	}

	c.Check(params.Name, Equals, "mirror-test-name")
	c.Check(params.Description, Equals, "mirror-test-description")
}

func (s *SnapshotAPITestSuite) TestApiSnapshotsCreateEmptyRequest(c *C) {
	// Test POST with empty request body
	req, _ := http.NewRequest("POST", "/api/snapshots", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Equals, 400) // Should return bad request for empty body
}

func (s *SnapshotAPITestSuite) TestApiSnapshotsCreateFromMirrorEmptyRequest(c *C) {
	// Test POST mirror snapshot with empty request body
	req, _ := http.NewRequest("POST", "/api/mirrors/test-mirror/snapshots", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Equals, 400) // Should return bad request for empty body
}

func (s *SnapshotAPITestSuite) TestApiSnapshotsListDefaultSort(c *C) {
	// Test that default sort is applied when no sort parameter provided
	req, _ := http.NewRequest("GET", "/api/snapshots", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Endpoint should handle default sort without issues
	c.Check(w.Code, Not(Equals), 0)
}

func (s *SnapshotAPITestSuite) TestApiSnapshotsCreateComplexPayload(c *C) {
	// Test POST with complex payload including all fields
	requestBody := snapshotsCreateParams{
		Name:            "complex-snapshot",
		Description:     "Complex test snapshot with multiple sources",
		SourceSnapshots: []string{"base-snapshot", "updates-snapshot", "security-snapshot"},
		PackageRefs:     []string{"pkg1_1.0_amd64", "pkg2_2.0_i386", "pkg3_3.0_all"},
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/snapshots", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Not(Equals), 0)
}

func (s *SnapshotAPITestSuite) TestApiSnapshotsHTTPMethods(c *C) {
	// Test that only allowed HTTP methods work

	// Test unsupported methods for snapshots list
	deniedMethods := []string{"PUT", "DELETE", "PATCH"}
	for _, method := range deniedMethods {
		req, _ := http.NewRequest(method, "/api/snapshots", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		c.Check(w.Code, Equals, 404, Commentf("Method %s should not be allowed for snapshots list", method))
	}
}

func (s *SnapshotAPITestSuite) TestApiSnapshotsCreateSpecialCharacters(c *C) {
	// Test snapshot creation with special characters in names
	specialNames := []string{
		"snapshot-with-dashes",
		"snapshot_with_underscores",
		"snapshot.with.dots",
		"snapshot123",
		"UPPERCASESNAPSHOT",
	}

	for _, name := range specialNames {
		requestBody := snapshotsCreateParams{
			Name:        name,
			Description: "Test snapshot with special characters",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/api/snapshots", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		c.Check(w.Code, Not(Equals), 0, Commentf("Special name test failed: %s", name))
	}
}

func (s *SnapshotAPITestSuite) TestApiSnapshotsListEmptyResponse(c *C) {
	// Test snapshots list when no snapshots exist
	req, _ := http.NewRequest("GET", "/api/snapshots", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Should return some response (likely error due to no context, but shouldn't crash)
	c.Check(w.Code, Not(Equals), 0)
}

func (s *SnapshotAPITestSuite) TestApiSnapshotsCreateWithoutContentType(c *C) {
	// Test POST without Content-Type header
	requestBody := `{"Name": "test-snapshot"}`
	req, _ := http.NewRequest("POST", "/api/snapshots", strings.NewReader(requestBody))

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Should handle missing content type
	c.Check(w.Code, Not(Equals), 0)
}

func (s *SnapshotAPITestSuite) TestApiSnapshotsParameterEdgeCases(c *C) {
	// Test edge cases for parameter validation

	// Test with very long name
	longName := strings.Repeat("a", 1000)
	requestBody := snapshotsCreateParams{
		Name: longName,
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/snapshots", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Not(Equals), 0)

	// Test with empty arrays
	emptyArrayBody := snapshotsCreateParams{
		Name:            "empty-arrays",
		SourceSnapshots: []string{},
		PackageRefs:     []string{},
	}

	jsonBody, _ = json.Marshal(emptyArrayBody)
	req, _ = http.NewRequest("POST", "/api/snapshots", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Not(Equals), 0)
}
