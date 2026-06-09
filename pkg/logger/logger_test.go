package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLogLevel(t *testing.T) {
	// Test default log level (should be info)
	level := GetLogLevel()
	assert.Equal(t, 0, int(level)) // slog.LevelInfo

	// Test debug level
	LogLevel = "debug"
	defer func() { LogLevel = "" }()
	level = GetLogLevel()
	assert.Equal(t, -4, int(level)) // slog.LevelDebug

	// Test warn level
	LogLevel = "warn"
	level = GetLogLevel()
	assert.Equal(t, 4, int(level)) // slog.LevelWarn

	// Test error level
	LogLevel = "error"
	level = GetLogLevel()
	assert.Equal(t, 8, int(level)) // slog.LevelError
}

func TestNewLogger(t *testing.T) {
	logger := NewLogger()
	assert.NotNil(t, logger)
}

func TestNewKitLogger(t *testing.T) {
	kitLogger := NewKitLogger()
	assert.NotNil(t, kitLogger)
	assert.IsType(t, &SlogToKitLogger{}, kitLogger)
}

func TestSlogToKitLogger_Log(t *testing.T) {
	slogLogger := NewLogger()
	kitLogger := &SlogToKitLogger{Logger: slogLogger}

	// Test logging with key-value pairs
	err := kitLogger.Log("key1", "value1", "key2", 42)
	assert.NoError(t, err)

	// Test logging with odd number of arguments (should handle gracefully)
	err = kitLogger.Log("key1", "value1", "key2")
	assert.NoError(t, err)
}
