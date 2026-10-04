/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/nicholas-fedor/go-remove/internal/errmsg"
)

// operation is one unit of work in flight.
//
// Its own result carries it back, so a result that arrives after an interrupt
// let a newer operation start can be told apart from the one the model is
// waiting on.
type operation struct {
	// name is shown on the status line while it runs.
	name string

	// cancel stops the work.
	cancel context.CancelFunc

	// done is closed when the work has finished.
	done chan struct{}

	// interrupted records that the user stopped this operation, so its own
	// result cannot claim success.
	interrupted bool
}

// opResultMsg carries the outcome of work that must not block the render loop.
type opResultMsg struct {
	// op is the operation the result belongs to, which is what makes a stale
	// result recognizable as one the model is no longer waiting on.
	op *operation

	// err is the outcome, nil on success.
	err error

	// Without it the generic "Error <op>" form is used, losing the distinction
	// the history layer makes between an unrestorable entry and a real fault.
	errStatus func(error) string

	// okStatus is the operation's own success message. Only the operation knows
	// its result, so it supplies the wording.
	okStatus string

	// refresh updates the model once the operation has finished. It runs on the
	// update goroutine, never in the operation's own goroutine.
	refresh func(m *Model)
}

// restoreErrorStatus maps a restore failure to a message that names the entry.
//
// Without this every failure collapses into one generic message, losing the
// difference between an entry that cannot be restored and a real fault.
//
// Parameters:
//   - name: Binary name the restore was for.
//
// Returns:
//   - A function mapping an error to a status message.
func restoreErrorStatus(name string) func(error) string {
	return func(err error) string {
		// The error is already wrapped with the binary name.
		fallback := "Error " + err.Error()

		switch errmsg.Classify(err) {
		case errmsg.KindAlreadyRestored:
			return name + " has already been restored"
		case errmsg.KindNotInTrash:
			return name + " is no longer in trash"
		case errmsg.KindRestoreCollision:
			return fmt.Sprintf("Cannot restore %s: file already exists", name)
		case errmsg.KindNoHistory, errmsg.KindUnknown:
			// A restore has no history to consult, so a no-history error lands
			// here along with anything unclassified.
			return fallback
		default:
			// A Kind was added without a message for this operation.
			return fallback
		}
	}
}

// loadHistory returns a command that loads deletion history.
//
// Returns:
//   - Command that produces a HistoryMsg.
func (m *Model) loadHistory() tea.Cmd {
	return m.loadHistoryReporting(false)
}

// loadHistoryReporting loads the history, optionally as a background refresh.
//
// Parameters:
//   - quiet: When true the result leaves the status line alone.
//
// Returns:
//   - Command that produces a HistoryMsg.
func (m *Model) loadHistoryReporting(quiet bool) tea.Cmd {
	return func() tea.Msg {
		if m.historyManager == nil {
			return HistoryMsg{Entries: nil, Error: ErrHistoryNotInitialized, quiet: quiet}
		}

		ctx := m.context()
		entries, err := m.historyManager.GetHistory(ctx, maxHistoryEntries)

		return HistoryMsg{Entries: entries, Error: err, quiet: quiet}
	}
}

// runAsync starts an operation outside the update goroutine and reports the
// outcome back as a message.
//
// The work runs with a context derived from the model's own, so an interrupt
// reaches it and the shutdown path can wait for it to finish.
//
// Parameters:
//   - name: Human-readable name of the operation, shown while it runs.
//   - work: The operation to perform.
//   - refresh: Applied to the model on the update goroutine after success.
//
// Returns:
//   - A command that performs the work and yields the result.
func (m *Model) runAsync(
	name string,
	work func(context.Context) error,
	refresh func(m *Model),
) tea.Cmd {
	return m.runAsyncReporting(name, work, refresh, nil, nil)
}

// runAsyncReporting starts an operation that maps its own failures to status
// messages.
//
// Parameters:
//   - name: Human-readable name of the operation, shown while it runs.
//   - work: The operation to perform.
//   - refresh: Applied to the model on the update goroutine after success.
//   - errStatus: Maps a failure to a status message, or nil for the generic form.
//   - okStatus: Supplies the success message from the operation's own result.
//
// Returns:
//   - A command that performs the work and yields the result.
func (m *Model) runAsyncReporting(
	name string,
	work func(context.Context) error,
	refresh func(m *Model),
	errStatus func(error) string,
	okStatus func() string,
) tea.Cmd {
	// Only one operation at a time, so a second keypress cannot start another
	// while the first is still in flight.
	if m.busy != "" {
		return nil
	}

	ctx, cancel := context.WithCancel(m.context())

	tracked := &operation{
		name:   name,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	m.busy = name
	m.cur = tracked

	// An interrupted operation keeps running, so shutdown has to wait for it too.
	m.inFlight = append(m.inFlight, tracked)

	// The busy line costs a row, so the grid has to be laid out again or the
	// choices overflow the terminal for as long as the work runs.
	m.updateGrid()

	return func() tea.Msg {
		defer close(tracked.done)

		err := work(ctx)

		result := opResultMsg{
			op:        tracked,
			err:       err,
			refresh:   refresh,
			errStatus: errStatus,
		}

		if err == nil && okStatus != nil {
			result.okStatus = okStatus()
		}

		return result
	}
}

// cancelInFlight stops the running operation and reports progress.
func (m *Model) cancelInFlight() {
	if op := m.cur; op != nil {
		op.cancel()

		// The work may not observe the cancellation at all, so remember that the
		// user stopped it rather than trusting the result it eventually reports.
		op.interrupted = true
	}

	// The operation stops owning the status line, but it stays in flight until
	// its own result arrives, because the work itself may still be running.
	m.cur = nil
	m.busy = ""

	// Clearing busy releases the row it reserved, rather than waiting for the result.
	m.updateGrid()
}

// waitForOperation blocks until every operation started outside Update has
// finished.
//
// It exists so shutdown does not close the history manager under a recovery
// that is still running, including one an interrupt released the status line
// for while it kept going.
//
// Returns:
//   - True if an operation had to be waited on.
func (m *Model) waitForOperation() bool {
	if len(m.inFlight) == 0 {
		return false
	}

	// Every operation has to be waited on, because an interrupted one is still
	// in flight and can reach the history manager just as the newest one can.
	for _, op := range m.inFlight {
		<-op.done
	}

	m.inFlight = nil

	return true
}
