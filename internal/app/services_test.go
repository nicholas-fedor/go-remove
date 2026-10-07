/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package app

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/go-remove/internal/cli/flags"
	fsmocks "github.com/nicholas-fedor/go-remove/internal/fs/mocks"
	"github.com/nicholas-fedor/go-remove/internal/history"
	historymocks "github.com/nicholas-fedor/go-remove/internal/history/mocks"
	"github.com/nicholas-fedor/go-remove/internal/logger"
	loggermocks "github.com/nicholas-fedor/go-remove/internal/logger/mocks"
	"github.com/nicholas-fedor/go-remove/internal/tui"
	"github.com/nicholas-fedor/go-remove/internal/tui/models"
)

// Errors the fakes return.
var (
	// errBinDir is a bin directory resolution failure.
	errBinDir = errors.New("GOROOT is not set")

	// errLogger is a logger construction failure.
	errLogger = errors.New("logger failed")

	// errOpen is a history open failure.
	errOpen = errors.New("database locked")

	// errOperation is a failure of the work itself.
	errOperation = errors.New("operation failed")
)

// testSettings are the settings passed to every service call.
var testSettings = flags.Settings{
	LogLevel: "warn",
	Level:    logger.WarnLevel,
	Verbose:  false,
	Goroot:   true,
}

// servicesFixture holds services wired to mocks, plus what they recorded.
type servicesFixture struct {
	// svc is the services under test.
	svc *services

	// fs backs bin directory resolution.
	fs *fsmocks.MockFS

	// manager is the history manager openHistory returns.
	manager *historymocks.MockManager

	// log is the logger both logger factories return.
	log *loggermocks.MockLogger

	// tuiOpts records the options runTUI was called with.
	tuiOpts *tui.Options

	// captured records whether the capture logger factory was used.
	captured bool
}

// newServicesFixture returns services whose dependencies are all mocks.
//
// The logger accepts any level and any warning. Tests that care about the
// close warning set their own expectation first.
//
// Parameters:
//   - t: the running test.
//
// Returns:
//   - *servicesFixture: the services and their mocks.
func newServicesFixture(t *testing.T) *servicesFixture {
	t.Helper()

	f := &servicesFixture{
		svc:      nil,
		fs:       fsmocks.NewMockFS(t),
		manager:  historymocks.NewMockManager(t),
		log:      loggermocks.NewMockLogger(t),
		tuiOpts:  nil,
		captured: false,
	}

	f.svc = &services{
		fs: f.fs,
		newLogger: func() (logger.Logger, error) {
			return f.log, nil
		},
		newCaptureLogger: func() (logger.Logger, error) {
			f.captured = true

			return f.log, nil
		},
		openHistory: func(logger.Logger) (history.Manager, error) {
			return f.manager, nil
		},
		runTUI: func(_ context.Context, opts tui.Options) error {
			f.tuiOpts = &opts

			return nil
		},
	}

	return f
}

// expectOpen sets the expectations every successful open meets: the level is
// applied and the manager is closed.
func (f *servicesFixture) expectOpen() {
	f.log.EXPECT().Level(logger.WarnLevel).Once()
	f.manager.EXPECT().Close().Return(nil).Once()
}

// TestNewServices verifies the production services have every dependency.
func TestNewServices(t *testing.T) {
	t.Parallel()

	svc := newServices()

	assert.NotNil(t, svc.fs)
	assert.NotNil(t, svc.newLogger)
	assert.NotNil(t, svc.newCaptureLogger)
	assert.NotNil(t, svc.openHistory)
	assert.NotNil(t, svc.runTUI)
}

// TestNewCaptureLogger verifies the capturing logger builds.
func TestNewCaptureLogger(t *testing.T) {
	t.Parallel()

	log, err := newCaptureLogger()

	require.NoError(t, err)
	assert.NotNil(t, log)
}

