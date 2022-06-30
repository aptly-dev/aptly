package utils

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type logLevel int8

var logLevels = map[string]logLevel{
	"debug":   -1,
	"info":    0,
	"warn":    1,
	"warning": 1,
	"error":   2,
}

func levelFromZeroLevel(level zerolog.Level) logLevel {
	switch level {
	case zerolog.DebugLevel:
		return logLevels["debug"]
	case zerolog.InfoLevel:
		return logLevels["info"]
	case zerolog.WarnLevel:
		return logLevels["warning"]
	case zerolog.ErrorLevel:
		return logLevels["error"]
	}

	return -2
}

func (l logLevel) String() string {
	switch l {
	case logLevels["debug"]:
		return "debug"
	case logLevels["info"]:
		return "info"
	case logLevels["warning"]:
		return "warning"
	case logLevels["error"]:
		return "error"
	}

	return "unknown"
}

type logField struct {
	key   string
	value string
}

// Logger writes log messages at the specified levels.
type Logger interface {
	// Level returns a string representation of the currently applied log level
	Level() string

	// Writer returns an implementation of io.Writer that writes messages to the logger instances
	// at the provided level.
	Writer(levelStr string) io.Writer

	// Debug logs a message at debug level. The log entry includes all key value pairs added via the
	// WithField function.
	Debug(args ...interface{})

	// Debugf logs a message at debug level using a format string. The log entry includes all key value
	// pairs added via the WithField function.
	Debugf(template string, args ...interface{})

	// Info logs a message at info level. The log entry includes all key value pairs added via the
	// WithField function.
	Info(args ...interface{})

	// Infof logs a message at info level using a format string. The log entry includes all key value
	// pairs added via the WithField function.
	Infof(template string, args ...interface{})

	// Warn logs a message at warning level. The log entry includes all key value pairs added via the
	// WithField function.
	Warn(args ...interface{})

	// Warnf logs a message at warning level using a format string. The log entry includes all key value
	// pairs added via the WithField function.
	Warnf(template string, args ...interface{})

	// Error logs a message at error level. The log entry includes all key value pairs added via the
	// WithField function.
	Error(args ...interface{})

	// Errorf logs a message at error level using a format string. The log entry includes all key value
	// pairs added via the WithField function.
	Errorf(template string, args ...interface{})

	// Fatal logs a message at error level. The log entry includes all key value pairs added via the
	// WithField function. After writing the message it calls os.Exit().
	Fatal(args ...interface{})

	// Fatalf logs a message at error level using a format string. The log entry includes all key value
	// pairs added via the WithField function. After writing the message it calls os.Exit().
	Fatalf(template string, args ...interface{})

	// Panic logs a message at error level. The log entry includes all key value pairs added via the
	// WithField function. After writing the message it calls panic().
	Panic(args ...interface{})

	// Panicf logs a message at error level using a format string. The log entry includes all key value
	// pairs added via the WithField function. After writing the message it calls panic().
	Panicf(template string, args ...interface{})

	// WithField adds a key value pair to the logger which will be printed alongside the log message.
	// For example:
	//
	//	 logger.WithField("remote", c.ClientIP()).
	//	        WithField("method", c.Request.Method).
	//          Error("Some error message")
	//
	WithField(key string, value string) Logger
}

// LoggerFactory returns an implementation of the Logger interface based on the log format and log level.
func LoggerFactory(logFormat string, logLevel string) Logger {
	if logFormat == "json" {
		return NewZeroJSONLogger(logLevel)
	}

	return NewPlainLogger(logLevel)
}

type PlainLogger struct {
	level  logLevel
	fields []logField
}

// NewPlainLogger returns an instance of PlainLogger configured with the provided log level.
// Accepted log levels are debug, info, warn, warning and error. The log level is case-insensitive.
func NewPlainLogger(level string) *PlainLogger {
	logger := &PlainLogger{level: logLevels["debug"]} // default to debug log level
	logger.initLogger(level)
	logger.Debug("Initialized plain text logger.")
	return logger
}

func (l *PlainLogger) Level() string {
	return l.level.String()
}

