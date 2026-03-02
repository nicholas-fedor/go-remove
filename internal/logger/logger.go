/*
Copyright © 2025 Nicholas Fedor <nick@nickfedor.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
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

// ErrLoggerNil indicates that the logger instance is nil when an operation is attempted.
var ErrLoggerNil = errors.New("logger is nil")

// LogCaptureFunc is a callback function that receives captured log messages.
// The level parameter contains the log level string (e.g., "DBG", "INF", "WRN", "ERR").
// The msg parameter contains the formatted log message.
type LogCaptureFunc func(level, msg string)

// Logger defines the logging operations required by the application.
type Logger interface {
	// Debug returns a debug-level event for logging.
	Debug() *zerolog.Event
	// Info returns an info-level event for logging.
	Info() *zerolog.Event
	// Warn returns a warn-level event for logging.
	Warn() *zerolog.Event
	// Error returns an error-level event for logging.
	Error() *zerolog.Event
	// Sync flushes any buffered log entries (no-op for zerolog, kept for compatibility).
	Sync() error
	// Level sets the minimum log level dynamically.
	Level(level zerolog.Level)
	// SetCaptureFunc sets a callback function to capture log messages for TUI display.
	// When set, log messages are sent to this callback in addition to normal output.
	SetCaptureFunc(captureFunc LogCaptureFunc)
}

// ZerologLogger wraps zerolog.Logger to implement the Logger interface.
type ZerologLogger struct {
	logger        zerolog.Logger
	mu            sync.RWMutex
	output        io.Writer
	captureFunc   LogCaptureFunc
	captureWriter *captureWriter
}

// captureWriter wraps an io.Writer and captures written data for TUI display.
type captureWriter struct {
	mu             sync.RWMutex
	output         io.Writer
	captureFunc    LogCaptureFunc
	captureEnabled bool
}

// Write implements io.Writer, writing to the underlying output and capturing the data.
// When capture is enabled, output is discarded to prevent duplicate logs in TUI mode.
func (w *captureWriter) Write(data []byte) (int, error) {
	w.mu.RLock()
	output := w.output
	enabled := w.captureEnabled
	w.mu.RUnlock()

	// If capture is enabled, discard the output to prevent duplicate logs.
	// The log message will only be sent through the capture mechanism to the TUI log panel.
	if enabled {
		w.captureLogMessage(string(data))

		// Write to io.Discard to satisfy the writer interface without producing output.
		bytesWritten, err := io.Discard.Write(data)
		if err != nil {
			return bytesWritten, fmt.Errorf("failed to write to discard: %w", err)
		}

		return bytesWritten, nil
	}

	// Capture is not enabled, write to the underlying output normally.
	bytesWritten, err := output.Write(data)
	if err != nil {
		return bytesWritten, fmt.Errorf("failed to write to output: %w", err)
	}

	return bytesWritten, nil
}

// captureLogMessage parses and captures the log message.
// Expected format from ConsoleWriter: "<timestamp> <LEVEL> <message>".
func (w *captureWriter) captureLogMessage(logLine string) {
	w.mu.RLock()
	capture := w.captureFunc
	w.mu.RUnlock()

	if capture == nil {
		return
	}

	// Parse the log line to extract level and message
	// Format: "2006-01-02T15:04:05Z07:00 DBG message here"
	parts := strings.Fields(logLine)

	const minLogParts = 3 // timestamp, level, message
	if len(parts) < minLogParts {
		// Not enough parts, use the whole line
		capture("LOG", strings.TrimSpace(logLine))

		return
	}

	// The level is typically the second field (index 1)
	level := parts[1]

	// The message is everything after the level
	msgStart := strings.Index(logLine, level) + len(level)
	msg := strings.TrimSpace(logLine[msgStart:])

	capture(level, msg)
}

// SetCaptureFunc sets the capture function for the underlying capture writer.
// When a non-nil captureFunc is provided, capture is enabled and log messages
// will be sent to the callback instead of the normal output.
func (w *captureWriter) SetCaptureFunc(captureFunc LogCaptureFunc) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.captureFunc = captureFunc
	w.captureEnabled = captureFunc != nil
}

// SetupCaptureBridge configures the capture writer with a bridge function that
// calls the provided callback. This is used internally to connect the capture
// writer to the logger's capture function.
func (w *captureWriter) SetupCaptureBridge(bridge func(level, msg string)) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.captureFunc = bridge
}

// NewLogger creates a new zerolog-based logger with console output.
//
// The logger is configured with:
//   - ConsoleWriter output to os.Stderr
//   - RFC3339 timestamp format
//   - Info level as default
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

// NewLoggerWithCapture creates a new zerolog-based logger that supports log capture.
//
// The returned logger uses a captureWriter that can send log messages to a callback
// for display in the TUI. This is useful for verbose mode where debug logs should
// appear within the TUI interface rather than being written directly to stderr.
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
		captureFunc:   nil,
		captureWriter: captureWriter,
	}

	return logger, captureWriter, nil
}

// Debug returns a debug-level event for logging.
func (z *ZerologLogger) Debug() *zerolog.Event {
	z.mu.RLock()
	defer z.mu.RUnlock()

	//nolint:zerologlint // Factory method returns event for chaining by design
	return z.logger.Debug()
}

// Info returns an info-level event for logging.
func (z *ZerologLogger) Info() *zerolog.Event {
	z.mu.RLock()
	defer z.mu.RUnlock()

	//nolint:zerologlint // Factory method returns event for chaining by design
	return z.logger.Info()
}

// Warn returns a warn-level event for logging.
func (z *ZerologLogger) Warn() *zerolog.Event {
	z.mu.RLock()
	defer z.mu.RUnlock()

	//nolint:zerologlint // Factory method returns event for chaining by design
	return z.logger.Warn()
}

// Error returns an error-level event for logging.
func (z *ZerologLogger) Error() *zerolog.Event {
	z.mu.RLock()
	defer z.mu.RUnlock()

	//nolint:zerologlint // Factory method returns event for chaining by design
	return z.logger.Error()
}

// Sync is a no-op for zerolog, kept for interface compatibility.
// zerolog writes directly to the output without buffering.
func (z *ZerologLogger) Sync() error {
	z.mu.RLock()
	defer z.mu.RUnlock()

	return nil
}

// Level sets the minimum log level dynamically.
func (z *ZerologLogger) Level(level zerolog.Level) {
	z.mu.Lock()
	defer z.mu.Unlock()

	// Create a new logger with the specified level.
	z.logger = z.logger.Level(level)
}

// SetCaptureFunc sets a callback function to capture log messages for TUI display.
// When set, log messages are parsed and sent to this callback.
func (z *ZerologLogger) SetCaptureFunc(captureFunc LogCaptureFunc) {
	z.mu.Lock()
	z.captureFunc = captureFunc
	z.mu.Unlock()

	// Set up the bridge function on the capture writer when a callback is provided.
	// The bridge function reads the current captureFunc from the logger and calls it.
	if z.captureWriter != nil && captureFunc != nil {
		z.captureWriter.SetupCaptureBridge(func(level, msg string) {
			z.mu.RLock()
			capture := z.captureFunc
			z.mu.RUnlock()

			if capture != nil {
				capture(level, msg)
			}
		})
		z.captureWriter.SetCaptureFunc(captureFunc)
	}
}

// ParseLevel parses a string log level into a zerolog.Level.
//
// Supported levels (case-insensitive):
//   - "debug" -> DebugLevel
//   - "info" -> InfoLevel
//   - "warn" -> WarnLevel
//   - "error" -> ErrorLevel
//
// Defaults to InfoLevel if the level is unrecognized.
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
