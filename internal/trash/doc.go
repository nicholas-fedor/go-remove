/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package trash moves deleted binaries to a per-user trash and puts them back.
//
// Every platform uses a trash directory go-remove owns, laid out to the XDG
// Trash specification with files and info subdirectories and a .trashinfo file
// per entry. Listing, restoring and permanent deletion are all carried out
// against that directory, because the platform trash offers no interface
// go-remove could drive them through.
package trash
