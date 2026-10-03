/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package buildinfo reads build metadata out of Go binaries.
//
// It reports whether a file is a Go binary at all, and for those that are,
// their module path, version, VCS revision and build settings. It also
// computes the SHA256 checksum the history record keeps, which is how a binary
// altered in trash is told apart from an untouched one.
package buildinfo
