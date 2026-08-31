package deb

import (
	. "gopkg.in/check.v1"
)

type IndexFilesSuite struct{}

var _ = Suite(&IndexFilesSuite{})

func (s *IndexFilesSuite) TestHashAlgorithmsDefaultsToAll(c *C) {
	// Unset must behave exactly as aptly did before the set was configurable,
	// so an existing config keeps writing all four.
	for _, files := range []*indexFiles{{}, {byHashAlgorithms: []string{}}} {
		got, err := files.hashAlgorithms()
		c.Check(err, IsNil)
		c.Check(got, DeepEquals, []string{"MD5Sum", "SHA1", "SHA256", "SHA512"})
	}
}

func (s *IndexFilesSuite) TestHashAlgorithmsHonoursTheConfiguredSet(c *C) {
	got, err := (&indexFiles{byHashAlgorithms: []string{"SHA256"}}).hashAlgorithms()
	c.Check(err, IsNil)
	c.Check(got, DeepEquals, []string{"SHA256"})

	got, err = (&indexFiles{byHashAlgorithms: []string{"SHA256", "SHA512"}}).hashAlgorithms()
	c.Check(err, IsNil)
	c.Check(got, DeepEquals, []string{"SHA256", "SHA512"})
}

func (s *IndexFilesSuite) TestHashAlgorithmsIsOrderedIndependentlyOfConfig(c *C) {
	// The directories are created in one loop and the entries written in
	// another. If the order came from the config those two could disagree,
	// so it is fixed here whatever order the user listed.
	got, err := (&indexFiles{byHashAlgorithms: []string{"SHA512", "MD5Sum", "SHA256"}}).hashAlgorithms()
	c.Check(err, IsNil)
	c.Check(got, DeepEquals, []string{"MD5Sum", "SHA256", "SHA512"})
}

func (s *IndexFilesSuite) TestHashAlgorithmsRejectsUnknownNames(c *C) {
	// Loudly, not by skipping. Filtering a typo would produce "no by-hash
	// entries for that algorithm", which is indistinguishable from having
	// deliberately configured it away.
	_, err := (&indexFiles{byHashAlgorithms: []string{"SHA256", "SHA3"}}).hashAlgorithms()
	c.Check(err, ErrorMatches, `unknown Acquire-By-Hash algorithm "SHA3".*`)

	_, err = (&indexFiles{byHashAlgorithms: []string{"sha256"}}).hashAlgorithms()
	c.Check(err, NotNil)
}
