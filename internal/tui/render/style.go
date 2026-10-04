/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package render

// StyleConfig holds TUI appearance settings.
//
// The colors are ANSI 256-color codes rather than resolved values. color
// downsampling happens when the view is written, so a value here is a request
// for a color, not a promise of one.
type StyleConfig struct {
	TitleColor    string
	CursorColor   string
	FooterColor   string
	StatusColor   string
	LogColor      string
	HistoryColor  string
	TrashYesColor string
	TrashNoColor  string
	Cursor        string
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
