package utils

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

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

	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		Level(GetLogLevelOrDebug(levelStr)).
		With().
		Timestamp().
		Logger()
}

func GetLogLevelOrDebug(levelStr string) zerolog.Level {
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

func (h *timestampHook) Run(e *zerolog.Event, l zerolog.Level, msg string) {
	t := time.Now()
	ts := t.Format(time.RFC3339)
	e.Str("time", ts)
}
