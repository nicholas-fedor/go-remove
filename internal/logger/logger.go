/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package logger provides a logging interface and implementation for go-remove.
package logger

import (
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

// Logger defines the logging operations required by the application.
type Logger interface {
	// Debug returns a debug-level event for logging.
	//
	// Returns:
	//   - Zerolog event for a debug message.
	Debug() *zerolog.Event

	// Info returns an info-level event for logging.
	//
	// Returns:
	//   - Zerolog event for an info message.
	Info() *zerolog.Event

	// Warn returns a warn-level event for logging.
	//
	// Returns:
	//   - Zerolog event for a warning message.
	Warn() *zerolog.Event

	// Error returns an error-level event for logging.
	//
	// Returns:
	//   - Zerolog event for an error message.
	Error() *zerolog.Event

	// Sync flushes any buffered log entries.
	//
	// This is a no-op for zerolog and exists for interface compatibility.
	//
	// Returns:
	//   - Always nil.
	Sync() error

	// Level sets the minimum log level dynamically.
	//
	// Parameters:
	//   - level: Minimum zerolog level to emit.
	Level(level zerolog.Level)

	// SetCaptureFunc sets a callback that receives log messages for TUI display.
	//
	// Parameters:
	//   - captureFunc: Callback invoked with level and message, or nil to disable capture.
	SetCaptureFunc(captureFunc LogCaptureFunc)
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

	return &ZerologLogger{
		logger: zerologLogger,
		output: output,
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
		NoColor:    false,
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

// Debug returns a debug-level event for logging.
//
// Returns:
//   - Zerolog event for a debug message.
func (z *ZerologLogger) Debug() *zerolog.Event {
	z.mu.RLock()
	defer z.mu.RUnlock()

	//nolint:zerologlint // Factory method returns event for chaining by design
	return z.logger.Debug()
}

// Info returns an info-level event for logging.
//
// Returns:
//   - Zerolog event for an info message.
func (z *ZerologLogger) Info() *zerolog.Event {
	z.mu.RLock()
	defer z.mu.RUnlock()

	//nolint:zerologlint // Factory method returns event for chaining by design
	return z.logger.Info()
}

// Warn returns a warn-level event for logging.
//
// Returns:
//   - Zerolog event for a warning message.
func (z *ZerologLogger) Warn() *zerolog.Event {
	z.mu.RLock()
	defer z.mu.RUnlock()

	//nolint:zerologlint // Factory method returns event for chaining by design
	return z.logger.Warn()
}

// Error returns an error-level event for logging.
//
// Returns:
//   - Zerolog event for an error message.
func (z *ZerologLogger) Error() *zerolog.Event {
	z.mu.RLock()
	defer z.mu.RUnlock()

	//nolint:zerologlint // Factory method returns event for chaining by design
	return z.logger.Error()
}

// Sync flushes any buffered log entries.
//
// This is a no-op for zerolog and exists for interface compatibility.
//
// Returns:
//   - Always nil.
func (z *ZerologLogger) Sync() error {
	z.mu.RLock()
	defer z.mu.RUnlock()

	return nil
}

// Level sets the minimum log level dynamically.
//
// Parameters:
//   - level: Minimum zerolog level to emit.
func (z *ZerologLogger) Level(level zerolog.Level) {
	z.mu.Lock()
	defer z.mu.Unlock()

	// Create a new logger with the specified level.
	z.logger = z.logger.Level(level)
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

// ParseLevel parses a string log level into a zerolog.Level.
//
// Supported levels are debug, info, warn, and error. Unrecognized values default to info.
//
// Parameters:
//   - level: Case-insensitive level name.
//
// Returns:
//   - Matching zerolog level, or InfoLevel when unrecognized.
func ParseLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}
