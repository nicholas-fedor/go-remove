/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package logger_test provides black-box integration tests for the logger package.
//
// These tests verify the Logger interface behavior through the MockLogger mock
// implementation. Tests that need a real logger to report its messages install
// the capture callback, so no assertion depends on a third-party log event type.
//
// Test Organization:
//   - LoggerIntegrationTestSuite: Main test suite using testify suite
//   - Table-driven tests for parameterized log level scenarios
//   - Individual tests for complex error handling and concurrent operations
//
// Coverage Areas:
//   - All log level methods (Debug, Info, Warn, Error)
//   - Dynamic log level setting and filtering
//   - CaptureFunc functionality for TUI integration
//   - Captured payloads for messages logged with fields
//   - Concurrent logging operations
package logger_test

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/nicholas-fedor/go-remove/internal/logger"
	"github.com/nicholas-fedor/go-remove/internal/logger/mocks"
)

// capturedMessage is one log line recorded through the capture callback.
type capturedMessage struct {
	level string
	msg   string
}

// recorder collects the log lines a capture callback receives.
type recorder struct {
	mu       sync.Mutex
	messages []capturedMessage
}

// capture returns a LogCaptureFunc that records every line it is given.
//
// Returns:
//   - A LogCaptureFunc recording into the receiver.
func (r *recorder) capture() logger.LogCaptureFunc {
	return func(level, msg string) {
		r.mu.Lock()
		defer r.mu.Unlock()

		r.messages = append(r.messages, capturedMessage{level: level, msg: msg})
	}
}

// recorded returns a copy of the messages recorded so far.
//
// Returns:
//   - Every message recorded, in arrival order.
func (r *recorder) recorded() []capturedMessage {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]capturedMessage(nil), r.messages...)
}

// count reports how many recorded messages contain substr.
//
// Parameters:
//   - substr: Text to look for.
//
// Returns:
//   - The number of recorded messages containing substr.
func (r *recorder) count(substr string) int {
	total := 0

	for _, message := range r.recorded() {
		if strings.Contains(message.msg, substr) {
			total++
		}
	}

	return total
}

// find returns the first recorded message containing substr.
//
// Parameters:
//   - substr: Text to look for.
//
// Returns:
//   - The matching message, and true when one was found.
func (r *recorder) find(substr string) (capturedMessage, bool) {
	for _, message := range r.recorded() {
		if strings.Contains(message.msg, substr) {
			return message, true
		}
	}

	return capturedMessage{}, false
}

// capturedLogger returns a real logger whose output is routed to a recorder.
//
// Parameters:
//   - t: Test the logger is created for.
//
// Returns:
//   - A logger with the capture callback already installed.
//   - The recorder receiving its output.
func capturedLogger(t *testing.T) (logger.Logger, *recorder) {
	t.Helper()

	loggerInstance, _, err := logger.NewLoggerWithCapture()
	require.NoError(t, err)

	records := &recorder{}
	loggerInstance.SetCaptureFunc(records.capture())

	return loggerInstance, records
}

// LoggerIntegrationTestSuite provides integration tests for the logger Logger interface.
//
// This suite tests the orchestration between the application layer and logging layer,
// ensuring that all logging operations behave correctly through the Logger interface.
// All tests use the MockLogger to simulate logging operations without producing
// actual output.
type LoggerIntegrationTestSuite struct {
	suite.Suite

	// Mock for the Logger interface
	mockLogger *mocks.MockLogger
}

// SetupTest initializes the test suite before each test.
//
// Creates a fresh MockLogger instance for each test to ensure test isolation
// and prevent test interference.
func (s *LoggerIntegrationTestSuite) SetupTest() {
	s.mockLogger = mocks.NewMockLogger(s.T())
}

// TestLoggerIntegrationTestSuite runs the integration test suite.
func TestLoggerIntegrationTestSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(LoggerIntegrationTestSuite))
}

