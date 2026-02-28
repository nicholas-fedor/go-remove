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

package logger

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewLogger verifies the NewLogger function creates a valid logger.
func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful creation",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewLogger()
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)

			// Verify the returned logger is a *ZerologLogger.
			_, ok := got.(*ZerologLogger)
			assert.True(t, ok, "expected *ZerologLogger, got %T", got)
		})
	}
}

// TestZerologLogger_LogLevelMethods verifies all log level methods work correctly.
func TestZerologLogger_LogLevelMethods(t *testing.T) {
	tests := []struct {
		name     string
		level    zerolog.Level
		logFunc  func(l Logger) *zerolog.Event
		wantLog  bool
		contains string
	}{
		{
			name:     "debug level with debug enabled",
			level:    zerolog.DebugLevel,
			logFunc:  func(l Logger) *zerolog.Event { return l.Debug() },
			wantLog:  true,
			contains: "DBG",
		},
		{
			name:     "info level",
			level:    zerolog.InfoLevel,
			logFunc:  func(l Logger) *zerolog.Event { return l.Info() },
			wantLog:  true,
			contains: "INF",
		},
		{
			name:     "warn level",
			level:    zerolog.WarnLevel,
			logFunc:  func(l Logger) *zerolog.Event { return l.Warn() },
			wantLog:  true,
			contains: "WRN",
		},
		{
			name:     "error level",
			level:    zerolog.ErrorLevel,
			logFunc:  func(l Logger) *zerolog.Event { return l.Error() },
			wantLog:  true,
			contains: "ERR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a buffer to capture log output.
			var buf bytes.Buffer

			output := zerolog.ConsoleWriter{
				Out:        &buf,
				TimeFormat: "2006-01-02",
				NoColor:    true,
			}

			// Create logger with specific level.
			zl := zerolog.New(output).
				With().
				Timestamp().
				Logger().
				Level(tt.level)

			logger := &ZerologLogger{logger: zl, output: output}

			// Call the log function with a test message.
			tt.logFunc(logger).Msg("test message")

			outputStr := buf.String()

			if tt.wantLog {
				assert.Contains(t, outputStr, tt.contains)
				assert.Contains(t, outputStr, "test message")
			}
		})
	}
}

// TestZerologLogger_Sync verifies the Sync method's behavior.
func TestZerologLogger_Sync(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *ZerologLogger
		wantErr bool
	}{
		{
			name: "sync with valid logger",
			setup: func() *ZerologLogger {
				zl := zerolog.New(nil).With().Logger()

				return &ZerologLogger{logger: zl}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := tt.setup()
			err := z.Sync()
			assert.NoError(t, err)
		})
	}
}

// TestZerologLogger_Level verifies the Level method changes log level dynamically.
func TestZerologLogger_Level(t *testing.T) {
	tests := []struct {
		name         string
		initialLevel zerolog.Level
		newLevel     zerolog.Level
		logLevel     zerolog.Level
		shouldLog    bool
	}{
		{
			name:         "change from info to debug",
			initialLevel: zerolog.InfoLevel,
			newLevel:     zerolog.DebugLevel,
			logLevel:     zerolog.DebugLevel,
			shouldLog:    true,
		},
		{
			name:         "change from debug to error",
			initialLevel: zerolog.DebugLevel,
			newLevel:     zerolog.ErrorLevel,
			logLevel:     zerolog.InfoLevel,
			shouldLog:    false,
		},
		{
			name:         "no change in level",
			initialLevel: zerolog.InfoLevel,
			newLevel:     zerolog.InfoLevel,
			logLevel:     zerolog.InfoLevel,
			shouldLog:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			output := zerolog.ConsoleWriter{
				Out:        &buf,
				TimeFormat: "2006-01-02",
				NoColor:    true,
			}

			// Create logger with initial level.
			zl := zerolog.New(output).
				With().
				Timestamp().
				Logger().
				Level(tt.initialLevel)

			logger := &ZerologLogger{logger: zl, output: output}

			// Change the level.
			logger.Level(tt.newLevel)

			// Try to log at the specified level.
			switch tt.logLevel {
			case zerolog.DebugLevel:
				logger.Debug().Msg("test debug")
			case zerolog.InfoLevel:
				logger.Info().Msg("test info")
			case zerolog.ErrorLevel:
				logger.Error().Msg("test error")
			}

			outputStr := buf.String()

			if tt.shouldLog {
				assert.NotEmpty(t, outputStr)
			} else {
				assert.Empty(t, outputStr)
			}
		})
	}
}