func (l *PlainLogger) Writer(levelStr string) io.Writer {
	logLevel, err := l.parseLogLevel(levelStr)
	if err != nil {
		l.Warn(err.Error())
		defaultLvl := logLevels["debug"]
		logLevel = &defaultLvl
	}

	return LogWriter{lvl: *logLevel, logger: l}
}

func (l *PlainLogger) Debug(args ...interface{}) {
	l.log(logLevels["debug"], args...)
}

func (l *PlainLogger) Debugf(template string, args ...interface{}) {
	l.logf(logLevels["debug"], template, args...)
}

func (l *PlainLogger) Info(args ...interface{}) {
	l.log(logLevels["info"], args...)
}

func (l *PlainLogger) Infof(template string, args ...interface{}) {
	l.logf(logLevels["info"], template, args...)
}

func (l *PlainLogger) Warn(args ...interface{}) {
	l.log(logLevels["warning"], args...)
}

func (l *PlainLogger) Warnf(template string, args ...interface{}) {
	l.logf(logLevels["warning"], template, args...)
}

func (l *PlainLogger) Error(args ...interface{}) {
	l.log(logLevels["error"], args...)
}

func (l *PlainLogger) Errorf(template string, args ...interface{}) {
	l.logf(logLevels["error"], template, args...)
}

func (l *PlainLogger) Fatal(args ...interface{}) {
	l.log(logLevels["error"], args...)
	os.Exit(1)
}

func (l *PlainLogger) Fatalf(template string, args ...interface{}) {
	l.logf(logLevels["error"], template, args...)
	os.Exit(1)
}

func (l *PlainLogger) Panic(args ...interface{}) {
	l.log(logLevels["error"], args...)
	s := fmt.Sprintln(args...)
	panic(s)
}

func (l *PlainLogger) Panicf(template string, args ...interface{}) {
	l.logf(logLevels["error"], template, args...)
	s := fmt.Sprintln(args...)
	panic(s)
}

func (l *PlainLogger) WithField(key string, value string) Logger {
	lp := *l
	newL := &lp
	f := logField{key: key, value: value}
	newL.fields = append(newL.fields, f)
	return newL
}

func (l *PlainLogger) initLogger(levelStr string) {
	logLevel, err := l.parseLogLevel(levelStr)
	if err == nil {
		l.level = *logLevel
	} else {
		l.Warn(err.Error())
	}

	l.fields = make([]logField, 0)
}

func (l *PlainLogger) parseLogLevel(levelStr string) (*logLevel, error) {
	level, exist := logLevels[strings.ToLower(levelStr)]
	if !exist {
		msg := fmt.Sprintf("Unknown log level '%s', defaulting to debug level", levelStr)
		return nil, errors.New(msg)
	}

	return &level, nil
}

func (l *PlainLogger) log(level logLevel, args ...interface{}) {
	if l.levelEnabled(level) {
		for _, f := range l.fields {
			fmt.Printf("%s: %s ", f.key, f.value)
		}

		fmt.Println(args...)
	}
}

func (l *PlainLogger) logf(level logLevel, template string, args ...interface{}) {
	if l.levelEnabled(level) {
		for _, f := range l.fields {
			fmt.Printf("%s: %s ", f.key, f.value)
		}

		fmt.Printf(template, args...)
	}
}

func (l *PlainLogger) levelEnabled(level logLevel) bool {
	return l.level <= level
}

type ZeroJSONLogger struct {
	level  zerolog.Level
	logger *zerolog.Logger
}

type timestampHook struct{}

func (h *timestampHook) Run(e *zerolog.Event, l zerolog.Level, msg string) {
	t := time.Now()
	ts := t.Format(time.RFC3339)
	e.Str("ts", ts)
}

// NewZeroJSONLogger returns an instance of ZeroJSONLogger configured with the provided log level.
// Accepted log levels are debug, info, warn, warning and error. The log level is case-insensitive.
func NewZeroJSONLogger(level string) *ZeroJSONLogger {
	logger := &ZeroJSONLogger{}
	logger.initLogger(level, os.Stdout)
	logger.Debug("Initialized json logger with stdout writer.")
	return logger
}

// NewZeroJSONLoggerWithWriter returns an instance of ZeroJSONLogger configured with the provided
// log level. Itwrites logs to the provided io.Writer instead of Stdout.
// Accepted log levels are debug, info, warn, warning and error. The log level is case-insensitive.
func NewZeroJSONLoggerWithWriter(level string, writer io.Writer) *ZeroJSONLogger {
	logger := &ZeroJSONLogger{}
	logger.initLogger(level, writer)
	return logger
}

