package api

import (
	"bytes"
	"encoding/json"

	"github.com/gin-gonic/gin"
	. "gopkg.in/check.v1"
)

type MirrorSuite struct {
	APISuite
}

var _ = Suite(&MirrorSuite{})

func (s *MirrorSuite) TestGetMirrors(c *C) {
	response, _ := s.HTTPRequest("GET", "/api/mirrors", nil)
	c.Check(response.Code, Equals, 200)
	c.Check(response.Body.String(), Equals, "[]")
}

func (s *MirrorSuite) TestDeleteMirrorNonExisting(c *C) {
	response, _ := s.HTTPRequest("DELETE", "/api/mirrors/does-not-exist", nil)
	c.Check(response.Code, Equals, 404)
	c.Check(response.Body.String(), Equals, "{\"error\":\"unable to drop: mirror with name does-not-exist not found\"}")
}

func (s *MirrorSuite) TestCreateMirror(c *C) {
	c.ExpectFailure("Need to mock downloads")
	body, err := json.Marshal(gin.H{
		"Name":       "dummy",
		"ArchiveURL": "foobar",
	})
	c.Assert(err, IsNil)
	response, err := s.HTTPRequest("POST", "/api/mirrors", bytes.NewReader(body))
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, 400)
	c.Check(response.Body.String(), Equals, "")
}

func (s *MirrorSuite) TestMirrorShow(c *C) {
	// Test showing a specific mirror
	response, _ := s.HTTPRequest("GET", "/api/mirrors/test-mirror", nil)
	c.Check(response.Code, Equals, 404)
}

func (s *MirrorSuite) TestMirrorUpdate(c *C) {
	// Test updating a mirror
	body, _ := json.Marshal(gin.H{
		"ArchiveURL": "http://new.archive.url/debian",
	})
	response, _ := s.HTTPRequest("PUT", "/api/mirrors/test-mirror", bytes.NewReader(body))
	c.Check(response.Code, Equals, 404)
}

func (s *MirrorSuite) TestMirrorPackages(c *C) {
	// Test listing packages in a mirror
	response, _ := s.HTTPRequest("GET", "/api/mirrors/test-mirror/packages", nil)
	c.Check(response.Code, Equals, 404)
}

func (s *MirrorSuite) TestMirrorUpdateRun(c *C) {
	// Test running mirror update
	response, _ := s.HTTPRequest("PUT", "/api/mirrors/test-mirror/update", nil)
	c.Check(response.Code, Equals, 404)
}
