package api

import (
	"bytes"
	"encoding/json"
	
	"github.com/gin-gonic/gin"
	. "gopkg.in/check.v1"
)

type PackagesSuite struct {
	APISuite
}

var _ = Suite(&PackagesSuite{})

func (s *PackagesSuite) TestPackageShow(c *C) {
	// Test showing a specific package
	response, _ := s.HTTPRequest("GET", "/api/packages/Pamd64%20test%201.0%20abc123", nil)
	// Will return 404 as the package doesn't exist
	c.Check(response.Code, Equals, 404)
}

func (s *PackagesSuite) TestPackagesList(c *C) {
	// Test listing all packages
	response, _ := s.HTTPRequest("GET", "/api/packages", nil)
	c.Check(response.Code, Equals, 200)
	
	var result []interface{}
	err := json.Unmarshal(response.Body.Bytes(), &result)
	c.Check(err, IsNil)
	c.Check(result, NotNil)
}

func (s *PackagesSuite) TestPackagesGetMaximumVersion(c *C) {
	// Create dummy repo first
	body, _ := json.Marshal(gin.H{"Name": "dummy"})
	resp, err := s.HTTPRequest("POST", "/api/repos", bytes.NewReader(body))
	c.Assert(err, IsNil)
	c.Check(resp.Code, Equals, 201)
	
	// Now test packages with maximumVersion
	response, err := s.HTTPRequest("GET", "/api/repos/dummy/packages?maximumVersion=1", nil)
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, 200)
	c.Check(response.Body.String(), Equals, "[]")
	
	// Clean up
	resp, err = s.HTTPRequest("DELETE", "/api/repos/dummy?force=1", nil)
	c.Assert(err, IsNil)
	c.Check(resp.Code, Equals, 200)
}
