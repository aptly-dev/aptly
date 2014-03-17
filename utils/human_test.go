package utils

import (
	. "launchpad.net/gocheck"
)

type HumanSuite struct{}

var _ = Suite(&HumanSuite{})

func (s *HumanSuite) TestHumanBytes(c *C) {
	c.Check(HumanBytes(50), Equals, "50 B")
	c.Check(HumanBytes(968), Equals, "0.95 KiB")
	c.Check(HumanBytes(20480), Equals, "20.00 KiB")
	c.Check(HumanBytes(700480), Equals, "0.67 MiB")
	c.Check(HumanBytes(7000480), Equals, "6.68 MiB")
	c.Check(HumanBytes(824000480), Equals, "0.77 GiB")
	c.Check(HumanBytes(82400000480), Equals, "76.74 GiB")
	c.Check(HumanBytes(824000000480), Equals, "0.75 TiB")
	c.Check(HumanBytes(824000000000480), Equals, "749.42 TiB")
}
