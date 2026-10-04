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
	if runtime.GOOS == "windows" {
		t.Skip("XDG_DATA_HOME is not consulted on Windows")
	}

	home := isolateHome(t)
	xdg := filepath.Join(home, "custom", "data")
	t.Setenv("XDG_DATA_HOME", xdg)

	candidates := DataHomeCandidates()

	require.NotEmpty(t, candidates)
	assert.Equal(t, xdg, candidates[0], "an absolute XDG_DATA_HOME must come first")

	// The platform default differs: macOS uses Application Support, the other
	// Unix-like systems use the XDG default.
	fallback := filepath.Join(home, ".local", "share")
	if runtime.GOOS == "darwin" {
		fallback = filepath.Join(home, "Library", "Application Support")
	}

	assert.Contains(t, candidates, fallback,
		"the platform default must remain a candidate")
}

// TestDataHomeCandidates_RelativeXDGIgnored verifies a relative XDG_DATA_HOME is
// treated as unset rather than joined against the working directory.
//
// The XDG base directory specification requires an absolute path. A relative
// value would otherwise place the data directory inside the current directory,
// which is where the tool happens to be running.
func TestDataHomeCandidates_RelativeXDGIgnored(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("XDG_DATA_HOME is not consulted on Windows")
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
	if runtime.GOOS == "windows" {
		t.Skip("XDG_DATA_HOME is not consulted on Windows")
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
func TestWritableDataHome_RejectsRelativeXDG(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("XDG_DATA_HOME is not consulted on Windows")
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
	if runtime.GOOS == "windows" {
		t.Skip("XDG_DATA_HOME is not consulted on Windows")
	}

	isolateHome(t)

	xdg := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdg)

	dir, err := WritableDataHome()
	require.NoError(t, err)
	assert.Equal(t, xdg, dir)
}

// TestWritableDataHome_NoWritableCandidate verifies the search reports failure
// rather than falling back to the directory holding the executable.
//
// That directory is typically a system location such as /usr/local/bin, so a
// database written there is removed by the next package upgrade.
func TestWritableDataHome_NoWritableCandidate(t *testing.T) {
	// Point every candidate at a path that cannot be created, by nesting it
	// beneath a regular file.
	base := t.TempDir()

	blocker := filepath.Join(base, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))

	isolated := isolateHome(t)

	t.Setenv("XDG_DATA_HOME", filepath.Join(blocker, "data"))
	t.Setenv("HOME", filepath.Join(blocker, "home"))
	t.Setenv("USERPROFILE", filepath.Join(blocker, "home"))
	t.Setenv("LOCALAPPDATA", filepath.Join(blocker, "local"))

	dir, err := WritableDataHome()

	require.ErrorIs(t, err, ErrNoWritableStorage)
	assert.Empty(t, dir)
	assert.NotEmpty(t, isolated, "the home directory is still redirected")
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
//
// Every case names GOBIN, GOPATH, HOME and USERPROFILE explicitly, so a value
// inherited from the environment running the suite cannot decide the outcome.
// Both home variables are set because os.UserHomeDir reads $HOME on Unix and
// %USERPROFILE% on Windows, and a case that set only one would resolve the
// real home of whichever platform the suite runs on.
func TestBinDir(t *testing.T) {
	// A literal such as "/gobin" is not absolute on Windows, so the absolute
	// inputs are built from a temporary directory.
	root := t.TempDir()
	goBinAbs := filepath.Join(root, "gobin")
	goPathAbs := filepath.Join(root, "gopath")
	goRootAbs := filepath.Join(root, "goroot")

	home := filepath.Join(root, "home", "u")
	homeGoBin := filepath.Join(home, "go", "bin")

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
			env: map[string]string{
				"GOROOT":      goRootAbs,
				"GOBIN":       goBinAbs,
				"GOPATH":      goPathAbs,
				"HOME":        home,
				"USERPROFILE": home,
			},
			want: filepath.Join(goRootAbs, "bin"),
		},
		{
			name:      "goroot unset is an error",
			useGoroot: true,
			env: map[string]string{
				"GOROOT":      "",
				"GOBIN":       goBinAbs,
				"GOPATH":      goPathAbs,
				"HOME":        home,
				"USERPROFILE": home,
			},
			wantErr: true,
		},
		{
			name: "gobin wins over gopath",
			env: map[string]string{
				"GOBIN":       goBinAbs,
				"GOPATH":      goPathAbs,
				"HOME":        home,
				"USERPROFILE": home,
			},
			want: goBinAbs,
		},
		{
			name: "gopath is used when gobin is unset",
			env: map[string]string{
				"GOBIN":       "",
				"GOPATH":      goPathAbs,
				"HOME":        home,
				"USERPROFILE": home,
			},
			want: filepath.Join(goPathAbs, "bin"),
		},
		{
			name: "home go path is the last resort",
			env: map[string]string{
				"GOBIN":       "",
				"GOPATH":      "",
				"HOME":        home,
				"USERPROFILE": home,
			},
			want: homeGoBin,
		},
		{
			// A relative GOBIN would put the binary directory inside the working
			// directory, so it is skipped exactly as an unset value is.
			name: "relative gobin is skipped",
			env: map[string]string{
				"GOBIN":       filepath.FromSlash("relative/gobin"),
				"GOPATH":      goPathAbs,
				"HOME":        home,
				"USERPROFILE": home,
			},
			want: filepath.Join(goPathAbs, "bin"),
		},
		{
			// The same rule for GOPATH, which falls through to the home
			// directory rather than to GOBIN.
			name: "relative gopath is skipped",
			env: map[string]string{
				"GOBIN":       "",
				"GOPATH":      filepath.FromSlash("relative/gopath"),
				"HOME":        home,
				"USERPROFILE": home,
			},
			want: homeGoBin,
		},
		{
			// Only the first entry is the search root, so a trailing entry must
			// not glue a list separator onto an absolute first entry.
			name: "a gopath list uses its first entry",
			env: map[string]string{
				"GOBIN":       "",
				"GOPATH":      goPathAbs + string(os.PathListSeparator) + goRootAbs,
				"HOME":        home,
				"USERPROFILE": home,
			},
			want: filepath.Join(goPathAbs, "bin"),
		},
		{
			// The first entry is what counts, so a list that starts relative is
			// rejected just as a single relative value is.
			name: "a gopath list with a relative first entry is skipped",
			env: map[string]string{
				"GOBIN": "",
				"GOPATH": filepath.FromSlash("relative/gopath") +
					string(os.PathListSeparator) + goPathAbs,
				"HOME":        home,
				"USERPROFILE": home,
			},
			want: homeGoBin,
		},
		{
			name: "a relative home is rejected",
			env: map[string]string{
				"GOBIN":       "",
				"GOPATH":      "",
				"HOME":        filepath.FromSlash("relative/home"),
				"USERPROFILE": filepath.FromSlash("relative/home"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			got, err := BinDir(tt.useGoroot)

			if tt.wantErr {
				// The GOROOT case names its own sentinel; any other case
				// only requires an error.
				if tt.useGoroot {
					require.ErrorIs(t, err, ErrGorootNotSet)
				} else {
					require.Error(t, err)
				}

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.True(t, filepath.IsAbs(got),
				"the binary directory must be absolute, got %q", got)
		})
	}
}

// TestBinDir_UnresolvableHome verifies an error when neither GOPATH nor the
// home directory can be resolved, rather than a relative path that would be
// resolved against the working directory.
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
