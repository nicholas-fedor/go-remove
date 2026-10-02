/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"context"
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/nicholas-fedor/go-remove/internal/errmsg"
	"github.com/nicholas-fedor/go-remove/internal/history"
)

// Key names as Bubble Tea reports them on a KeyPressMsg.
//
// Update matches on these, so they are named once here instead of being
// repeated as literals in each switch, and keyPressString turns one back into
// the press that produced it.
const (
	keyUp    = "up"
	keyDown  = "down"
	keyLeft  = "left"
	keyRight = "right"
	keyEnter = "enter"
)

// keyPress builds a key press from the single-rune shortcut the view advertises.
//
// Parameters:
//   - r: Shortcut character, such as 'j'.
//
// Returns:
//   - A key press for that character.
func keyPress(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Text: string(r), Code: r, ShiftedCode: r}
}

// keyPressString builds a key press from the name Bubble Tea reports for it,
// which is the inverse of the matching Update performs.
//
// Parameters:
//   - name: Key name, such as "up" or "enter".
//
// Returns:
//   - A key press reporting that name, or an empty press for an unknown name.
func keyPressString(name string) tea.KeyPressMsg {
	switch name {
	case keyEnter:
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case keyUp:
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case keyDown:
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case keyLeft:
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case keyRight:
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case keyCtrlC:
		// A modified key cannot fall through to the single-rune branch below,
		// so it has to be built explicitly.
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	default:
		if len(name) == 1 {
			return keyPress(rune(name[0]))
		}

		return tea.KeyPressMsg{}
	}
}

// Update processes TUI events and updates the model state.
//
// Parameters:
//   - msg: Incoming Bubble Tea message.
//
// Returns:
//   - Updated model.
//   - Follow-up command, if any.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// An interrupt always works, so a long operation can be stopped.
		if msg.String() == keyCtrlC {
			m.cancelInFlight()
			m.status = "Interrupted"
		}

		// While an operation is running the model is not in a state a keypress
		// can act on, so the key is ignored rather than acted on halfway.
		if m.busy != "" && msg.String() != keyCtrlC {
			return m, nil
		}

		// Ignore KeyReleaseMsg. KeyMsg matches both, so a single physical
		// keystroke would otherwise undo or restore twice.
		if m.confirmation != confirmNone {
			return m.handleConfirmation(msg)
		}

		if m.mode == modeHistory {
			return m.updateHistoryMode(msg)
		}

		return m.updateBinaryMode(msg)

	case tea.WindowSizeMsg:
		// Update dimensions and recalculate grid layout on resize.
		m.width = msg.Width
		m.height = msg.Height
		m.updateGrid()

		return m, nil

	case pollLogTickMsg:
		// Continue polling for log messages when a tick occurs.
		cmd := m.pollLogChannel()

		return m, cmd

	case opResultMsg:
		// The operation has finished, so the model is interactive again.
		m.busy = ""
		m.cancelOp = nil

		// The grid is recalculated below, once the status line is final,
		// because the layout reserves a row for a status that is present.

		// An interrupted operation still applies its side effects, because the
		// work may not have observed the cancellation, but it must not then
		// report success the user has already been told they stopped.
		interrupted := m.interrupted
		m.interrupted = false

		if msg.err != nil {
			if !interrupted {
				if msg.errStatus != nil {
					m.status = msg.errStatus(msg.err)
				} else {
					// The error is already wrapped with the operation.
					m.status = "Error " + msg.err.Error()
				}
			}

			m.updateGrid()

			if m.historyManager == nil {
				return m, nil
			}

			reload := m.loadHistoryReporting(true)

			return m, reload
		}

		// The operation may report a more specific outcome than the generic
		// form, so its own status wins when it sets one. An interrupted
		// operation keeps the status the interrupt already showed, even though
		// the refresh below reports the work as done.
		preserved := m.status

		if !interrupted {
			m.status = msg.okStatus
		}

		if msg.refresh != nil {
			msg.refresh(m)
		}

		switch {
		case interrupted:
			m.status = preserved
		case m.status == "":
			m.status = "Done " + msg.operation
		}

		m.updateGrid()

		if m.historyManager == nil {
			return m, nil
		}

		reload := m.loadHistoryReporting(true)

		return m, reload

	case LogMsg:
		// Add log message to the circular buffer.
		m.addLogEntry(msg)
		m.updateGrid()

		// Continue polling for more log messages.
		// This ensures all pending logs are captured.
		cmd := m.pollLogChannel()

		return m, cmd

	case HistoryMsg:
		// Handle history loading result
		m.historyLoading = false

		if msg.quiet {
			// A background refresh keeps whatever the operation reported.
			if msg.Error == nil {
				m.historyEntries = msg.Entries
				m.clampHistoryCursor()
			}

			return m, nil
		}

		if msg.Error != nil {
			m.status = fmt.Sprintf("Error loading history: %v", msg.Error)
		} else {
			m.historyEntries = msg.Entries
			if len(m.historyEntries) == 0 {
				m.status = "No deletion history found"
			} else {
				m.status = fmt.Sprintf("Loaded %d history entries", len(m.historyEntries))
			}

			// The list may have shrunk, leaving the cursor past the end, in
			// which case no row renders as selected and the view looks frozen on
			// a phantom row until the user moves up.
			m.clampHistoryCursor()
		}

		return m, nil
	}

	return m, nil
}

