package logger

import (
	"io"
	"log/slog"
	"strings"
)

var L *slog.Logger

// Init initializes the global logger
func Init(out io.Writer, levelStr string, format string) {
	level := ParseLevel(levelStr)
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}

	var handler slog.Handler
	if strings.ToLower(format) == "json" {
		handler = slog.NewJSONHandler(out, opts)
	} else {
		handler = slog.NewTextHandler(out, opts)
	}

	L = slog.New(handler)
	slog.SetDefault(L)
}

// ParseLevel converts a string level to slog.Level
func ParseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
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

// Info logs at LevelInfo
func Info(msg string, args ...any) {
	L.Info(msg, args...)
}

// Error logs at LevelError
func Error(msg string, args ...any) {
	L.Error(msg, args...)
}

// Debug logs at LevelDebug
func Debug(msg string, args ...any) {
	L.Debug(msg, args...)
}

// Warn logs at LevelWarn
func Warn(msg string, args ...any) {
	L.Warn(msg, args...)
}
