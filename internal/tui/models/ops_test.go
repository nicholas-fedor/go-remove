/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	tea "charm.land/bubbletea/v2"

	mockFS "github.com/nicholas-fedor/go-remove/internal/fs/mocks"
	"github.com/nicholas-fedor/go-remove/internal/logger"
	"github.com/nicholas-fedor/go-remove/internal/tui/render"
)

const (
	// opStartTimeout bounds a wait for work the test has already started, so it
	// only fires when a goroutine is not making progress.
	opStartTimeout = 30 * time.Second

	// blockedGrace is how long a correct wait is given to stay blocked. It is not
	// a timing assumption: a wait covering every in-flight operation cannot
	// return while the test is still holding one of them.
	blockedGrace = 100 * time.Millisecond
)

// newLifecycleModel builds the smallest model that can start an operation, so a
// lifecycle test does not have to describe the whole view.
func newLifecycleModel() *Model {
	return &Model{
		choices:       []string{"vhs"},
		dir:           "/bin",
		cols:          1,
		rows:          1,
		fs:            fuzzFS{},
		logger:        logger.NopLogger(),
		logs:          make([]string, 0, maxLogLines),
		logChan:       make(chan LogMsg, maxLogLines),
		mode:          modeBinaries,
		width:         80,
		height:        24,
		sortAscending: true,
		styles:        render.DefaultStyleConfig(),
	}
}

// awaitSignal waits for a channel the test is holding work on, failing the test
// when the work never reaches it.
func awaitSignal(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(opStartTimeout):
		t.Fatal(message)
	}
}

// drainOperation runs the command an operation returned and applies its result,
// so a test can assert the outcome of work that now happens outside Update.
//
// Parameters:
//   - t: Test that owns the model.
//   - m: The model that produced the command.
//   - cmd: Command returned alongside the updated model.
//
// Returns:
//   - The model after the result has been applied.
func drainOperation(t *testing.T, m *Model, cmd tea.Cmd) *Model {
	t.Helper()

	if cmd == nil {
		return m
	}

	msg := cmd()

	// Commands that are not an operation, such as a quit, need no draining.
	switch msg.(type) {
	case opResultMsg:
	default:
		return m
	}

	result := msg.(opResultMsg)
	if result.op == nil {
		return m
	}

	updated, _ := m.Update(result)
	got, ok := updated.(*Model)
	require.True(t, ok)

	return got
}

// TestCtrlC_KeepsInterruptedStatus verifies the status an interrupt showed is
// not replaced when the operation it stopped later reports success.
//
// The work is not necessarily context aware, so it can run to completion after
// the user has been told they stopped it.
func TestCtrlC_KeepsInterruptedStatus(t *testing.T) {
	t.Parallel()

	fsMock := mockFS.NewMockFS(t)
	fsMock.On("AdjustBinaryPath", "/bin", "vhs").Return("/bin/vhs")
	fsMock.On("RemoveBinary", "/bin/vhs", "vhs", false, mock.Anything).Return(nil)
	fsMock.On("ListBinaries", "/bin").Return([]string{"vhs"}, nil).Maybe()

	m := &Model{
		choices:       []string{"vhs"},
		dir:           "/bin",
		cols:          1,
		rows:          1,
		fs:            fsMock,
		logger:        logger.NopLogger(),
		logs:          make([]string, 0, maxLogLines),
		logChan:       make(chan LogMsg, maxLogLines),
		width:         80,
		height:        24,
		sortAscending: true,
		styles:        render.DefaultStyleConfig(),
	}

	// Start a removal, then interrupt it the way the key handler does.
	started, opCmd := m.Update(keyPressString(keyEnter))
	interrupting, _ := started.Update(keyPressString(keyCtrlC))

	interrupted, ok := interrupting.(*Model)
	require.True(t, ok)
	require.Equal(t, "Interrupted", interrupted.status)
	require.NotEmpty(t, opCmd, "the removal must still be in flight")

	// The work finishes without observing the cancellation.
	final := drainOperation(t, interrupted, opCmd)

	assert.Equal(t, "Interrupted", final.status,
		"a stopped operation must not then report that it completed")

	// The work is not context aware, so it still ran and the list was refreshed
	// from the store rather than left stale.
	fsMock.AssertExpectations(t)
}

