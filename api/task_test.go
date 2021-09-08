package api

import (
	"encoding/json"
	"fmt"

	"github.com/aptly-dev/aptly/task"

	. "gopkg.in/check.v1"
)

type TaskSuite struct {
	ApiSuite
}

var _ = Suite(&TaskSuite{})

func (s *TaskSuite) TestTasksDummy(c *C) {
	response, _ := s.HTTPRequest("POST", "/api/tasks-dummy", nil)
	c.Check(response.Code, Equals, 418)
	c.Check(response.Body.String(), Equals, "[1,2,3]")
}

func (s *TaskSuite) TestTasksDummyAsync(c *C) {
	response, _ := s.HTTPRequest("POST", "/api/tasks-dummy?_async=true", nil)
	c.Check(response.Code, Equals, 202)
	var t task.Task
	err := json.Unmarshal(response.Body.Bytes(), &t)
	c.Assert(err, IsNil)
	c.Check(t.Name, Equals, "Dummy task")
	response, _ = s.HTTPRequest("GET", fmt.Sprintf("/api/tasks/%d/wait", t.ID), nil)
	err = json.Unmarshal(response.Body.Bytes(), &t)
	c.Assert(err, IsNil)
	c.Check(t.State, Equals, task.SUCCEEDED)
	response, _ = s.HTTPRequest("GET", fmt.Sprintf("/api/tasks/%d/detail", t.ID), nil)
	c.Check(response.Code, Equals, 200)
	c.Check(response.Body.String(), Equals, "[1,2,3]")
	response, _ = s.HTTPRequest("GET", fmt.Sprintf("/api/tasks/%d/output", t.ID), nil)
	c.Check(response.Code, Equals, 200)
	c.Check(response.Body.String(), Matches, "\"Dummy task started.*")
}

func (s *TaskSuite) TestTaskDelete(c *C) {
	response, _ := s.HTTPRequest("POST", "/api/tasks-dummy?_async=true", nil)
	c.Check(response.Code, Equals, 202)
	c.Check(response.Body.String(), Equals, "{\"Name\":\"Dummy task\",\"ID\":1,\"State\":0}")
	response, _ = s.HTTPRequest("DELETE", "/api/tasks/1", nil)
	c.Check(response.Code, Equals, 200)
}

func (s *TaskSuite) TestTasksClear(c *C) {
	response, _ := s.HTTPRequest("POST", "/api/tasks-dummy?_async=true", nil)
	c.Check(response.Code, Equals, 202)
	var t task.Task
	err := json.Unmarshal(response.Body.Bytes(), &t)
	c.Assert(err, IsNil)
	c.Check(t.Name, Equals, "Dummy task")
	response, _ = s.HTTPRequest("GET", "/api/tasks-wait", nil)
	c.Check(response.Code, Equals, 200)
	response, _ = s.HTTPRequest("GET", "/api/tasks", nil)
	c.Check(response.Code, Equals, 200)
	var ts []task.Task
	err = json.Unmarshal(response.Body.Bytes(), &ts)
	c.Assert(err, IsNil)
	c.Check(len(ts), Equals, 1)
	c.Check(ts[0].State, Equals, task.SUCCEEDED)
	response, _ = s.HTTPRequest("POST", "/api/tasks-clear", nil)
	c.Check(response.Code, Equals, 200)
	response, _ = s.HTTPRequest("GET", "/api/tasks", nil)
	c.Check(response.Code, Equals, 200)
	c.Check(response.Body.String(), Equals, "null")
}
