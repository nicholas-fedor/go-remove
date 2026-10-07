/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package version holds the build metadata stamped into the go-remove binary.
//
// Release builds set Version, CommitSHA, and BuildTime through -ldflags. A
// build that was never stamped, such as one from go install, reports the
// module version Go recorded instead.
package version
