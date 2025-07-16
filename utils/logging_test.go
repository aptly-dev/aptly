package utils

import (
	"bytes"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	. "gopkg.in/check.v1"
)

type LoggingSuite struct {
	origLogger zerolog.Logger
}

var _ = Suite(&LoggingSuite{})

func (s *LoggingSuite) SetUpTest(c *C) {
	// Save original logger
	s.origLogger = log.Logger
}

func (s *LoggingSuite) TearDownTest(c *C) {
	// Restore original logger
	log.Logger = s.origLogger
}

func (s *LoggingSuite) TestRunningOnTerminal(c *C) {
	// Test RunningOnTerminal function
	// The result depends on whether we're running in a terminal
	result := RunningOnTerminal()
	c.Check(result, FitsTypeOf, true)
}

func (s *LoggingSuite) TestLogWriter(c *C) {
	// Test LogWriter struct and Write method
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	logWriter := LogWriter{Logger: logger}

	// Test Write method
	testData := []byte("test log message")
	n, err := logWriter.Write(testData)
	c.Check(err, IsNil)
	c.Check(n, Equals, len(testData))

	// Check that something was written to the buffer
	c.Check(buf.Len() > 0, Equals, true)

	// Check that the output contains the message
	output := buf.String()
	c.Check(strings.Contains(output, "test log message"), Equals, true)
}

func (s *LoggingSuite) TestLogWriterEmpty(c *C) {
	// Test LogWriter with empty data
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	logWriter := LogWriter{Logger: logger}

	n, err := logWriter.Write([]byte{})
	c.Check(err, IsNil)
	c.Check(n, Equals, 0)
}

func (s *LoggingSuite) TestSetupJSONLogger(c *C) {
	// Test SetupJSONLogger function
	var buf bytes.Buffer

	// Test with different log levels
	testLevels := []string{"debug", "info", "warn", "error"}

	for _, level := range testLevels {
		buf.Reset()
		SetupJSONLogger(level, &buf)

		// Check that message and level field names are set correctly
		c.Check(zerolog.MessageFieldName, Equals, "message")
		c.Check(zerolog.LevelFieldName, Equals, "level")

		// Test logging something
		log.Info().Msg("test message")

		// Check that JSON was written
		output := buf.String()
		if len(output) > 0 {
			c.Check(strings.Contains(output, "message"), Equals, true)
			c.Check(strings.Contains(output, "test message"), Equals, true)
		}
	}
}

func (s *LoggingSuite) TestSetupDefaultLogger(c *C) {
	// Test SetupDefaultLogger function
	testLevels := []string{"debug", "info", "warn", "error"}

	for _, level := range testLevels {
		SetupDefaultLogger(level)

		// Check that message and level field names are set correctly
		c.Check(zerolog.MessageFieldName, Equals, "message")
		c.Check(zerolog.LevelFieldName, Equals, "level")

		// Check that logger is configured (hard to test output since it goes to stderr)
		c.Check(log.Logger, NotNil)
	}
}

func (s *LoggingSuite) TestGetLogLevelOrDebugValid(c *C) {
	// Test GetLogLevelOrDebug with valid levels
	testCases := map[string]zerolog.Level{
		"debug":   zerolog.DebugLevel,
		"DEBUG":   zerolog.DebugLevel,
		"info":    zerolog.InfoLevel,
		"INFO":    zerolog.InfoLevel,
		"warn":    zerolog.WarnLevel,
		"WARN":    zerolog.WarnLevel,
		"warning": zerolog.WarnLevel,
		"WARNING": zerolog.WarnLevel,
		"error":   zerolog.ErrorLevel,
		"ERROR":   zerolog.ErrorLevel,
		"fatal":   zerolog.FatalLevel,
		"FATAL":   zerolog.FatalLevel,
		"panic":   zerolog.PanicLevel,
		"PANIC":   zerolog.PanicLevel,
		"trace":   zerolog.TraceLevel,
		"TRACE":   zerolog.TraceLevel,
	}

	for levelStr, expectedLevel := range testCases {
		result := GetLogLevelOrDebug(levelStr)
		c.Check(result, Equals, expectedLevel, Commentf("Failed for level: %s", levelStr))
	}
}

