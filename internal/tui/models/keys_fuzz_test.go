/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"testing"

	"github.com/nicholas-fedor/go-remove/internal/history"
	"github.com/nicholas-fedor/go-remove/internal/logger"
	"github.com/nicholas-fedor/go-remove/internal/tui/render"
)

type fuzzFS struct{}

func (fuzzFS) DetermineBinDir(bool) (string, error) { return "/bin", nil }

func (fuzzFS) AdjustBinaryPath(dir, binary string) string { return dir + "/" + binary }

func (fuzzFS) RemoveBinary(string, string, bool, logger.Logger) error { return nil }

func (fuzzFS) ListBinaries(string) ([]string, error) {
	return []string{"test1", "test2", "test3"}, nil
}

// Fuzz_keyPressString verifies keyPressString accepts arbitrary key names.
func Fuzz_keyPressString(f *testing.F) {
	f.Add("enter")
	f.Add("up")
	f.Add("q")
	f.Add("")
	f.Add("ctrl+c")
	f.Add("down")
	f.Add("left")
	f.Add("right")
	f.Add(" ")
	f.Add("\n")
	f.Add("esc")

	f.Fuzz(func(t *testing.T, key string) {
		msg := keyPressString(key)
		_ = msg.String()
	})
}

// Fuzz_model_Update fuzz tests the Update() method with random key sequences.
// It verifies that Update() doesn't panic with any input and maintains state consistency.
func Fuzz_model_Update(f *testing.F) {
	// Seed corpus with valid inputs representing common key presses
	f.Add("enter")
	f.Add("esc")
	f.Add("up")
	f.Add("down")
	f.Add("left")
	f.Add("right")
	f.Add("r")
	f.Add("b")
	f.Add("q")
	f.Add("s")
	f.Add("L")
	f.Add("u")
	f.Add("k")
	f.Add("j")
	f.Add("h")
	f.Add("l")
	f.Add("y")
	f.Add("n")
	f.Add("c")
	f.Add("C")
	f.Add("d")

	f.Fuzz(func(t *testing.T, key string) {
		// Initialize a model with test data
		m := &Model{
			choices:       []string{"test1", "test2", "test3"},
			cursorY:       0,
			cursorX:       0,
			rows:          3,
			cols:          1,
			mode:          modeBinaries,
			sortAscending: true,
			logs:          make([]string, 0, maxLogLines),
			showLogs:      false,
			width:         80,
			height:        24,
			dir:           "/bin",
			fs:            fuzzFS{},
			logger:        logger.NopLogger(),
			styles:        render.DefaultStyleConfig(),
		}

		msg := keyPressString(key)
		result, _ := m.Update(msg)
		resultModel := result.(*Model)

		// Cursor should never be negative
		if resultModel.cursorX < 0 {
			t.Errorf("cursorX became negative: %d", resultModel.cursorX)
		}

		if resultModel.cursorY < 0 {
			t.Errorf("cursorY became negative: %d", resultModel.cursorY)
		}

		// Mode should be one of the valid modes
		if resultModel.mode != modeBinaries && resultModel.mode != modeHistory {
			t.Errorf("invalid mode: %s", resultModel.mode)
		}

		// Confirmation should be a valid value
		if resultModel.confirmation != confirmNone &&
			resultModel.confirmation != confirmClearAll &&
			resultModel.confirmation != confirmDeletePerm {
			t.Errorf("invalid confirmation state: %s", resultModel.confirmation)
		}

		// Logs slice should never be nil after Update
		if resultModel.logs == nil {
			t.Error("logs slice became nil after Update")
		}
	})
}

// Fuzz_keyMatches fuzz tests the key matching logic by testing key comparison
// against all valid key types to ensure consistent matching behavior.
func Fuzz_keyMatches(f *testing.F) {
	// Seed with various key patterns
	f.Add("a")
	f.Add("A")
	f.Add("enter")
	f.Add("up")
	f.Add("")
	f.Add("special_key_123")

	f.Fuzz(func(t *testing.T, key string) {
		// Test creating KeyPressMsg from string using helper
		msg := keyPressString(key)

		// Test that String() method works without panic
		_ = msg.String()

		// Test various key comparisons that are used in the actual code
		// These match the patterns used in updateBinaryMode and updateHistoryMode
		switch msg.String() {
		case "q", keyCtrlC:
			// Quit keys
		case keyUp, "k":
			// Up navigation
		case keyDown, "j":
			// Down navigation
		case keyLeft, "h":
			// Left navigation
		case keyRight, "l":
			// Right navigation
		case keyEnter:
			// Enter key
		case "s":
			// Sort toggle
		case "r":
			// History mode
		case "b":
			// Back to binaries
		case "L":
			// Toggle logs
		case "u":
			// Undo
		case "y", "Y":
			// Confirm
		case "n", "N":
			// Cancel
		case "c":
			// Clear entry
		case "C":
			// Clear all
		case "d":
			// Delete permanently
		default:
			// Unknown key, handled by the default branch above.
		}

		_ = msg.Code
	})
}

// Fuzz_model_stateConsistency fuzz tests state transitions and consistency
// across multiple Update() calls with various inputs.
func Fuzz_model_stateConsistency(f *testing.F) {
	// Seed with sequences of key inputs (space-separated)
	f.Add("up down left right")
	f.Add("s ")      // Multiple sort toggles
	f.Add("r b r b") // Mode switching
	f.Add("q")       // Quit

	f.Fuzz(func(t *testing.T, keySequence string) {
		m := &Model{
			choices:        []string{"bin1", "bin2", "bin3", "bin4"},
			cursorY:        0,
			cursorX:        0,
			rows:           2,
			cols:           2,
			mode:           modeBinaries,
			sortAscending:  true,
			logs:           make([]string, 0, maxLogLines),
			showLogs:       false,
			width:          80,
			height:         24,
			dir:            "/bin",
			fs:             fuzzFS{},
			logger:         logger.NopLogger(),
			styles:         render.DefaultStyleConfig(),
			historyEntries: make([]*history.HistoryEntry, 0),
			historyCursor:  0,
			confirmation:   confirmNone,
		}

		keys := splitKeys(keySequence)
		for _, key := range keys {
			msg := keyPressString(key)
			result, _ := m.Update(msg)
			m = result.(*Model)

			// State consistency checks
			if m.cursorX < 0 {
				t.Errorf("cursorX became negative after key %q: %d", key, m.cursorX)
			}

			if m.cursorY < 0 {
				t.Errorf("cursorY became negative after key %q: %d", key, m.cursorY)
			}

			if m.mode != modeBinaries && m.mode != modeHistory {
				t.Errorf("invalid mode after key %q: %s", key, m.mode)
			}

			if m.historyCursor < 0 {
				t.Errorf("historyCursor became negative after key %q: %d", key, m.historyCursor)
			}
		}

		// Final state consistency check
		if m.logs == nil {
			t.Error("logs became nil after sequence")
		}

		if m.historyEntries == nil {
			t.Error("historyEntries became nil after sequence")
		}
	})
}

// splitKeys splits a space-separated key sequence into individual keys.
// It handles empty strings and multiple spaces gracefully.
func splitKeys(sequence string) []string {
	if sequence == "" {
		return []string{}
	}

	var keys []string

	start := 0
	inWord := false

	for i, ch := range sequence {
		if ch == ' ' {
			if inWord {
				keys = append(keys, sequence[start:i])
				inWord = false
			}
		} else {
			if !inWord {
				start = i
				inWord = true
			}
		}
	}

	if inWord {
		keys = append(keys, sequence[start:])
	}

	return keys
}
