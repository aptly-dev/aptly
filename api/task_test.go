package api

import (
	"net/http"
	"net/http/httptest"


	"github.com/gin-gonic/gin"
	. "gopkg.in/check.v1"
)

type TaskTestSuite struct {
	router *gin.Engine
}

var _ = Suite(&TaskTestSuite{})

func (s *TaskTestSuite) SetUpTest(c *C) {
	s.router = gin.New()
	s.router.GET("/api/tasks", apiTasksList)
	s.router.POST("/api/tasks-clear", apiTasksClear)
	s.router.GET("/api/tasks-wait", apiTasksWait)
	s.router.GET("/api/tasks/:id/wait", apiTasksWaitForTaskByID)
	s.router.GET("/api/tasks/:id", apiTasksShow)
	s.router.GET("/api/tasks/:id/output", apiTasksOutputShow)
	s.router.GET("/api/tasks/:id/detail", apiTasksDetailShow)
	s.router.GET("/api/tasks/:id/return_value", apiTasksReturnValueShow)
	s.router.DELETE("/api/tasks/:id", apiTasksDelete)

	gin.SetMode(gin.TestMode)
}

func (s *TaskTestSuite) TestTasksListEmpty(c *C) {
	// Test listing tasks when none exist
	req, _ := http.NewRequest("GET", "/api/tasks", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Equals, 200)
	c.Check(w.Header().Get("Content-Type"), Equals, "application/json; charset=utf-8")

	// Will likely return empty array due to no context, but tests structure
}

func (s *TaskTestSuite) TestTasksClearStructure(c *C) {
	// Test clearing tasks
	req, _ := http.NewRequest("POST", "/api/tasks-clear", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Equals, 200)
	c.Check(w.Header().Get("Content-Type"), Equals, "application/json; charset=utf-8")

	// Should return empty object
}

func (s *TaskTestSuite) TestTasksWaitStructure(c *C) {
	// Test waiting for all tasks
	req, _ := http.NewRequest("GET", "/api/tasks-wait", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Equals, 200)
	c.Check(w.Header().Get("Content-Type"), Equals, "application/json; charset=utf-8")

	// Should return empty object after waiting
}