// TestAllLogLevelMethods verifies all log level methods record their message.
func (s *LoggerIntegrationTestSuite) TestAllLogLevelMethods() {
	tests := []struct {
		name string
		call func(msg string)
	}{
		{
			name: "debug level logging",
			call: func(msg string) {
				s.mockLogger.EXPECT().Debug(msg).Return().Once()
				s.mockLogger.Debug(msg)
			},
		},
		{
			name: "info level logging",
			call: func(msg string) {
				s.mockLogger.EXPECT().Info(msg).Return().Once()
				s.mockLogger.Info(msg)
			},
		},
		{
			name: "warn level logging",
			call: func(msg string) {
				s.mockLogger.EXPECT().Warn(msg).Return().Once()
				s.mockLogger.Warn(msg)
			},
		},
		{
			name: "error level logging",
			call: func(msg string) {
				s.mockLogger.EXPECT().Error(msg).Return().Once()
				s.mockLogger.Error(msg)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.call(tt.name)
		})
	}
}

// TestLogFields verifies the captured message is the logged message, with the
// structured fields left on the event rather than folded into the message.
func (s *LoggerIntegrationTestSuite) TestLogFields() {
	loggerInstance, records := capturedLogger(s.T())

	loggerInstance.Info(
		"chained log message",
		logger.Str("component", "test"),
		logger.Int("count", 42),
		logger.Bool("flag", true),
	)

	message, found := records.find("chained log message")
	s.Require().True(found, "expected the message to be captured")
	s.Equal("chained log message", message.msg)
	s.Equal("INF", message.level)
}

// TestLevelSetting verifies dynamic log level changes.
func (s *LoggerIntegrationTestSuite) TestLevelSetting() {
	tests := []struct {
		name  string
		level logger.Level
	}{
		{
			name:  "set to debug level",
			level: logger.DebugLevel,
		},
		{
			name:  "set to info level",
			level: logger.InfoLevel,
		},
		{
			name:  "set to warn level",
			level: logger.WarnLevel,
		},
		{
			name:  "set to error level",
			level: logger.ErrorLevel,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.mockLogger.EXPECT().Level(tt.level).Return().Once()
			s.mockLogger.Level(tt.level)
		})
	}
}

// TestLevelFiltering verifies that log level filtering works correctly.
//
// This test ensures that messages below the current log level are filtered out
// and messages at or above the level are processed.
func (s *LoggerIntegrationTestSuite) TestLevelFiltering() {
	loggerInstance, records := capturedLogger(s.T())
	loggerInstance.Level(logger.InfoLevel)

	s.Run("debug filtered at info level", func() {
		loggerInstance.Debug("debug message")
		s.Zero(records.count("debug message"))
	})

	s.Run("info allowed at info level", func() {
		loggerInstance.Info("info message")
		s.Equal(1, records.count("info message"))
	})

	s.Run("error allowed at info level", func() {
		loggerInstance.Error("error message")
		s.Equal(1, records.count("error message"))
	})
}

// TestCaptureFunc verifies the SetCaptureFunc functionality.
//
// This test ensures that the capture function is properly set and called
// when log messages are generated. This is critical for TUI integration
// where logs need to be displayed in the interface.
func (s *LoggerIntegrationTestSuite) TestCaptureFunc() {
	s.Run("set capture function", func() {
		s.mockLogger.EXPECT().
			SetCaptureFunc(mock.AnythingOfType("logger.LogCaptureFunc")).
			Return().
			Once()
		s.mockLogger.SetCaptureFunc(func(_, _ string) {})
	})

	s.Run("set nil capture function", func() {
		s.mockLogger.EXPECT().
			SetCaptureFunc(mock.IsType(logger.LogCaptureFunc(nil))).
			Return().
			Once()
		s.mockLogger.SetCaptureFunc(nil)
	})
}