// TestOpResultMsg_StaleResultKeepsNewerOperation verifies a result belonging to
// an interrupted operation leaves a newer operation's state alone.
//
// The interrupt releases the status line without waiting, so the released
// operation's result arrives after the next one has already started.
func TestOpResultMsg_StaleResultKeepsNewerOperation(t *testing.T) {
	t.Parallel()

	m := newLifecycleModel()

	staleCmd := m.runAsync(
		"removing stale",
		func(context.Context) error { return nil },
		nil,
	)
	require.NotNil(t, staleCmd)
	require.Equal(t, "removing stale", m.busy)

	// The user interrupts, which releases the status line at once.
	interrupting, _ := m.Update(keyPressString(keyCtrlC))
	m, _ = interrupting.(*Model)
	require.Empty(t, m.busy)

	currentCmd := m.runAsync(
		"removing current",
		func(context.Context) error { return nil },
		nil,
	)
	require.NotNil(t, currentCmd)
	require.Equal(t, "removing current", m.busy)

	// The released operation now reports success, well after it stopped owning
	// the status line.
	stale, ok := staleCmd().(opResultMsg)
	require.True(t, ok)

	updated, _ := m.Update(stale)
	got, ok := updated.(*Model)
	require.True(t, ok)

	assert.Equal(t, "removing current", got.busy,
		"a stale result must not release the newer operation")

	// The newer operation still owns the status line, so it reports its own
	// outcome rather than being orphaned.
	current, ok := currentCmd().(opResultMsg)
	require.True(t, ok)

	updated, _ = got.Update(current)
	got, ok = updated.(*Model)
	require.True(t, ok)

	assert.Equal(t, "Done removing current", got.status)
	assert.Empty(t, got.busy)
}

// TestOpResultMsg_InterruptedDoesNotSuppressLaterSuccess verifies the interrupt
// that stopped one operation cannot withhold the success message of an operation
// that started after it.
func TestOpResultMsg_InterruptedDoesNotSuppressLaterSuccess(t *testing.T) {
	t.Parallel()

	m := newLifecycleModel()

	stoppedCmd := m.runAsync(
		"removing stopped",
		func(context.Context) error { return nil },
		nil,
	)
	require.NotNil(t, stoppedCmd)

	interrupting, _ := m.Update(keyPressString(keyCtrlC))
	m, _ = interrupting.(*Model)
	require.Equal(t, "Interrupted", m.status)

	// The next operation starts while the stopped one is still in flight, so it
	// reports back before the stopped operation does.
	currentCmd := m.runAsyncReporting(
		"removing current",
		func(context.Context) error { return nil },
		nil,
		nil,
		func() string { return "Removed current" },
	)
	require.NotNil(t, currentCmd)

	current, ok := currentCmd().(opResultMsg)
	require.True(t, ok)

	updated, _ := m.Update(current)
	got, ok := updated.(*Model)
	require.True(t, ok)

	assert.Equal(t, "Removed current", got.status,
		"an operation that was never interrupted must report its own success")

	// The stopped operation reports afterwards, and must not claim the status
	// line for work the user was told had finished.
	stopped, ok := stoppedCmd().(opResultMsg)
	require.True(t, ok)

	updated, _ = got.Update(stopped)
	got, ok = updated.(*Model)
	require.True(t, ok)

	assert.Equal(t, "Removed current", got.status,
		"a stopped operation must not overwrite a newer operation's success")
}

// TestWaitForOperation_WaitsForEveryInFlightOperation verifies shutdown waits for
// an operation an interrupt released as well as for the newest one.
func TestWaitForOperation_WaitsForEveryInFlightOperation(t *testing.T) {
	t.Parallel()

	m := newLifecycleModel()

	// Each operation blocks on its own channel, so the test decides when it
	// finishes instead of racing a timeout. Neither observes its context, which
	// is what lets an interrupted operation keep running.
	type operationRun struct {
		started  <-chan struct{}
		returned <-chan struct{}
	}

	start := func(name string, release <-chan struct{}) operationRun {
		startedCh := make(chan struct{})
		returnedCh := make(chan struct{})

		cmd := m.runAsync(name, func(context.Context) error {
			close(startedCh)
			defer close(returnedCh)

			<-release

			return nil
		}, nil)
		require.NotNil(t, cmd)

		// The work runs on its own goroutine, as Bubble Tea runs a command.
		go func() { _ = cmd() }()

		return operationRun{started: startedCh, returned: returnedCh}
	}

	releaseStopped := make(chan struct{})
	stopped := start("removing stopped", releaseStopped)
	awaitSignal(t, stopped.started, "the interrupted operation never started")

	// The user interrupts, which releases the status line while the work carries
	// on.
	m.cancelInFlight()

	releaseCurrent := make(chan struct{})
	current := start("removing current", releaseCurrent)
	awaitSignal(t, current.started, "the newest operation never started")

	waited := make(chan bool, 1)
	go func() { waited <- m.waitForOperation() }()

	// The newest operation finishes first, which is precisely what a wait on
	// only the newest one would already be satisfied by.
	close(releaseCurrent)
	awaitSignal(t, current.returned, "the newest operation never finished")

	select {
	case <-waited:
		t.Fatal("waitForOperation returned while an interrupted operation was still running")
	case <-time.After(blockedGrace):
	}

	// Releasing the interrupted operation lets the wait finish.
	close(releaseStopped)
	awaitSignal(t, stopped.returned, "the interrupted operation never finished")

	select {
	case returned := <-waited:
		assert.True(t, returned, "the wait must report that it waited on an operation")
	case <-time.After(opStartTimeout):
		t.Fatal("waitForOperation did not return after every operation finished")
	}
}
