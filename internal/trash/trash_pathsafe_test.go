package trash

import (
	"strings"
	"testing"
)

// TestTrashPathSafe_MatchesGLib pins the safe set against the characters
// g_filename_to_uri was observed to leave unescaped.
//
// The list is transcribed from running GLib 2.88.3 over every printable ASCII
// byte. It is deliberately not taken from GLib's own header, whose
// G_URI_RESERVED_CHARS_ALLOWED_IN_PATH comment also lists a semicolon, which
// g_filename_to_uri in fact escapes to %3B.
func TestTrashPathSafe_MatchesGLib(t *testing.T) {
	t.Parallel()

	// Verified against g_filename_to_uri on GLib 2.88.3, compared as a set
	// since trashPathSafe is ordered for readability.
	const want = "!$&'()*+,-./0123456789:=@ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz~"

	for _, c := range want {
		if !strings.ContainsRune(trashPathSafe, c) {
			t.Errorf("trashPathSafe is missing %q, which glib leaves unescaped", c)
		}
	}

	for _, c := range trashPathSafe {
		if !strings.ContainsRune(want, c) {
			t.Errorf("trashPathSafe wrongly includes %q, which glib escapes", c)
		}
	}

	// The delimiter cases are the ones that caused wrong restores, so guard them
	// by name rather than relying only on the set comparison.
	for _, c := range []rune{';', '?', '#', '[', ']', ' ', '%', '"', '<', '>', '\\', '^', '`', '{', '|', '}'} {
		if strings.ContainsRune(trashPathSafe, c) {
			t.Errorf("%q must be escaped, but is in the safe set", c)
		}
	}
}
