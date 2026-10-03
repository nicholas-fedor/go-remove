/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package logger provides a logging interface and implementation for go-remove.
package logger

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// LogCaptureFunc is a callback function that receives captured log messages.
// The level parameter contains the log level string (e.g., "DBG", "INF", "WRN", "ERR").
// The msg parameter contains the formatted log message.
type LogCaptureFunc func(level, msg string)

// Level is a log severity.
//
// Levels are ordered by increasing severity, so a logger set to a level emits
// every message logged at that level or above.
type Level int8

// Supported log levels, ordered by increasing severity.
const (
	// DebugLevel reports detail that is only useful while diagnosing a problem.
	DebugLevel Level = iota

	// InfoLevel reports ordinary operational progress.
	InfoLevel

	// WarnLevel reports a condition the caller needs to know about.
	WarnLevel

	// ErrorLevel reports a failed operation.
	ErrorLevel
)

// errorFieldKey is the field name an error recorded through Err is written under.
const errorFieldKey = "error"

// Logger defines the logging operations required by the application.
type Logger interface {
	// Debug logs a message at debug level.
	//
	// Parameters:
	//   - msg: Message to record.
	//   - fields: Structured pairs to attach to the message.
	Debug(msg string, fields ...Field)

	// Info logs a message at info level.
	//
	// Parameters:
	//   - msg: Message to record.
	//   - fields: Structured pairs to attach to the message.
	Info(msg string, fields ...Field)

	// Warn logs a message at warn level.
	//
	// Parameters:
	//   - msg: Message to record.
	//   - fields: Structured pairs to attach to the message.
	Warn(msg string, fields ...Field)

	// Error logs a message at error level.
	//
	// Parameters:
	//   - msg: Message to record.
	//   - fields: Structured pairs to attach to the message.
	Error(msg string, fields ...Field)

	// Level sets the minimum log level dynamically.
	//
	// Parameters:
	//   - level: Minimum level to emit.
	Level(level Level)

	// SetCaptureFunc sets a callback that receives log messages for TUI display.
	//
	// Parameters:
	//   - captureFunc: Callback invoked with level and message, or nil to disable capture.
	SetCaptureFunc(captureFunc LogCaptureFunc)
}

// fieldKind identifies the kind of value a Field carries.
type fieldKind uint8

// Field kinds, one for each value type a Field can carry.
const (
	fieldString fieldKind = iota
	fieldInt
	fieldBool
	fieldError
)

// Field is a structured pair attached to a log message.
//
// A Field is built with Str, Int, Bool, or Err and passed to one of the level
// methods. The zero Field carries no value and writes nothing.
type Field struct {
	kind  fieldKind
	key   string
	value any
}

// Str returns a Field holding a string value.
//
// Parameters:
//   - key: Field name.
//   - value: Field value.
//
// Returns:
//   - A Field for a log message.
func Str(key, value string) Field {
	return Field{kind: fieldString, key: key, value: value}
}

// Int returns a Field holding an integer value.
//
// Parameters:
//   - key: Field name.
//   - value: Field value.
//
// Returns:
//   - A Field for a log message.
func Int(key string, value int) Field {
	return Field{kind: fieldInt, key: key, value: value}
}

// Bool returns a Field holding a boolean value.
//
// Parameters:
//   - key: Field name.
//   - value: Field value.
//
// Returns:
//   - A Field for a log message.
func Bool(key string, value bool) Field {
	return Field{kind: fieldBool, key: key, value: value}
}

// Err returns a Field holding an error value.
//
// A nil error writes nothing, so a caller with nothing to report can pass it
// straight through.
//
// Parameters:
//   - err: Error to record.
//
// Returns:
//   - A Field for a log message.
func Err(err error) Field {
	return Field{kind: fieldError, key: errorFieldKey, value: err}
}

// apply writes the field onto a zerolog event.
//
// Parameters:
//   - event: Event the field is written to.
func (f Field) apply(event *zerolog.Event) {
	switch f.kind {
	case fieldString:
		if value, ok := f.value.(string); ok {
			event.Str(f.key, value)
		}
	case fieldInt:
		if value, ok := f.value.(int); ok {
			event.Int(f.key, value)
		}
	case fieldBool:
		if value, ok := f.value.(bool); ok {
			event.Bool(f.key, value)
		}
	case fieldError:
		if value, ok := f.value.(error); ok {
			event.Err(value)
		}
	}
}

// ZerologLogger wraps zerolog.Logger to implement the Logger interface.
type ZerologLogger struct {
	logger        zerolog.Logger
	mu            sync.RWMutex
	output        io.Writer
	captureWriter *captureWriter
}

var _ Logger = (*ZerologLogger)(nil)

