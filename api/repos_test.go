package api

import (
	"fmt"
	"github.com/aptly-dev/aptly/deb"
	"net/http"

	. "gopkg.in/check.v1"
)

type ReposSuite struct {
	ApiSuite
}

const (
	repoMainName string = "test-api-repo-repo1"
	repoAuxName  string = "test-api-repo-repo2"
)

var _ = Suite(&ReposSuite{})

func (s *ReposSuite) SetUpSuite(c *C) {
	err := s.setupContext()
	c.Assert(err, IsNil)

	collection := s.context.NewCollectionFactory().LocalRepoCollection()
	repo1 := deb.NewLocalRepo(repoMainName, "Testing purpose repo 1")
	err = collection.Add(repo1)
	c.Assert(err, IsNil)

	repo2 := deb.NewLocalRepo(repoAuxName, "Testing purpose repo 2")
	err = collection.Add(repo2)
	c.Assert(err, IsNil)

	// TODO : Add some packages inside repo1
}

func (s *ReposSuite) TestReposCopyPackage(c *C) {
	response, err := s.HTTPRequest(http.MethodPost,
		fmt.Sprintf("/repos/%s/copy/%s/package1", repoAuxName, repoMainName), nil)
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, http.StatusOK)
	// TODO : Make sure that the package is available in repo2.
}
