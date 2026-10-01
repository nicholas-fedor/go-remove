/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isolateHome points HOME and USERPROFILE at a temporary directory so a test
// never reads or writes the real user home.
//
// Parameters:
//   - t: test handle supplying the temporary directory.
func isolateHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	return home
}

// TestDataHomeCandidates_AbsoluteXDG verifies an absolute XDG_DATA_HOME is
// preferred on Unix-like systems.
func TestDataHomeCandidates_AbsoluteXDG(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		t.Skip("XDG_DATA_HOME is not consulted on this platform")
	}

	home := isolateHome(t)
	xdg := filepath.Join(home, "custom", "data")
	t.Setenv("XDG_DATA_HOME", xdg)

	candidates := DataHomeCandidates()

	require.NotEmpty(t, candidates)
	assert.Equal(t, xdg, candidates[0], "an absolute XDG_DATA_HOME must come first")
	assert.Contains(t, candidates, filepath.Join(home, ".local", "share"),
		"the XDG default must remain a candidate")
}

// TestDataHomeCandidates_RelativeXDGIgnored verifies a relative XDG_DATA_HOME is
// treated as unset rather than joined against the working directory.
//
// The XDG base directory specification requires an absolute path. A relative
// value would otherwise place the data directory inside the current directory,
// which is where the tool happens to be running.
func TestDataHomeCandidates_RelativeXDGIgnored(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		t.Skip("XDG_DATA_HOME is not consulted on this platform")
	}

	isolateHome(t)
	t.Setenv("XDG_DATA_HOME", "relative/data")

	for _, candidate := range DataHomeCandidates() {
		assert.NotContains(t, candidate, filepath.Join("relative", "data"),
			"a relative XDG_DATA_HOME must never be used")
	}
}

// TestDataHomeCandidates_EmptyXDGIgnored verifies an empty XDG_DATA_HOME is
// treated as unset.
func TestDataHomeCandidates_EmptyXDGIgnored(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		t.Skip("XDG_DATA_HOME is not consulted on this platform")
	}

	isolateHome(t)
	t.Setenv("XDG_DATA_HOME", "")

	for _, candidate := range DataHomeCandidates() {
		assert.NotEmpty(t, candidate, "an empty candidate must never be returned")
	}
}

// TestDataHomeCandidates_NoEmptyEntries verifies no platform returns an empty
// candidate, since WritableDataHome would have to skip it.
func TestDataHomeCandidates_NoEmptyEntries(t *testing.T) {
	isolateHome(t)

	for _, candidate := range DataHomeCandidates() {
		assert.NotEmpty(t, candidate)
	}
}

// TestWritableDataHome_RejectsRelativeXDG verifies a relative XDG_DATA_HOME is
// ignored and the resolved data home is absolute.
//
// Moved from cmd/root_test.go, which could no longer reach the logic once it
// moved here.
func TestWritableDataHome_RejectsRelativeXDG(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		t.Skip("XDG_DATA_HOME is not consulted on this platform")
	}

	isolateHome(t)
	t.Setenv("XDG_DATA_HOME", "relative/data")

	dir, err := WritableDataHome()
	require.NoError(t, err)

	require.NotEmpty(t, dir)
	assert.True(t, filepath.IsAbs(dir), "the data home must be absolute, got %q", dir)
	assert.NotContains(t, dir, filepath.Join("relative", "data"),
		"a relative XDG_DATA_HOME must be ignored")
}

// TestWritableDataHome_PrefersWritableCandidate verifies the first writable
// candidate wins, and that an absolute XDG_DATA_HOME is the one selected.
func TestWritableDataHome_PrefersWritableCandidate(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		t.Skip("XDG_DATA_HOME is not consulted on this platform")
	}

	isolateHome(t)

	xdg := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdg)

	dir, err := WritableDataHome()
	require.NoError(t, err)
	assert.Equal(t, xdg, dir)
}

