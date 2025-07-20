package api

import (
	"encoding/json"
	"net/http/httptest"

	"github.com/aptly-dev/aptly/deb"
	"github.com/gin-gonic/gin"
	. "gopkg.in/check.v1"
)

type ApiPackagesSuite struct {
	APISuite
}

var _ = Suite(&ApiPackagesSuite{})

func (s *ApiPackagesSuite) TestShowPackages(c *C) {
	// Test showPackages function with nil reflist
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequest("GET", "/api/test", nil)
	
	showPackages(ginCtx, nil, s.context.NewCollectionFactory())
	
	// Should return 404 for nil reflist
	c.Check(w.Code, Equals, 404)
}

func (s *ApiPackagesSuite) TestShowPackagesWithEmptyList(c *C) {
	// Test showPackages with empty package reflist
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequest("GET", "/api/test", nil)
	
	reflist := deb.NewPackageRefList()
	showPackages(ginCtx, reflist, s.context.NewCollectionFactory())
	
	c.Check(w.Code, Equals, 200)
	
	var result []string
	err := json.Unmarshal(w.Body.Bytes(), &result)
	c.Check(err, IsNil)
	c.Check(len(result), Equals, 0)
}

func (s *ApiPackagesSuite) TestShowPackagesCompact(c *C) {
	// Test showPackages with compact format (default)
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequest("GET", "/api/test", nil)
	
	reflist := deb.NewPackageRefList()
	showPackages(ginCtx, reflist, s.context.NewCollectionFactory())
	
	c.Check(w.Code, Equals, 200)
}

func (s *ApiPackagesSuite) TestShowPackagesDetails(c *C) {
	// Test showPackages with details format
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequest("GET", "/api/test?format=details", nil)
	
	reflist := deb.NewPackageRefList()
	showPackages(ginCtx, reflist, s.context.NewCollectionFactory())
	
	c.Check(w.Code, Equals, 200)
	
	var result []*deb.Package
	err := json.Unmarshal(w.Body.Bytes(), &result)
	c.Check(err, IsNil)
	c.Check(len(result), Equals, 0)
}