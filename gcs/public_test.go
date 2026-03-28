package gcs

import (
	"path/filepath"

	"github.com/aptly-dev/aptly/utils"
	. "gopkg.in/check.v1"
)

type PublishedStorageSuite struct{}

var _ = Suite(&PublishedStorageSuite{})

func (s *PublishedStorageSuite) TestString(c *C) {
	storage := &PublishedStorage{bucketName: "bucket-1", prefix: "prefix/a"}
	c.Check(storage.String(), Equals, "GCS: bucket-1/prefix/a")
}

func (s *PublishedStorageSuite) TestObjectPath(c *C) {
	storage := &PublishedStorage{prefix: "root"}
	c.Check(storage.objectPath("dists/stable/Release"), Equals, filepath.Join("root", "dists/stable/Release"))
}

func (s *PublishedStorageSuite) TestApplyACLNoOpModes(c *C) {
	for _, acl := range []string{"", "none", "private"} {
		storage := &PublishedStorage{acl: acl}
		err := storage.applyACL(nil)
		c.Check(err, IsNil)
	}
}

func (s *PublishedStorageSuite) TestApplyACLUnsupported(c *C) {
	storage := &PublishedStorage{acl: "bucket-owner-full-control"}
	err := storage.applyACL(nil)
	c.Assert(err, NotNil)
	c.Check(err, ErrorMatches, "unsupported GCS ACL value: bucket-owner-full-control")
}

func (s *PublishedStorageSuite) TestLinkFromPoolMissingMD5(c *C) {
	publishedPrefix := "repo"
	publishedRelPath := "pool/main/a/aptly"
	fileName := "pkg.deb"
	relPath := filepath.Join(filepath.Join(publishedPrefix, publishedRelPath), fileName)

	storage := &PublishedStorage{pathCache: map[string]string{relPath: "0123456789abcdef0123456789abcdef"}}

	err := storage.LinkFromPool(publishedPrefix, publishedRelPath, fileName, nil, "", utils.ChecksumInfo{}, false)
	c.Assert(err, NotNil)
	c.Check(err, ErrorMatches, "unable to compare object, MD5 checksum missing")
}

func (s *PublishedStorageSuite) TestLinkFromPoolDifferentMD5NoForce(c *C) {
	publishedPrefix := "repo"
	publishedRelPath := "pool/main/a/aptly"
	fileName := "pkg.deb"
	relPath := filepath.Join(filepath.Join(publishedPrefix, publishedRelPath), fileName)

	storage := &PublishedStorage{pathCache: map[string]string{relPath: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}

	err := storage.LinkFromPool(publishedPrefix, publishedRelPath, fileName, nil, "", utils.ChecksumInfo{MD5: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}, false)
	c.Assert(err, NotNil)
	c.Check(err, ErrorMatches, ".*file already exists and is different.*")
}

func (s *PublishedStorageSuite) TestLinkFromPoolSameMD5NoUpload(c *C) {
	publishedPrefix := "repo"
	publishedRelPath := "pool/main/a/aptly"
	fileName := "pkg.deb"
	relPath := filepath.Join(filepath.Join(publishedPrefix, publishedRelPath), fileName)

	storage := &PublishedStorage{pathCache: map[string]string{relPath: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}

	err := storage.LinkFromPool(publishedPrefix, publishedRelPath, fileName, nil, "", utils.ChecksumInfo{MD5: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}, false)
	c.Check(err, IsNil)
}
