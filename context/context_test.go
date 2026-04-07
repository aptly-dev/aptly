package context

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/aptly-dev/aptly/utils"
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
	prevConfig := utils.Config
	defer func() { utils.Config = prevConfig }()

	s.context.configLoaded = true
	utils.Config.FileSystemPublishRoots = map[string]utils.FileSystemPublishRoot{}

	c.Assert(func() { s.context.GetPublishedStorage("filesystem:fuji") },
		FatalErrorPanicMatches,
		&FatalError{ReturnCode: 1, Message: "published local storage fuji not configured"})
}

func (s *AptlyContextSuite) TestGetPublishedStorageJFrogConfigured(c *C) {
	prevConfig := utils.Config
	defer func() { utils.Config = prevConfig }()

	s.context.configLoaded = true
	utils.Config.RootDir = c.MkDir()
	utils.Config.JFrogPublishRoots = map[string]utils.JFrogPublishRoot{
		"test": {
			Repository:  "aptly-repo",
			Url:         "https://example.jfrog.local/artifactory",
			AccessToken: "token",
			Prefix:      "public",
		},
	}

	storage := s.context.GetPublishedStorage("jfrog:test")
	c.Assert(storage, NotNil)
	c.Assert(fmt.Sprintf("%v", storage), Equals, "jfrog:aptly-repo:public")

	// Ensure we get the cached object on repeated lookups.
	storageAgain := s.context.GetPublishedStorage("jfrog:test")
	c.Assert(storageAgain, Equals, storage)
}

func (s *AptlyContextSuite) TestGetPublishedStorageJFrogMissing(c *C) {
	prevConfig := utils.Config
	defer func() { utils.Config = prevConfig }()

	s.context.configLoaded = true
	utils.Config.JFrogPublishRoots = map[string]utils.JFrogPublishRoot{}

	c.Assert(func() { s.context.GetPublishedStorage("jfrog:missing") },
		FatalErrorPanicMatches,
		&FatalError{ReturnCode: 1, Message: "published JFrog storage missing not configured"})
}
