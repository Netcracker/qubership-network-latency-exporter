package logger

import (
	"log/slog"
	"os"

	"github.com/go-kit/log"
)

var (
	// Log configuration
	LogLevel  string
	LogFormat string
)

// GetLogLevel returns the slog.Level based on the configured LogLevel
func GetLogLevel() slog.Level {
	switch LogLevel {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// SlogToKitLogger adapts slog.Logger to go-kit/log.Logger
type SlogToKitLogger struct {
	*slog.Logger
}

func (l *SlogToKitLogger) Log(keyvals ...interface{}) error {
	if len(keyvals) == 0 {
		return nil
	}
	msg, ok := keyvals[0].(string)
	if !ok {
		msg = ""
	}
	args := make([]any, 0, len(keyvals)-1)
	for i := 1; i < len(keyvals); i += 2 {
		if i+1 < len(keyvals) {
			args = append(args, keyvals[i], keyvals[i+1])
		}
	}
	l.Info(msg, args...)
	return nil
}

// NewLogger creates a new slog.Logger with the configured level and format
func NewLogger() *slog.Logger {
	var handler slog.Handler
	if LogFormat == "text" {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: GetLogLevel(),
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: GetLogLevel(),
		})
	}
	return slog.New(handler)
}

// NewKitLogger creates a new go-kit compatible logger
func NewKitLogger() log.Logger {
	return &SlogToKitLogger{NewLogger()}
}
