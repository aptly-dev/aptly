package api

import (
	"bytes"
	"encoding/json"

	"github.com/aptly-dev/aptly/deb"
	"github.com/gin-gonic/gin"
	. "gopkg.in/check.v1"
)

type ReposSuite struct {
	APISuite
}

var _ = Suite(&ReposSuite{})

func (s *ReposSuite) TestGetReposIncludesNumPackages(c *C) {
	collectionFactory := s.context.NewCollectionFactory()
	collection := collectionFactory.LocalRepoCollection()
	reflistCollection := collectionFactory.RefListCollection()

	repo := deb.NewLocalRepo("count-repo-list", "")
	repo.UpdateRefList(makePackageRefList(c))
	c.Assert(collection.Add(repo, reflistCollection), IsNil)

	response, err := s.HTTPRequest("GET", "/api/repos", nil)
	c.Assert(err, IsNil)
	c.Assert(response.Code, Equals, 200)

	var repos []map[string]interface{}
	err = json.Unmarshal(response.Body.Bytes(), &repos)
	c.Assert(err, IsNil)

	found := false
	for _, repo := range repos {
		if repo["Name"] == "count-repo-list" {
			found = true
			value, ok := repo["NumPackages"]
			c.Assert(ok, Equals, true)
			c.Assert(value, Equals, float64(2))
			break
		}
	}

	c.Assert(found, Equals, true)
}

func (s *ReposSuite) TestGetReposReturns500OnCorruptRefList(c *C) {
	body, err := json.Marshal(gin.H{"Name": "broken-repo-list"})
	c.Assert(err, IsNil)

	response, err := s.HTTPRequest("POST", "/api/repos", bytes.NewReader(body))
	c.Assert(err, IsNil)
	c.Assert(response.Code, Equals, 201)

	collection := s.context.NewCollectionFactory().LocalRepoCollection()
	repo, err := collection.ByName("broken-repo-list")
	c.Assert(err, IsNil)
	putRawDBValue(c, &s.APISuite, repo.RefKey(), []byte("not-msgpack"))

	response, err = s.HTTPRequest("GET", "/api/repos", nil)
	c.Assert(err, IsNil)
	c.Assert(response.Code, Equals, 500)
	c.Assert(response.Body.String(), Matches, ".*msgpack.*|.*decode.*")
}