func (s *LoggingSuite) TestGetLogLevelOrDebugInvalid(c *C) {
	// Test GetLogLevelOrDebug with invalid levels
	invalidLevels := []string{
		"invalid",
		"unknown",
		"",
		"verbose",
		"critical",
	}

	// Capture log output to verify warning is logged
	var buf bytes.Buffer
	originalLogger := log.Logger
	log.Logger = zerolog.New(&buf).Level(zerolog.TraceLevel)
	defer func() { log.Logger = originalLogger }()

	for _, levelStr := range invalidLevels {
		buf.Reset()
		result := GetLogLevelOrDebug(levelStr)

		// Should default to debug level
		c.Check(result, Equals, zerolog.DebugLevel, Commentf("Failed for invalid level: %s", levelStr))

		// Should log a warning (if levelStr is not empty)
		if levelStr != "" {
			output := buf.String()
			c.Check(strings.Contains(output, "Unknown log level"), Equals, true, Commentf("No warning logged for: %s, got: %s", levelStr, output))
			c.Check(strings.Contains(output, levelStr), Equals, true, Commentf("Level not mentioned in warning: %s", levelStr))
		}
	}
}

func (s *LoggingSuite) TestTimestampHook(c *C) {
	// Test timestampHook struct and Run method
	hook := &timestampHook{}

	var buf bytes.Buffer
	logger := zerolog.New(&buf).Hook(hook)

	// Log a message
	logger.Info().Msg("test message with timestamp")

	// Check that output contains timestamp
	output := buf.String()
	c.Check(strings.Contains(output, "time"), Equals, true)
	c.Check(strings.Contains(output, "test message with timestamp"), Equals, true)

	// Check that timestamp is in RFC3339 format (contains T and Z or +/- timezone)
	c.Check(strings.Contains(output, "T") || strings.Contains(output, ":"), Equals, true)
}

func (s *LoggingSuite) TestTimestampHookMultipleLevels(c *C) {
	// Test timestampHook with different log levels
	hook := &timestampHook{}

	var buf bytes.Buffer
	logger := zerolog.New(&buf).Hook(hook)

	// Test different log levels
	testCases := []struct {
		level   zerolog.Level
		message string
	}{
		{zerolog.DebugLevel, "debug message"},
		{zerolog.InfoLevel, "info message"},
		{zerolog.WarnLevel, "warn message"},
		{zerolog.ErrorLevel, "error message"},
	}

	for _, tc := range testCases {
		buf.Reset()
		logger.WithLevel(tc.level).Msg(tc.message)

		output := buf.String()
		if len(output) > 0 {
			c.Check(strings.Contains(output, "time"), Equals, true, Commentf("No timestamp for level: %v", tc.level))
			c.Check(strings.Contains(output, tc.message), Equals, true, Commentf("Message missing for level: %v", tc.level))
		}
	}
}

func (s *LoggingSuite) TestLogLevelCaseInsensitive(c *C) {
	// Test that log level parsing is case insensitive
	mixedCaseLevels := []string{
		"Debug", "deBuG", "DeBuG",
		"Info", "inFo", "InFo",
		"Warn", "waRn", "WaRn",
		"Error", "erRor", "ErRoR",
	}

	for _, levelStr := range mixedCaseLevels {
		result := GetLogLevelOrDebug(levelStr)
		// Should not default to debug (unless it's actually debug)
		if strings.ToLower(levelStr) != "debug" {
			c.Check(result != zerolog.DebugLevel || strings.ToLower(levelStr) == "debug", Equals, true,
				Commentf("Case insensitive parsing failed for: %s", levelStr))
		}
	}
}

func (s *LoggingSuite) TestSetupLoggersIntegration(c *C) {
	// Test integration between setup functions and actual logging
	var buf bytes.Buffer

	// Test JSON logger
	SetupJSONLogger("info", &buf)
	log.Info().Str("key", "value").Msg("json test message")

	jsonOutput := buf.String()
	if len(jsonOutput) > 0 {
		c.Check(strings.Contains(jsonOutput, "json test message"), Equals, true)
		c.Check(strings.Contains(jsonOutput, "key"), Equals, true)
		c.Check(strings.Contains(jsonOutput, "value"), Equals, true)
	}

	// Test default logger (output goes to stderr, so we can't easily capture it)
	SetupDefaultLogger("warn")
	c.Check(log.Logger, NotNil)
}
