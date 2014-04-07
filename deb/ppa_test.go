package deb

import (
	"github.com/smira/aptly/utils"
	. "launchpad.net/gocheck"
)

type PpaSuite struct {
	config utils.ConfigStructure
}

var _ = Suite(&PpaSuite{})

func (s *PpaSuite) TestParsePPA(c *C) {
	_, _, _, err := ParsePPA("ppa:dedeed", &s.config)
	c.Check(err, ErrorMatches, "unable to parse ppa URL.*")

	s.config.PpaDistributorID = "debian"
	s.config.PpaCodename = "wheezy"

	url, distribution, components, err := ParsePPA("ppa:user/project", &s.config)
	c.Check(err, IsNil)
	c.Check(url, Equals, "http://ppa.launchpad.net/user/project/debian")
	c.Check(distribution, Equals, "wheezy")
	c.Check(components, DeepEquals, []string{"main"})
}
