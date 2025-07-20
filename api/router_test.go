package api

import (
	. "gopkg.in/check.v1"
)

type RouterSuite struct {
	APISuite
}

var _ = Suite(&RouterSuite{})

func (s *RouterSuite) TestRedirectSwagger(c *C) {
	// Test redirect from /docs to /docs/index.html
	response, _ := s.HTTPRequest("GET", "/docs", nil)
	c.Check(response.Code, Equals, 301)
	c.Check(response.Header().Get("Location"), Equals, "/docs/")
}