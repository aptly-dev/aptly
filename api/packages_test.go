package api

import (
	. "gopkg.in/check.v1"
)

type PackagesSuite struct {
	ApiSuite
}

var _ = Suite(&PackagesSuite{})

func (s *PackagesSuite) TestPackagesGetMaximumVersion(c *C) {
	response, err := s.HTTPRequest("GET", "/api/repos/dummy/packages?maximumVersion=1", nil)
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, 200)
	c.Check(response.Body.String(), Equals, "[]")
}
