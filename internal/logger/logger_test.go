/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package logger

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// consoleLineFields is the number of fields a console line of an event with a
// single-token message holds: a timestamp, a level, and the message.
const consoleLineFields = 3

// syncBuffer is a thread-safe wrapper around bytes.Buffer for concurrent writes.
type syncBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

// capturedEntry is one log message the capture callback reported.
type capturedEntry struct {
	level string
	msg   string
}

// messageCounter counts messages per text, for capture callbacks and output
// that are read while other goroutines are still logging.
type messageCounter struct {
	mu     sync.Mutex
	counts map[string]int
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

// newMessageCounter builds an empty counter.
func newMessageCounter() *messageCounter {
	return &messageCounter{counts: make(map[string]int)}
}

// record counts one occurrence of msg.
func (mc *messageCounter) record(msg string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.counts[msg]++
}

// count reports how many times msg was recorded.
func (mc *messageCounter) count(msg string) int {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	return mc.counts[msg]
}

// loggedMessages counts the messages present in console output.
//
// Every logged event produces one console line whose last field is the message.
//
// Parameters:
//   - out: Console output the logger produced.
//
// Returns:
//   - Occurrence count per logged message.
func loggedMessages(out string) map[string]int {
	counts := make(map[string]int)

	for line := range strings.SplitSeq(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < consoleLineFields {
			continue
		}

		counts[fields[len(fields)-1]]++
	}

	return counts
}

// eventName names the nth event of a concurrency test.
//
// The names are distinct and none of them contains another, so counting names
// in console output cannot mistake one event for another.
func eventName(event int) string {
	return fmt.Sprintf("event-%04d", event)
}

// newTestLogger builds a ZerologLogger writing console output into out.
//
// The logger is wired the same way as the ones the constructors build, so
// capture suppresses and restores out exactly as it does in production.
//
// Parameters:
//   - out: Destination for the console output.
//   - level: Minimum level the logger emits.
//
// Returns:
//   - A logger ready to log into out.
func newTestLogger(out io.Writer, level Level) *ZerologLogger {
	capture := &captureWriter{output: out}

	output := zerolog.ConsoleWriter{
		Out:        capture,
		TimeFormat: "2006-01-02",
		NoColor:    true,
	}

	zerologLogger := zerolog.New(output).
		With().
		Timestamp().
		Logger().
		Level(level.zerologLevel()).
		Hook(zerolog.HookFunc(capture.Run))

	return &ZerologLogger{logger: zerologLogger, output: output, captureWriter: capture}
}

// TestNewLogger verifies the NewLogger function creates a valid logger.
func TestNewLogger(t *testing.T) {
	t.Parallel()

	got, err := NewLogger()
	require.NoError(t, err)
	assert.NotNil(t, got)

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

// TestZerologLogger_SetCaptureFunc verifies capture delivers the level and message
// of every logged call, and that output is suppressed while capture is on.
func TestZerologLogger_SetCaptureFunc(t *testing.T) {
	t.Parallel()

	var (
		buf      bytes.Buffer
		captured []capturedEntry
	)

	logger := newTestLogger(&buf, DebugLevel)
	logger.SetCaptureFunc(func(level, msg string) {
		captured = append(captured, capturedEntry{level: level, msg: msg})
	})

	logger.Debug("first message")
	logger.Info("second message", Str("component", "test"))
	logger.Warn("third message")
	logger.Error("fourth message")

	assert.Equal(t, []capturedEntry{
		{level: "DBG", msg: "first message"},
		{level: "INF", msg: "second message"},
		{level: "WRN", msg: "third message"},
		{level: "ERR", msg: "fourth message"},
	}, captured)
	assert.Empty(t, buf.String(), "output must be suppressed while capture is enabled")
}

// TestZerologLogger_SetCaptureFuncNilDisablesCapture verifies a nil callback stops
// capture and restores the logger's output.
func TestZerologLogger_SetCaptureFuncNilDisablesCapture(t *testing.T) {
	t.Parallel()

	var (
		buf      bytes.Buffer
		captured []capturedEntry
	)

	logger := newTestLogger(&buf, InfoLevel)
	logger.SetCaptureFunc(func(level, msg string) {
		captured = append(captured, capturedEntry{level: level, msg: msg})
	})
	logger.Info("message with capture")

	require.Len(t, captured, 1, "expected the message to be captured before disabling")
	assert.Empty(t, buf.String(), "output must be suppressed while capture is enabled")

	logger.SetCaptureFunc(nil)
	logger.Info("message without capture")

	assert.Len(t, captured, 1, "no message may be captured once capture is disabled")
	assert.Contains(t, buf.String(), "message without capture")
	assert.NotContains(t, buf.String(), "message with capture")
}

// TestZerologLogger_CaptureRespectsLevelFilter verifies filtered messages reach
// neither the capture callback nor the output.
func TestZerologLogger_CaptureRespectsLevelFilter(t *testing.T) {
	t.Parallel()

	var (
		buf      bytes.Buffer
		captured []capturedEntry
	)

	logger := newTestLogger(&buf, InfoLevel)
	logger.SetCaptureFunc(func(level, msg string) {
		captured = append(captured, capturedEntry{level: level, msg: msg})
	})

	logger.Debug("filtered out")
	logger.Info("emitted")

	assert.Equal(t, []capturedEntry{{level: "INF", msg: "emitted"}}, captured)
	assert.NotContains(t, buf.String(), "filtered out")
}

// TestZerologLogger_CaptureRepeatedCalls verifies every logged call reaches the
// capture callback.
func TestZerologLogger_CaptureRepeatedCalls(t *testing.T) {
	t.Parallel()

	var (
		buf      bytes.Buffer
		captured int
	)

	logger := newTestLogger(&buf, InfoLevel)
	logger.SetCaptureFunc(func(_, _ string) {
		captured++
	})

	const calls = 50

	for range calls {
		logger.Info("repeated message")
	}

	assert.Equal(t, calls, captured)
}

// TestZerologLogger_EnableCaptureWhileLogging verifies that enabling capture
// while events are being emitted leaves every event accounted for once.
//
// Whichever side of the swap an event was emitted on, it is either captured or
// written to the output, never both and never neither.
func TestZerologLogger_EnableCaptureWhileLogging(t *testing.T) {
	t.Parallel()

	assertEventsAccountedFor(t, false, func(logger *ZerologLogger, captured *messageCounter) {
		logger.SetCaptureFunc(func(_, msg string) {
			captured.record(msg)
		})
	})
}

// TestZerologLogger_DisableCaptureWhileLogging verifies that setting the capture
// callback to nil while events are being emitted leaves every event accounted
// for once.
//
// Whichever side of the swap an event was emitted on, it is either captured or
// written to the output, never both and never neither.
func TestZerologLogger_DisableCaptureWhileLogging(t *testing.T) {
	t.Parallel()

	assertEventsAccountedFor(t, true, func(logger *ZerologLogger, _ *messageCounter) {
		logger.SetCaptureFunc(nil)
	})
}

// assertEventsAccountedFor changes a logger's capture state with swap while
// several goroutines log, then checks that every event reached one destination.
//
// Capture delivers a message to the callback and drops the logger's own copy,
// while a disabled callback leaves the logger's output alone. Each event must
// therefore appear in exactly one of the two: missing from both means it was
// dropped, and present in both means it was shown twice.
//
// Parameters:
//   - t: Testing handle.
//   - captureEnabled: Whether capture is on before the swap.
//   - swap: Capture state change to apply while logging is in progress.
func assertEventsAccountedFor(
	t *testing.T,
	captureEnabled bool,
	swap func(logger *ZerologLogger, captured *messageCounter),
) {
	t.Helper()

	const (
		writers   = 4
		perWriter = 250
		events    = writers * perWriter
	)

	buf := &syncBuffer{}
	logger := newTestLogger(buf, InfoLevel)
	captured := newMessageCounter()

	if captureEnabled {
		logger.SetCaptureFunc(func(_, msg string) {
			captured.record(msg)
		})
	}

	var wg sync.WaitGroup

	for writer := range writers {
		base := writer * perWriter

		wg.Go(func() {
			for event := range perWriter {
				logger.Info(eventName(base + event))
			}
		})
	}

	wg.Go(func() {
		swap(logger, captured)
	})

	wg.Wait()

	written := loggedMessages(buf.String())

	for event := range events {
		name := eventName(event)
		delivered := captured.count(name) + written[name]

		assert.Equal(
			t,
			1,
			delivered,
			"event %q was delivered %d times, want exactly one of capture or output",
			name,
			delivered,
		)
	}
}

// TestCaptureLevel verifies each severity maps onto its capture level name.
func TestCaptureLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level zerolog.Level
		want  string
	}{
		{name: "debug", level: zerolog.DebugLevel, want: "DBG"},
		{name: "info", level: zerolog.InfoLevel, want: "INF"},
		{name: "warn", level: zerolog.WarnLevel, want: "WRN"},
		{name: "error", level: zerolog.ErrorLevel, want: "ERR"},
		{name: "fatal", level: zerolog.FatalLevel, want: "FTL"},
		{name: "trace falls back", level: zerolog.TraceLevel, want: "LOG"},
		{name: "panic falls back", level: zerolog.PanicLevel, want: "LOG"},
		{name: "no level falls back", level: zerolog.NoLevel, want: "LOG"},
		{name: "disabled falls back", level: zerolog.Disabled, want: "LOG"},
		{name: "unknown value falls back", level: zerolog.Level(99), want: "LOG"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, captureLevel(tt.level))
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

// TestParseLogLevel verifies ParseLogLevel rejects unrecognized names.
func TestParseLogLevel(t *testing.T) {
	t.Parallel()

	got, err := ParseLogLevel("warn")
	require.NoError(t, err)
	assert.Equal(t, WarnLevel, got)

	got, err = ParseLogLevel("nonsense")
	require.ErrorIs(t, err, ErrInvalidLogLevel)
	assert.Equal(t, InfoLevel, got, "an unrecognized name reports info alongside the error")
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
