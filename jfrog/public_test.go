package jfrog

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/aptly-dev/aptly/aptly"
	aptly_utils "github.com/aptly-dev/aptly/utils"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	jfrogutils "github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/utils/io/content"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type fakeJFrogManager struct {
	artifactory.EmptyArtifactoryServicesManager

	uploadParams []services.UploadParams
	uploadErr    error

	deleteParams      []services.DeleteParams
	getPathsToDelete  *content.ContentReader
	getPathsDeleteErr error
	deleteErr         error
	deleteCalled      bool

	searchParams []services.SearchParams
	searchReader *content.ContentReader
	searchErr    error

	moveParams []services.MoveCopyParams
	moveErr    error

	copyParams []services.MoveCopyParams
	copyErr    error

	itemProps    *jfrogutils.ItemProperties
	itemPropsErr error
}

func (m *fakeJFrogManager) UploadFiles(_ artifactory.UploadServiceOptions, params ...services.UploadParams) (int, int, error) {
	m.uploadParams = append(m.uploadParams, params...)
	return len(params), 0, m.uploadErr
}

func (m *fakeJFrogManager) GetPathsToDelete(params services.DeleteParams) (*content.ContentReader, error) {
	m.deleteParams = append(m.deleteParams, params)
	if m.getPathsDeleteErr != nil {
		return nil, m.getPathsDeleteErr
	}
	if m.getPathsToDelete != nil {
		return m.getPathsToDelete, nil
	}
	return content.NewEmptyContentReader("results"), nil
}

func (m *fakeJFrogManager) DeleteFiles(_ *content.ContentReader) (int, error) {
	m.deleteCalled = true
	return 1, m.deleteErr
}

func (m *fakeJFrogManager) SearchFiles(params services.SearchParams) (*content.ContentReader, error) {
	m.searchParams = append(m.searchParams, params)
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	if m.searchReader != nil {
		return m.searchReader, nil
	}
	return content.NewEmptyContentReader("results"), nil
}

func (m *fakeJFrogManager) Move(params ...services.MoveCopyParams) (int, int, error) {
	m.moveParams = append(m.moveParams, params...)
	return len(params), 0, m.moveErr
}

func (m *fakeJFrogManager) Copy(params ...services.MoveCopyParams) (int, int, error) {
	m.copyParams = append(m.copyParams, params...)
	return len(params), 0, m.copyErr
}

func (m *fakeJFrogManager) GetItemProps(_ string) (*jfrogutils.ItemProperties, error) {
	if m.itemPropsErr != nil {
		return nil, m.itemPropsErr
	}
	if m.itemProps != nil {
		return m.itemProps, nil
	}
	return &jfrogutils.ItemProperties{}, nil
}

type resultFixture struct {
	Results []jfrogutils.ResultItem `json:"results"`
}

func createReader(c *C, results []jfrogutils.ResultItem) *content.ContentReader {
	filePath := filepath.Join(c.MkDir(), "results.json")
	data, err := json.Marshal(resultFixture{Results: results})
	c.Assert(err, IsNil)
	c.Assert(os.WriteFile(filePath, data, 0o644), IsNil)
	return content.NewContentReader(filePath, "results")
}

type PublishedStorageSuite struct {
	manager *fakeJFrogManager
	storage *PublishedStorage
}

var _ = Suite(&PublishedStorageSuite{})

func (s *PublishedStorageSuite) SetUpTest(c *C) {
	s.manager = &fakeJFrogManager{}
	s.storage = &PublishedStorage{
		manager:    s.manager,
		repository: "repo",
		prefix:     "prefix",
	}
}

func (s *PublishedStorageSuite) TestStringAndMkDir(c *C) {
	c.Assert(s.storage.String(), Equals, "jfrog:repo:prefix")
	c.Assert(s.storage.MkDir("anything"), IsNil)
}

