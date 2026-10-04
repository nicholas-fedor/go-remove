/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package render

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestViewBinaries_FitsNarrowTerminal verifies the grid shrinks to the window
// rather than overflowing it.
func TestViewBinaries_FitsNarrowTerminal(t *testing.T) {
	t.Parallel()

	for _, width := range []int{30, 40, 80, 160} {
		t.Run(fmt.Sprintf("width=%d", width), func(t *testing.T) {
			t.Parallel()

			state := &State{
				Mode:    ModeBinaries,
				Choices: []string{"a-fairly-long-binary-name", "short", "日本語のツール"},
				Cols:    1,
				Rows:    3,
				CursorX: 0,
				CursorY: 0,
				Width:   width,
				Height:  24,
				Styles:  DefaultStyleConfig(),
			}
			withGrid(state)

			rendered := stripANSI(Binaries(state).Content)

			for line := range strings.SplitSeq(rendered, "\n") {
				assert.LessOrEqual(t, DisplayWidth(line), width,
					"no line may exceed the terminal width, got %d for %q",
					DisplayWidth(line), line)
			}
		})
	}
}