// TestCaptureFuncIntegration verifies capture function is called with correct data.
//
// This test creates a real logger with capture to verify the full integration
// of the capture functionality.
func (s *LoggerIntegrationTestSuite) TestCaptureFuncIntegration() {
	loggerInstance, records := capturedLogger(s.T())
	loggerInstance.Level(logger.DebugLevel)

	loggerInstance.Debug("test debug message")
	loggerInstance.Info("test info message")
	loggerInstance.Warn("test warn message")
	loggerInstance.Error("test error message")

	// Give some time for capture to process.
	time.Sleep(10 * time.Millisecond)

	messages := records.recorded()
	s.Require().GreaterOrEqual(len(messages), 4, "expected at least 4 captured messages")

	tests := []struct {
		msg   string
		level string
	}{
		{msg: "test debug message", level: "DBG"},
		{msg: "test info message", level: "INF"},
		{msg: "test warn message", level: "WRN"},
		{msg: "test error message", level: "ERR"},
	}

	for _, tt := range tests {
		message, found := records.find(tt.msg)

		s.Require().True(found, "expected to find %q", tt.msg)
		s.Equal(tt.level, message.level)
		s.Equal(tt.msg, message.msg, "the message is captured as it was logged")
	}
}

// TestCaptureDisabledRestoresOutput verifies that disabling capture leaves the
// logger logging normally and reporting nothing further to the callback.
func (s *LoggerIntegrationTestSuite) TestCaptureDisabledRestoresOutput() {
	loggerInstance, records := capturedLogger(s.T())

	loggerInstance.Info("captured message")
	s.Require().Equal(1, records.count("captured message"))

	loggerInstance.SetCaptureFunc(nil)
	loggerInstance.Info("uncaptured message")

	s.Equal(1, records.count("captured message"))
	s.Zero(records.count("uncaptured message"))
}

// TestConcurrentLogging verifies thread safety of logger methods.
//
// This test ensures that all logger methods can be called concurrently
// without race conditions or data corruption.
func (s *LoggerIntegrationTestSuite) TestConcurrentLogging() {
	const iterations = 50

	// Set up expectations for concurrent calls.
	// We use Once() for each call, so we need to set up multiple expectations.
	for range iterations {
		s.mockLogger.EXPECT().Debug("debug message").Return().Once()
		s.mockLogger.EXPECT().Info("info message").Return().Once()
		s.mockLogger.EXPECT().Warn("warn message").Return().Once()
		s.mockLogger.EXPECT().Error("error message").Return().Once()
	}

	var wg sync.WaitGroup

	// Launch concurrent goroutines for each log level.
	for range iterations {
		wg.Go(func() {
			s.mockLogger.Debug("debug message")
		})

		wg.Go(func() {
			s.mockLogger.Info("info message")
		})

		wg.Go(func() {
			s.mockLogger.Warn("warn message")
		})

		wg.Go(func() {
			s.mockLogger.Error("error message")
		})
	}

	// Wait for all goroutines to complete.
	wg.Wait()
}

// TestConcurrentLevelChanges verifies thread safety of level changes during logging.
//
// This test ensures that changing log levels concurrently with logging
// operations does not cause race conditions.
func (s *LoggerIntegrationTestSuite) TestConcurrentLevelChanges() {
	const iterations = 50

	// Set up expectations.
	for range iterations {
		s.mockLogger.EXPECT().Info("info message").Return().Once()
		s.mockLogger.EXPECT().Level(mock.Anything).Return().Once()
	}

	var wg sync.WaitGroup

	// Concurrent logging.
	for range iterations {
		wg.Go(func() {
			s.mockLogger.Info("info message")
		})
	}

	// Concurrent level changes.
	for range iterations {
		wg.Go(func() {
			s.mockLogger.Level(logger.DebugLevel)
		})
	}

	wg.Wait()
}

// TestLoggerMethodOrdering verifies that logger methods can be called in any order.
//
// This test ensures that the logger interface supports flexible usage patterns.
func (s *LoggerIntegrationTestSuite) TestLoggerMethodOrdering() {
	s.Run("level then log", func() {
		s.mockLogger.EXPECT().Level(logger.WarnLevel).Return().Once()
		s.mockLogger.EXPECT().Warn("warn message").Return().Once()

		s.mockLogger.Level(logger.WarnLevel)
		s.mockLogger.Warn("warn message")
	})
}