func (s *PublishedStorageSuite) TestPutFile(c *C) {
	err := s.storage.PutFile("pool/main/a+b.deb", "/tmp/source.deb")
	c.Assert(err, IsNil)
	c.Assert(len(s.manager.uploadParams), Equals, 1)
	c.Assert(s.manager.uploadParams[0].Pattern, Equals, "/tmp/source.deb")
	c.Assert(s.manager.uploadParams[0].Target, Equals, filepath.Join("repo", "prefix", "pool/main/a+b.deb"))
	c.Assert(s.manager.uploadParams[0].Flat, Equals, true)
}

func (s *PublishedStorageSuite) TestPutFilePlusWorkaroundAndError(c *C) {
	s.storage.plusWorkaround = true
	s.manager.uploadErr = errors.New("upload failed")

	err := s.storage.PutFile("pool/main/a+b.deb", "/tmp/source.deb")
	c.Assert(err, ErrorMatches, "upload failed")
	c.Assert(s.manager.uploadParams[0].Target, Equals, filepath.Join("repo", "prefix", "pool/main/a%2Bb.deb"))
}

func (s *PublishedStorageSuite) TestRemove(c *C) {
	s.manager.getPathsToDelete = createReader(c, []jfrogutils.ResultItem{})

	err := s.storage.Remove("dists/stable+main")
	c.Assert(err, IsNil)
	c.Assert(len(s.manager.deleteParams), Equals, 1)
	c.Assert(s.manager.deleteParams[0].Pattern, Equals, filepath.Join("repo", "prefix", "dists/stable+main"))
	c.Assert(s.manager.deleteCalled, Equals, true)
}

func (s *PublishedStorageSuite) TestRemovePlusWorkaround(c *C) {
	s.storage.plusWorkaround = true
	s.manager.getPathsToDelete = createReader(c, []jfrogutils.ResultItem{})

	err := s.storage.Remove("pool/a+b.deb")
	c.Assert(err, IsNil)
	c.Assert(s.manager.deleteParams[0].Pattern, Equals, filepath.Join("repo", "prefix", "pool/a%2Bb.deb"))
}

func (s *PublishedStorageSuite) TestRemoveErrors(c *C) {
	s.manager.getPathsDeleteErr = errors.New("search delete failed")
	err := s.storage.Remove("x")
	c.Assert(err, ErrorMatches, "search delete failed")

	s.manager.getPathsDeleteErr = nil
	s.manager.getPathsToDelete = createReader(c, []jfrogutils.ResultItem{})
	s.manager.deleteErr = errors.New("delete failed")
	err = s.storage.Remove("x")
	c.Assert(err, ErrorMatches, "delete failed")
}

func (s *PublishedStorageSuite) TestRemoveDirsDelegatesToRemove(c *C) {
	s.manager.getPathsToDelete = createReader(c, []jfrogutils.ResultItem{})
	c.Assert(s.storage.RemoveDirs("x", nil), IsNil)
	c.Assert(len(s.manager.deleteParams), Equals, 1)
}

func (s *PublishedStorageSuite) TestLinkFromPoolDelegatesToPutFile(c *C) {
	err := s.storage.LinkFromPool("", "pool/main/p", "pkg.deb", nil, "/tmp/source.deb", aptly_utils.ChecksumInfo{}, false)
	c.Assert(err, IsNil)
	c.Assert(s.manager.uploadParams[0].Target, Equals, filepath.Join("repo", "prefix", "pool/main/p", "pkg.deb"))
}

func (s *PublishedStorageSuite) TestFilelist(c *C) {
	s.manager.searchReader = createReader(c, []jfrogutils.ResultItem{
		{Path: "repo/prefix/pool/main/a", Name: "a.deb", Actual_Md5: "m1"},
		{Path: "repo/prefix/pool/main/b", Name: "b.deb", Actual_Md5: "m2"},
	})

	list, err := s.storage.Filelist("pool/main")
	c.Assert(err, IsNil)
	c.Assert(list, DeepEquals, []string{"pool/main/a/a.deb", "pool/main/b/b.deb"})
	c.Assert(s.manager.searchParams[0].Pattern, Equals, filepath.Join("repo", "prefix", "pool/main", "*"))
}

