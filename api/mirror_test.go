package api

import (
	"bytes"
	"encoding/json"

	"github.com/aptly-dev/aptly/deb"
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

	var mirrors []map[string]interface{}
	err := json.Unmarshal(response.Body.Bytes(), &mirrors)
	c.Assert(err, IsNil)
}

func (s *MirrorSuite) TestDeleteMirrorNonExisting(c *C) {
	response, _ := s.HTTPRequest("DELETE", "/api/mirrors/does-not-exist", nil)
	c.Check(response.Code, Equals, 404)
	c.Check(response.Body.String(), Equals, "{\"error\":\"unable to drop: mirror with name does-not-exist not found\"}")
}

func (s *MirrorSuite) TestCreateMirrorFlatWithAppStream(c *C) {
	body, err := json.Marshal(gin.H{
		"Name":              "test-flat-appstream",
		"ArchiveURL":        "http://example.com/repo/",
		"Distribution":      "./",
		"DownloadAppStream": true,
	})
	c.Assert(err, IsNil)

	response, err := s.HTTPRequest("POST", "/api/mirrors", bytes.NewReader(body))
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, 400)
	c.Check(response.Body.String(), Matches, ".*AppStream.*flat.*")
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

func (s *MirrorSuite) TestGetMirrorsIncludesNumPackages(c *C) {
	collectionFactory := s.context.NewCollectionFactory()
	collection := collectionFactory.RemoteRepoCollection()
	reflistCollection := collectionFactory.RefListCollection()

	repo, err := deb.NewRemoteRepo("count-mirror", "http://example.com/debian", "stable", []string{"main"}, []string{}, false, false, false, false)
	c.Assert(err, IsNil)

	err = collection.Add(repo, reflistCollection)
	c.Assert(err, IsNil)
	err = reflistCollection.Update(makePackageRefList(c), repo.RefKey())
	c.Assert(err, IsNil)

	response, err := s.HTTPRequest("GET", "/api/mirrors", nil)
	c.Assert(err, IsNil)
	c.Assert(response.Code, Equals, 200)

	var mirrors []map[string]interface{}
	err = json.Unmarshal(response.Body.Bytes(), &mirrors)
	c.Assert(err, IsNil)

	found := false
	for _, mirror := range mirrors {
		if mirror["Name"] == "count-mirror" {
			found = true
			value, ok := mirror["NumPackages"]
			c.Assert(ok, Equals, true)
			c.Assert(value, Equals, float64(2))
			break
		}
	}

	c.Assert(found, Equals, true)
}

func (s *MirrorSuite) TestGetMirrorsReturns500OnCorruptRefList(c *C) {
	collectionFactory := s.context.NewCollectionFactory()
	collection := collectionFactory.RemoteRepoCollection()
	reflistCollection := collectionFactory.RefListCollection()

	repo, err := deb.NewRemoteRepo("broken-mirror", "http://example.com/debian", "stable", []string{"main"}, []string{}, false, false, false, false)
	c.Assert(err, IsNil)
	c.Assert(collection.Add(repo, reflistCollection), IsNil)
	putRawDBValue(c, &s.APISuite, repo.RefKey(), []byte("not-msgpack"))

	response, err := s.HTTPRequest("GET", "/api/mirrors", nil)
	c.Assert(err, IsNil)
	c.Assert(response.Code, Equals, 500)
	c.Assert(response.Body.String(), Matches, ".*unable to show:.*")
}
