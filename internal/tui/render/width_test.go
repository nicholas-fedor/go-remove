/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package render

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

// TestTruncateToWidth verifies truncation ends on a grapheme cluster boundary.
//
// Slicing at a byte index splits a multi-byte rune and renders invalid UTF-8.
// Accumulating per rune splits a cluster that spans runes, such as a joined
// emoji, and leaves a dangling sequence the terminal renders as tofu.
func TestTruncateToWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		width int
		want  string
	}{
		{name: "fits", input: "tool", width: 10, want: "tool"},
		{name: "ascii truncation", input: "a-very-long-tool-name", width: 8, want: "a-very-l"},
		{name: "multibyte whole runes", input: "héllo", width: 4, want: "héll"},
		{name: "multibyte never splits", input: "日本語のツール", width: 5, want: "日本"},
		{name: "cjk is two cells per rune", input: "日本語", width: 3, want: "日"},
		{name: "zero width", input: "tool", width: 0, want: ""},
		{
			name:  "joined emoji cluster fits whole",
			input: "👨‍👩‍👧‍👦x",
			width: 2,
			want:  "👨‍👩‍👧‍👦",
		},
		{
			// Cutting after the man would leave a dangling zero-width joiner.
			name:  "cluster cut off is dropped whole",
			input: "ab👨‍👩‍👧‍👦",
			width: 3,
			want:  "ab",
		},
		{
			name:  "skin tone modifier stays attached",
			input: "👍🏽x",
			width: 2,
			want:  "👍🏽",
		},
		{
			// An "e" plus a combining acute (U+0301): one cluster, one cell.
			// Per rune it would split, leaving a dangling accent.
			name:  "combining mark stays attached",
			input: "e" + string(rune(0x301)) + "x",
			width: 1,
			want:  "e" + string(rune(0x301)),
		},
		{
			name:  "flag sequence is one cluster",
			input: "🇺🇸🇬🇧",
			width: 2,
			want:  "🇺🇸",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := TruncateToWidth(tt.input, tt.width)

			assert.Equal(t, tt.want, got)
			assert.True(t, utf8.ValidString(got),
				"truncation must not leave invalid UTF-8")
			assert.LessOrEqual(t, DisplayWidth(got), tt.width,
				"the result must fit the width")
		})
	}
}

// TestDisplayWidth_IgnoresRuneCount verifies a wide rune counts as two cells, so
// a grid sized on runes would overflow.
func TestDisplayWidth_IgnoresRuneCount(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 4, DisplayWidth("tool"))
	assert.Equal(t, 5, DisplayWidth("héllo"), "accents stay single width")
	assert.Equal(t, 6, DisplayWidth("日本語"), "CJK runes are two cells each")
}