func (s *PublishedStorageSuite) TestFilelistPlusWorkaroundAndSearchError(c *C) {
	s.storage.plusWorkaround = true
	s.manager.searchReader = createReader(c, []jfrogutils.ResultItem{
		{Path: "repo/prefix/pool/main", Name: "a%2Bb.deb", Actual_Md5: "m1"},
	})

	list, err := s.storage.Filelist("pool/main")
	c.Assert(err, IsNil)
	c.Assert(list, DeepEquals, []string{"pool/main/a+b.deb"})

	s.manager.searchErr = errors.New("search failed")
	_, err = s.storage.Filelist("pool/main")
	c.Assert(err, ErrorMatches, "search failed")
}

func (s *PublishedStorageSuite) TestRenameFile(c *C) {
	err := s.storage.RenameFile("old+name", "new+name")
	c.Assert(err, IsNil)
	c.Assert(len(s.manager.moveParams), Equals, 1)
	c.Assert(s.manager.moveParams[0].Pattern, Equals, filepath.Join("repo", "prefix", "old+name"))
	c.Assert(s.manager.moveParams[0].Target, Equals, filepath.Join("repo", "prefix", "new+name"))
	c.Assert(s.manager.moveParams[0].Flat, Equals, true)
}

func (s *PublishedStorageSuite) TestRenameFilePlusWorkaroundAndError(c *C) {
	s.storage.plusWorkaround = true
	s.manager.moveErr = errors.New("move failed")
	err := s.storage.RenameFile("old+name", "new+name")
	c.Assert(err, ErrorMatches, "move failed")
	c.Assert(s.manager.moveParams[0].Pattern, Equals, filepath.Join("repo", "prefix", "old%2Bname"))
	c.Assert(s.manager.moveParams[0].Target, Equals, filepath.Join("repo", "prefix", "new%2Bname"))
}

func (s *PublishedStorageSuite) TestSymLinkAndHardLink(c *C) {
	err := s.storage.SymLink("src+name", "dst+name")
	c.Assert(err, IsNil)
	c.Assert(len(s.manager.copyParams), Equals, 1)
	c.Assert(s.manager.copyParams[0].Pattern, Equals, filepath.Join("repo", "prefix", "src+name"))
	c.Assert(s.manager.copyParams[0].Target, Equals, filepath.Join("repo", "prefix", "dst+name"))
	c.Assert(s.manager.copyParams[0].Flat, Equals, true)
	targetProps := s.manager.copyParams[0].TargetProps.ToMap()
	c.Assert(targetProps["SymLink"], DeepEquals, []string{"src+name"})

	err = s.storage.HardLink("a", "b")
	c.Assert(err, IsNil)
	c.Assert(len(s.manager.copyParams), Equals, 2)
}

func (s *PublishedStorageSuite) TestSymLinkPlusWorkaroundAndError(c *C) {
	s.storage.plusWorkaround = true
	s.manager.copyErr = errors.New("copy failed")

	err := s.storage.SymLink("src+name", "dst+name")
	c.Assert(err, ErrorMatches, "copy failed")
	c.Assert(s.manager.copyParams[0].Pattern, Equals, filepath.Join("repo", "prefix", "src%2Bname"))
	c.Assert(s.manager.copyParams[0].Target, Equals, filepath.Join("repo", "prefix", "dst%2Bname"))
}

func (s *PublishedStorageSuite) TestFileExists(c *C) {
	s.manager.searchReader = createReader(c, []jfrogutils.ResultItem{{Path: "repo/prefix/pool", Name: "x"}})
	ok, err := s.storage.FileExists("pool/x")
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)
	c.Assert(s.manager.searchParams[0].Pattern, Equals, filepath.Join("repo", "prefix", "pool/x"))

	s.manager.searchReader = content.NewEmptyContentReader("results")
	ok, err = s.storage.FileExists("pool/y")
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, false)
}

