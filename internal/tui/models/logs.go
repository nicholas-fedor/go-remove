/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/nicholas-fedor/go-remove/internal/logger"
)

// pollInterval is the duration between log channel polls.
const pollInterval = 50 * time.Millisecond

// LogMsg is a Bubble Tea message that carries a log entry to be displayed in the TUI.
type LogMsg struct {
	Level   string // Log level (e.g., "DBG", "INF", "WRN", "ERR")
	Message string // Log message content
}

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
		// The channel is bounded, so a full one drops the message rather than
		// blocking the logger that is trying to write it.
		select {
		case m.logChan <- LogMsg{Level: level, Message: msg}:
		default:
		}
	})
}

// toggleVerboseLogging toggles log panel visibility and logging level.
//
// Polling already runs for the whole session, so this only changes what is
// shown and how much detail is emitted.
func (m *Model) toggleVerboseLogging() {
	m.showLogs = !m.showLogs

	// Closing the panel returns to the configured level, except when the user
	// asked for verbose at startup, which is independent of the panel.
	if m.showLogs || m.config.Verbose {
		m.logger.Level(logger.DebugLevel)
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
		select {
		case msg := <-m.logChan:
			return msg
		case <-time.After(pollInterval):
			return pollLogTickMsg{}
		}
	}
}

// addLogEntry records a log message, dropping the oldest entries past the buffer.
//
// Parameters:
//   - msg: Log message to display.
func (m *Model) addLogEntry(msg LogMsg) {
	entry := fmt.Sprintf("[%s] %s", msg.Level, msg.Message)

	m.logs = append(m.logs, entry)

	if len(m.logs) > maxLogLines {
		m.logs = m.logs[len(m.logs)-maxLogLines:]
	}
}
