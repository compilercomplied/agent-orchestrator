package logging

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
)

// Logger defines the interface for logging
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Printf(format string, v ...any)
	Fatalf(format string, v ...any)
}

// globalLogger is the instance used by the package-level functions.
// It defaults to the plain logger until Init is called.
var globalLogger Logger = &DefaultLogger{}

// Allowed values: "structured" (JSON), "plain" (standard text).
func Init(format string) {
	switch strings.ToLower(format) {
	case "structured":
		globalLogger = NewStructuredLogger()
	case "plain":
		globalLogger = &DefaultLogger{}
	default:
		panic(fmt.Sprintf("invalid LOG_FORMAT: %q. Must be 'structured' or 'plain'", format))
	}
}

// --- Plain Logger (Wrapper around standard log) ---

type DefaultLogger struct{}

func (l *DefaultLogger) Info(msg string, args ...any) {
	// Simple key-value formatting for plain text
	log.Printf("INFO: %s %v", msg, args)
}

func (l *DefaultLogger) Error(msg string, args ...any) {
	log.Printf("ERROR: %s %v", msg, args)
}

func (l *DefaultLogger) Printf(format string, v ...any) {
	log.Printf(format, v...)
}

func (l *DefaultLogger) Fatalf(format string, v ...any) {
	log.Fatalf(format, v...)
}

// --- Structured Logger (Wrapper around slog) ---

type StructuredLogger struct {
	logger *slog.Logger
}

func NewStructuredLogger() *StructuredLogger {
	return &StructuredLogger{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

func (l *StructuredLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *StructuredLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

func (l *StructuredLogger) Printf(format string, v ...any) {
	// sLog doesn't have printf, so we treat the formatted string as the message
	l.logger.Info(fmt.Sprintf(format, v...))
}

func (l *StructuredLogger) Fatalf(format string, v ...any) {
	l.logger.Error(fmt.Sprintf(format, v...))
	os.Exit(1)
}

// --- Global Accessors ---

func Info(msg string, args ...any) {
	globalLogger.Info(msg, args...)
}

func Error(msg string, args ...any) {
	globalLogger.Error(msg, args...)
}

func Printf(format string, v ...any) {
	globalLogger.Printf(format, v...)
}

func Fatalf(format string, v ...any) {
	globalLogger.Fatalf(format, v...)
}

// With creates a child logger with pre-set attributes (Context).
// This is a helper specific to structured logging, but we can make it safe for plain logger too.
// For simplicity in this interface, we aren't exposing With() yet in the main interface,
// but you might want to add it later for request-scoped loggers.
func With(args ...any) Logger {
	if sl, ok := globalLogger.(*StructuredLogger); ok {
		return &StructuredLogger{logger: sl.logger.With(args...)}
	}
	// For plain logger, we just return it as is (ignoring attributes for now)
	// or we could try to prepend them to messages.
	return globalLogger
}
