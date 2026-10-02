/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package render

// StyleConfig holds TUI appearance settings.
//
// The colours are ANSI 256-color codes rather than resolved values. Colour
// downsampling happens when the view is written, so a value here is a request
// for a colour, not a promise of one.
type StyleConfig struct {
	TitleColor    string // ANSI 256-color code for title
	CursorColor   string // ANSI 256-color code for cursor
	FooterColor   string // ANSI 256-color code for footer
	StatusColor   string // ANSI 256-color code for status
	LogColor      string // ANSI 256-color code for log messages
	HistoryColor  string // ANSI 256-color code for history table header
	TrashYesColor string // ANSI 256-color code for "Yes" in trash available column
	TrashNoColor  string // ANSI 256-color code for "No" in trash available column
	Cursor        string // Symbol used for the cursor
}

// DefaultStyleConfig returns the appearance used when none is configured.
//
// Returns:
//   - The default StyleConfig.
func DefaultStyleConfig() StyleConfig {
	return StyleConfig{
		TitleColor:    "39",  // Bright blue
		CursorColor:   "214", // Orange
		FooterColor:   "245", // Light gray
		StatusColor:   "46",  // Lime green
		LogColor:      "240", // Dark gray for subtle log display
		HistoryColor:  "141", // Purple for history header
		TrashYesColor: "46",  // Green for "Yes"
		TrashNoColor:  "196", // Red for "No"
		Cursor:        "❯ ",
	}
}
