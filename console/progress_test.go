package console

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/aptly-dev/aptly/utils"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

type ProgressSuite struct {}

var _ = Suite(&ProgressSuite{})

func (s *ProgressSuite) TestProgressWorkerFactroy(c *C) {
	l := utils.LoggerFactory("default", "debug")
	p := NewProgress(&l)
	c.Check(fmt.Sprintf("%T", p.worker), Equals, fmt.Sprintf("%T", &standardProgressWorker{}))

	l = utils.LoggerFactory("json", "debug")
	p = NewProgress(&l)
	c.Check(fmt.Sprintf("%T", p.worker), Equals, fmt.Sprintf("%T", &loggerProgressWorker{}))
}

func (s *ProgressSuite) TestProgressLoggerWorker(c *C) {
	l := utils.LoggerFactory("default", "debug")
	p := NewProgress(&l)
	c.Check(fmt.Sprintf("%T", p.worker), Equals, fmt.Sprintf("%T", &standardProgressWorker{}))
	p.Start()

}

func (s *ProgressSuite) TestProgressLoggerWorkerPrintf(c *C) {
	r, w, err := os.Pipe()
	if err != nil {
		c.Error(err.Error())
	}

	zl := utils.NewZeroJSONLoggerWithWriter("debug", w)
	l := utils.Logger(zl)
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		fmt.Println(buf.String())
		outC <- buf.String()
	}()
	
	p := NewProgress(&l)
	p.Start()

	expected := "TestProgressMessage"
	p.Printf(expected)
	p.Flush()

	w.Close()
	capturedOutput := <-outC

	var jsonMap map[string]interface{}
	json.Unmarshal([]byte(capturedOutput), &jsonMap)

	if val, ok := jsonMap["message"]; ok {
		c.Check(val, Equals, expected)
	} else {
		c.Errorf("Log message didn't have a 'message' key, obtained %s", capturedOutput)
	}
}

func (s *ProgressSuite) TestProgressLoggerWorkerPrintfStdErr(c *C) {
	r, w, err := os.Pipe()
	if err != nil {
		c.Error(err.Error())
	}

	zl := utils.NewZeroJSONLoggerWithWriter("debug", w)
	l := utils.Logger(zl)
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		fmt.Println(buf.String())
		outC <- buf.String()
	}()
	
	p := NewProgress(&l)
	p.Start()

	expected := "TestProgressMessage"
	p.PrintfStdErr(expected)
	p.Flush()

	w.Close()
	capturedOutput := <-outC

	var jsonMap map[string]interface{}
	json.Unmarshal([]byte(capturedOutput), &jsonMap)

	if val, ok := jsonMap["message"]; ok {
		c.Check(val, Equals, expected)
	} else {
		c.Errorf("Log message didn't have a 'message' key, obtained %s", capturedOutput)
	}
}

func (s *ProgressSuite) TestProgressLoggerWorkerColoredPrintf(c *C) {
	r, w, err := os.Pipe()
	if err != nil {
		c.Error(err.Error())
	}

	zl := utils.NewZeroJSONLoggerWithWriter("debug", w)
	l := utils.Logger(zl)
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		fmt.Println(buf.String())
		outC <- buf.String()
	}()
	
	p := NewProgress(&l)
	p.Start()

	expected := "TestProgressMessage"
	p.ColoredPrintf(expected)
	p.Flush()

	w.Close()
	capturedOutput := <-outC

	var jsonMap map[string]interface{}
	json.Unmarshal([]byte(capturedOutput), &jsonMap)

	if val, ok := jsonMap["message"]; ok {
		c.Check(val, Equals, expected)
	} else {
		c.Errorf("Log message didn't have a 'message' key, obtained %s", capturedOutput)
	}
}