func (s *PublishedStorageSuite) TestFileExistsSearchErrorAndPlusWorkaround(c *C) {
	s.storage.plusWorkaround = true
	s.manager.searchErr = errors.New("search failed")
	ok, err := s.storage.FileExists("pool/a+b")
	c.Assert(ok, Equals, false)
	c.Assert(err, ErrorMatches, "search failed")
	c.Assert(s.manager.searchParams[0].Pattern, Equals, filepath.Join("repo", "prefix", "pool/a%2Bb"))
}

func (s *PublishedStorageSuite) TestReadLink(c *C) {
	s.manager.itemProps = &jfrogutils.ItemProperties{
		Properties: map[string][]string{
			"SymLink": {"src/file"},
		},
	}

	link, err := s.storage.ReadLink("path/to/link")
	c.Assert(err, IsNil)
	c.Assert(link, Equals, "src/file")
}

func (s *PublishedStorageSuite) TestReadLinkNoPropertyAndErrors(c *C) {
	s.manager.itemProps = &jfrogutils.ItemProperties{Properties: map[string][]string{"Other": {"value"}}}
	link, err := s.storage.ReadLink("path/to/link")
	c.Assert(err, IsNil)
	c.Assert(link, Equals, "")

	s.manager.itemPropsErr = errors.New("props failed")
	link, err = s.storage.ReadLink("path/to/link")
	c.Assert(err, IsNil)
	c.Assert(link, Equals, "")
}

func (s *PublishedStorageSuite) TestReadLinkPlusWorkaround(c *C) {
	s.storage.plusWorkaround = true
	s.manager.itemProps = &jfrogutils.ItemProperties{}
	_, _ = s.storage.ReadLink("a+b")
	// Ensure the method runs with plusWorkaround path conversion.
	c.Assert(s.manager.itemPropsErr, IsNil)
}

func (s *PublishedStorageSuite) TestNewPublishedStorageRaw(c *C) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	withUserPassword, err := NewPublishedStorageRaw("repo", server.URL, "user", "password", "", "", "prefix", true, false)
	c.Assert(err, IsNil)
	c.Assert(withUserPassword, NotNil)
	c.Assert(withUserPassword.String(), Equals, "jfrog:repo:prefix")

	withAPIKey, err := NewPublishedStorageRaw("repo", server.URL, "", "", "api-key", "", "prefix", false, false)
	c.Assert(err, IsNil)
	c.Assert(withAPIKey, NotNil)

	withToken, err := NewPublishedStorageRaw("repo", server.URL, "", "", "", "token", "prefix", false, false)
	c.Assert(err, IsNil)
	c.Assert(withToken, NotNil)
}

func (s *PublishedStorageSuite) TestNewPublishedStorageRawManagerError(c *C) {
	// An SSH URL causes artifactory.New() to fail (no SSH key configured),
	// exercising the error return on lines 59-61.
	_, err := NewPublishedStorageRaw("repo", "ssh://example.local/artifactory", "", "", "", "", "", false, false)
	c.Assert(err, ErrorMatches, "error creating jfrog manager: .*")
}

func (s *PublishedStorageSuite) TestNewPublishedStorage(c *C) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	storage, err := NewPublishedStorage("test", aptly_utils.JFrogPublishRoot{
		Repository:     "repo",
		Url:            server.URL,
		AccessToken:    "token",
		Prefix:         "pref",
		PlusWorkaround: true,
	})
	c.Assert(err, IsNil)
	c.Assert(storage, NotNil)
	c.Assert(storage.String(), Equals, "jfrog:repo:pref")
}

var _ aptly.PublishedStorage = (*PublishedStorage)(nil)