// handleConfirmation processes key presses during confirmation dialogs.
//
// Parameters:
//   - msg: Key press from the user.
//
// Returns:
//   - Updated model.
//   - Follow-up command, if any.
func (m *Model) handleConfirmation(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// Confirmed - execute the operation
		return m.executeConfirmation()
	case "ctrl+c":
		// The one key a dialog must never swallow, otherwise the prompt
		// becomes a trap for anyone who reaches for it.
		m.confirmation = confirmNone
		m.status = "Operation cancelled"

		return m, tea.Quit
	case "n", "N", "q", "esc":
		// Cancelled - clear confirmation
		m.confirmation = confirmNone
		m.status = "Operation cancelled"
	}

	return m, nil
}

// clampHistoryCursor keeps the history cursor within the loaded entries.
func (m *Model) clampHistoryCursor() {
	if len(m.historyEntries) == 0 {
		m.historyCursor = 0

		return
	}

	if m.historyCursor >= len(m.historyEntries) {
		m.historyCursor = len(m.historyEntries) - 1
	}

	if m.historyCursor < 0 {
		m.historyCursor = 0
	}
}

// executeConfirmation executes the pending confirmation operation.
//
// Returns:
//   - Updated model.
//   - Follow-up command, if any.
func (m *Model) executeConfirmation() (tea.Model, tea.Cmd) {
	confirmation := m.confirmation

	// The user has confirmed, so the dialog closes as the work starts. Keys are
	// ignored while it runs.
	m.confirmation = confirmNone

	manager := m.historyManager

	var entry *history.HistoryEntry

	if m.historyCursor < len(m.historyEntries) {
		entry = m.historyEntries[m.historyCursor]
	}

	// Clearing the history iterates and permanently deletes every trashed
	// binary, the longest operation in the tool. All confirmations run outside
	// Update so the view keeps rendering and stays interruptible.
	switch confirmation {
	case confirmClearAll:
		if manager == nil {
			return m, nil
		}

		clearCmd := m.runAsyncReporting(
			"clearing history",
			func(ctx context.Context) error {
				if err := manager.ClearHistory(ctx, false); err != nil {
					return fmt.Errorf("clearing history: %w", err)
				}

				return nil
			},
			func(m *Model) {
				m.historyEntries = make([]*history.HistoryEntry, 0)
				m.historyCursor = 0
			},
			nil,
			func() string { return "History cleared" },
		)

		return m, clearCmd

	case confirmDeletePerm:
		if manager == nil || entry == nil {
			return m, nil
		}

		entryID := entry.ID
		name := entry.BinaryName

		deleteCmd := m.runAsyncReporting(
			"deleting "+name+" permanently",
			func(ctx context.Context) error {
				if err := manager.DeletePermanently(ctx, entryID); err != nil {
					return fmt.Errorf("deleting permanently: %w", err)
				}

				return nil
			},
			nil,
			nil,
			func() string { return "Permanently deleted " + name },
		)

		return m, deleteCmd
	}

	return m, nil
}

// updateHistoryMode processes key events in history mode.
//
// Parameters:
//   - msg: Key press from the user.
//
// Returns:
//   - Updated model.
//   - Follow-up command, if any.
func (m *Model) updateHistoryMode(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit // Exit the TUI

	case "b":
		// Back to binary mode
		m.mode = modeBinaries

		// Clear the history-view status first, so a failed rescan below can set
		// its own rather than being wiped here.
		m.status = ""
		m.refreshChoices()
		m.sortChoices()
		m.updateGrid()

	case keyUp, "k":
		// Move cursor up in history list
		if m.historyCursor > 0 {
			m.historyCursor--
		}

	case keyDown, "j":
		// Move cursor down in history list
		if m.historyCursor < len(m.historyEntries)-1 {
			m.historyCursor++
		}

	case keyEnter:
		// Restore selected entry
		return m.handleRestore()

	case "d":
		// Delete permanently (with confirmation)
		if m.historyCursor < len(m.historyEntries) {
			m.confirmation = confirmDeletePerm
		}

	case "c":
		// Clear this entry (keep in trash)
		return m.handleClearEntry(false)

	case "C":
		// Clear all history (with confirmation)
		if len(m.historyEntries) > 0 {
			m.confirmation = confirmClearAll
		}

	case "u":
		// Undo most recent deletion
		return m.handleUndo()

	case "L":
		// Toggle verbose logging and log panel visibility
		m.toggleVerboseLogging()

		return m, nil
	}

	return m, nil
}

