/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package paths resolves the filesystem locations go-remove reads and writes.
//
// Three kinds of location are resolved here, all of which are environment
// dependent and were previously resolved in more than one place:
//
//   - Application data. DataHomeCandidates applies the XDG base directory
//     specification on Unix, Application Support on macOS, and %LOCALAPPDATA% on
//     Windows. WritableDataHome picks the first candidate that accepts a write,
//     and StoragePath builds the history database location on top of it.
//   - The trash root. TrashRoot follows the same XDG rule and therefore shares
//     one definition of what a usable data home is.
//   - The Go toolchain binary directory. BinDir follows $GOBIN, $GOPATH and
//     $GOROOT.
//
// Sharing the rules matters because they had already drifted: the command layer
// required an absolute XDG_DATA_HOME while the trash layer applied a different
// fallback. Any platform-specific resolution lives in a build-tagged file so
// that the rules for one platform are compiled only for that platform.
package paths
