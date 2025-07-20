package api

import (
	. "gopkg.in/check.v1"
)

type S3Suite struct {
	APISuite
}

var _ = Suite(&S3Suite{})

func (s *S3Suite) TestS3List(c *C) {
	// Test listing S3 endpoints
	response, _ := s.HTTPRequest("GET", "/api/s3", nil)
	c.Check(response.Code, Equals, 200)
	c.Check(response.Header().Get("Content-Type"), Equals, "application/json; charset=utf-8")
}