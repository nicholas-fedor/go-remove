/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package render

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/rivo/uniseg"
)

// DisplayWidth returns the number of terminal cells a string occupies.
//
// Rune count is not sufficient: a CJK character occupies two cells, so a grid
// sized on runes overflows the terminal and its padding comes out short.
func DisplayWidth(s string) int {
	return lipgloss.Width(s)
}

// TruncateToWidth shortens s to at most width cells, ending on a grapheme cluster
// boundary.
func TruncateToWidth(s string, width int) string {
	if DisplayWidth(s) <= width {
		return s
	}

	var (
		builder strings.Builder
		used    int
	)

	// Clusters are walked whole and measured by lipgloss, so one is kept or
	// dropped whole. Cutting at a byte index would split a rune, and cutting per
	// rune would leave a dangling ZWJ sequence or skin-tone modifier behind,
	// which the terminal renders as tofu.
	clusters := uniseg.NewGraphemes(s)
	for clusters.Next() {
		cluster := clusters.Str()

		cellWidth := lipgloss.Width(cluster)
		if used+cellWidth > width {
			break
		}

		builder.WriteString(cluster)

		used += cellWidth
	}

	return builder.String()
}

// PadToWidth appends spaces so s occupies exactly width terminal cells.
//
// A styled string measures by its printable text, so this also works on a
// rendered value.
func PadToWidth(s string, width int) string {
	// fmt's %-*s would pad by rune count, which is too few for a wide rune and
	// leaves the column overflowing.
	return s + strings.Repeat(" ", max(width-DisplayWidth(s), 0))
}
