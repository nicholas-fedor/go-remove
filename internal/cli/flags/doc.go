/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package flags defines the go-remove command-line flags and validates them.
//
// global.go holds the persistent flags every command shares. Each command
// with flags or arguments of its own has a file named after it, such as
// rm.go. Commands read resolved values from this package and never query
// cobra's flag sets directly.
package flags
