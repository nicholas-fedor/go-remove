/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cli

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	mockRunner "github.com/nicholas-fedor/go-remove/internal/cli/mocks"
	mockFS "github.com/nicholas-fedor/go-remove/internal/fs/mocks"
	"github.com/nicholas-fedor/go-remove/internal/logger"
)

// TestRunTUI verifies the RunTUI function's behavior under various conditions.
func TestRunTUI(t *testing.T) {
	// go test attaches no terminal, so without this every subtest would take
	// the no-terminal branch and never reach the runner or the binary listing.
	withTerminal(t)

	type args struct {
		dir    string
		config Config
		logger logger.Logger
		fs     *mockFS.MockFS
		runner ProgramRunner
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success with binaries",
			args: args{
				dir:    "/bin",
				config: Config{},
				logger: logger.NopLogger(),
				fs: func() *mockFS.MockFS {
					m := mockFS.NewMockFS(t)
					m.On("ListBinaries", "/bin").Return([]string{"vhs"}, nil)

					return m
				}(),
				runner: func() *mockRunner.MockProgramRunner {
					runner := mockRunner.NewMockProgramRunner(t)
					runner.On("RunProgram", mock.Anything, mock.Anything).
						Return(nil, nil)

					return runner
				}(),
			},
			wantErr: false,
		},
		{
			name: "no binaries",
			args: args{
				dir:    "/bin",
				config: Config{},
				logger: logger.NopLogger(),
				fs: func() *mockFS.MockFS {
					m := mockFS.NewMockFS(t)
					m.On("ListBinaries", "/bin").Return([]string{}, nil)

					return m
				}(),
				runner: func() *mockRunner.MockProgramRunner {
					runner := mockRunner.NewMockProgramRunner(t)
					// The runner stays optional because RunTUI reports the
					// empty listing before it ever starts a program.
					runner.On("RunProgram", mock.Anything, mock.Anything).
						Return(nil, nil).
						Maybe()

					return runner
				}(),
			},
			wantErr: true,
		},
		{
			name: "runner error",
			args: args{
				dir:    "/bin",
				config: Config{},
				logger: logger.NopLogger(),
				fs: func() *mockFS.MockFS {
					m := mockFS.NewMockFS(t)
					m.On("ListBinaries", "/bin").Return([]string{"vhs"}, nil)

					return m
				}(),
				runner: func() *mockRunner.MockProgramRunner {
					runner := mockRunner.NewMockProgramRunner(t)
					runner.On("RunProgram", mock.Anything, mock.Anything).
						Return(nil, errors.New("runner failed"))

					return runner
				}(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RunTUI(
				t.Context(),
				tt.args.dir,
				tt.args.config,
				tt.args.logger,
				tt.args.fs,
				tt.args.runner,
				nil,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunTUI() error = %v, wantErr %v", err, tt.wantErr)
			}

			tt.args.fs.AssertExpectations(t)
		})
	}
}