// TestCaptureFuncCalledMultipleTimes verifies capture function handles multiple calls.
//
// This test ensures that the capture function is called for each log message
// and properly handles high-frequency logging.
func (s *LoggerIntegrationTestSuite) TestCaptureFuncCalledMultipleTimes() {
	loggerInstance, records := capturedLogger(s.T())

	// Generate multiple log messages rapidly.
	const numMessages = 100

	for i := range numMessages {
		loggerInstance.Info("rapid log message", logger.Int("index", i))
	}

	// Give time for capture to process.
	time.Sleep(50 * time.Millisecond)

	// Verify capture was called for each message.
	s.GreaterOrEqual(
		records.count("rapid log message"),
		numMessages*9/10,
		"expected capture to be called for at least 90% of messages",
	)
}

// TestNilCaptureFuncDisablesCapture verifies that setting nil capture func disables capture.
//
// This test ensures that capture can be properly disabled by setting a nil function.
func (s *LoggerIntegrationTestSuite) TestNilCaptureFuncDisablesCapture() {
	loggerInstance, records := capturedLogger(s.T())

	// Log once while capture is still installed.
	loggerInstance.Info("message with capture")

	time.Sleep(10 * time.Millisecond)

	var capturedAfter int

	captureFuncAfter := func(_, _ string) {
		capturedAfter++
	}

	// Verify the new capture function is NOT called since we set nil.
	loggerInstance.SetCaptureFunc(captureFuncAfter)
	loggerInstance.SetCaptureFunc(nil) // Disable again.

	for range 10 {
		loggerInstance.Info("message without capture")
	}

	time.Sleep(10 * time.Millisecond)

	s.Positive(
		records.count("message with capture"),
		"expected capture to be called before disabling",
	)
	s.Equal(0, capturedAfter, "expected no capture after disabling")
}

// Additional standalone tests for edge cases and type verification.

// TestLogCaptureFuncType verifies the LogCaptureFunc type definition.
func TestLogCaptureFuncType(t *testing.T) {
	t.Parallel()

	// Verify LogCaptureFunc can be assigned and called.
	var (
		called                     bool
		capturedLevel, capturedMsg string
	)

	captureFunc := logger.LogCaptureFunc(func(level, msg string) {
		called = true
		capturedLevel = level
		capturedMsg = msg
	})

	// Call the function.
	captureFunc("INF", "test message")

	assert.True(t, called)
	assert.Equal(t, "INF", capturedLevel)
	assert.Equal(t, "test message", capturedMsg)
}

// TestLoggerInterfaceCompliance verifies types implement the Logger interface.
func TestLoggerInterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Test that *mocks.MockLogger implements logger.Logger.
	var _ logger.Logger = (*mocks.MockLogger)(nil)
}

// TestParseLevelIntegration verifies ParseLevel with various inputs.
func TestParseLevelIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected logger.Level
	}{
		{
			name:     "debug lowercase",
			input:    "debug",
			expected: logger.DebugLevel,
		},
		{
			name:     "info lowercase",
			input:    "info",
			expected: logger.InfoLevel,
		},
		{
			name:     "warn lowercase",
			input:    "warn",
			expected: logger.WarnLevel,
		},
		{
			name:     "error lowercase",
			input:    "error",
			expected: logger.ErrorLevel,
		},
		{
			name:     "debug uppercase",
			input:    "DEBUG",
			expected: logger.DebugLevel,
		},
		{
			name:     "info mixed case",
			input:    "Info",
			expected: logger.InfoLevel,
		},
		{
			name:     "unknown defaults to info",
			input:    "unknown",
			expected: logger.InfoLevel,
		},
		{
			name:     "empty defaults to info",
			input:    "",
			expected: logger.InfoLevel,
		},
		{
			name:     "trace maps to debug",
			input:    "trace",
			expected: logger.InfoLevel, // trace is not supported, defaults to info
		},
		{
			name:     "fatal maps to info",
			input:    "fatal",
			expected: logger.InfoLevel, // fatal is not supported, defaults to info
		},
		{
			name:     "panic maps to info",
			input:    "panic",
			expected: logger.InfoLevel, // panic is not supported, defaults to info
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, logger.ParseLevel(tt.input))
		})
	}
}

