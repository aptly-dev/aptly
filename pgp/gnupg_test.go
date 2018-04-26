package pgp

import (
	"os"
	"path/filepath"
	"runtime"

	. "gopkg.in/check.v1"
)

type GnupgSuite struct {
	verifier Verifier
	bins     string
}

var _ = Suite(&GnupgSuite{})

func (s *GnupgSuite) SetUpSuite(c *C) {
	_, _File, _, _ := runtime.Caller(0)
	s.bins = filepath.Join(filepath.Dir(_File), "test-bins")
}

// If gpg == gpg1 = pick gpg
func (s *GnupgSuite) TestGPG1(c *C) {
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", filepath.Join(s.bins, "gpg1"))
	defer func() { os.Setenv("PATH", origPath) }()

	signer := NewGpgSigner()
	c.Assert(signer.gpg, Equals, "gpg")
}

// gpg(2) + gpg1 installed = pick gpg1
func (s *GnupgSuite) TestGPG1Not2(c *C) {
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", filepath.Join(s.bins, "gpg2-and-1"))
	defer func() { os.Setenv("PATH", origPath) }()

	signer := NewGpgSigner()
	c.Assert(signer.gpg, Equals, "gpg1")
}

// If gpg == gpg2 and no gpg1 is available = error
func (s *GnupgSuite) TestGPGNothing(c *C) {
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", filepath.Join(s.bins, "gpg2-only"))
	defer func() { os.Setenv("PATH", origPath) }()

	c.Assert(func() { NewGpgSigner() }, PanicMatches, `Couldn't find a suitable gpg executable.+`)
}

// If gpgv == gpgv1 = pick gpgv
func (s *GnupgSuite) TestGPGV1(c *C) {
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", filepath.Join(s.bins, "gpgv1")+":"+filepath.Join(s.bins, "gpg1"))
	defer func() { os.Setenv("PATH", origPath) }()

	verifier := NewGpgVerifier()
	c.Assert(verifier.gpgv, Equals, "gpgv")
}

// gpgv(2) + gpgv1 installed = pick gpgv1
func (s *GnupgSuite) TestGPGV1Not2(c *C) {
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", filepath.Join(s.bins, "gpgv2-and-1")+":"+filepath.Join(s.bins, "gpg1"))
	defer func() { os.Setenv("PATH", origPath) }()

	verifier := NewGpgVerifier()
	c.Assert(verifier.gpgv, Equals, "gpgv1")
}

// If gpgv == gpgv2 and no gpgv1 is available = error
func (s *GnupgSuite) TestGPGVNothing(c *C) {
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", filepath.Join(s.bins, "gpgv2-only")+":"+filepath.Join(s.bins, "gpg1"))
	defer func() { os.Setenv("PATH", origPath) }()

	c.Assert(func() { NewGpgVerifier() }, PanicMatches, `Couldn't find a suitable gpgv executable.+`)
}
