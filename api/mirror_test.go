package api

import (
	"bytes"
	"encoding/json"

	"github.com/gin-gonic/gin"
	. "gopkg.in/check.v1"
)

type MirrorSuite struct {
	ApiSuite
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
