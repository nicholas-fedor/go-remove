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
//
// Slicing at a byte index would split a multi-byte rune in half and render
// invalid UTF-8 as mojibake. Accumulating per rune is not enough either: an
// emoji joined into a single cluster (a ZWJ sequence such as a family, or a
// skin-tone modifier) spans several runes, and cutting between them leaves a
// dangling cluster that the terminal renders as tofu. uniseg walks whole
// clusters, and lipgloss measures them, so a cluster is kept or dropped whole.
func TruncateToWidth(s string, width int) string {
	if DisplayWidth(s) <= width {
		return s
	}

	var (
		builder strings.Builder
		used    int
	)

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
// fmt's %-*s pads by rune count, which is too few for a wide rune and leaves
// the column overflowing. A styled string measures by its printable text, so
// this also works on a rendered value.
func PadToWidth(s string, width int) string {
	return s + strings.Repeat(" ", max(width-DisplayWidth(s), 0))
}
