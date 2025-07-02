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
	"os"
	"runtime"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Sampling constants for logger configuration.
const (
	samplingInitial    = 100 // Initial number of log entries before sampling
	samplingThereafter = 100 // Number of log entries to sample after initial
)

// ErrLoggerNil indicates that the logger instance is nil when an operation is attempted.
var ErrLoggerNil = errors.New("logger is nil")

// Logger defines the logging operations required by the application.
type Logger interface {
	Sync() error               // Flush any buffered log entries
	Sugar() *zap.SugaredLogger // Return a sugared logger for key-value logging
}

// ZapLogger adapts zap.Logger to implement the Logger interface.
type ZapLogger struct {
	*zap.Logger // Embedded zap.Logger for core logging functionality
}

// NewZapLogger creates a new production-ready Zap logger.
func NewZapLogger() (Logger, error) {
	// Define a custom configuration for the logger to ensure cross-platform compatibility.
	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(zapcore.InfoLevel),
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    samplingInitial,
			Thereafter: samplingThereafter,
		},
		Encoding: "console", // Use console encoding for human-readable output
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder, // Use uppercase level names (e.g., INFO)
			EncodeTime:     zapcore.ISO8601TimeEncoder,  // Use ISO 8601 format for timestamps
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stderr"}, // Explicitly write to stderr
		ErrorOutputPaths: []string{"stderr"},
	}

	// Build the logger with a locked writer to ensure thread-safe writes to stderr.
	logger, err := config.Build(zap.WrapCore(func(_ zapcore.Core) zapcore.Core {
		writer := zapcore.Lock(os.Stderr)

		return zapcore.NewCore(
			zapcore.NewConsoleEncoder(config.EncoderConfig),
			writer,
			config.Level,
		)
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	return &ZapLogger{logger}, nil
}

// Sync flushes any buffered log entries to the underlying output.
func (z *ZapLogger) Sync() error {
	// Check for nil logger to avoid panics on uninitialized instances.
	if z.Logger == nil {
		return ErrLoggerNil
	}

	// On Windows, ignore sync errors that are benign (e.g., invalid handle).
	if err := z.Logger.Sync(); err != nil && runtime.GOOS != "windows" {
		return fmt.Errorf("failed to sync logger: %w", err)
	}

	return nil
}

// Sugar returns a sugared logger for convenient key-value logging.
func (z *ZapLogger) Sugar() *zap.SugaredLogger {
	// Return nil if the logger is uninitialized to prevent nil pointer dereference.
	if z.Logger == nil {
		return nil
	}

	// Delegate to zap.Logger’s Sugar method for sugared logging.
	return z.Logger.Sugar()
}
