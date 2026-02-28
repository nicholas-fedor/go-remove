/*
Copyright © 2025 Nicholas Fedor <nick@nickfedor.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/

package fs

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"

	"github.com/rs/zerolog"

	"github.com/nicholas-fedor/go-remove/internal/logger"
	"github.com/nicholas-fedor/go-remove/internal/logger/mocks"
)

// nopLogger creates a mock logger for testing.
func nopLogger(t *testing.T) logger.Logger {
	t.Helper()

	return mocks.NewMockLogger(t)
}

// TestNewRealFS verifies the NewRealFS function's instance creation.
func TestNewRealFS(t *testing.T) {
	tests := []struct {
		name string
		want FS
	}{
		{
			name: "creates RealFS",
			want: &RealFS{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRealFS()
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Errorf("NewRealFS() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRealFS_DetermineBinDir verifies the DetermineBinDir method's directory resolution.
func TestRealFS_DetermineBinDir(t *testing.T) {
	type args struct {
		useGoroot bool
	}

	tests := []struct {
		name    string
		r       *RealFS
		args    args
		env     map[string]string
		want    string
		wantErr bool
	}{
		{
			name:    "useGoroot with GOROOT set",
			r:       &RealFS{},
			args:    args{useGoroot: true},
			env:     map[string]string{"GOROOT": filepath.FromSlash("/go")},
			want:    filepath.FromSlash("/go/bin"),
			wantErr: false,
		},
		{
			name:    "useGoroot with GOROOT unset",
			r:       &RealFS{},
			args:    args{useGoroot: true},
			env:     map[string]string{"GOROOT": ""}, // Explicitly unset GOROOT
			want:    "",
			wantErr: true,
		},
		{
			name:    "use GOBIN",
			r:       &RealFS{},
			args:    args{useGoroot: false},
			env:     map[string]string{"GOBIN": filepath.FromSlash("/custom/bin")},
			want:    filepath.FromSlash("/custom/bin"),
			wantErr: false,
		},
		{
			name:    "use GOPATH/bin when GOBIN unset",
			r:       &RealFS{},
			args:    args{useGoroot: false},
			env:     map[string]string{"GOPATH": filepath.FromSlash("/gopath"), "GOBIN": ""},
			want:    filepath.FromSlash("/gopath/bin"),
			wantErr: false,
		},
		{
			name:    "use default ~/go/bin when GOBIN and GOPATH unset",
			r:       &RealFS{},
			args:    args{useGoroot: false},
			env:     map[string]string{"GOBIN": "", "GOPATH": ""},
			want:    filepath.Join(os.Getenv("HOME"), "go", "bin"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear existing env vars that might interfere.
			os.Unsetenv("GOROOT")
			os.Unsetenv("GOBIN")
			os.Unsetenv("GOPATH")
			// Set environment variables for the test case.
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			// Adjust expected path for the platform's HOME directory.
			if tt.name == "use default ~/go/bin when GOBIN and GOPATH unset" {
				home := os.Getenv("HOME")
				if runtime.GOOS == windowsOS {
					if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
						home = userProfile
					}
				}

				tt.want = filepath.Join(home, "go", "bin")
			}

			got, err := tt.r.DetermineBinDir(tt.args.useGoroot)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetermineBinDir() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if got != tt.want {
				t.Errorf("DetermineBinDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRealFS_AdjustBinaryPath verifies the AdjustBinaryPath method's path construction.
func TestRealFS_AdjustBinaryPath(t *testing.T) {
	type args struct {
		dir    string
		binary string
	}

	tests := []struct {
		name string
		r    *RealFS
		args args
		want string
	}{
		{
			name: "basic path",
			r:    &RealFS{},
			args: args{dir: filepath.FromSlash("/bin"), binary: "tool"},
			want: filepath.FromSlash("/bin/tool") + func() string {
				if runtime.GOOS == windowsOS {
					return windowsExt
				}

				return ""
			}(),
		},
		{
			name: "empty binary",
			r:    &RealFS{},
			args: args{dir: filepath.FromSlash("/bin"), binary: ""},
			want: filepath.FromSlash("/bin"),
		},
	}
	if runtime.GOOS == windowsOS {
		tests = append(tests, struct {
			name string
			r    *RealFS
			args args
			want string
		}{
			name: "windows adds .exe",
			r:    &RealFS{},
			args: args{dir: filepath.FromSlash("/bin"), binary: "tool"},
			want: filepath.FromSlash("/bin/tool.exe"),
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.AdjustBinaryPath(tt.args.dir, tt.args.binary); got != tt.want {
				t.Errorf("AdjustBinaryPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRealFS_RemoveBinary verifies the RemoveBinary method's file removal behavior.
func TestRealFS_RemoveBinary(t *testing.T) {
	type args struct {
		binaryPath string
		name       string
		verbose    bool
		logger     func() logger.Logger // Factory function to create logger
	}

	tests := []struct {
		name    string
		r       *RealFS
		args    args
		setup   func() string // Returns temp file path
		wantErr bool
	}{
		{
			name: "remove existing binary",
			r:    &RealFS{},
			args: args{
				name:    "testbin",
				verbose: false,
				logger: func() logger.Logger {
					return nopLogger(t)
				},
			},
			setup: func() string {
				tmpDir := t.TempDir()
				tmpFile := filepath.Join(tmpDir, "testbin")
				os.WriteFile(tmpFile, []byte("test"), 0o755)

				return tmpFile
			},
			wantErr: false,
		},
		{
			name: "remove non-existent binary",
			r:    &RealFS{},
			args: args{
				binaryPath: "/nonexistent/testbin",
				name:       "testbin",
				verbose:    false,
				logger: func() logger.Logger {
					return nopLogger(t)
				},
			},
			wantErr: true,
		},
		{
			name: "verbose logging",
			r:    &RealFS{},
			args: args{
				name:    "testbin",
				verbose: true,
				logger: func() logger.Logger {
					// Create mock with expectations for verbose logging.
					log := mocks.NewMockLogger(t)
					zl := zerolog.New(nil).With().Logger()
					dummyEvent := zl.Debug()
					log.On("Debug").Return(dummyEvent)
					log.On("Info").Return(dummyEvent)

					return log
				},
			},
			setup: func() string {
				tmpDir := t.TempDir()
				tmpFile := filepath.Join(tmpDir, "testbin")
				os.WriteFile(tmpFile, []byte("test"), 0o755)

				return tmpFile
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up temporary file if provided.
			if tt.setup != nil {
				tt.args.binaryPath = tt.setup()
			}

			// Execute the RemoveBinary method and verify error behavior.
			err := tt.r.RemoveBinary(
				tt.args.binaryPath,
				tt.args.name,
				tt.args.verbose,
				tt.args.logger(), // Call factory to get logger
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveBinary() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRealFS_RemoveBinary_VerboseLogging verifies verbose logging behavior.
func TestRealFS_RemoveBinary_VerboseLogging(t *testing.T) {
	log := mocks.NewMockLogger(t)

	// Create a dummy zerolog logger and event for return values.
	zl := zerolog.New(nil).With().Logger()
	dummyEvent := zl.Debug()

	// Set up expectations for verbose logging calls.
	log.On("Debug").Return(dummyEvent)
	log.On("Info").Return(dummyEvent)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "testbin")
	os.WriteFile(tmpFile, []byte("test"), 0o755)

	fs := &RealFS{}

	err := fs.RemoveBinary(tmpFile, "testbin", true, log)
	if err != nil {
		t.Errorf("RemoveBinary() unexpected error = %v", err)
	}

	// Verify that Debug and Info were called.
	log.AssertCalled(t, "Debug")
	log.AssertCalled(t, "Info")
}

// TestRealFS_ListBinaries verifies the ListBinaries method's directory listing.
func TestRealFS_ListBinaries(t *testing.T) {
	type args struct {
		dir string
	}

	tests := []struct {
		name  string
		r     *RealFS
		args  args
		setup func() string // Returns temp dir
		want  []string
	}{
		{
			name: "list binaries",
			r:    &RealFS{},
			args: args{},
			setup: func() string {
				tmpDir := t.TempDir()

				ext := ""
				if runtime.GOOS == windowsOS {
					ext = windowsExt
				}

				os.WriteFile(filepath.Join(tmpDir, "tool2"+ext), []byte("test"), 0o755)
				os.WriteFile(filepath.Join(tmpDir, "tool1"+ext), []byte("test"), 0o755)

				if runtime.GOOS == windowsOS {
					os.WriteFile(filepath.Join(tmpDir, "tool3.exe"), []byte("test"), 0o755)
				}

				os.Mkdir(filepath.Join(tmpDir, "dir"), 0o755)

				return tmpDir
			},
			want: func() []string {
				if runtime.GOOS == windowsOS {
					return []string{"tool1.exe", "tool2.exe", "tool3.exe"}
				}

				return []string{"tool1", "tool2"}
			}(),
		},
		{
			name: "empty dir",
			r:    &RealFS{},
			args: args{},
			setup: func() string {
				return t.TempDir()
			},
			want: []string{},
		},
		{
			name: "non-existent dir",
			r:    &RealFS{},
			args: args{dir: "/nonexistent"},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up temporary directory if provided.
			if tt.setup != nil {
				tt.args.dir = tt.setup()
			}

			got := tt.r.ListBinaries(tt.args.dir)

			if tt.name == "list binaries" {
				sortedGot := make([]string, len(got))
				copy(sortedGot, got)
				sort.Strings(sortedGot)
				got = sortedGot
			}

			if tt.name == "empty dir" {
				files, err := os.ReadDir(tt.args.dir)
				t.Logf("Directory %s contents: %v, err: %v", tt.args.dir, files, err)
				t.Logf("Got: %v (len: %d, cap: %d), Want: %v (len: %d, cap: %d)",
					got, len(got), cap(got), tt.want, len(tt.want), cap(tt.want))

				if len(got) == 0 && len(tt.want) == 0 {
					return
				}
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListBinaries() = %v, want %v", got, tt.want)
			}
		})
	}
}
