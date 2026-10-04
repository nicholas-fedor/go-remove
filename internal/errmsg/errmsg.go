/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package errmsg

import (
	"errors"

	"github.com/nicholas-fedor/go-remove/internal/history"
)

const (
	// KindUnknown is an error the history layer does not classify.
	KindUnknown Kind = iota

	// KindNoHistory means no deletion history exists to operate on.
	KindNoHistory

	// KindAlreadyRestored means the entry was restored by an earlier attempt.
	KindAlreadyRestored

	// KindNotInTrash means the binary is no longer in the trash.
	KindNotInTrash

	// KindRestoreCollision means a file already occupies the restore location.
	KindRestoreCollision
)

// Kind identifies the category of a failure reported by the history layer.
//
// The zero value is KindUnknown, which every unmapped error reports. Kinds
// describe what happened, never how it should be worded.
type Kind uint8

// Classify maps a history error to its category.
//
// Wrapped errors are matched, so a caller may add context with %w without
// losing the classification.
//
// Parameters:
//   - err: Error returned by a history operation. A nil error is
//     KindUnknown.
//
// Returns:
//   - The matching category, or KindUnknown when nothing matches.
func Classify(err error) Kind {
	switch {
	case errors.Is(err, history.ErrNoHistory):
		return KindNoHistory
	case errors.Is(err, history.ErrAlreadyRestored):
		return KindAlreadyRestored
	case errors.Is(err, history.ErrNotInTrash):
		return KindNotInTrash
	case errors.Is(err, history.ErrRestoreCollision):
		return KindRestoreCollision
	default:
		return KindUnknown
	}
}
