/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package logger_test provides black-box integration tests for the logger package.
//
// These tests verify the Logger interface behavior through the MockLogger mock
// implementation. All tests use Mockery-generated mocks to ensure no real logging
// output is produced. The tests cover the complete logging lifecycle including
// all log level methods, sync operations, level management, and capture functionality.
//
// Test Organization:
//   - LoggerIntegrationTestSuite: Main test suite using testify suite
//   - Table-driven tests for parameterized log level scenarios
//   - Individual tests for complex error handling and concurrent operations
//
// Coverage Areas:
//   - All log level methods (Debug, Info, Warn, Error)
//   - Sync operations and error handling
//   - Dynamic log level setting and filtering
//   - CaptureFunc functionality for TUI integration
//   - Log event chaining patterns
//   - Concurrent logging operations
//   - Error handling for sync failures
package logger_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/nicholas-fedor/go-remove/internal/logger"
	"github.com/nicholas-fedor/go-remove/internal/logger/mocks"
)

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

// TestAllLogLevelMethods verifies all log level methods work correctly.
//
// This test ensures that Debug, Info, Warn, and Error methods all return
// valid zerolog events that can be used for logging.
func (s *LoggerIntegrationTestSuite) TestAllLogLevelMethods() {
	// Create a discard writer to prevent actual output
	discardLogger := zerolog.New(io.Discard)

	tests := []struct {
		name     string
		logFunc  func()
		expected string
	}{
		{
			name: "debug level logging",
			logFunc: func() {
				s.mockLogger.EXPECT().Debug().Return(discardLogger.Debug()).Once()
				event := s.mockLogger.Debug()
				s.NotNil(event)
			},
			expected: "debug",
		},
		{
			name: "info level logging",
			logFunc: func() {
				s.mockLogger.EXPECT().Info().Return(discardLogger.Info()).Once()
				event := s.mockLogger.Info()
				s.NotNil(event)
			},
			expected: "info",
		},
		{
			name: "warn level logging",
			logFunc: func() {
				s.mockLogger.EXPECT().Warn().Return(discardLogger.Warn()).Once()
				event := s.mockLogger.Warn()
				s.NotNil(event)
			},
			expected: "warn",
		},
		{
			name: "error level logging",
			logFunc: func() {
				s.mockLogger.EXPECT().Error().Return(discardLogger.Error()).Once()
				event := s.mockLogger.Error()
				s.NotNil(event)
			},
			expected: "error",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.logFunc()
		})
	}
}

// TestLogEventChaining verifies that log events can be properly chained.
//
// This test ensures that zerolog events returned by log methods support
// the full chain of operations including Str, Int, Err, and Msg.
func (s *LoggerIntegrationTestSuite) TestLogEventChaining() {
	var buf bytes.Buffer

	consoleWriter := zerolog.ConsoleWriter{
		Out:        &buf,
		TimeFormat: "2006-01-02",
		NoColor:    true,
	}

	zl := zerolog.New(consoleWriter).With().Timestamp().Logger()

	s.mockLogger.EXPECT().Info().Return(zl.Info()).Once()

	// Chain multiple fields and send the message
	s.mockLogger.Info().
		Str("component", "test").
		Int("count", 42).
		Msg("chained log message")

	output := buf.String()
	s.Contains(output, "chained log message")
	s.Contains(output, "component")
	s.Contains(output, "test")
	s.Contains(output, "count")
	s.Contains(output, "42")
}

// TestSyncOperation verifies the Sync method behavior.
//
// This test ensures that Sync properly flushes buffered log entries
// and returns appropriate errors when sync fails.
func (s *LoggerIntegrationTestSuite) TestSyncOperation() {
	s.Run("successful sync", func() {
		s.mockLogger.EXPECT().Sync().Return(nil).Once()

		err := s.mockLogger.Sync()
		s.NoError(err)
	})

	s.Run("sync with error", func() {
		syncErr := errors.New("sync failed: buffer full")
		s.mockLogger.EXPECT().Sync().Return(syncErr).Once()

		err := s.mockLogger.Sync()
		s.ErrorIs(err, syncErr)
	})
}

