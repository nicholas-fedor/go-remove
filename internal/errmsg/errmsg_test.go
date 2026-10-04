/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package errmsg

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/go-remove/internal/history"
)

// TestClassify verifies each sentinel is recognized, including through a wrap.
func TestClassify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want Kind
	}{
		{name: "nil", err: nil, want: KindUnknown},
		{name: "no history", err: history.ErrNoHistory, want: KindNoHistory},
		{
			name: "already restored",
			err:  history.ErrAlreadyRestored,
			want: KindAlreadyRestored,
		},
		{name: "not in trash", err: history.ErrNotInTrash, want: KindNotInTrash},
		{
			name: "restore collision",
			err:  history.ErrRestoreCollision,
			want: KindRestoreCollision,
		},
		{name: "unrelated", err: errors.New("disk on fire"), want: KindUnknown},
		{
			name: "wrapped no history",
			err:  fmt.Errorf("undo failed: %w", history.ErrNoHistory),
			want: KindNoHistory,
		},
		{
			name: "wrapped already restored",
			err:  fmt.Errorf("undo failed: %w", history.ErrAlreadyRestored),
			want: KindAlreadyRestored,
		},
		{
			name: "wrapped not in trash",
			err:  fmt.Errorf("restore failed: %w", history.ErrNotInTrash),
			want: KindNotInTrash,
		},
		{
			name: "wrapped restore collision",
			err:  fmt.Errorf("restore failed: %w", history.ErrRestoreCollision),
			want: KindRestoreCollision,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, Classify(tt.err))
		})
	}
}

// TestClassify_KindsAreDistinct guards against two categories collapsing onto
// the same value, which would silently merge two different user-facing
// outcomes.
func TestClassify_KindsAreDistinct(t *testing.T) {
	t.Parallel()

	kinds := []Kind{
		KindUnknown,
		KindNoHistory,
		KindAlreadyRestored,
		KindNotInTrash,
		KindRestoreCollision,
	}

	seen := make(map[Kind]bool, len(kinds))

	for _, kind := range kinds {
		assert.False(t, seen[kind], "kind %d is declared more than once", kind)
		seen[kind] = true
	}
}