// TestIsDirWritable verifies writability probing, including that the probe
// leaves no temporary file behind.
func TestIsDirWritable(t *testing.T) {
	base := t.TempDir()

	t.Run("creates a missing directory", func(t *testing.T) {
		target := filepath.Join(base, "missing", "nested")

		assert.True(t, IsDirWritable(target))
		assert.DirExists(t, target)
	})

	t.Run("leaves no probe file behind", func(t *testing.T) {
		target := filepath.Join(base, "clean")

		require.True(t, IsDirWritable(target))

		entries, err := os.ReadDir(target)
		require.NoError(t, err)
		assert.Empty(t, entries, "the writability probe must clean up after itself")
	})

	t.Run("reports false for a path under a file", func(t *testing.T) {
		file := filepath.Join(base, "a-file")
		require.NoError(t, os.WriteFile(file, []byte("x"), 0o600))

		assert.False(t, IsDirWritable(filepath.Join(file, "child")),
			"a directory cannot be created beneath a regular file")
	})
}

// TestStoragePath verifies the history database path is built beneath the data
// home.
//
// Moved from cmd/root_test.go.
func TestStoragePath(t *testing.T) {
	switch runtime.GOOS {
	case "windows":
		// Windows reads LOCALAPPDATA directly, so redirecting it is what keeps
		// the test out of the real application data directory.
		t.Setenv("LOCALAPPDATA", t.TempDir())
	case "darwin":
		// macOS ignores LOCALAPPDATA and derives the location from the home
		// directory, so isolateHome below is the only thing that redirects it.
	default:
		t.Setenv("XDG_DATA_HOME", t.TempDir())
	}

	isolateHome(t)

	path, err := StoragePath()
	require.NoError(t, err)
	require.NotEmpty(t, path)

	assert.Contains(t, path, "go-remove",
		"the storage path must live under the application directory, got %q", path)
	assert.Contains(t, path, "history.badger",
		"the storage path must name the database, got %q", path)
}

// TestBinDir verifies the Go toolchain lookup order.
func TestBinDir(t *testing.T) {
	tests := []struct {
		name      string
		useGoroot bool
		env       map[string]string
		want      string
		wantErr   bool
	}{
		{
			name:      "goroot is used when requested",
			useGoroot: true,
			env:       map[string]string{"GOROOT": "/opt/go", "GOBIN": "/ignored/bin"},
			want:      filepath.FromSlash("/opt/go/bin"),
		},
		{
			name:      "goroot unset is an error",
			useGoroot: true,
			env:       map[string]string{"GOROOT": ""},
			wantErr:   true,
		},
		{
			name: "gobin wins over gopath",
			env:  map[string]string{"GOBIN": "/gobin", "GOPATH": "/gopath", "HOME": "/home/u"},
			want: "/gobin",
		},
		{
			name: "gopath is used when gobin is unset",
			env:  map[string]string{"GOBIN": "", "GOPATH": "/gopath", "HOME": "/home/u"},
			want: filepath.FromSlash("/gopath/bin"),
		},
		{
			name: "home go path is the last resort",
			env:  map[string]string{"GOBIN": "", "GOPATH": "", "HOME": "/home/u"},
			want: filepath.FromSlash("/home/u/go/bin"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			got, err := BinDir(tt.useGoroot)

			if tt.wantErr {
				require.ErrorIs(t, err, ErrGorootNotSet)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestBinDir_UnresolvableHome verifies an error is returned when neither
// GOPATH nor the home directory can be resolved.
//
// The previous implementation read $HOME and %USERPROFILE% directly, so an
// unset home produced filepath.Join("", "go") and a relative "go/bin" result
// that resolved against the working directory.
func TestBinDir_UnresolvableHome(t *testing.T) {
	t.Setenv("GOBIN", "")
	t.Setenv("GOPATH", "")
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")

	got, err := BinDir(false)

	require.ErrorContains(t, err, "home directory")
	assert.Empty(t, got, "no path may be returned when resolution fails")
}

// FuzzIsDirWritable verifies IsDirWritable does not panic on arbitrary paths.
//
// Moved from cmd/cmd_fuzz_test.go.
func FuzzIsDirWritable(f *testing.F) {
	f.Add("")
	f.Add("subdir")
	f.Add("missing")
	f.Add(".")
	f.Add("..")
	f.Add("a/b")

	f.Fuzz(func(t *testing.T, rel string) {
		dir := t.TempDir()
		_ = IsDirWritable(dir)

		name := filepath.Base(rel)
		if name == "" || name == "." || name == ".." {
			return
		}

		_ = IsDirWritable(filepath.Join(dir, name))
	})
}