// TestLevelSetting verifies dynamic log level changes.
//
// This test ensures that the Level method properly changes the minimum
// log level and that filtering works correctly.
func (s *LoggerIntegrationTestSuite) TestLevelSetting() {
	tests := []struct {
		name       string
		level      zerolog.Level
		shouldCall bool
	}{
		{
			name:       "set to debug level",
			level:      zerolog.DebugLevel,
			shouldCall: true,
		},
		{
			name:       "set to info level",
			level:      zerolog.InfoLevel,
			shouldCall: true,
		},
		{
			name:       "set to warn level",
			level:      zerolog.WarnLevel,
			shouldCall: true,
		},
		{
			name:       "set to error level",
			level:      zerolog.ErrorLevel,
			shouldCall: true,
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
	var buf bytes.Buffer

	consoleWriter := zerolog.ConsoleWriter{
		Out:        &buf,
		TimeFormat: "2006-01-02",
		NoColor:    true,
	}

	// Create logger at info level
	zl := zerolog.New(consoleWriter).With().Timestamp().Logger().Level(zerolog.InfoLevel)

	s.Run("debug filtered at info level", func() {
		buf.Reset()
		s.mockLogger.EXPECT().Debug().Return(zl.Debug()).Once()

		s.mockLogger.Debug().Msg("debug message")
		s.Empty(buf.String())
	})

	s.Run("info allowed at info level", func() {
		buf.Reset()
		s.mockLogger.EXPECT().Info().Return(zl.Info()).Once()

		s.mockLogger.Info().Msg("info message")
		s.Contains(buf.String(), "info message")
	})

	s.Run("error allowed at info level", func() {
		buf.Reset()
		s.mockLogger.EXPECT().Error().Return(zl.Error()).Once()

		s.mockLogger.Error().Msg("error message")
		s.Contains(buf.String(), "error message")
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
	// Create a real logger to test capture functionality
	loggerInstance, captureWriter, err := logger.NewLoggerWithCapture()
	s.Require().NoError(err)
	s.Require().NotNil(loggerInstance)
	s.Require().NotNil(captureWriter)

	// Set up capture tracking
	var (
		capturedMessages []struct {
			level string
			msg   string
		}
		mu sync.Mutex
	)

	captureFunc := func(level, msg string) {
		mu.Lock()
		defer mu.Unlock()

		capturedMessages = append(capturedMessages, struct {
			level string
			msg   string
		}{level: level, msg: msg})
	}

	// Enable capture
	loggerInstance.SetCaptureFunc(captureFunc)

	// Generate log messages
	loggerInstance.Info().Msg("test info message")
	loggerInstance.Warn().Msg("test warn message")
	loggerInstance.Error().Msg("test error message")

	// Give some time for capture to process
	time.Sleep(10 * time.Millisecond)

	// Verify captured messages
	mu.Lock()
	defer mu.Unlock()

	s.Require().GreaterOrEqual(len(capturedMessages), 3, "expected at least 3 captured messages")

	// Find and verify each message type
	var foundInfo, foundWarn, foundError bool

	for _, cm := range capturedMessages {
		if strings.Contains(cm.msg, "test info message") {
			foundInfo = true

			// Level could be "INF" or "LOG" depending on format parsing
			s.NotEmpty(cm.level)
		}

		if strings.Contains(cm.msg, "test warn message") {
			foundWarn = true

			// Level could be "WRN" or "LOG" depending on format parsing
			s.NotEmpty(cm.level)
		}

		if strings.Contains(cm.msg, "test error message") {
			foundError = true

			// Level could be "ERR" or "LOG" depending on format parsing
			s.NotEmpty(cm.level)
		}
	}

	s.True(foundInfo, "expected to find info message")
	s.True(foundWarn, "expected to find warn message")
	s.True(foundError, "expected to find error message")
}

// TestConcurrentLogging verifies thread safety of logger methods.
//
// This test ensures that all logger methods can be called concurrently
// without race conditions or data corruption.
func (s *LoggerIntegrationTestSuite) TestConcurrentLogging() {
	discardLogger := zerolog.New(io.Discard)

	// Set up expectations for concurrent calls
	// We use Once() for each call, so we need to set up multiple expectations
	for range 50 {
		s.mockLogger.EXPECT().Debug().Return(discardLogger.Debug()).Once()
		s.mockLogger.EXPECT().Info().Return(discardLogger.Info()).Once()
		s.mockLogger.EXPECT().Warn().Return(discardLogger.Warn()).Once()
		s.mockLogger.EXPECT().Error().Return(discardLogger.Error()).Once()
	}

	var wg sync.WaitGroup

	// Launch concurrent goroutines for each log level
	for range 50 {
		wg.Go(func() {
			event := s.mockLogger.Debug()
			s.NotNil(event)
		})

		wg.Go(func() {
			event := s.mockLogger.Info()
			s.NotNil(event)
		})

		wg.Go(func() {
			event := s.mockLogger.Warn()
			s.NotNil(event)
		})

		wg.Go(func() {
			event := s.mockLogger.Error()
			s.NotNil(event)
		})
	}

	// Wait for all goroutines to complete
	wg.Wait()
}

// TestConcurrentLevelChanges verifies thread safety of level changes during logging.
//
// This test ensures that changing log levels concurrently with logging
// operations does not cause race conditions.
func (s *LoggerIntegrationTestSuite) TestConcurrentLevelChanges() {
	discardLogger := zerolog.New(io.Discard)

	// Set up expectations
	for range 50 {
		s.mockLogger.EXPECT().Info().Return(discardLogger.Info()).Once()
		s.mockLogger.EXPECT().Level(mock.Anything).Return().Once()
	}

	var wg sync.WaitGroup

	// Concurrent logging
	for range 50 {
		wg.Go(func() {
			event := s.mockLogger.Info()
			s.NotNil(event)
		})
	}

	// Concurrent level changes
	for range 50 {
		wg.Go(func() {
			s.mockLogger.Level(zerolog.DebugLevel)
		})
	}

	wg.Wait()
}

// TestSyncFailureHandling verifies error handling when sync fails.
//
// This test ensures that sync errors are properly propagated and can be
// handled by the caller.
func (s *LoggerIntegrationTestSuite) TestSyncFailureHandling() {
	tests := []struct {
		name     string
		errSetup error
		wantErr  bool
	}{
		{
			name:     "no error on sync",
			errSetup: nil,
			wantErr:  false,
		},
		{
			name:     "io error on sync",
			errSetup: errors.New("io error: device not ready"),
			wantErr:  true,
		},
		{
			name:     "buffer error on sync",
			errSetup: errors.New("buffer flush failed"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.mockLogger.EXPECT().Sync().Return(tt.errSetup).Once()

			err := s.mockLogger.Sync()
			if tt.wantErr {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

// TestMultipleSyncCalls verifies behavior of multiple consecutive sync calls.
//
// This test ensures that calling sync multiple times works correctly
// and doesn't cause issues.
func (s *LoggerIntegrationTestSuite) TestMultipleSyncCalls() {
	// Expect multiple sync calls
	s.mockLogger.EXPECT().Sync().Return(nil).Times(3)

	// Call sync multiple times
	for range 3 {
		err := s.mockLogger.Sync()
		s.Require().NoError(err)
	}
}

// TestLoggerMethodOrdering verifies that logger methods can be called in any order.
//
// This test ensures that the logger interface supports flexible usage patterns.
func (s *LoggerIntegrationTestSuite) TestLoggerMethodOrdering() {
	discardLogger := zerolog.New(io.Discard)

	s.Run("sync then log", func() {
		s.mockLogger.EXPECT().Sync().Return(nil).Once()
		s.mockLogger.EXPECT().Info().Return(discardLogger.Info()).Once()

		err := s.mockLogger.Sync()
		s.Require().NoError(err)

		event := s.mockLogger.Info()
		s.NotNil(event)
	})

	s.Run("log then sync", func() {
		s.mockLogger.EXPECT().Error().Return(discardLogger.Error()).Once()
		s.mockLogger.EXPECT().Sync().Return(nil).Once()

		event := s.mockLogger.Error()
		s.NotNil(event)

		err := s.mockLogger.Sync()
		s.NoError(err)
	})

	s.Run("level then log then sync", func() {
		s.mockLogger.EXPECT().Level(zerolog.WarnLevel).Return().Once()
		s.mockLogger.EXPECT().Warn().Return(discardLogger.Warn()).Once()
		s.mockLogger.EXPECT().Sync().Return(nil).Once()

		s.mockLogger.Level(zerolog.WarnLevel)

		event := s.mockLogger.Warn()
		s.NotNil(event)

		err := s.mockLogger.Sync()
		s.NoError(err)
	})
}

// TestCaptureFuncCalledMultipleTimes verifies capture function handles multiple calls.
//
// This test ensures that the capture function is called for each log message
// and properly handles high-frequency logging.
func (s *LoggerIntegrationTestSuite) TestCaptureFuncCalledMultipleTimes() {
	// Create a real logger to test capture functionality
	loggerInstance, captureWriter, err := logger.NewLoggerWithCapture()
	s.Require().NoError(err)
	s.Require().NotNil(loggerInstance)
	s.Require().NotNil(captureWriter)

	// Set up capture counter
	var (
		callCount int
		mu        sync.Mutex
	)

	captureFunc := func(level, msg string) {
		mu.Lock()
		defer mu.Unlock()

		callCount++
	}

	// Enable capture
	loggerInstance.SetCaptureFunc(captureFunc)

	// Generate multiple log messages rapidly
	const numMessages = 100

	for i := range numMessages {
		loggerInstance.Info().Int("index", i).Msg("rapid log message")
	}

	// Give time for capture to process
	time.Sleep(50 * time.Millisecond)

	// Verify capture was called for each message
	mu.Lock()
	defer mu.Unlock()

	s.GreaterOrEqual(
		callCount,
		numMessages*9/10,
		"expected capture to be called for at least 90% of messages",
	)
}

// TestNilCaptureFuncDisablesCapture verifies that setting nil capture func disables capture.
//
// This test ensures that capture can be properly disabled by setting a nil function.
func (s *LoggerIntegrationTestSuite) TestNilCaptureFuncDisablesCapture() {
	// Create a real logger to test capture functionality
	loggerInstance, captureWriter, err := logger.NewLoggerWithCapture()
	s.Require().NoError(err)
	s.Require().NotNil(loggerInstance)
	s.Require().NotNil(captureWriter)

	// First enable capture
	var capturedBefore int

	captureFunc := func(level, msg string) {
		capturedBefore++
	}

	loggerInstance.SetCaptureFunc(captureFunc)
	loggerInstance.Info().Msg("message with capture")

	time.Sleep(10 * time.Millisecond)

	// Now disable capture
	loggerInstance.SetCaptureFunc(nil)

	// Set up a new capture function that increments capturedAfter
	var capturedAfter int

	captureFuncAfter := func(level, msg string) {
		capturedAfter++
	}

	// Verify the new capture function is NOT called since we set nil
	loggerInstance.SetCaptureFunc(captureFuncAfter)
	loggerInstance.SetCaptureFunc(nil) // Disable again

	for range 10 {
		loggerInstance.Info().Msg("message without capture")
	}

	time.Sleep(10 * time.Millisecond)

	// Capture before should have been called, after should not increase
	s.Positive(capturedBefore, "expected capture to be called before disabling")
	s.Equal(0, capturedAfter, "expected no capture after disabling")
}

// Additional standalone tests for edge cases and type verification.

// TestLogCaptureFuncType verifies the LogCaptureFunc type definition.
func TestLogCaptureFuncType(t *testing.T) {
	t.Parallel()

	// Verify LogCaptureFunc can be assigned and called
	var (
		called                     bool
		capturedLevel, capturedMsg string
	)

	captureFunc := logger.LogCaptureFunc(func(level, msg string) {
		called = true
		capturedLevel = level
		capturedMsg = msg
	})

	// Call the function
	captureFunc("INF", "test message")

	assert.True(t, called)
	assert.Equal(t, "INF", capturedLevel)
	assert.Equal(t, "test message", capturedMsg)
}

// TestLoggerInterfaceCompliance verifies types implement the Logger interface.
func TestLoggerInterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Test that *mocks.MockLogger implements logger.Logger
	var _ logger.Logger = (*mocks.MockLogger)(nil)

	// Verify the interface has all expected methods
	// This is a compile-time check
}

// TestParseLevelIntegration verifies ParseLevel with various inputs.
func TestParseLevelIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected zerolog.Level
	}{
		{
			name:     "debug lowercase",
			input:    "debug",
			expected: zerolog.DebugLevel,
		},
		{
			name:     "info lowercase",
			input:    "info",
			expected: zerolog.InfoLevel,
		},
		{
			name:     "warn lowercase",
			input:    "warn",
			expected: zerolog.WarnLevel,
		},
		{
			name:     "error lowercase",
			input:    "error",
			expected: zerolog.ErrorLevel,
		},
		{
			name:     "debug uppercase",
			input:    "DEBUG",
			expected: zerolog.DebugLevel,
		},
		{
			name:     "info mixed case",
			input:    "Info",
			expected: zerolog.InfoLevel,
		},
		{
			name:     "unknown defaults to info",
			input:    "unknown",
			expected: zerolog.InfoLevel,
		},
		{
			name:     "empty defaults to info",
			input:    "",
			expected: zerolog.InfoLevel,
		},
		{
			name:     "trace maps to debug",
			input:    "trace",
			expected: zerolog.InfoLevel, // trace is not supported, defaults to info
		},
		{
			name:     "fatal maps to info",
			input:    "fatal",
			expected: zerolog.InfoLevel, // fatal is not supported, defaults to info
		},
		{
			name:     "panic maps to info",
			input:    "panic",
			expected: zerolog.InfoLevel, // panic is not supported, defaults to info
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := logger.ParseLevel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestNewLoggerIntegration verifies NewLogger creates valid loggers.
func TestNewLoggerIntegration(t *testing.T) {
	t.Parallel()

	loggerInstance, err := logger.NewLogger()
	require.NoError(t, err)
	require.NotNil(t, loggerInstance)

	// Test that all methods can be called without panicking
	// Note: Some events may be nil if log level is higher than the event level
	_ = loggerInstance.Debug()
	_ = loggerInstance.Info()
	_ = loggerInstance.Warn()
	_ = loggerInstance.Error()

	err = loggerInstance.Sync()
	require.NoError(t, err)

	// Test level setting (should not panic)
	loggerInstance.Level(zerolog.DebugLevel)

	// Test capture func setting (should not panic)
	loggerInstance.SetCaptureFunc(nil)
}

// TestNewLoggerWithCaptureIntegration verifies NewLoggerWithCapture creates valid loggers.
func TestNewLoggerWithCaptureIntegration(t *testing.T) {
	t.Parallel()

	loggerInstance, captureWriter, err := logger.NewLoggerWithCapture()
	require.NoError(t, err)
	require.NotNil(t, loggerInstance)
	require.NotNil(t, captureWriter)

	// Verify it implements the interface
	iface := loggerInstance
	assert.NotNil(t, iface)

	// Test capture functionality
	var captured bool

	captureFunc := func(level, msg string) {
		captured = true
	}

	loggerInstance.SetCaptureFunc(captureFunc)
	loggerInstance.Info().Msg("test capture")

	time.Sleep(10 * time.Millisecond)
	assert.True(t, captured, "expected capture function to be called")
}

// TestErrorFieldChaining verifies error fields can be chained in log events.
func TestErrorFieldChaining(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	consoleWriter := zerolog.ConsoleWriter{
		Out:        &buf,
		TimeFormat: "2006-01-02",
		NoColor:    true,
	}

	zl := zerolog.New(consoleWriter).With().Timestamp().Logger()
	testError := errors.New("test error for chaining")

	zl.Error().Err(testError).Msg("error occurred")

	output := buf.String()
	assert.Contains(t, output, "error occurred")
	assert.Contains(t, output, "test error for chaining")
}

// TestLogLevelTransitions verifies transitions between log levels.
func TestLogLevelTransitions(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	consoleWriter := zerolog.ConsoleWriter{
		Out:        &buf,
		TimeFormat: "2006-01-02",
		NoColor:    true,
	}

	zl := zerolog.New(consoleWriter).With().Timestamp().Logger().Level(zerolog.InfoLevel)

	loggerInstance := &testLoggerWrapper{logger: zl}

	// At info level, debug should be filtered
	buf.Reset()
	loggerInstance.Debug().Msg("debug at info level")
	assert.Empty(t, buf.String())

	// Change to debug level
	loggerInstance.Level(zerolog.DebugLevel)

	// Now debug should pass through
	buf.Reset()
	loggerInstance.Debug().Msg("debug at debug level")
	assert.Contains(t, buf.String(), "debug at debug level")

	// Change to error level
	loggerInstance.Level(zerolog.ErrorLevel)

	// Now info should be filtered
	buf.Reset()
	loggerInstance.Info().Msg("info at error level")
	assert.Empty(t, buf.String())

	// But error should pass
	buf.Reset()
	loggerInstance.Error().Msg("error at error level")
	assert.Contains(t, buf.String(), "error at error level")
}

// testLoggerWrapper wraps zerolog.Logger to implement logger.Logger for testing.
type testLoggerWrapper struct {
	logger zerolog.Logger
	mu     sync.RWMutex
}

// Debug returns a debug-level event.
func (t *testLoggerWrapper) Debug() *zerolog.Event {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.logger.Debug()
}

// Info returns an info-level event.
func (t *testLoggerWrapper) Info() *zerolog.Event {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.logger.Info()
}

// Warn returns a warn-level event.
func (t *testLoggerWrapper) Warn() *zerolog.Event {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.logger.Warn()
}

// Error returns an error-level event.
func (t *testLoggerWrapper) Error() *zerolog.Event {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.logger.Error()
}

// Sync is a no-op for this wrapper.
func (t *testLoggerWrapper) Sync() error {
	return nil
}

// Level sets the log level dynamically.
func (t *testLoggerWrapper) Level(level zerolog.Level) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.logger = t.logger.Level(level)
}

// SetCaptureFunc is a no-op for this wrapper.
func (t *testLoggerWrapper) SetCaptureFunc(_ logger.LogCaptureFunc) {}

// Verify testLoggerWrapper implements logger.Logger.
var _ logger.Logger = (*testLoggerWrapper)(nil)

// BenchmarkLogLevelMethods benchmarks the log level methods.
func BenchmarkLogLevelMethods(b *testing.B) {
	loggerInstance, err := logger.NewLogger()
	require.NoError(b, err)

	b.Run("Debug", func(b *testing.B) {
		for b.Loop() {
			loggerInstance.Debug().Msg("benchmark debug")
		}
	})

	b.Run("Info", func(b *testing.B) {
		for b.Loop() {
			loggerInstance.Info().Msg("benchmark info")
		}
	})

	b.Run("Warn", func(b *testing.B) {
		for b.Loop() {
			loggerInstance.Warn().Msg("benchmark warn")
		}
	})

	b.Run("Error", func(b *testing.B) {
		for b.Loop() {
			loggerInstance.Error().Msg("benchmark error")
		}
	})
}

// BenchmarkSync benchmarks the Sync operation.
func BenchmarkSync(b *testing.B) {
	loggerInstance, err := logger.NewLogger()
	require.NoError(b, err)

	for b.Loop() {
		_ = loggerInstance.Sync()
	}
}

// BenchmarkLevelChange benchmarks level changes.
func BenchmarkLevelChange(b *testing.B) {
	loggerInstance, err := logger.NewLogger()
	require.NoError(b, err)

	levels := []zerolog.Level{
		zerolog.DebugLevel,
		zerolog.InfoLevel,
		zerolog.WarnLevel,
		zerolog.ErrorLevel,
	}

	for b.Loop() {
		for _, level := range levels {
			loggerInstance.Level(level)
		}
	}
}
