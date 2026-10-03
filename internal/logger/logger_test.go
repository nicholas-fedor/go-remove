/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package logger

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// syncBuffer is a thread-safe wrapper around bytes.Buffer for concurrent writes.
type syncBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

// Write writes p to the buffer in a thread-safe manner.
// It implements the io.Writer interface.
func (sb *syncBuffer) Write(p []byte) (int, error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	return sb.buf.Write(p)
}

// String returns the buffer's contents in a thread-safe manner.
func (sb *syncBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	return sb.buf.String()
}

// newTestLogger builds a ZerologLogger writing console output into out.
//
// Parameters:
//   - out: Destination for the console output.
//   - level: Minimum level the logger emits.
//
// Returns:
//   - A logger ready to log into out.
func newTestLogger(out io.Writer, level Level) *ZerologLogger {
	output := zerolog.ConsoleWriter{
		Out:        out,
		TimeFormat: "2006-01-02",
		NoColor:    true,
	}

	zerologLogger := zerolog.New(output).
		With().
		Timestamp().
		Logger().
		Level(level.zerologLevel())

	return &ZerologLogger{logger: zerologLogger, output: output}
}

// TestNewLogger verifies the NewLogger function creates a valid logger.
func TestNewLogger(t *testing.T) {
	t.Parallel()

	got, err := NewLogger()
	require.NoError(t, err)
	assert.NotNil(t, got)

	// Verify the returned logger is a *ZerologLogger.
	_, ok := got.(*ZerologLogger)
	assert.True(t, ok, "expected *ZerologLogger, got %T", got)
}

// TestZerologLogger_LogLevelMethods verifies all log level methods write correctly.
func TestZerologLogger_LogLevelMethods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		level    Level
		logFunc  func(l Logger, msg string, fields ...Field)
		contains string
	}{
		{
			name:     "debug level",
			level:    DebugLevel,
			logFunc:  func(l Logger, msg string, fields ...Field) { l.Debug(msg, fields...) },
			contains: "DBG",
		},
		{
			name:     "info level",
			level:    InfoLevel,
			logFunc:  func(l Logger, msg string, fields ...Field) { l.Info(msg, fields...) },
			contains: "INF",
		},
		{
			name:     "warn level",
			level:    WarnLevel,
			logFunc:  func(l Logger, msg string, fields ...Field) { l.Warn(msg, fields...) },
			contains: "WRN",
		},
		{
			name:     "error level",
			level:    ErrorLevel,
			logFunc:  func(l Logger, msg string, fields ...Field) { l.Error(msg, fields...) },
			contains: "ERR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			logger := newTestLogger(&buf, tt.level)
			tt.logFunc(logger, "test message")

			assert.Contains(t, buf.String(), tt.contains)
			assert.Contains(t, buf.String(), "test message")
		})
	}
}

// TestZerologLogger_Fields verifies every field constructor reaches the output.
func TestZerologLogger_Fields(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	logger := newTestLogger(&buf, DebugLevel)
	logger.Info(
		"field message",
		Str("component", "test"),
		Int("count", 42),
		Bool("flag", true),
		Err(errors.New("boom")),
	)

	output := buf.String()
	assert.Contains(t, output, "field message")
	assert.Contains(t, output, "component=test")
	assert.Contains(t, output, "count=42")
	assert.Contains(t, output, "flag=true")
	assert.Contains(t, output, "error=boom")
}

// TestZerologLogger_NilErrField verifies a nil error writes no error field.
func TestZerologLogger_NilErrField(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	logger := newTestLogger(&buf, DebugLevel)
	logger.Warn("no error attached", Err(nil))

	output := buf.String()
	assert.Contains(t, output, "no error attached")
	assert.NotContains(t, output, "error=")
}

