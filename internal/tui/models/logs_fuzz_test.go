/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"testing"
)

// Fuzz_addLogEntry fuzz tests log entry handling by verifying that addLogEntry()
// handles any input gracefully and maintains the circular buffer behavior.
func Fuzz_addLogEntry(f *testing.F) {
	// Seed with typical log messages
	f.Add("INF", "test message")
	f.Add("DBG", "debug information")
	f.Add("WRN", "warning message")
	f.Add("ERR", "error occurred")
	f.Add("", "")
	f.Add("INF", "")
	f.Add("", "message only")

	f.Fuzz(func(t *testing.T, level string, message string) {
		m := &Model{
			logs:     make([]string, 0, maxLogLines),
			showLogs: true,
		}

		// Create log message with fuzzed inputs
		logMsg := LogMsg{Level: level, Message: message}

		// Should not panic with any input
		m.addLogEntry(logMsg)

		// Verify log was added
		if len(m.logs) == 0 {
			t.Error("log entry was not added")
		}

		// Verify the log entry contains expected format
		lastEntry := m.logs[len(m.logs)-1]
		if lastEntry == "" && (level != "" || message != "") {
			t.Error("log entry format is incorrect")
		}

		// Test circular buffer by adding many entries
		for range maxLogLines + 10 {
			m.addLogEntry(LogMsg{Level: level, Message: message})
		}

		// Verify buffer size is capped at maxLogLines
		if len(m.logs) > maxLogLines {
			t.Errorf(
				"log buffer exceeded maxLogLines: got %d, want <= %d",
				len(m.logs),
				maxLogLines,
			)
		}

		// Verify buffer is not empty
		if len(m.logs) == 0 {
			t.Error("log buffer should not be empty after adding entries")
		}
	})
}

// Fuzz_pollLogChannel fuzz tests log polling with random log channel states
// to ensure pollLogChannel() handles various channel conditions without panicking.
func Fuzz_pollLogChannel(f *testing.F) {
	// Seed with flags indicating channel state
	// First bool: whether to send a message
	// Second bool: whether to close the channel after sending
	f.Add(true, false)
	f.Add(false, false)
	f.Add(true, true)

	f.Fuzz(func(t *testing.T, hasMessage bool, closeChannel bool) {
		logChan := make(chan LogMsg, 10)
		m := &Model{logChan: logChan}

		if hasMessage {
			// Send a message to the channel
			select {
			case logChan <- LogMsg{Level: "INF", Message: "test"}:
			default:
			}
		}

		if closeChannel {
			close(logChan)
		}

		cmd := m.pollLogChannel()
		if cmd != nil {
			_ = cmd()
		}

		// Clean up if not already closed
		if !closeChannel {
			close(logChan)
		}
	})
}
