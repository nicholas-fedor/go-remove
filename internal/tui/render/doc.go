/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package render builds the TUI's view output.
//
// Everything here is a pure function of the state it is given. Widths are
// measured in terminal cells rather than bytes or runes, so a name containing
// CJK or an emoji occupies the width it actually renders at.
package render