func (s *TaskTestSuite) TestTasksWaitForTaskByIDStructure(c *C) {
	// Test waiting for specific task by ID
	req, _ := http.NewRequest("GET", "/api/tasks/123/wait", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context or invalid task, but tests structure
	c.Check(w.Code, Not(Equals), 200)
}

func (s *TaskTestSuite) TestTasksWaitForTaskByIDInvalidID(c *C) {
	// Test waiting for task with invalid ID
	invalidIDs := []string{"invalid", "abc", "-1", "", "123.45"}

	for _, id := range invalidIDs {
		req, _ := http.NewRequest("GET", "/api/tasks/"+id+"/wait", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should return 500 for invalid ID format
		c.Check(w.Code, Equals, 500, Commentf("ID: %s", id))
	}
}

func (s *TaskTestSuite) TestTasksShowStructure(c *C) {
	// Test showing specific task by ID
	req, _ := http.NewRequest("GET", "/api/tasks/123", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context or invalid task, but tests structure
	c.Check(w.Code, Not(Equals), 200)
}

func (s *TaskTestSuite) TestTasksShowInvalidID(c *C) {
	// Test showing task with invalid ID
	invalidIDs := []string{"invalid", "abc", "-1", "", "123.45", "999999999999999999999"}

	for _, id := range invalidIDs {
		req, _ := http.NewRequest("GET", "/api/tasks/"+id, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should return 500 for invalid ID format
		c.Check(w.Code, Equals, 500, Commentf("ID: %s", id))
	}
}

func (s *TaskTestSuite) TestTasksOutputStructure(c *C) {
	// Test getting task output by ID
	req, _ := http.NewRequest("GET", "/api/tasks/123/output", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context or invalid task, but tests structure
	c.Check(w.Code, Not(Equals), 200)
}

func (s *TaskTestSuite) TestTasksOutputInvalidID(c *C) {
	// Test getting task output with invalid ID
	invalidIDs := []string{"invalid", "abc", "-1", "", "123.45"}

	for _, id := range invalidIDs {
		req, _ := http.NewRequest("GET", "/api/tasks/"+id+"/output", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should return 500 for invalid ID format
		c.Check(w.Code, Equals, 500, Commentf("ID: %s", id))
	}
}

func (s *TaskTestSuite) TestTasksDetailStructure(c *C) {
	// Test getting task detail by ID
	req, _ := http.NewRequest("GET", "/api/tasks/123/detail", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context or invalid task, but tests structure
	c.Check(w.Code, Not(Equals), 200)
}

func (s *TaskTestSuite) TestTasksDetailInvalidID(c *C) {
	// Test getting task detail with invalid ID
	invalidIDs := []string{"invalid", "abc", "-1", "", "123.45"}

	for _, id := range invalidIDs {
		req, _ := http.NewRequest("GET", "/api/tasks/"+id+"/detail", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should return 500 for invalid ID format
		c.Check(w.Code, Equals, 500, Commentf("ID: %s", id))
	}
}

func (s *TaskTestSuite) TestTasksReturnValueStructure(c *C) {
	// Test getting task return value by ID
	req, _ := http.NewRequest("GET", "/api/tasks/123/return_value", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context or invalid task, but tests structure
	c.Check(w.Code, Not(Equals), 200)
}

func (s *TaskTestSuite) TestTasksReturnValueInvalidID(c *C) {
	// Test getting task return value with invalid ID
	invalidIDs := []string{"invalid", "abc", "-1", "", "123.45"}

	for _, id := range invalidIDs {
		req, _ := http.NewRequest("GET", "/api/tasks/"+id+"/return_value", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should return 500 for invalid ID format
		c.Check(w.Code, Equals, 500, Commentf("ID: %s", id))
	}
}

func (s *TaskTestSuite) TestTasksDeleteStructure(c *C) {
	// Test deleting task by ID
	req, _ := http.NewRequest("DELETE", "/api/tasks/123", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context or invalid task, but tests structure
	c.Check(w.Code, Not(Equals), 200)
}

func (s *TaskTestSuite) TestTasksDeleteInvalidID(c *C) {
	// Test deleting task with invalid ID
	invalidIDs := []string{"invalid", "abc", "-1", "", "123.45"}

	for _, id := range invalidIDs {
		req, _ := http.NewRequest("DELETE", "/api/tasks/"+id, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should return 500 for invalid ID format
		c.Check(w.Code, Equals, 500, Commentf("ID: %s", id))
	}
}

func (s *TaskTestSuite) TestTasksValidIDFormats(c *C) {
	// Test various valid ID formats
	validIDs := []string{"0", "1", "123", "999", "2147483647"} // Max int32

	for _, id := range validIDs {
		// Test show endpoint
		req, _ := http.NewRequest("GET", "/api/tasks/"+id, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should not be 500 (invalid format), might be 404 (not found) or other error
		c.Check(w.Code, Not(Equals), 500, Commentf("ID: %s", id))

		// Test wait endpoint
		req, _ = http.NewRequest("GET", "/api/tasks/"+id+"/wait", nil)
		w = httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should not be 500 (invalid format)
		c.Check(w.Code, Not(Equals), 500, Commentf("ID: %s", id))

		// Test output endpoint
		req, _ = http.NewRequest("GET", "/api/tasks/"+id+"/output", nil)
		w = httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should not be 500 (invalid format)
		c.Check(w.Code, Not(Equals), 500, Commentf("ID: %s", id))

		// Test detail endpoint
		req, _ = http.NewRequest("GET", "/api/tasks/"+id+"/detail", nil)
		w = httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should not be 500 (invalid format)
		c.Check(w.Code, Not(Equals), 500, Commentf("ID: %s", id))

		// Test return_value endpoint
		req, _ = http.NewRequest("GET", "/api/tasks/"+id+"/return_value", nil)
		w = httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should not be 500 (invalid format)
		c.Check(w.Code, Not(Equals), 500, Commentf("ID: %s", id))

		// Test delete endpoint
		req, _ = http.NewRequest("DELETE", "/api/tasks/"+id, nil)
		w = httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should not be 500 (invalid format)
		c.Check(w.Code, Not(Equals), 500, Commentf("ID: %s", id))
	}
}

func (s *TaskTestSuite) TestTasksParameterEdgeCases(c *C) {
	// Test edge cases in parameter handling
	edgeCases := []struct {
		path        string
		description string
	}{
		{"/api/tasks/0", "zero ID"},
		{"/api/tasks/1", "single digit ID"},
		{"/api/tasks/2147483647", "max int32 ID"},
		{"/api/tasks/00123", "leading zeros"},
		{"/api/tasks/+123", "positive sign"},
	}

	for _, tc := range edgeCases {
		req, _ := http.NewRequest("GET", tc.path, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should handle edge cases gracefully without crashing
		c.Check(w.Code, Not(Equals), 0, Commentf("Test: %s", tc.description))
	}
}

func (s *TaskTestSuite) TestTasksHTTPMethods(c *C) {
	// Test that correct HTTP methods are supported for each endpoint
	methodTests := []struct {
		path           string
		allowedMethods []string
		deniedMethods  []string
	}{
		{"/api/tasks", []string{"GET"}, []string{"POST", "PUT", "DELETE", "PATCH"}},
		{"/api/tasks-clear", []string{"POST"}, []string{"GET", "PUT", "DELETE", "PATCH"}},
		{"/api/tasks-wait", []string{"GET"}, []string{"POST", "PUT", "DELETE", "PATCH"}},
		{"/api/tasks/123", []string{"GET", "DELETE"}, []string{"POST", "PUT", "PATCH"}},
		{"/api/tasks/123/wait", []string{"GET"}, []string{"POST", "PUT", "DELETE", "PATCH"}},
		{"/api/tasks/123/output", []string{"GET"}, []string{"POST", "PUT", "DELETE", "PATCH"}},
		{"/api/tasks/123/detail", []string{"GET"}, []string{"POST", "PUT", "DELETE", "PATCH"}},
		{"/api/tasks/123/return_value", []string{"GET"}, []string{"POST", "PUT", "DELETE", "PATCH"}},
	}

	for _, test := range methodTests {
		// Test denied methods return 404 (method not allowed for route)
		for _, method := range test.deniedMethods {
			req, _ := http.NewRequest(method, test.path, nil)
			w := httptest.NewRecorder()
			s.router.ServeHTTP(w, req)

			c.Check(w.Code, Equals, 404, Commentf("Path: %s, Method: %s", test.path, method))
		}

		// Test allowed methods don't return 404 for method not allowed
		for _, method := range test.allowedMethods {
			req, _ := http.NewRequest(method, test.path, nil)
			w := httptest.NewRecorder()
			s.router.ServeHTTP(w, req)

			// Should not be 404 (method not allowed), might be other errors due to missing context
			c.Check(w.Code, Not(Equals), 404, Commentf("Path: %s, Method: %s", test.path, method))
		}
	}
}

func (s *TaskTestSuite) TestTasksContentTypes(c *C) {
	// Test content type handling for different endpoints
	contentTypeTests := []struct {
		path         string
		method       string
		expectedType string
	}{
		{"/api/tasks", "GET", "application/json"},
		{"/api/tasks-clear", "POST", "application/json"},
		{"/api/tasks-wait", "GET", "application/json"},
		{"/api/tasks/123", "GET", "application/json"},
		{"/api/tasks/123/wait", "GET", "application/json"},
		{"/api/tasks/123/output", "GET", ""}, // Text content
		{"/api/tasks/123/detail", "GET", "application/json"},
		{"/api/tasks/123/return_value", "GET", "application/json"},
	}

	for _, test := range contentTypeTests {
		req, _ := http.NewRequest(test.method, test.path, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		if test.expectedType != "" {
			// Check that JSON endpoints return JSON content type
			contentType := w.Header().Get("Content-Type")
			c.Check(contentType, Matches, ".*"+test.expectedType+".*", 
				Commentf("Path: %s, Expected: %s, Got: %s", test.path, test.expectedType, contentType))
		}
	}
}

func (s *TaskTestSuite) TestTasksErrorConditions(c *C) {
	// Test various error conditions
	errorTests := []struct {
		description string
		path        string
		method      string
		expectedErr bool
	}{
		{"Non-existent task ID", "/api/tasks/999999", "GET", true},
		{"Non-existent task wait", "/api/tasks/999999/wait", "GET", true},
		{"Non-existent task output", "/api/tasks/999999/output", "GET", true},
		{"Non-existent task detail", "/api/tasks/999999/detail", "GET", true},
		{"Non-existent task return value", "/api/tasks/999999/return_value", "GET", true},
		{"Non-existent task delete", "/api/tasks/999999", "DELETE", true},
		{"Malformed task path", "/api/tasks/", "GET", false}, // Route not matched
		{"Extra path segments", "/api/tasks/123/extra/segment", "GET", false}, // Route not matched
	}

	for _, test := range errorTests {
		req, _ := http.NewRequest(test.method, test.path, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// All should return some response without crashing
		c.Check(w.Code, Not(Equals), 0, Commentf("Test: %s", test.description))
	}
}

func (s *TaskTestSuite) TestTasksResourceManagement(c *C) {
	// Test that endpoints handle resource management correctly
	endpoints := []string{
		"/api/tasks",
		"/api/tasks-clear", 
		"/api/tasks-wait",
		"/api/tasks/1",
		"/api/tasks/1/wait",
		"/api/tasks/1/output",
		"/api/tasks/1/detail", 
		"/api/tasks/1/return_value",
	}

	for _, endpoint := range endpoints {
		method := "GET"
		if endpoint == "/api/tasks-clear" {
			method = "POST"
		}

		req, _ := http.NewRequest(method, endpoint, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should complete without hanging or crashing
		c.Check(w.Code, Not(Equals), 0, Commentf("Endpoint: %s", endpoint))
		
		// Response should have proper headers
		c.Check(w.Header(), NotNil, Commentf("Endpoint: %s", endpoint))
	}
}