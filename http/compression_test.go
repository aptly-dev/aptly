package http

import (
	"errors"
	"io"
	"net/url"

	"github.com/smira/aptly/utils"

	. "gopkg.in/check.v1"
)

type CompressionSuite struct {
	baseURL *url.URL
}

var _ = Suite(&CompressionSuite{})

const (
	bzipData = "BZh91AY&SY\xcc\xc3q\xd4\x00\x00\x02A\x80\x00\x10\x02\x00\x0c\x00 \x00!\x9ah3M\x19\x97\x8b\xb9\"\x9c(Hfa\xb8\xea\x00"
	gzipData = "\x1f\x8b\x08\x00\xc8j\xb0R\x00\x03+I-.\xe1\x02\x00\xc65\xb9;\x05\x00\x00\x00"
	xzData   = "\xfd\x37\x7a\x58\x5a\x00\x00\x04\xe6\xd6\xb4\x46\x02\x00\x21\x01\x16\x00\x00\x00\x74\x2f\xe5\xa3\x01\x00\x04\x74\x65\x73\x74\x0a\x00\x00\x00\x00\x9d\xed\x31\x1d\x0f\x9f\xd7\xe6\x00\x01\x1d\x05\xb8\x2d\x80\xaf\x1f\xb6\xf3\x7d\x01\x00\x00\x00\x00\x04\x59\x5a"
	rawData  = "test"
)

func (s *CompressionSuite) SetUpTest(c *C) {
	s.baseURL, _ = url.Parse("http://example.com/")
}

func (s *CompressionSuite) TestDownloadTryCompression(c *C) {
	var buf []byte

	expectedChecksums := map[string]utils.ChecksumInfo{
		"file.bz2": {Size: int64(len(bzipData))},
		"file.gz":  {Size: int64(len(gzipData))},
		"file.xz":  {Size: int64(len(xzData))},
		"file":     {Size: int64(len(rawData))},
	}

	// bzip2 only available
	buf = make([]byte, 4)
	d := NewFakeDownloader()
	d.ExpectResponse("http://example.com/file.bz2", bzipData)
	r, file, err := DownloadTryCompression(d, s.baseURL, "file", expectedChecksums, false, 1)
	c.Assert(err, IsNil)
	defer file.Close()
	io.ReadFull(r, buf)
	c.Assert(string(buf), Equals, rawData)
	c.Assert(d.Empty(), Equals, true)

	// bzip2 not available, but gz is
	buf = make([]byte, 4)
	d = NewFakeDownloader()
	d.ExpectError("http://example.com/file.bz2", &Error{Code: 404})
	d.ExpectResponse("http://example.com/file.gz", gzipData)
	r, file, err = DownloadTryCompression(d, s.baseURL, "file", expectedChecksums, false, 1)
	c.Assert(err, IsNil)
	defer file.Close()
	io.ReadFull(r, buf)
	c.Assert(string(buf), Equals, rawData)
	c.Assert(d.Empty(), Equals, true)

	// bzip2 & gzip not available, but xz is
	buf = make([]byte, 4)
	d = NewFakeDownloader()
	d.ExpectError("http://example.com/file.bz2", &Error{Code: 404})
	d.ExpectError("http://example.com/file.gz", &Error{Code: 404})
	d.ExpectResponse("http://example.com/file.xz", xzData)
	r, file, err = DownloadTryCompression(d, s.baseURL, "file", expectedChecksums, false, 1)
	c.Assert(err, IsNil)
	defer file.Close()
	io.ReadFull(r, buf)
	c.Assert(string(buf), Equals, rawData)
	c.Assert(d.Empty(), Equals, true)

	// bzip2, gzip & xz not available, but raw is
	buf = make([]byte, 4)
	d = NewFakeDownloader()
	d.ExpectError("http://example.com/file.bz2", &Error{Code: 404})
	d.ExpectError("http://example.com/file.gz", &Error{Code: 404})
	d.ExpectError("http://example.com/file.xz", &Error{Code: 404})
	d.ExpectResponse("http://example.com/file", rawData)
	r, file, err = DownloadTryCompression(d, s.baseURL, "file", expectedChecksums, false, 1)
	c.Assert(err, IsNil)
	defer file.Close()
	io.ReadFull(r, buf)
	c.Assert(string(buf), Equals, rawData)
	c.Assert(d.Empty(), Equals, true)

	// gzip available, but broken
	d = NewFakeDownloader()
	d.ExpectError("http://example.com/file.bz2", &Error{Code: 404})
	d.ExpectResponse("http://example.com/file.gz", "x")
	_, _, err = DownloadTryCompression(d, s.baseURL, "file", nil, true, 1)
	c.Assert(err, ErrorMatches, "unexpected EOF")
	c.Assert(d.Empty(), Equals, true)
}

func (s *CompressionSuite) TestDownloadTryCompressionLongestSuffix(c *C) {
	var buf []byte

	expectedChecksums := map[string]utils.ChecksumInfo{
		"file.bz2":          {Size: 1},
		"subdir/file.bz2":   {Size: int64(len(bzipData))},
		"otherdir/file.bz2": {Size: 1},
	}

	// longest suffix should be picked up
	buf = make([]byte, 4)
	d := NewFakeDownloader()
	d.ExpectResponse("http://example.com/subdir/file.bz2", bzipData)
	r, file, err := DownloadTryCompression(d, s.baseURL, "subdir/file", expectedChecksums, false, 1)
	c.Assert(err, IsNil)
	defer file.Close()
	io.ReadFull(r, buf)
	c.Assert(string(buf), Equals, rawData)
	c.Assert(d.Empty(), Equals, true)
}

func (s *CompressionSuite) TestDownloadTryCompressionErrors(c *C) {
	d := NewFakeDownloader()
	_, _, err := DownloadTryCompression(d, s.baseURL, "file", nil, true, 1)
	c.Assert(err, ErrorMatches, "unexpected request.*")

	d = NewFakeDownloader()
	d.ExpectError("http://example.com/file.bz2", &Error{Code: 404})
	d.ExpectError("http://example.com/file.gz", &Error{Code: 404})
	d.ExpectError("http://example.com/file.xz", &Error{Code: 404})
	d.ExpectError("http://example.com/file", errors.New("403"))
	_, _, err = DownloadTryCompression(d, s.baseURL, "file", nil, true, 1)
	c.Assert(err, ErrorMatches, "403")

	d = NewFakeDownloader()
	d.ExpectError("http://example.com/file.bz2", &Error{Code: 404})
	d.ExpectError("http://example.com/file.gz", &Error{Code: 404})
	d.ExpectError("http://example.com/file.xz", &Error{Code: 404})
	d.ExpectResponse("http://example.com/file", rawData)
	expectedChecksums := map[string]utils.ChecksumInfo{
		"file.bz2": {Size: 7},
		"file.gz":  {Size: 7},
		"file.xz":  {Size: 7},
		"file":     {Size: 7},
	}
	_, _, err = DownloadTryCompression(d, s.baseURL, "file", expectedChecksums, false, 1)
	c.Assert(err, ErrorMatches, "checksums don't match.*")
}