// TestServices_Remove verifies the binary is resolved in the bin directory and
// recorded in history, and the manager is closed.
func TestServices_Remove(t *testing.T) {
	t.Parallel()

	f := newServicesFixture(t)
	f.expectOpen()
	f.fs.EXPECT().DetermineBinDir(true).Return("/go/bin", nil)
	f.fs.EXPECT().AdjustBinaryPath("/go/bin", "vhs").Return("/go/bin/vhs")
	f.manager.EXPECT().RecordDeletion(mock.Anything, "/go/bin/vhs").Return(nil, nil)

	require.NoError(t, f.svc.Remove(t.Context(), "vhs", testSettings))
	assert.False(t, f.captured, "a direct command does not need the capturing logger")
}

// TestServices_Remove_BinDirError verifies a bad bin directory fails before
// the logger or the history database is opened.
func TestServices_Remove_BinDirError(t *testing.T) {
	t.Parallel()

	f := newServicesFixture(t)
	f.fs.EXPECT().DetermineBinDir(true).Return("", errBinDir)
	f.svc.openHistory = func(logger.Logger) (history.Manager, error) {
		t.Fatal("the history database must not be opened")

		return nil, errOpen
	}

	err := f.svc.Remove(t.Context(), "vhs", testSettings)

	require.ErrorIs(t, err, errBinDir)
	assert.Contains(t, err.Error(), "determining binary directory")
}

// TestServices_Remove_RecordError verifies a failed removal is reported and
// the manager is still closed.
func TestServices_Remove_RecordError(t *testing.T) {
	t.Parallel()

	f := newServicesFixture(t)
	f.expectOpen()
	f.fs.EXPECT().DetermineBinDir(true).Return("/go/bin", nil)
	f.fs.EXPECT().AdjustBinaryPath("/go/bin", "vhs").Return("/go/bin/vhs")
	f.manager.EXPECT().RecordDeletion(mock.Anything, "/go/bin/vhs").Return(nil, errOperation)

	err := f.svc.Remove(t.Context(), "vhs", testSettings)

	require.ErrorIs(t, err, errOperation)
	assert.Contains(t, err.Error(), "recording deletion")
}

// TestServices_OpenErrors verifies logger and history failures are reported
// by every service.
func TestServices_OpenErrors(t *testing.T) {
	t.Parallel()

	calls := map[string]func(*services, context.Context) error{
		"remove": func(svc *services, ctx context.Context) error {
			return svc.Remove(ctx, "vhs", testSettings)
		},
		"undo": func(svc *services, ctx context.Context) error {
			_, err := svc.Undo(ctx, testSettings)

			return err
		},
		"browse": func(svc *services, ctx context.Context) error {
			return svc.Browse(ctx, testSettings)
		},
	}

	for name, call := range calls {
		t.Run(name+"/logger", func(t *testing.T) {
			t.Parallel()

			f := newServicesFixture(t)
			f.fs.EXPECT().DetermineBinDir(true).Return("/go/bin", nil).Maybe()

			failing := func() (logger.Logger, error) { return nil, errLogger }
			f.svc.newLogger = failing
			f.svc.newCaptureLogger = failing

			err := call(f.svc, t.Context())

			require.ErrorIs(t, err, errLogger)
			assert.Contains(t, err.Error(), "initializing logger")
		})

		t.Run(name+"/history", func(t *testing.T) {
			t.Parallel()

			f := newServicesFixture(t)
			f.log.EXPECT().Level(logger.WarnLevel).Once()
			f.fs.EXPECT().DetermineBinDir(true).Return("/go/bin", nil).Maybe()
			f.svc.openHistory = func(logger.Logger) (history.Manager, error) {
				return nil, errOpen
			}

			err := call(f.svc, t.Context())

			require.ErrorIs(t, err, errOpen)
			assert.Contains(t, err.Error(), "initializing history manager")
		})
	}
}

