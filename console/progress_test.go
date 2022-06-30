package console

import (
	"fmt"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

type ProgressSuite struct {}

var _ = Suite(&ProgressSuite{})

func (s *ProgressSuite) TestNewProgress(c *C) {
	p := NewProgress(false)
	c.Check(fmt.Sprintf("%T", p.worker), Equals, fmt.Sprintf("%T", &standardProgressWorker{}))

	p = NewProgress(true)
	c.Check(fmt.Sprintf("%T", p.worker), Equals, fmt.Sprintf("%T", &loggerProgressWorker{}))
}
