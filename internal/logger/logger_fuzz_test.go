/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package logger

import (
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

// captureLevelNames lists every level name the capture hook can report.
var captureLevelNames = []string{
	captureDebugLevel,
	captureInfoLevel,
	captureWarnLevel,
	captureErrorLevel,
	captureFatalLevel,
	captureOtherLevel,
}

// FuzzCaptureWriterRun verifies the capture hook reports a known level name and the
// message unchanged, for any severity and message.
func FuzzCaptureWriterRun(f *testing.F) {
	f.Add(int8(0), "hello")
	f.Add(int8(1), "")
	f.Add(int8(2), "  spaced message  ")
	f.Add(int8(3), "DBG INF ERR")
	f.Add(int8(4), "line\nbreak")
	f.Add(int8(5), "not a log line")
	f.Add(int8(6), "no level")
	f.Add(int8(7), "disabled")
	f.Add(int8(-1), "trace")
	f.Add(int8(99), "out of range")

	f.Fuzz(func(t *testing.T, raw int8, msg string) {
		var captured []capturedEntry

		w := &captureWriter{
			output:      io.Discard,
			captureFunc: func(level, message string) { captured = append(captured, capturedEntry{level: level, msg: message}) },
		}

		w.Run(nil, zerolog.Level(raw), msg)

		if len(captured) != 1 {
			t.Fatalf("capture recorded %d entries, want 1", len(captured))
		}

		if !slices.Contains(captureLevelNames, captured[0].level) {
			t.Errorf("capture level %q is not a known level name", captured[0].level)
		}

		if captured[0].msg != msg {
			t.Errorf("capture message %q, want %q", captured[0].msg, msg)
		}
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
			if got != DebugLevel {
				t.Errorf("ParseLevel(%q) = %v, want debug", level, got)
			}
		case "info":
			if got != InfoLevel {
				t.Errorf("ParseLevel(%q) = %v, want info", level, got)
			}
		case "warn":
			if got != WarnLevel {
				t.Errorf("ParseLevel(%q) = %v, want warn", level, got)
			}
		case "error":
			if got != ErrorLevel {
				t.Errorf("ParseLevel(%q) = %v, want error", level, got)
			}
		default:
			if got != InfoLevel {
				t.Errorf("ParseLevel(%q) = %v, want info default", level, got)
			}
		}
	})
}
