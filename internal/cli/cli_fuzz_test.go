/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cli

import (
	"testing"

	"github.com/nicholas-fedor/go-remove/internal/history"
)

// FuzzView verifies View rendering does not panic for arbitrary terminal sizes.
func FuzzView(f *testing.F) {
	f.Add(80, 24)
	f.Add(0, 0)
	f.Add(1, 1)
	f.Add(200, 50)

	f.Fuzz(func(t *testing.T, width, height int) {
		m := &model{
			choices:       []string{"tool"},
			rows:          1,
			cols:          1,
			width:         width,
			height:        height,
			logs:          []string{},
			styles:        defaultStyleConfig(),
			sortAscending: true,
			mode:          modeBinaries,
		}
		_ = m.View()

		m.mode = modeHistory
		m.historyEntries = []*history.HistoryEntry{{BinaryName: "tool"}}
		_ = m.View()
	})
}

// FuzzKeyPressString verifies keyPressString accepts arbitrary key names.
func FuzzKeyPressString(f *testing.F) {
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
