package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	. "gopkg.in/check.v1"
)

type LoggingSuite struct{}

var _ = Suite(&LoggingSuite{})



func (s *LoggingSuite) TestLoggerFactory(c *C) {
	l := LoggerFactory("default", "debug")
	c.Check(fmt.Sprintf("%T", l), Equals, fmt.Sprintf("%T", &PlainLogger{}))
	c.Check(l == nil, Equals, false)

	l = LoggerFactory("json", "debug")
	c.Check(fmt.Sprintf("%T", l), Equals, fmt.Sprintf("%T", &ZeroJSONLogger{}))
	c.Check(l == nil, Equals, false)

	l = LoggerFactory("undefined", "debug")
	c.Check(fmt.Sprintf("%T", l), Equals, fmt.Sprintf("%T", &PlainLogger{}))
	c.Check(l == nil, Equals, false)
}

func (s *LoggingSuite) TestPlainLoggerMessage(c *C) {
	l := LoggerFactory("default", "debug")

	stdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		c.Error(err.Error())
	}

	os.Stdout = w
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	expected := "TestMessage"
	l.Info(expected)

	w.Close()
	os.Stdout = stdout
	capturedOutput := <-outC

	c.Check(capturedOutput, Equals, expected+"\n")
}

func (s *LoggingSuite) TestZeroLoggerMessage(c *C) {
	r, w, err := os.Pipe()
	if err != nil {
		c.Error(err.Error())
	}

	l := NewZeroJSONLoggerWithWriter("debug", w)
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		fmt.Println(buf.String())
		outC <- buf.String()
	}()

	expected := "TestMessage"
	l.Info(expected)

	w.Close()
	capturedOutput := <-outC

	var jsonMap map[string]interface{}
	json.Unmarshal([]byte(capturedOutput), &jsonMap)

	c.Check(jsonMap["message"], Equals, expected)
	c.Check(jsonMap["level"], Equals, "info")

	if val, ok := jsonMap["ts"]; ok {
		layout := time.RFC3339
		_, err := time.Parse(layout, val.(string))

		if err != nil {
			c.Errorf("Logger uses invalid format for timestamp, expected RFC3339, obtained %s", val)
		}
	} else {
		c.Errorf("Log message didn't have a 'ts' key, obtained %s", capturedOutput)
	}
}

func (s *LoggingSuite) TestPlainLoggerLowerLogLevel(c *C) {
	l := LoggerFactory("default", "warning")

	stdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		c.Error(err.Error())
	}

	os.Stdout = w
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	expected := "ErrorTestMessage"
	l.Error(expected)
	l.Info("InfoTestMessage")

	w.Close()
	os.Stdout = stdout
	capturedOutput := <-outC

	logs := strings.Split(strings.TrimSuffix(capturedOutput, "\n"), "\n")
	testLog := logs[len(logs)-1]

	c.Check(testLog, Equals, expected)
	c.Check(len(logs), Equals, 1)
}

func (s *LoggingSuite) TestZeroLoggerLowerLogLevel(c *C) {
	r, w, err := os.Pipe()
	if err != nil {
		c.Error(err.Error())
	}

	l := NewZeroJSONLoggerWithWriter("warning", w)
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		fmt.Println(buf.String())
		outC <- buf.String()
	}()

	expected := "ErrorTestMessage"
	l.Error(expected)
	l.Info("InfoTestMessage")

	w.Close()
	capturedOutput := <-outC

	logs := strings.Split(strings.TrimSuffix(capturedOutput, "\n"), "\n")
	testLog := logs[len(logs)-1]

	var jsonMap map[string]interface{}
	json.Unmarshal([]byte(testLog), &jsonMap)

	c.Check(jsonMap["message"], Equals, expected)
	c.Check(len(logs), Equals, 1)
}

func (s *LoggingSuite) TestZeroLoggerWithField(c *C) {
	r, w, err := os.Pipe()
	if err != nil {
		c.Error(err.Error())
	}

	l := NewZeroJSONLoggerWithWriter("debug", w)
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		fmt.Println(buf.String())
		outC <- buf.String()
	}()

	expected := "testValue"
	l.WithField("test", "testValue").Info("InfoTestMessage")

	w.Close()
	capturedOutput := <-outC

	var jsonMap map[string]interface{}
	json.Unmarshal([]byte(capturedOutput), &jsonMap)

	if val, ok := jsonMap["test"]; ok {
		c.Check(val, Equals, expected)
	} else {
		c.Errorf("Log message didn't have a 'test' key, obtained %s", capturedOutput)
	}
}

func (s *LoggingSuite) TestPlainLoggerWriter(c *C) {
	l := LoggerFactory("default", "debug")
	lw := l.Writer("debug")

	stdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		c.Error(err.Error())
	}

	os.Stdout = w
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	expected := "ErrorTestMessage"
	lw.Write([]byte(expected))

	w.Close()
	os.Stdout = stdout
	capturedOutput := <-outC
	c.Check(capturedOutput, Equals, expected+"\n")
}

func (s *LoggingSuite) TestZeroLoggerWriter(c *C) {
	r, w, err := os.Pipe()
	if err != nil {
		c.Error(err.Error())
	}

	l := NewZeroJSONLoggerWithWriter("debug", w)
	lw := l.Writer("debug")
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	expected := "ErrorTestMessage"
	lw.Write([]byte(expected))

	w.Close()
	capturedOutput := <-outC

	var jsonMap map[string]interface{}
	json.Unmarshal([]byte(capturedOutput), &jsonMap)

	c.Check(jsonMap["level"], Equals, "debug")
	c.Check(jsonMap["message"], Equals, expected)
}

func (s *LoggingSuite) TestPlainLoggerLogLevel(c *C) {
	l := LoggerFactory("default", "debug")
	c.Check(l.Level(), Equals, "debug")

	l = LoggerFactory("default", "info")
	c.Check(l.Level(), Equals, "info")

	l = LoggerFactory("default", "warning")
	c.Check(l.Level(), Equals, "warning")

	l = LoggerFactory("default", "warn")
	c.Check(l.Level(), Equals, "warning")

	l = LoggerFactory("default", "error")
	c.Check(l.Level(), Equals, "error")
}

func (s *LoggingSuite) TestZeroLoggerLogLevel(c *C) {
	l := LoggerFactory("json", "debug")
	c.Check(l.Level(), Equals, "debug")

	l = LoggerFactory("json", "info")
	c.Check(l.Level(), Equals, "info")

	l = LoggerFactory("json", "warning")
	c.Check(l.Level(), Equals, "warning")

	l = LoggerFactory("json", "warn")
	c.Check(l.Level(), Equals, "warning")

	l = LoggerFactory("json", "error")
	c.Check(l.Level(), Equals, "error")
}

func (s *LoggingSuite) TestPlainLoggerDefaultLogLevel(c *C) {
	l := LoggerFactory("default", "undefined")
	c.Check(l.Level(), Equals, "debug")
}

func (s *LoggingSuite) TestZeroLoggerDefaultLogLevel(c *C) {
	l := LoggerFactory("json", "undefined")
	c.Check(l.Level(), Equals, "debug")
}
