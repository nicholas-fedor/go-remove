/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package paths resolves the environment-dependent locations go-remove uses:
// the application data directory, the trash root, and the Go toolchain binary
// directory. Platform differences are confined to build-tagged files.
package paths
