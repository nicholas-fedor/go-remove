/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package rm defines the rm command, which removes one named binary.
//
// The removal is recorded in history, so the undo command or the restore
// view can bring the binary back.
package rm
