package debian

import (
	"github.com/smira/aptly/utils"
	. "launchpad.net/gocheck"
)

type PpaSuite struct {
	savedConfig utils.ConfigStructure
}

var _ = Suite(&PpaSuite{})

func (s *PpaSuite) SetUpTest(c *C) {
	s.savedConfig = utils.Config
}

func (s *PpaSuite) TearDownTest(c *C) {
	utils.Config = s.savedConfig
}

func (s *PpaSuite) TestParsePPA(c *C) {
	_, _, _, err := ParsePPA("ppa:dedeed")
	c.Check(err, ErrorMatches, "unable to parse ppa URL.*")

	utils.Config.PpaDistributorID = "debian"
	utils.Config.PpaCodename = "wheezy"

	url, distribution, components, err := ParsePPA("ppa:user/project")
	c.Check(err, IsNil)
	c.Check(url, Equals, "http://ppa.launchpad.net/user/project/debian")
	c.Check(distribution, Equals, "wheezy")
	c.Check(components, DeepEquals, []string{"main"})
}
