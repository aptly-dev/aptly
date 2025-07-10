package utils

import (
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/term"
)

// RunningOnTerminal checks whether stdout is terminal
func RunningOnTerminal() bool {
	return term.IsTerminal(syscall.Stdout)
}

type LogWriter struct {
	Logger zerolog.Logger
}

func (lw LogWriter) Write(bs []byte) (int, error) {
	return lw.Logger.With().Str("level", "info").Logger().Write(bs)
}

func SetupJSONLogger(levelStr string, w io.Writer) {
	zerolog.MessageFieldName = "message"
	zerolog.LevelFieldName = "level"

	var tsHook timestampHook
	log.Logger = zerolog.New(w).
		Hook(&tsHook).
		Level(GetLogLevelOrDebug(levelStr))
}

func SetupDefaultLogger(levelStr string) {
	zerolog.MessageFieldName = "message"
	zerolog.LevelFieldName = "level"

	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr,
		FormatTimestamp: func(_ interface{}) string { return "" },
		NoColor:         !RunningOnTerminal(),
	}).Level(GetLogLevelOrDebug(levelStr))
}

func GetLogLevelOrDebug(levelStr string) zerolog.Level {
	if levelStr == "" {
		return zerolog.DebugLevel
	}
	
	levelStr = strings.ToLower(levelStr)
	if levelStr == "warning" {
		levelStr = "warn"
	}

	var level zerolog.Level

	err := level.UnmarshalText([]byte(levelStr))
	if err == nil {
		return level
	}

	log.Warn().Msgf("Unknown log level '%s', defaulting to debug", levelStr)
	return zerolog.DebugLevel
}

type timestampHook struct{}

func (h *timestampHook) Run(e *zerolog.Event, _ zerolog.Level, _ string) {
	t := time.Now()
	ts := t.Format(time.RFC3339)
	e.Str("time", ts)
}