// captureWriter wraps an io.Writer and captures written data for TUI display.
type captureWriter struct {
	mu             sync.RWMutex
	output         io.Writer
	captureFunc    LogCaptureFunc
	captureEnabled bool
}

// Write implements io.Writer, writing to the underlying output or capturing the data.
//
// When capture is enabled, output is discarded so the TUI does not show duplicate logs.
//
// Parameters:
//   - data: Log bytes produced by zerolog.
//
// Returns:
//   - Number of bytes accepted.
//   - An error if the underlying writer fails.
func (w *captureWriter) Write(data []byte) (int, error) {
	w.mu.RLock()
	output := w.output
	enabled := w.captureEnabled
	w.mu.RUnlock()

	// If capture is enabled, discard the output to prevent duplicate logs.
	// The log message will only be sent through the capture mechanism to the TUI log panel.
	if enabled {
		w.captureLogMessage(string(data))

		return len(data), nil
	}

	// Capture is not enabled, write to the underlying output normally.
	bytesWritten, err := output.Write(data)
	if err != nil {
		return bytesWritten, fmt.Errorf("failed to write to output: %w", err)
	}

	return bytesWritten, nil
}

// captureLogMessage parses a ConsoleWriter line and sends it to the capture callback.
//
// Expected format is "<timestamp> <LEVEL> <message>", where LEVEL is DBG, INF, WRN, ERR, or FTL.
//
// Parameters:
//   - logLine: Console-formatted log line.
func (w *captureWriter) captureLogMessage(logLine string) {
	w.mu.RLock()
	capture := w.captureFunc
	w.mu.RUnlock()

	if capture == nil {
		return
	}

	parts := strings.Fields(logLine)

	const minLogParts = 3
	if len(parts) < minLogParts {
		capture("LOG", strings.TrimSpace(logLine))

		return
	}

	level := "LOG"
	levelIndex := -1

	for i, part := range parts {
		switch part {
		case "DBG", "INF", "WRN", "ERR", "FTL":
			level = part
			levelIndex = i
		}

		if levelIndex >= 0 {
			break
		}
	}

	msg := strings.TrimSpace(logLine)
	if levelIndex >= 0 {
		msg = strings.TrimSpace(strings.Join(parts[levelIndex+1:], " "))
	}

	capture(level, msg)
}

// SetCaptureFunc sets the capture callback for this writer.
//
// A non-nil callback enables capture and discards normal output.
//
// Parameters:
//   - captureFunc: Callback invoked with level and message, or nil to disable capture.
func (w *captureWriter) SetCaptureFunc(captureFunc LogCaptureFunc) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.captureFunc = captureFunc
	w.captureEnabled = captureFunc != nil
}

// NewLogger creates a zerolog-based logger with console output to stderr.
//
// The logger uses RFC3339 timestamps and defaults to info level.
//
// Returns:
//   - Configured logger.
//   - Always nil error.
func NewLogger() (Logger, error) {
	output := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	}

	// Create the base logger with info level.
	zerologLogger := zerolog.New(output).
		With().
		Timestamp().
		Logger().
		Level(zerolog.InfoLevel)

	// Install a capture writer so SetCaptureFunc always has somewhere to write.
	// Without it the call was a silent no-op, and behaviour then depended on
	// which constructor the caller happened to pick.
	capture := &captureWriter{output: output}

	zerologLogger = zerologLogger.Output(capture)

	return &ZerologLogger{
		logger:        zerologLogger,
		output:        output,
		captureWriter: capture,
	}, nil
}

// NewLoggerWithCapture creates a logger that can send messages to the TUI.
//
// The returned writer can be used to inspect capture state in tests.
//
// Returns:
//   - Logger that supports SetCaptureFunc.
//   - Underlying capture writer.
//   - Always nil error.
func NewLoggerWithCapture() (Logger, *captureWriter, error) {
	// Create a captureWriter that wraps stderr.
	// captureFunc and captureEnabled are left as zero values (nil and false).
	captureWriter := &captureWriter{
		output: os.Stderr,
	}

	// Create a ConsoleWriter that writes to the captureWriter.
	consoleWriter := zerolog.ConsoleWriter{
		Out:        captureWriter,
		TimeFormat: time.RFC3339,
		NoColor:    true,
	}

	// Create the base logger with info level.
	zerologLogger := zerolog.New(consoleWriter).
		With().
		Timestamp().
		Logger().
		Level(zerolog.InfoLevel)

	logger := &ZerologLogger{
		logger:        zerologLogger,
		output:        consoleWriter,
		captureWriter: captureWriter,
	}

	return logger, captureWriter, nil
}

