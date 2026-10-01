/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package paths resolves the filesystem locations go-remove reads and writes.
//
// Every location resolved here is environment dependent:
//
//   - Application data. DataHomeCandidates applies the XDG base directory
//     specification on Unix, Application Support on macOS, and %LOCALAPPDATA% on
//     Windows. WritableDataHome picks the first candidate that accepts a write,
//     and StoragePath builds the history database location on top of it.
//   - The trash root. TrashRoot applies the same XDG rule, so it resolves
//     against the same base as DataHomeCandidates.
//   - The Go toolchain binary directory. BinDir follows $GOBIN, $GOPATH and
//     $GOROOT.
//
// A relative XDG_DATA_HOME is treated as unset, since joining it would place
// the result inside the working directory. Platform-specific resolution lives in
// build-tagged files, so only one platform's rules are compiled.
package paths
