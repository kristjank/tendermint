package log

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

var _ Logger = (*defaultLogger)(nil)

type defaultLogger struct {
	zerolog.Logger

	trace bool
}

// NewDefaultLogger returns a default logger that can be used within Tendermint
// and that fulfills the Logger interface. The underlying logging provider is a
// zerolog logger that supports typical log levels along with JSON and plain/text
// log formats.
//
// Since zerolog supports typed structured logging and it is difficult to reflect
// that in a generic interface, all logging methods accept a series of key/value
// pair tuples, where the key must be a string.
func NewDefaultLogger(format, level string, trace bool) (Logger, error) {
	var logWriter io.Writer
	switch strings.ToLower(format) {
	case LogFormatPlain, LogFormatText:
		logWriter = zerolog.ConsoleWriter{Out: os.Stderr}

	case LogFormatJSON:
		logWriter = os.Stderr

	default:
		return nil, fmt.Errorf("unsupported log format: %s", format)
	}

	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("failed to parse log level (%s): %w", level, err)
	}

	// make the writer thread-safe
	logWriter = newSyncWriter(logWriter)

	return defaultLogger{
		Logger: zerolog.New(logWriter).Level(logLevel).With().Timestamp().Logger(),
		trace:  trace,
	}, nil
}

// MustNewDefaultLogger delegates a call NewDefaultLogger where it panics on
// error.
func MustNewDefaultLogger(format, level string, trace bool) Logger {
	logger, err := NewDefaultLogger(format, level, trace)
	if err != nil {
		panic(err)
	}

	return logger
}

func (l defaultLogger) Info(msg string, keyVals ...interface{}) {
	l.Logger.Info().Fields(getLogFields(keyVals...)).Msg(msg)
}

func (l defaultLogger) Error(msg string, keyVals ...interface{}) {
	e := l.Logger.Error()
	if l.trace {
		e = e.Stack()
	}

	e.Fields(getLogFields(keyVals...)).Msg(msg)
}

func (l defaultLogger) Debug(msg string, keyVals ...interface{}) {
	l.Logger.Debug().Fields(getLogFields(keyVals...)).Msg(msg)
}

func (l defaultLogger) With(keyVals ...interface{}) Logger {
	return defaultLogger{
		Logger: l.Logger.With().Fields(getLogFields(keyVals...)).Logger(),
		trace:  l.trace,
	}
}

func getLogFields(keyVals ...interface{}) map[string]interface{} {
	if len(keyVals)%2 != 0 {
		return nil
	}

	fields := make(map[string]interface{}, len(keyVals))
	for i := 0; i < len(keyVals); i += 2 {
		fields[fmt.Sprint(keyVals[i])] = keyVals[i+1]
	}

	return fields
}