func (l *ZeroJSONLogger) Level() string {
	return levelFromZeroLevel(l.level).String()
}

func (l *ZeroJSONLogger) Writer(levelStr string) io.Writer {
	level := zerolog.DebugLevel
	err := level.UnmarshalText([]byte(levelStr))

	if err != nil {
		l.Warn(err.Error())
	}

	logLevel := levelFromZeroLevel(level)
	return LogWriter{lvl: logLevel, logger: l}
}

func (l *ZeroJSONLogger) Debug(args ...interface{}) {
	s := fmt.Sprint(args...)
	l.logger.Debug().Msg(s)
}

func (l *ZeroJSONLogger) Debugf(template string, args ...interface{}) {
	s := fmt.Sprintf(template, args...)
	l.logger.Debug().Msg(s)
}

func (l *ZeroJSONLogger) Info(args ...interface{}) {
	s := fmt.Sprint(args...)
	l.logger.Info().Msg(s)
}

func (l *ZeroJSONLogger) Infof(template string, args ...interface{}) {
	s := fmt.Sprintf(template, args...)
	l.logger.Info().Msg(s)
}

func (l *ZeroJSONLogger) Warn(args ...interface{}) {
	s := fmt.Sprint(args...)
	l.logger.Warn().Msg(s)
}

func (l *ZeroJSONLogger) Warnf(template string, args ...interface{}) {
	s := fmt.Sprintf(template, args...)
	l.logger.Warn().Msg(s)
}

func (l *ZeroJSONLogger) Error(args ...interface{}) {
	s := fmt.Sprint(args...)
	l.logger.Error().Msg(s)
}

func (l *ZeroJSONLogger) Errorf(template string, args ...interface{}) {
	s := fmt.Sprintf(template, args...)
	l.logger.Error().Msg(s)
}

func (l *ZeroJSONLogger) Fatal(args ...interface{}) {
	s := fmt.Sprint(args...)
	l.logger.Error().Msg(s)
	os.Exit(1)
}

func (l *ZeroJSONLogger) Fatalf(template string, args ...interface{}) {
	s := fmt.Sprintf(template, args...)
	l.logger.Error().Msg(s)
	os.Exit(1)
}

func (l *ZeroJSONLogger) Panic(args ...interface{}) {
	s := fmt.Sprint(args...)
	l.logger.Error().Msg(s)
	panic(s)
}

func (l *ZeroJSONLogger) Panicf(template string, args ...interface{}) {
	s := fmt.Sprintf(template, args...)
	l.logger.Error().Msg(s)
	panic(s)
}

func (l *ZeroJSONLogger) WithField(key string, value string) Logger {
	newLogger := l.logger.With().Str(key, value).Logger()
	return &ZeroJSONLogger{level: l.level, logger: &newLogger}
}

func (l *ZeroJSONLogger) initLogger(levelStr string, writer io.Writer) {
	var level zerolog.Level

	levelStr = strings.ToLower(levelStr)
	if levelStr == "warning" {
		levelStr = "warn"
	}

	err := level.UnmarshalText([]byte(levelStr))
	if err == nil {
		zerolog.SetGlobalLevel(level)
		l.level = level
	} else {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		l.level = zerolog.DebugLevel
	}

	zerolog.MessageFieldName = "msg"
	zerolog.LevelFieldName = "lvl"

	var tsHook timestampHook
	zeroLogger := zerolog.New(writer).Hook(&tsHook)
	l.logger = &zeroLogger

	if err != nil {
		l.Warnf("Unknown log level '%s', defaulting to debug level", levelStr)
	}
}

type LogWriter struct {
	lvl    logLevel
	logger Logger
}

func (lw LogWriter) Write(bs []byte) (int, error) {
	msg := strings.TrimSuffix(string(bs), "\n")

	switch lw.lvl {
	case -1:
		lw.logger.Debug(msg)
	case 0:
		lw.logger.Info(msg)
	case 1:
		lw.logger.Warn(msg)
	case 2:
		lw.logger.Error(msg)
	}

	return len(msg), nil
}