// TestParseLevel verifies the ParseLevel function's string parsing.
func TestParseLevel(t *testing.T) {
	tests := []struct {
		name  string
		level string
		want  zerolog.Level
	}{
		{
			name:  "debug lowercase",
			level: "debug",
			want:  zerolog.DebugLevel,
		},
		{
			name:  "info lowercase",
			level: "info",
			want:  zerolog.InfoLevel,
		},
		{
			name:  "warn lowercase",
			level: "warn",
			want:  zerolog.WarnLevel,
		},
		{
			name:  "error lowercase",
			level: "error",
			want:  zerolog.ErrorLevel,
		},
		{
			name:  "debug uppercase",
			level: "DEBUG",
			want:  zerolog.DebugLevel,
		},
		{
			name:  "info mixed case",
			level: "Info",
			want:  zerolog.InfoLevel,
		},
		{
			name:  "unknown level defaults to info",
			level: "unknown",
			want:  zerolog.InfoLevel,
		},
		{
			name:  "empty string defaults to info",
			level: "",
			want:  zerolog.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseLevel(tt.level)
			assert.Equal(t, tt.want, got)
		})
	}
}

// BenchmarkZerologLogger_Debug benchmarks the Debug method.
func BenchmarkZerologLogger_Debug(b *testing.B) {
	zl := zerolog.New(nil).With().Logger()
	logger := &ZerologLogger{logger: zl}

	b.ResetTimer()

	for b.Loop() {
		logger.Debug().Msg("benchmark message")
	}
}

// BenchmarkZerologLogger_Info benchmarks the Info method.
func BenchmarkZerologLogger_Info(b *testing.B) {
	zl := zerolog.New(nil).With().Logger()
	logger := &ZerologLogger{logger: zl}

	b.ResetTimer()

	for b.Loop() {
		logger.Info().Msg("benchmark message")
	}
}

// BenchmarkParseLevel benchmarks the ParseLevel function.
func BenchmarkParseLevel(b *testing.B) {
	levels := []string{"debug", "info", "warn", "error", "unknown"}

	b.ResetTimer()

	for b.Loop() {
		for _, level := range levels {
			ParseLevel(level)
		}
	}
}

// TestZerologLogger_ConcurrentAccess verifies thread safety of logger methods.
func TestZerologLogger_ConcurrentAccess(t *testing.T) {
	var buf bytes.Buffer

	output := zerolog.ConsoleWriter{
		Out:        &buf,
		TimeFormat: "2006-01-02",
		NoColor:    true,
	}

	zl := zerolog.New(output).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	logger := &ZerologLogger{logger: zl, output: output}

	// Run concurrent logging operations.
	done := make(chan bool, 4)

	go func() {
		for range 100 {
			logger.Debug().Msg("debug message")
		}

		done <- true
	}()

	go func() {
		for range 100 {
			logger.Info().Msg("info message")
		}

		done <- true
	}()

	go func() {
		for range 100 {
			logger.Warn().Msg("warn message")
		}

		done <- true
	}()

	go func() {
		for range 100 {
			logger.Error().Msg("error message")
		}

		done <- true
	}()

	// Wait for all goroutines to complete.
	for range 4 {
		<-done
	}

	// Verify output contains messages from all levels.
	outputStr := buf.String()
	assert.Positive(t, strings.Count(outputStr, "debug message"))
	assert.Positive(t, strings.Count(outputStr, "info message"))
	assert.Positive(t, strings.Count(outputStr, "warn message"))
	assert.Positive(t, strings.Count(outputStr, "error message"))
}