// TestZerologLogger_Level verifies the Level method changes log level dynamically.
func TestZerologLogger_Level(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		initialLevel Level
		newLevel     Level
		logLevel     Level
		shouldLog    bool
	}{
		{
			name:         "change from info to debug",
			initialLevel: InfoLevel,
			newLevel:     DebugLevel,
			logLevel:     DebugLevel,
			shouldLog:    true,
		},
		{
			name:         "change from debug to error",
			initialLevel: DebugLevel,
			newLevel:     ErrorLevel,
			logLevel:     InfoLevel,
			shouldLog:    false,
		},
		{
			name:         "no change in level",
			initialLevel: InfoLevel,
			newLevel:     InfoLevel,
			logLevel:     InfoLevel,
			shouldLog:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			logger := newTestLogger(&buf, tt.initialLevel)
			logger.Level(tt.newLevel)

			switch tt.logLevel {
			case DebugLevel:
				logger.Debug("test debug")
			case InfoLevel:
				logger.Info("test info")
			case ErrorLevel:
				logger.Error("test error")
			}

			if tt.shouldLog {
				assert.NotEmpty(t, buf.String())
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}

// TestLevel_zerologLevel verifies Level maps onto the matching zerolog severity.
func TestLevel_zerologLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level Level
		want  zerolog.Level
	}{
		{name: "debug", level: DebugLevel, want: zerolog.DebugLevel},
		{name: "info", level: InfoLevel, want: zerolog.InfoLevel},
		{name: "warn", level: WarnLevel, want: zerolog.WarnLevel},
		{name: "error", level: ErrorLevel, want: zerolog.ErrorLevel},
		{name: "unknown value falls back to info", level: Level(99), want: zerolog.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.level.zerologLevel())
		})
	}
}

// TestParseLevel verifies the ParseLevel function's string parsing.
func TestParseLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level string
		want  Level
	}{
		{
			name:  "debug lowercase",
			level: "debug",
			want:  DebugLevel,
		},
		{
			name:  "info lowercase",
			level: "info",
			want:  InfoLevel,
		},
		{
			name:  "warn lowercase",
			level: "warn",
			want:  WarnLevel,
		},
		{
			name:  "error lowercase",
			level: "error",
			want:  ErrorLevel,
		},
		{
			name:  "debug uppercase",
			level: "DEBUG",
			want:  DebugLevel,
		},
		{
			name:  "info mixed case",
			level: "Info",
			want:  InfoLevel,
		},
		{
			name:  "unknown level defaults to info",
			level: "unknown",
			want:  InfoLevel,
		},
		{
			name:  "empty string defaults to info",
			level: "",
			want:  InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, ParseLevel(tt.level))
		})
	}
}

// TestParseLogLevel verifies ParseLogLevel rejects unrecognised names.
func TestParseLogLevel(t *testing.T) {
	t.Parallel()

	got, err := ParseLogLevel("warn")
	require.NoError(t, err)
	assert.Equal(t, WarnLevel, got)

	got, err = ParseLogLevel("nonsense")
	require.ErrorIs(t, err, ErrInvalidLogLevel)
	assert.Equal(t, InfoLevel, got, "an unrecognised name reports info alongside the error")
}

// BenchmarkZerologLogger_Debug benchmarks the Debug method.
func BenchmarkZerologLogger_Debug(b *testing.B) {
	logger := newTestLogger(io.Discard, DebugLevel)

	b.ResetTimer()

	for b.Loop() {
		logger.Debug("benchmark message")
	}
}

// BenchmarkZerologLogger_Info benchmarks the Info method.
func BenchmarkZerologLogger_Info(b *testing.B) {
	logger := newTestLogger(io.Discard, DebugLevel)

	b.ResetTimer()

	for b.Loop() {
		logger.Info("benchmark message")
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
	t.Parallel()

	buf := &syncBuffer{}
	logger := newTestLogger(buf, DebugLevel)

	var wg sync.WaitGroup

	wg.Go(func() {
		for range 100 {
			logger.Debug("debug message")
		}
	})
	wg.Go(func() {
		for range 100 {
			logger.Info("info message")
		}
	})
	wg.Go(func() {
		for range 100 {
			logger.Warn("warn message")
		}
	})
	wg.Go(func() {
		for range 100 {
			logger.Error("error message")
		}
	})
	wg.Wait()

	// Verify output contains messages from all levels.
	output := buf.String()
	assert.Positive(t, strings.Count(output, "debug message"))
	assert.Positive(t, strings.Count(output, "info message"))
	assert.Positive(t, strings.Count(output, "warn message"))
	assert.Positive(t, strings.Count(output, "error message"))
}

// TestZerologLogger_ConcurrentLevelChanges verifies level changes are safe
// while messages are being written.
func TestZerologLogger_ConcurrentLevelChanges(t *testing.T) {
	t.Parallel()

	logger := newTestLogger(io.Discard, InfoLevel)

	levels := []Level{DebugLevel, InfoLevel, WarnLevel, ErrorLevel}

	var wg sync.WaitGroup

	for range 50 {
		wg.Go(func() {
			for _, level := range levels {
				logger.Level(level)
			}
		})
		wg.Go(func() {
			for _, level := range levels {
				logger.log(level, "concurrent message", nil)
			}
		})
	}

	wg.Wait()
}
