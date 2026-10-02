/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"

	tea "charm.land/bubbletea/v2"

	"github.com/nicholas-fedor/go-remove/internal/logger"
)

// LogMsg is a Bubble Tea message that carries a log entry to be displayed in the TUI.
type LogMsg struct {
	Level   string // Log level (e.g., "DBG", "INF", "WRN", "ERR")
	Message string // Log message content
}

// pollInterval is the duration between log channel polls.
const pollInterval = 50 * time.Millisecond

// pollLogTickMsg is a message sent when it's time to poll for logs again.
type pollLogTickMsg struct{}

// setupLogCapture configures the logger's capture callback to send messages to the TUI.
//
// This method is idempotent and can be called multiple times safely.
//
// Parameters:
//   - log: Logger that will receive the capture callback.
func (m *Model) setupLogCapture(log logger.Logger) {
	log.SetCaptureFunc(func(level, msg string) {
		// Send to channel without blocking.
		// If channel is full, the message is dropped to prevent blocking.
		select {
		case m.logChan <- LogMsg{Level: level, Message: msg}:
		default:
			// Channel is full, message is dropped to prevent blocking.
		}
	})
}

// toggleVerboseLogging toggles log panel visibility and logging level.
//
// Polling already runs for the whole session, so this only changes what is
// shown and how much detail is emitted.
func (m *Model) toggleVerboseLogging() {
	m.showLogs = !m.showLogs

	// Opening the panel raises the level so the extra detail is worth having.
	// Closing it returns to the configured level, except when the user asked
	// for verbose at startup: that request is independent of the panel, so
	// hiding the panel must not quietly reduce verbosity below what was asked.
	if m.showLogs || m.config.Verbose {
		m.logger.Level(zerolog.DebugLevel)
	} else {
		m.logger.Level(logger.ParseLevel(m.config.LogLevel))
	}

	// The panel changes the space available to the grid, so the reserved height
	// has to be recalculated.
	m.updateGrid()
}

// pollLogChannel returns a command that polls for log messages.
//
// The TUI polls the channel instead of sending from the capture callback, which avoids deadlocks.
//
// Returns:
//   - Command that produces a LogMsg or poll tick.
func (m *Model) pollLogChannel() tea.Cmd {
	return func() tea.Msg {
		// Wait for either a log message or a timeout.
		// Using time.After ensures we return a proper tea.Msg (not a tea.Cmd),
		// which maintains the continuous polling loop.
		select {
		case msg := <-m.logChan:
			return msg
		case <-time.After(pollInterval):
			// No message available after interval, return tick to schedule next poll.
			return pollLogTickMsg{}
		}
	}
}

// addLogEntry adds a log message to the circular buffer.
//
// Parameters:
//   - msg: Log message to display.
func (m *Model) addLogEntry(msg LogMsg) {
	// Format the log entry: "[LEVEL] message"
	entry := fmt.Sprintf("[%s] %s", msg.Level, msg.Message)

	// Add to logs slice
	m.logs = append(m.logs, entry)

	// Maintain circular buffer size
	if len(m.logs) > maxLogLines {
		m.logs = m.logs[len(m.logs)-maxLogLines:]
	}
}
