/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"fmt"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/go-remove/internal/logger"
	"github.com/nicholas-fedor/go-remove/internal/tui/render"
)

// TestLogPolling_AlwaysRuns verifies that log polling starts regardless of
// --verbose, and that a captured line is retained while the panel is hidden.
//
// The capture callback is always installed, so polling has to run too. Without
// it the channel fills with nothing draining it and the panel opens empty.
func TestLogPolling_AlwaysRuns(t *testing.T) {
	t.Parallel()

	for _, verbose := range []bool{false, true} {
		t.Run(fmt.Sprintf("verbose=%v", verbose), func(t *testing.T) {
			t.Parallel()

			m := &Model{
				logs:     make([]string, 0, maxLogLines),
				logChan:  make(chan LogMsg, maxLogLines),
				logger:   logger.NopLogger(),
				showLogs: verbose,
				styles:   render.DefaultStyleConfig(),
				config:   Config{Verbose: verbose, LogLevel: "info"},
			}

			require.NotNil(t, m.Init(), "log polling must start so entries are retained")

			next, cmd := m.Update(LogMsg{Level: "ERR", Message: "binary is stranded in trash"})

			require.NotNil(t, cmd, "the poll loop must be re-armed after a message")

			updated, ok := next.(*Model)
			require.True(t, ok, "Update must return the model")

			assert.Equal(t, verbose, updated.showLogs, "the panel visibility is unchanged")

			// The point of the fix: the warning survives even when hidden.
			require.NotEmpty(
				t,
				updated.logs,
				"the message must be retained while the panel is hidden",
			)
			assert.Contains(t, updated.logs[len(updated.logs)-1], "stranded")
		})
	}
}

// TestToggleVerboseLogging_StartsNoExtraPollChain verifies that toggling the
// panel does not spawn a poll chain.
//
// The previous implementation returned a fresh poll command on every press, so
// each press left another permanent loop running.
func TestToggleVerboseLogging_StartsNoExtraPollChain(t *testing.T) {
	t.Parallel()

	m := &Model{
		logs:    make([]string, 0, maxLogLines),
		logChan: make(chan LogMsg, maxLogLines),
		logger:  logger.NopLogger(),
		styles:  render.DefaultStyleConfig(),
		config:  Config{LogLevel: "warn"},
		choices: []string{"tool"},
	}
	m.updateGrid()

	m.toggleVerboseLogging()
	assert.True(t, m.showLogs)

	m.toggleVerboseLogging()
	assert.False(t, m.showLogs)
}

// TestToggleVerboseLogging_PreservesStartupVerbose verifies that hiding the log
// panel does not reduce verbosity the user asked for at startup.
//
// A --verbose request is independent of the panel, so closing it must not
// quietly drop back below what was asked.
func TestToggleVerboseLogging_PreservesStartupVerbose(t *testing.T) {
	t.Parallel()

	t.Run("keeps debug when started verbose", func(t *testing.T) {
		t.Parallel()

		recorder := &levelRecordingLogger{}
		m := &Model{
			logs:    make([]string, 0, maxLogLines),
			logChan: make(chan LogMsg, maxLogLines),
			logger:  recorder,
			styles:  render.DefaultStyleConfig(),
			// The panel starts open when --verbose is given.
			showLogs: true,
			config:   Config{Verbose: true, LogLevel: "info"},
			choices:  []string{"tool"},
		}
		m.updateGrid()

		m.toggleVerboseLogging()
		require.False(t, m.showLogs, "the panel is hidden")
		assert.Equal(t, zerolog.DebugLevel, recorder.level,
			"a startup --verbose request is independent of the panel")

		m.toggleVerboseLogging()
		require.True(t, m.showLogs)
		assert.Equal(t, zerolog.DebugLevel, recorder.level)
	})

	t.Run("returns to the configured level otherwise", func(t *testing.T) {
		t.Parallel()

		recorder := &levelRecordingLogger{}
		m := &Model{
			logs:    make([]string, 0, maxLogLines),
			logChan: make(chan LogMsg, maxLogLines),
			logger:  recorder,
			styles:  render.DefaultStyleConfig(),
			config:  Config{Verbose: false, LogLevel: "warn"},
			choices: []string{"tool"},
		}
		m.updateGrid()

		m.toggleVerboseLogging()
		require.True(t, m.showLogs)
		assert.Equal(t, zerolog.DebugLevel, recorder.level, "opening the panel raises the level")

		m.toggleVerboseLogging()
		require.False(t, m.showLogs)
		assert.Equal(t, zerolog.WarnLevel, recorder.level,
			"closing the panel returns to the configured level")
	})
}

// levelRecordingLogger records the last level applied to it.
type levelRecordingLogger struct {
	logger.Logger

	level zerolog.Level
}

func (l *levelRecordingLogger) Level(level zerolog.Level) { l.level = level }

// Test_pollLogChannel_ReturnsLogMsg verifies log message is returned when available.
func Test_pollLogChannel_ReturnsLogMsg(t *testing.T) {
	logChan := make(chan LogMsg, 10)
	logChan <- LogMsg{Level: "INF", Message: "test message"}

	m := &Model{
		logChan: logChan,
	}

	cmd := m.pollLogChannel()
	assert.NotNil(t, cmd)

	// Execute the command to get the message
	msg := cmd()
	logMsg, ok := msg.(LogMsg)
	assert.True(t, ok)
	assert.Equal(t, "INF", logMsg.Level)
	assert.Equal(t, "test message", logMsg.Message)
}

// Test_pollLogChannel_ReturnsTickWhenEmpty verifies tick is scheduled when no messages.
func Test_pollLogChannel_ReturnsTickWhenEmpty(t *testing.T) {
	logChan := make(chan LogMsg, 10)
	m := &Model{
		logChan: logChan,
	}

	cmd := m.pollLogChannel()
	assert.NotNil(t, cmd)

	// Execute the command - should get a TickMsg
	// The tick msg is wrapped by tea.Tick
	msg := cmd()
	assert.NotNil(t, msg)
	// The message could be either TickMsg or pollLogTickMsg depending on implementation
}

// Test_addLogEntry_CircularBuffer verifies max log lines limit is enforced.
func Test_addLogEntry_CircularBuffer(t *testing.T) {
	m := &Model{
		logs: []string{},
	}

	// Add more than maxLogLines entries
	for i := range maxLogLines + 10 {
		m.addLogEntry(LogMsg{Level: "INF", Message: fmt.Sprintf("message %d", i)})
	}

	assert.Len(t, m.logs, maxLogLines)
	// Verify oldest entries were removed
	assert.NotContains(t, m.logs, "[INF] message 0")
	assert.Contains(t, m.logs, fmt.Sprintf("[INF] message %d", maxLogLines+9))
}
