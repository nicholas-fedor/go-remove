/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package render

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestVisibleLogs verifies the log buffer is trimmed to what fits in the panel.
//
// Whether the panel is drawn at all is the caller's decision, made before this
// is called.
func TestVisibleLogs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		logs    []string
		wantLen int
	}{
		{
			name:    "fewer than the limit",
			logs:    []string{"log1", "log2", "log3"},
			wantLen: 3,
		},
		{
			name:    "empty buffer",
			logs:    []string{},
			wantLen: 0,
		},
		{
			name:    "exceed the limit",
			logs:    []string{"1", "2", "3", "4", "5", "6", "7"},
			wantLen: MaxVisibleLogLines,
		},
		{
			name:    "nil buffer",
			logs:    nil,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Len(t, VisibleLogs(tt.logs), tt.wantLen)
		})
	}
}

// TestVisibleLogs_KeepsTheNewest verifies trimming keeps the most recent lines.
func TestVisibleLogs_KeepsTheNewest(t *testing.T) {
	t.Parallel()

	logs := []string{"1", "2", "3", "4", "5", "6", "7"}

	got := VisibleLogs(logs)

	assert.Len(t, got, MaxVisibleLogLines)
	assert.Equal(t, "7", got[len(got)-1], "the newest line must survive trimming")
	assert.NotContains(t, got, "1", "the oldest lines must be dropped")
}
