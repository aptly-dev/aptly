package utils

import (
	"fmt"
	"log"
	"os"
	"testing"

	. "gopkg.in/check.v1"
)

// Launch gocheck tests
func Test(t *testing.T) {
	TestingT(t)
}

type UtilsSuite struct {
	tempfile *os.File
}

var _ = Suite(&UtilsSuite{})

func (s *UtilsSuite) SetUpSuite(c *C) {
	s.tempfile, _ = os.CreateTemp(c.MkDir(), "aptly-test-inaccessible")
	if err := os.Chmod(s.tempfile.Name(), 0000); err != nil {
		log.Fatalln(err)
	}
}

func (s *UtilsSuite) TestDirIsAccessibleNotExist(c *C) {
	c.Check(DirIsAccessible("does/not/exist.invalid"), Equals, nil)
}

func (s *UtilsSuite) TestDirIsAccessibleNotAccessible(c *C) {
	c.Check(DirIsAccessible(s.tempfile.Name()).Error(), Equals, fmt.Errorf("'%s' is inaccessible, check access rights", s.tempfile.Name()).Error())
}
