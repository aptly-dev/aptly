package context

import (
	"reflect"
	"testing"

	"github.com/smira/flag"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type fatalErrorPanicChecker struct {
	*CheckerInfo
}

var FatalErrorPanicMatches Checker = &fatalErrorPanicChecker{
	&CheckerInfo{Name: "FatalErrorPanics", Params: []string{"function", "expected"}},
}

func (checker *fatalErrorPanicChecker) Check(params []interface{}, names []string) (result bool, errmsg string) {
	f := reflect.ValueOf(params[0])
	if f.Kind() != reflect.Func || f.Type().NumIn() != 0 {
		return false, "Function must take zero arguments"
	}
	defer func() {
		if errmsg != "" {
			return
		}

		obtained := recover()

		names[0] = "panic"

		var ok bool

		var e1 *FatalError
		if e1, ok = obtained.(*FatalError); ok {
			params[0] = e1
		} else {
			errmsg = "Panic value is not FatalError"
			return
		}

		var e2 *FatalError
		if e2, ok = params[1].(*FatalError); ok {
			params[1] = e2
		} else {
			errmsg = "Expected value is not FatalError"
			return
		}

		if *e1 == *e2 {
			result = true
		} else {
			result = false
			errmsg = "Not equal"
		}
	}()
	f.Call(nil)
	return false, "Function has not panicked"
}

type AptlyContextSuite struct {
	context *AptlyContext
}

var _ = Suite(&AptlyContextSuite{})

func (s *AptlyContextSuite) SetUpTest(c *C) {
	flags := flag.NewFlagSet("fakeFlags", flag.ContinueOnError)
	flags.String("config", "", "")
	context, err := NewContext(flags)
	c.Assert(err, IsNil)
	s.context = context
}

func (s *AptlyContextSuite) TestGetPublishedStorageBadFS(c *C) {
	// https://github.com/aptly-dev/aptly/issues/711
	// This will fail on account of us not having a config, so the
	// storage never exists.
	c.Assert(func() { s.context.GetPublishedStorage("filesystem:fuji") },
		FatalErrorPanicMatches,
		&FatalError{ReturnCode: 1, Message: "published local storage fuji not configured"})
}