// TestServices_CloseFailureIsLogged verifies a failed close is a warning, not
// an error, since the work itself succeeded.
func TestServices_CloseFailureIsLogged(t *testing.T) {
	t.Parallel()

	f := newServicesFixture(t)
	f.log.EXPECT().Level(logger.WarnLevel).Once()
	f.log.EXPECT().Warn("Failed to close history manager", mock.Anything).Once()
	f.manager.EXPECT().Close().Return(errOperation).Once()
	f.manager.EXPECT().UndoMostRecent(mock.Anything).Return(&history.RestoreResult{
		EntryID:    "1",
		BinaryName: "vhs",
		RestoredTo: "/go/bin/vhs",
		FromTrash:  true,
		ModulePath: "",
		Version:    "",
	}, nil)

	_, err := f.svc.Undo(t.Context(), testSettings)

	require.NoError(t, err)
}

// TestServices_Undo verifies the result of the most recent restore is
// returned, and a failure keeps its cause for classification.
func TestServices_Undo(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		want := &history.RestoreResult{
			EntryID:    "1",
			BinaryName: "vhs",
			RestoredTo: "/go/bin/vhs",
			FromTrash:  true,
			ModulePath: "github.com/charmbracelet/vhs",
			Version:    "v0.9.0",
		}

		f := newServicesFixture(t)
		f.expectOpen()
		f.manager.EXPECT().UndoMostRecent(mock.Anything).Return(want, nil)

		got, err := f.svc.Undo(t.Context(), testSettings)

		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("failure", func(t *testing.T) {
		t.Parallel()

		f := newServicesFixture(t)
		f.expectOpen()
		f.manager.EXPECT().UndoMostRecent(mock.Anything).Return(nil, history.ErrNoHistory)

		_, err := f.svc.Undo(t.Context(), testSettings)

		require.ErrorIs(t, err, history.ErrNoHistory,
			"the undo command classifies the cause, so it must stay in the chain")
	})
}

// TestServices_Browse verifies both TUI entry points pass the bin directory,
// the capturing logger, the history manager, and the starting view.
func TestServices_Browse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		call        func(*services, context.Context) error
		name        string
		wantRestore bool
	}{
		{
			name: "picker",
			call: func(svc *services, ctx context.Context) error {
				return svc.Browse(ctx, testSettings)
			},
			wantRestore: false,
		},
		{
			name: "history",
			call: func(svc *services, ctx context.Context) error {
				return svc.BrowseHistory(ctx, testSettings)
			},
			wantRestore: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newServicesFixture(t)
			f.expectOpen()
			f.fs.EXPECT().DetermineBinDir(true).Return("/go/bin", nil)

			require.NoError(t, tt.call(f.svc, t.Context()))
			require.NotNil(t, f.tuiOpts)

			assert.True(t, f.captured, "the TUI displays captured log output")
			assert.Equal(t, "/go/bin", f.tuiOpts.Dir)
			assert.Equal(t, models.Config{
				Verbose:     false,
				LogLevel:    "warn",
				RestoreMode: tt.wantRestore,
			}, f.tuiOpts.Config)
			assert.Same(t, f.log, f.tuiOpts.Logger)
			assert.Same(t, f.fs, f.tuiOpts.FS)
			assert.Same(t, f.manager, f.tuiOpts.HistoryManager)
		})
	}
}

// TestServices_Browse_Errors verifies bin directory and TUI failures are
// reported.
func TestServices_Browse_Errors(t *testing.T) {
	t.Parallel()

	t.Run("bin directory", func(t *testing.T) {
		t.Parallel()

		f := newServicesFixture(t)
		f.fs.EXPECT().DetermineBinDir(true).Return("", errBinDir)

		require.ErrorIs(t, f.svc.Browse(t.Context(), testSettings), errBinDir)
		assert.Nil(t, f.tuiOpts, "the TUI must not start")
	})

	t.Run("tui", func(t *testing.T) {
		t.Parallel()

		f := newServicesFixture(t)
		f.expectOpen()
		f.fs.EXPECT().DetermineBinDir(true).Return("/go/bin", nil)
		f.svc.runTUI = func(context.Context, tui.Options) error { return errOperation }

		err := f.svc.Browse(t.Context(), testSettings)

		require.ErrorIs(t, err, errOperation)
		assert.Contains(t, err.Error(), "running TUI")
	})
}
