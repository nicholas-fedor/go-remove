/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"testing"

	"github.com/nicholas-fedor/go-remove/internal/history"
	"github.com/nicholas-fedor/go-remove/internal/tui/render"
)

// Fuzz_model_View verifies View rendering does not panic for arbitrary terminal
// sizes, in either mode.
func Fuzz_model_View(f *testing.F) {
	f.Add(80, 24)
	f.Add(0, 0)
	f.Add(1, 1)
	f.Add(200, 50)

	f.Fuzz(func(t *testing.T, width, height int) {
		m := &Model{
			choices:       []string{"tool"},
			rows:          1,
			cols:          1,
			width:         width,
			height:        height,
			logs:          []string{},
			styles:        render.DefaultStyleConfig(),
			sortAscending: true,
			mode:          modeBinaries,
		}
		_ = m.View()

		m.mode = modeHistory
		m.historyEntries = []*history.HistoryEntry{{BinaryName: "tool"}}
		_ = m.View()
	})
}
