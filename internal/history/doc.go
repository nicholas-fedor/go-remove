/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package history records binary deletions and puts them back.
//
// A deletion is written to storage with the metadata needed to find and
// restore the binary later. Restoring and undoing return a binary to its
// original path, coordinating the trash and storage packages to do it.
package history