// updateBinaryMode processes key events in binary selection mode.
//
// Parameters:
//   - msg: Key press from the user.
//
// Returns:
//   - Updated model.
//   - Follow-up command, if any.
func (m *Model) updateBinaryMode(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit // Exit the TUI

	case keyUp, "k":
		// Move cursor up, stopping at the top row.
		if m.cursorY > 0 {
			m.cursorY--
		}

	case keyDown, "j":
		// Move cursor down, respecting grid bounds and item count.
		newY := m.cursorY + 1

		newIdx := newY + m.cursorX*m.rows // Column-major index (fill down columns)
		if newY < m.rows && newIdx < len(m.choices) {
			m.cursorY = newY
		}

	case keyLeft, "h":
		// Move cursor left, stopping at the first column.
		if m.cursorX > 0 {
			m.cursorX--
		}

	case keyRight, "l":
		// Move cursor right, respecting column bounds and item count.
		newX := m.cursorX + 1

		newIdx := m.cursorY + newX*m.rows // Column-major index
		if newX < m.cols && newIdx < len(m.choices) {
			m.cursorX = newX
		}

	case "s":
		// Toggle sort order and re-sort the choices.
		m.sortAscending = !m.sortAscending
		m.sortChoices()
		m.updateGrid()

	case "L":
		// Toggle verbose logging and log panel visibility.
		m.toggleVerboseLogging()

		return m, nil

	case "r":
		// Switch to history view
		m.mode = modeHistory
		m.historyLoading = true
		m.historyCursor = 0
		m.status = "Loading history..."

		cmd := m.loadHistory()

		return m, cmd

	case "u":
		// Undo most recent deletion
		return m.handleUndo()

	case keyEnter:
		// Remove the selected binary and update the TUI state.
		return m.handleRemove()
	}

	return m, nil
}

// handleRemove removes the binary under the cursor.
//
// Returns:
//   - Updated model.
//   - Command running the removal outside Update.
func (m *Model) handleRemove() (tea.Model, tea.Cmd) {
	if len(m.choices) == 0 {
		return m, nil
	}

	idx := m.cursorY + m.cursorX*m.rows // Column-major index
	if idx >= len(m.choices) {
		return m, nil
	}

	name := m.choices[idx]
	binaryPath := m.fs.AdjustBinaryPath(m.dir, name)
	manager := m.historyManager
	filesystem := m.fs
	verbose := m.config.Verbose
	log := m.logger

	// Moving the binary can take a while on a large file, so it runs outside
	// Update to keep the view responsive and interruptible.
	removeCmd := m.runAsync(
		"removing "+name,
		func(ctx context.Context) error {
			// Use history manager if available (it handles trash plus history).
			if manager != nil {
				if _, err := manager.RecordDeletion(ctx, binaryPath); err != nil {
					return fmt.Errorf("recording %s: %w", name, err)
				}

				return nil
			}

			// Fallback: permanent delete only without a manager.
			if err := filesystem.RemoveBinary(
				binaryPath,
				name,
				verbose,
				log,
			); err != nil {
				return fmt.Errorf("removing %s: %w", name, err)
			}

			return nil
		},
		func(m *Model) {
			// Drop the deleted entry before rescanning, so a failed rescan
			// cannot leave the view showing a binary that is already gone.
			m.choices = slices.DeleteFunc(m.choices, func(choice string) bool {
				return choice == name
			})
			m.refreshChoices()
			m.sortChoices()

			// Adjust cursor if it exceeds remaining choices.
			if m.cursorY+m.cursorX*m.rows >= len(m.choices) {
				lastIdx := len(m.choices) - 1
				m.cursorX = lastIdx / m.rows
				m.cursorY = lastIdx % m.rows
			}

			m.updateGrid()
			m.status = "Removed " + name
		},
	)

	return m, removeCmd
}