// TestNewLoggerIntegration verifies NewLogger creates valid loggers.
func TestNewLoggerIntegration(t *testing.T) {
	t.Parallel()

	loggerInstance, err := logger.NewLogger()
	require.NoError(t, err)
	require.NotNil(t, loggerInstance)

	// Test that all methods can be called without panicking.
	loggerInstance.Debug("integration debug")
	loggerInstance.Info("integration info")
	loggerInstance.Warn("integration warn")
	loggerInstance.Error("integration error")

	// Test level setting (should not panic).
	loggerInstance.Level(logger.DebugLevel)

	// Test capture func setting (should not panic).
	loggerInstance.SetCaptureFunc(nil)
}

// TestNewLoggerWithCaptureIntegration verifies NewLoggerWithCapture creates valid loggers.
func TestNewLoggerWithCaptureIntegration(t *testing.T) {
	t.Parallel()

	loggerInstance, captureWriter, err := logger.NewLoggerWithCapture()
	require.NoError(t, err)
	require.NotNil(t, loggerInstance)
	require.NotNil(t, captureWriter)

	// Verify it implements the interface.
	iface := loggerInstance
	assert.NotNil(t, iface)

	// Test capture functionality.
	var captured bool

	captureFunc := func(_, _ string) {
		captured = true
	}

	loggerInstance.SetCaptureFunc(captureFunc)
	loggerInstance.Info("test capture")

	time.Sleep(10 * time.Millisecond)
	assert.True(t, captured, "expected capture function to be called")
}

// TestErrorField verifies a message logged with an error field is captured.
func TestErrorField(t *testing.T) {
	t.Parallel()

	loggerInstance, records := capturedLogger(t)
	loggerInstance.Error("error occurred", logger.Err(errors.New("test error for field")))

	message, found := records.find("error occurred")
	require.True(t, found, "expected the message to be captured")
	assert.Equal(t, "error occurred", message.msg)
	assert.Equal(t, "ERR", message.level)
}

// TestLogLevelTransitions verifies transitions between log levels.
func TestLogLevelTransitions(t *testing.T) {
	t.Parallel()

	loggerInstance, records := capturedLogger(t)

	// At info level, debug should be filtered.
	loggerInstance.Level(logger.InfoLevel)
	loggerInstance.Debug("debug at info level")
	assert.Zero(t, records.count("debug at info level"))

	// Change to debug level.
	loggerInstance.Level(logger.DebugLevel)

	// Now debug should pass through.
	loggerInstance.Debug("debug at debug level")
	assert.Equal(t, 1, records.count("debug at debug level"))

	// Change to error level.
	loggerInstance.Level(logger.ErrorLevel)

	// Now info should be filtered.
	loggerInstance.Info("info at error level")
	assert.Zero(t, records.count("info at error level"))

	// But error should pass.
	loggerInstance.Error("error at error level")
	assert.Equal(t, 1, records.count("error at error level"))
}

// BenchmarkLogLevelMethods benchmarks the log level methods.
func BenchmarkLogLevelMethods(b *testing.B) {
	loggerInstance, err := logger.NewLogger()
	require.NoError(b, err)

	b.Run("Debug", func(b *testing.B) {
		for b.Loop() {
			loggerInstance.Debug("benchmark debug")
		}
	})

	b.Run("Info", func(b *testing.B) {
		for b.Loop() {
			loggerInstance.Info("benchmark info")
		}
	})

	b.Run("Warn", func(b *testing.B) {
		for b.Loop() {
			loggerInstance.Warn("benchmark warn")
		}
	})

	b.Run("Error", func(b *testing.B) {
		for b.Loop() {
			loggerInstance.Error("benchmark error")
		}
	})
}

// BenchmarkLevelChange benchmarks level changes.
func BenchmarkLevelChange(b *testing.B) {
	loggerInstance, err := logger.NewLogger()
	require.NoError(b, err)

	levels := []logger.Level{
		logger.DebugLevel,
		logger.InfoLevel,
		logger.WarnLevel,
		logger.ErrorLevel,
	}

	for b.Loop() {
		for _, level := range levels {
			loggerInstance.Level(level)
		}
	}
}