// NopLogger returns a logger that discards everything written to it.
//
// It wraps zerolog's disabled logger, so no message is ever formatted or
// serialised, whatever level it is logged at.
//
// Use it where logging is not the subject: tests, and any caller that wants a
// logger it can pass along without producing output.
//
// Returns:
//   - A logger that writes nothing.
func NopLogger() Logger {
	capture := &captureWriter{output: io.Discard}

	return &ZerologLogger{
		logger:        zerolog.Nop().Output(capture),
		output:        io.Discard,
		captureWriter: capture,
	}
}

// Debug logs a message at debug level.
//
// Parameters:
//   - msg: Message to record.
//   - fields: Structured pairs to attach to the message.
func (z *ZerologLogger) Debug(msg string, fields ...Field) {
	z.log(DebugLevel, msg, fields)
}

// Info logs a message at info level.
//
// Parameters:
//   - msg: Message to record.
//   - fields: Structured pairs to attach to the message.
func (z *ZerologLogger) Info(msg string, fields ...Field) {
	z.log(InfoLevel, msg, fields)
}

// Warn logs a message at warn level.
//
// Parameters:
//   - msg: Message to record.
//   - fields: Structured pairs to attach to the message.
func (z *ZerologLogger) Warn(msg string, fields ...Field) {
	z.log(WarnLevel, msg, fields)
}

// Error logs a message at error level.
//
// Parameters:
//   - msg: Message to record.
//   - fields: Structured pairs to attach to the message.
func (z *ZerologLogger) Error(msg string, fields ...Field) {
	z.log(ErrorLevel, msg, fields)
}

// log writes a message and its fields at the given level.
//
// The read lock is held across the whole write, so a concurrent Level change
// cannot swap the underlying logger out from under a message being emitted.
//
// Parameters:
//   - level: Severity to record the message at.
//   - msg: Message to record.
//   - fields: Structured pairs to attach to the message.
func (z *ZerologLogger) log(level Level, msg string, fields []Field) {
	z.mu.RLock()
	defer z.mu.RUnlock()

	event := z.logger.WithLevel(level.zerologLevel())
	if event == nil {
		// The level is filtered out, so there is nothing to attach fields to.
		return
	}

	for _, field := range fields {
		field.apply(event)
	}

	event.Msg(msg)
}

// Level sets the minimum log level dynamically.
//
// Parameters:
//   - level: Minimum level to emit.
func (z *ZerologLogger) Level(level Level) {
	z.mu.Lock()
	defer z.mu.Unlock()

	// Create a new logger with the specified level.
	z.logger = z.logger.Level(level.zerologLevel())
}

// SetCaptureFunc sets a callback that receives log messages for TUI display.
//
// Parameters:
//   - captureFunc: Callback invoked with level and message, or nil to disable capture.
func (z *ZerologLogger) SetCaptureFunc(captureFunc LogCaptureFunc) {
	if z.captureWriter != nil {
		z.captureWriter.SetCaptureFunc(captureFunc)
	}
}

// zerologLevel maps a Level onto the matching zerolog severity.
//
// An unrecognised value resolves to info, so a level outside the supported set
// cannot silence a logger.
//
// Parameters:
//   - level: Level to translate.
//
// Returns:
//   - Matching zerolog level.
func (l Level) zerologLevel() zerolog.Level {
	switch l {
	case DebugLevel:
		return zerolog.DebugLevel
	case InfoLevel:
		return zerolog.InfoLevel
	case WarnLevel:
		return zerolog.WarnLevel
	case ErrorLevel:
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// ErrInvalidLogLevel indicates a log level name is not recognised.
var ErrInvalidLogLevel = errors.New("invalid log level")

// ParseLogLevel parses a log level name, rejecting an unrecognised value.
//
// Supported levels are debug, info, warn, and error.
//
// Parameters:
//   - level: Case-insensitive level name.
//
// Returns:
//   - Matching level.
//   - An error wrapping ErrInvalidLogLevel for an unrecognised name.
func ParseLogLevel(level string) (Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return DebugLevel, nil
	case "info":
		return InfoLevel, nil
	case "warn":
		return WarnLevel, nil
	case "error":
		return ErrorLevel, nil
	default:
		return InfoLevel, fmt.Errorf(
			"%w: %q, expected one of debug, info, warn, error",
			ErrInvalidLogLevel,
			level,
		)
	}
}

// ParseLevel parses a string log level into a Level.
//
// Supported levels are debug, info, warn, and error. Unrecognized values default to info.
//
// Parameters:
//   - level: Case-insensitive level name.
//
// Returns:
//   - Matching level, or InfoLevel when unrecognized.
func ParseLevel(level string) Level {
	parsed, _ := ParseLogLevel(level)

	return parsed
}
