/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package logger

import (
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

// FuzzCaptureLogMessage verifies log-line parsing does not panic.
func FuzzCaptureLogMessage(f *testing.F) {
	f.Add("2026-01-01T00:00:00Z INF hello")
	f.Add("DBG only")
	f.Add("")
	f.Add("2026-01-01T00:00:00Z WRN path=/tmp/file")
	f.Add("not a log line")

	f.Fuzz(func(t *testing.T, line string) {
		w := &captureWriter{
			captureEnabled: true,
			captureFunc:    func(string, string) {},
		}
		w.captureLogMessage(line)
	})
}

// FuzzParseLevel verifies ParseLevel maps known names and defaults unknown values to info.
func FuzzParseLevel(f *testing.F) {
	f.Add("debug")
	f.Add("INFO")
	f.Add("Warn")
	f.Add("error")
	f.Add("")
	f.Add("trace")
	f.Add(" DEBUG ")
	f.Add("DeBuG")
	f.Add("fatal")
	f.Add("info\n")

	f.Fuzz(func(t *testing.T, level string) {
		got := ParseLevel(level)

		switch strings.ToLower(level) {
		case "debug":
			if got != zerolog.DebugLevel {
				t.Errorf("ParseLevel(%q) = %v, want debug", level, got)
			}
		case "info":
			if got != zerolog.InfoLevel {
				t.Errorf("ParseLevel(%q) = %v, want info", level, got)
			}
		case "warn":
			if got != zerolog.WarnLevel {
				t.Errorf("ParseLevel(%q) = %v, want warn", level, got)
			}
		case "error":
			if got != zerolog.ErrorLevel {
				t.Errorf("ParseLevel(%q) = %v, want error", level, got)
			}
		default:
			if got != zerolog.InfoLevel {
				t.Errorf("ParseLevel(%q) = %v, want info default", level, got)
			}
		}
	})
}
