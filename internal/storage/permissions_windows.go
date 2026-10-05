//go:build windows

/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package storage

// withRestrictiveUmask runs fn unchanged.
//
// Windows has no file creation mask, and the NTFS ACL it applies to a new file
// comes from the directory it is created in, which Badger already restricts to
// the owner.
//
// Parameters:
//   - fn: The work to run, which is expected to create the database files.
//
// Returns:
//   - Whatever fn returns.
func withRestrictiveUmask(fn func() error) error {
	return fn()
}