// handleRestore restores the selected history entry.
//
// Returns:
//   - Updated model.
//   - Command to refresh history after a successful restore.
func (m *Model) handleRestore() (tea.Model, tea.Cmd) {
	if m.historyManager == nil || m.historyCursor >= len(m.historyEntries) {
		m.status = "No history entry selected"

		return m, nil
	}

	entry := m.historyEntries[m.historyCursor]

	// Check if entry can be restored
	if !entry.InTrash {
		m.status = fmt.Sprintf("Cannot restore %s: not available in trash", entry.BinaryName)

		return m, nil
	}

	name := entry.BinaryName
	manager := m.historyManager

	// The destination is only known to the operation, which reads it from the
	// result on its own goroutine.
	var restoredTo string

	// The restore touches the filesystem and the history store, so it runs
	// outside Update to keep the view responsive.
	restoreCmd := m.runAsyncReporting(
		"restoring "+name,
		func(ctx context.Context) error {
			result, err := manager.Restore(ctx, entry.ID)
			if err != nil {
				return fmt.Errorf("restoring %s: %w", name, err)
			}

			if result != nil {
				restoredTo = fmt.Sprintf("Restored %s to %s", result.BinaryName, result.RestoredTo)
			}

			return nil
		},
		func(m *Model) {
			m.refreshChoices()
			m.sortChoices()
			m.updateGrid()
		},
		restoreErrorStatus(name),
		func() string { return restoredTo },
	)

	return m, restoreCmd
}

// handleUndo restores the most recently deleted binary.
//
// Returns:
//   - Updated model.
//   - Command to refresh history when viewing history mode.
func (m *Model) handleUndo() (tea.Model, tea.Cmd) {
	if m.historyManager == nil {
		m.status = "History manager not available"

		return m, nil
	}

	manager := m.historyManager

	// The destination is only known to the operation, which reads it from the
	// result on its own goroutine.
	var restoredTo string

	// Undo walks the history in pages and restores from the trash, so it runs
	// outside Update to keep the view responsive and interruptible.
	undoCmd := m.runAsyncReporting(
		"undoing the most recent deletion",
		func(ctx context.Context) error {
			result, err := manager.UndoMostRecent(ctx)
			if err != nil {
				return fmt.Errorf("undo failed: %w", err)
			}

			if result != nil {
				restoredTo = fmt.Sprintf("Restored %s to %s", result.BinaryName, result.RestoredTo)
			}

			return nil
		},
		func(m *Model) {
			// Refresh history and binaries if in binary mode.
			if m.mode == modeBinaries {
				m.refreshChoices()
				m.sortChoices()
				m.updateGrid()
			}
		},
		func(err error) string {
			// The error is already wrapped with the operation, and a status
			// line reads better capitalised.
			fallback := func() string {
				msg := err.Error()
				if msg == "" {
					return "Undo failed"
				}

				return strings.ToUpper(msg[:1]) + msg[1:]
			}

			switch errmsg.Classify(err) {
			case errmsg.KindNoHistory:
				return "No deletion history found - nothing to undo"
			case errmsg.KindAlreadyRestored:
				return "Binary has already been restored"
			case errmsg.KindNotInTrash:
				return "Binary is no longer in trash - cannot restore"
			case errmsg.KindRestoreCollision:
				return "A file already exists at the restore location"
			case errmsg.KindUnknown:
				return fallback()
			default:
				// A Kind was added without a message for this operation.
				return fallback()
			}
		},
		func() string { return restoredTo },
	)

	return m, undoCmd
}

// handleClearEntry removes a history entry and optionally deletes it from trash.
//
// Parameters:
//   - deleteFromTrash: When true, also permanently delete the trashed binary.
//
// Returns:
//   - Updated model.
//   - Command to refresh history after a successful clear.
func (m *Model) handleClearEntry(deleteFromTrash bool) (tea.Model, tea.Cmd) {
	if m.historyManager == nil || m.historyCursor >= len(m.historyEntries) {
		m.status = "No history entry selected"

		return m, nil
	}

	entry := m.historyEntries[m.historyCursor]
	ctx := m.context()

	if err := m.historyManager.ClearEntry(ctx, entry.ID, deleteFromTrash); err != nil {
		m.status = fmt.Sprintf("Error clearing entry: %v", err)
	} else {
		if deleteFromTrash {
			m.status = fmt.Sprintf("Permanently deleted %s and cleared history", entry.BinaryName)
		} else {
			m.status = "Cleared history entry for " + entry.BinaryName
		}
		// Refresh history
		cmd := m.loadHistory()

		return m, cmd
	}

	return m, nil
}
