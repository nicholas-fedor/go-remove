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

	"go.uber.org/zap"
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
	// Initialize a production-configured Zap logger.
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Wrap the Zap logger in ZapLogger to satisfy the Logger interface.
	return &ZapLogger{logger}, nil
}

// Sync flushes any buffered log entries to the underlying output.
func (z *ZapLogger) Sync() error {
	// Check for nil logger to avoid panics on uninitialized instances.
	if z.Logger == nil {
		return ErrLoggerNil
	}

	// Flush the logger and wrap any errors for context.
	if err := z.Logger.Sync(); err != nil {
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
