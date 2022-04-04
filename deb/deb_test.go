package deb

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/aptly-dev/aptly/pgp"

	. "gopkg.in/check.v1"
)

type DebSuite struct {
	debFile, debFile2, debFileWithXzControl, debFileWithZstdControl, dscFile, dscFileNoSign string
}

var _ = Suite(&DebSuite{})

func (s *DebSuite) SetUpSuite(c *C) {
	_, _File, _, _ := runtime.Caller(0)
	s.debFile = filepath.Join(filepath.Dir(_File), "../system/files/libboost-program-options-dev_1.49.0.1_i386.deb")
	s.debFile2 = filepath.Join(filepath.Dir(_File), "../system/changes/hardlink_0.2.1_amd64.deb")
	s.debFileWithXzControl = filepath.Join(filepath.Dir(_File), "../system/changes/libqt5concurrent5-dbgsym_5.9.1+dfsg-2+18.04+bionic+build4_amd64.ddeb")
	s.debFileWithZstdControl = filepath.Join(filepath.Dir(_File), "../system/changes/libqt5concurrent5-dbgsym_5.15.2+dfsg-12_amd64.ddeb")
	s.dscFile = filepath.Join(filepath.Dir(_File), "../system/files/pyspi_0.6.1-1.3.dsc")
	s.dscFileNoSign = filepath.Join(filepath.Dir(_File), "../system/files/pyspi-0.6.1-1.3.stripped.dsc")
}

func (s *DebSuite) TestGetControlFileFromDeb(c *C) {
	_, err := GetControlFileFromDeb("/no/such/file")
	c.Check(err, ErrorMatches, ".*no such file or directory")

	_, _File, _, _ := runtime.Caller(0)
	_, err = GetControlFileFromDeb(_File)
	c.Check(err, ErrorMatches, "^.+ar: missing global header")

	st, err := GetControlFileFromDeb(s.debFile)
	c.Check(err, IsNil)
	c.Check(st["Version"], Equals, "1.49.0.1")
	c.Check(st["Package"], Equals, "libboost-program-options-dev")
}

func (s *DebSuite) TestGetControlFileFromDebWithXzControl(c *C) {
	// Has control.tar.xz archive inside.
	st, err := GetControlFileFromDeb(s.debFileWithXzControl)
	c.Check(err, IsNil)
	c.Check(st["Version"], Equals, "5.9.1+dfsg-2+18.04+bionic+build4")
	c.Check(st["Package"], Equals, "libqt5concurrent5-dbgsym")
}

func (s *DebSuite) TestGetControlFileFromDebWithZstdControl(c *C) {
	// Has control.tar.zstd archive inside.
	st, err := GetControlFileFromDeb(s.debFileWithZstdControl)
	c.Check(err, IsNil)
	c.Check(st["Version"], Equals, "5.15.2+dfsg-12")
	c.Check(st["Package"], Equals, "libqt5concurrent5-dbgsym")
}

func (s *DebSuite) TestGetControlFileFromDsc(c *C) {
	verifier := &pgp.GoVerifier{}

	_, err := GetControlFileFromDsc("/no/such/file", verifier)
	c.Check(err, ErrorMatches, ".*no such file or directory")

	_, _File, _, _ := runtime.Caller(0)
	_, err = GetControlFileFromDsc(_File, verifier)
	c.Check(err, ErrorMatches, "malformed stanza syntax")

	st, err := GetControlFileFromDsc(s.dscFile, verifier)
	c.Check(err, IsNil)
	c.Check(st["Version"], Equals, "0.6.1-1.3")
	c.Check(st["Source"], Equals, "pyspi")

	st, err = GetControlFileFromDsc(s.dscFileNoSign, verifier)
	c.Check(err, IsNil)
	c.Check(st["Version"], Equals, "0.6.1-1.4")
	c.Check(st["Source"], Equals, "pyspi")
}

func (s *DebSuite) TestGetContentsFromDeb(c *C) {
	f, err := os.Open(s.debFile)
	c.Assert(err, IsNil)
	contents, err := GetContentsFromDeb(f, s.debFile)
	c.Check(err, IsNil)
	c.Check(contents, DeepEquals, []string{"usr/share/doc/libboost-program-options-dev/changelog.gz",
		"usr/share/doc/libboost-program-options-dev/copyright"})
	c.Assert(f.Close(), IsNil)

	f, err = os.Open(s.debFile2)
	c.Assert(err, IsNil)
	contents, err = GetContentsFromDeb(f, s.debFile2)
	c.Check(err, IsNil)
	c.Check(contents, DeepEquals, []string{"usr/bin/hardlink", "usr/share/man/man1/hardlink.1.gz",
		"usr/share/doc/hardlink/changelog.gz", "usr/share/doc/hardlink/copyright", "usr/share/doc/hardlink/NEWS.Debian.gz"})
	c.Assert(f.Close(), IsNil)
}
