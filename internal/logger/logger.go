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

// Level names reported to the capture callback, one per severity it names.
const (
	captureDebugLevel = "DBG"
	captureInfoLevel  = "INF"
	captureWarnLevel  = "WRN"
	captureErrorLevel = "ERR"
	captureFatalLevel = "FTL"
	captureOtherLevel = "LOG"
)

// Field kinds, one for each value type a Field can carry.
const (
	fieldString fieldKind = iota
	fieldInt
	fieldBool
	fieldError
)

// ErrInvalidLogLevel indicates a log level name is not recognized.
var ErrInvalidLogLevel = errors.New("invalid log level")

var _ Logger = (*ZerologLogger)(nil)

var _ zerolog.Hook = (*captureWriter)(nil)

// LogCaptureFunc receives captured log messages as a level name such as "DBG"
// or "INF", and the message.
//
// It runs inside the event it captures, while the logger still holds its read
// lock, so it must not call back into that logger. Nothing enforces this.
type LogCaptureFunc func(level, msg string)

// Level is a log severity.
//
// Levels are ordered by increasing severity, so a logger set to a level emits
// every message logged at that level or above.
type Level int8

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

// Field is a structured pair attached to a log message.
//
// A Field is built with Str, Int, Bool, or Err and passed to one of the level
// methods. The zero Field carries no value and writes nothing.
type Field struct {
	kind  fieldKind
	key   string
	value any
}

// ZerologLogger wraps zerolog.Logger to implement the Logger interface.
type ZerologLogger struct {
	logger        zerolog.Logger
	mu            sync.RWMutex
	output        io.Writer
	captureWriter *captureWriter
}

// captureState is one locked observation of a captureWriter's configuration.
//
// The hook and the writer each decide an event's fate from one of these, so
// neither reasons about a half-updated view of the configuration.
type captureState struct {
	output      io.Writer
	captureFunc LogCaptureFunc
}

// captureWriter covers both halves of capture. As the logger's writer it
// discards output while the TUI shows the message, and as a zerolog hook it
// hands the callback the level and message unformatted. The callback's presence
// is the only record of whether capture is on.
type captureWriter struct {
	mu          sync.RWMutex
	output      io.Writer
	captureFunc LogCaptureFunc
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

// snapshot returns the writer's output and capture callback as one locked
// observation.
//
// Both halves of capture call it, so each reads the configuration whole and at
// one moment rather than field by field.
//
// Returns:
//   - The output and capture callback as observed together.
func (w *captureWriter) snapshot() captureState {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return captureState{output: w.output, captureFunc: w.captureFunc}
}

// Write implements io.Writer, writing to the underlying output.
//
// When capture is enabled the hook has already handed this event to the capture
// callback, so this copy is discarded to keep the TUI from showing it twice.
//
// Parameters:
//   - data: Log bytes produced by zerolog.
//
// Returns:
//   - Number of bytes accepted.
//   - An error if the underlying writer fails.
func (w *captureWriter) Write(data []byte) (int, error) {
	state := w.snapshot()

	if state.captureFunc != nil {
		return len(data), nil
	}

	bytesWritten, err := state.output.Write(data)
	if err != nil {
		return bytesWritten, fmt.Errorf("failed to write to output: %w", err)
	}

	return bytesWritten, nil
}

// Run implements zerolog.Hook, sending a logged message to the capture callback.
//
// Parameters:
//   - event: Event being logged, unused.
//   - level: Severity the message was logged at.
//   - msg: Message that was logged.
func (w *captureWriter) Run(_ *zerolog.Event, level zerolog.Level, msg string) {
	capture := w.snapshot().captureFunc
	if capture == nil {
		return
	}

	capture(captureLevel(level), msg)
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
}

// captureLevel translates a zerolog severity into the level name reported to the
// capture callback.
//
// A severity with no name of its own, such as trace, is reported as
// captureOtherLevel.
//
// Parameters:
//   - level: Severity to translate.
//
// Returns:
//   - Level name for the severity.
func captureLevel(level zerolog.Level) string {
	switch level {
	case zerolog.DebugLevel:
		return captureDebugLevel
	case zerolog.InfoLevel:
		return captureInfoLevel
	case zerolog.WarnLevel:
		return captureWarnLevel
	case zerolog.ErrorLevel:
		return captureErrorLevel
	case zerolog.FatalLevel:
		return captureFatalLevel
	case zerolog.TraceLevel, zerolog.PanicLevel, zerolog.NoLevel, zerolog.Disabled:
		return captureOtherLevel
	default:
		return captureOtherLevel
	}
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

	zerologLogger := zerolog.New(output).
		With().
		Timestamp().
		Logger().
		Level(zerolog.InfoLevel)

	// Install a capture writer so SetCaptureFunc always has somewhere to write,
	// whichever constructor built the logger.
	capture := &captureWriter{output: output}

	zerologLogger = zerologLogger.
		Output(capture).
		Hook(zerolog.HookFunc(capture.Run))

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
	captureWriter := &captureWriter{
		output: os.Stderr,
	}

	consoleWriter := zerolog.ConsoleWriter{
		Out:        captureWriter,
		TimeFormat: time.RFC3339,
		NoColor:    true,
	}

	zerologLogger := zerolog.New(consoleWriter).
		With().
		Timestamp().
		Logger().
		Level(zerolog.InfoLevel).
		Hook(zerolog.HookFunc(captureWriter.Run))

	logger := &ZerologLogger{
		logger:        zerologLogger,
		output:        consoleWriter,
		captureWriter: captureWriter,
	}

	return logger, captureWriter, nil
}

// NopLogger returns a logger that discards everything written to it.
//
// Use it where logging is not the subject: tests, and any caller that needs a
// logger to pass along without producing output.
//
// Returns:
//   - A logger that writes nothing.
func NopLogger() Logger {
	capture := &captureWriter{output: io.Discard}

	return &ZerologLogger{
		logger: zerolog.Nop().
			Output(capture).
			Hook(zerolog.HookFunc(capture.Run)),
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
// Parameters:
//   - level: Severity to record the message at.
//   - msg: Message to record.
//   - fields: Structured pairs to attach to the message.
func (z *ZerologLogger) log(level Level, msg string, fields []Field) {
	// The read lock is held across the whole write, so a concurrent Level change
	// cannot swap the underlying logger out from under a message being emitted,
	// and a concurrent SetCaptureFunc cannot swap the capture callback out from
	// between the message's capture hook and its write.
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

	z.logger = z.logger.Level(level.zerologLevel())
}

// SetCaptureFunc sets a callback that receives log messages for TUI display.
// A nil callback disables capture, and output is written again.
//
// Parameters:
//   - captureFunc: Callback invoked with level and message, or nil to disable capture.
func (z *ZerologLogger) SetCaptureFunc(captureFunc LogCaptureFunc) {
	// The write lock, not a read one, so a swap cannot land between the capture
	// hook and the write of an event already being emitted.
	z.mu.Lock()
	defer z.mu.Unlock()

	if z.captureWriter != nil {
		z.captureWriter.SetCaptureFunc(captureFunc)
	}
}

// zerologLevel maps a Level onto the matching zerolog severity.
//
// An unrecognized value resolves to info, so a level outside the supported set
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

// ParseLogLevel parses a log level name, rejecting an unrecognized value.
//
// Supported levels are debug, info, warn, and error.
//
// Parameters:
//   - level: Case-insensitive level name.
//
// Returns:
//   - Matching level.
//   - An error wrapping ErrInvalidLogLevel for an unrecognized name.
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
// Supported levels are debug, info, warn, and error.
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
