/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"context"
	"errors"
	"fmt"
	"slices"

	tea "charm.land/bubbletea/v2"

	"github.com/nicholas-fedor/go-remove/internal/fs"
	"github.com/nicholas-fedor/go-remove/internal/history"
	"github.com/nicholas-fedor/go-remove/internal/logger"
	"github.com/nicholas-fedor/go-remove/internal/tui/render"
)

// Layout constants for TUI state that is not a rendering decision.
const (
	// keyCtrlC is the key that interrupts, and the one a dialog must honour.
	keyCtrlC = "ctrl+c"

	maxLogLines       = 50  // Maximum number of log lines to retain
	maxHistoryEntries = 100 // Maximum number of history entries to display
)

// Mode constants for TUI state.
const (
	modeBinaries = "binaries" // Mode for binary selection view
	modeHistory  = "history"  // Mode for history view
)

// Confirmation constants for destructive operations.
const (
	confirmNone       = render.ConfirmNone       // No confirmation pending
	confirmClearAll   = render.ConfirmClearAll   // Confirm clearing all history
	confirmDeletePerm = render.ConfirmDeletePerm // Confirm permanent deletion
)

// ErrHistoryNotInitialized indicates the history manager was not initialized.
var ErrHistoryNotInitialized = errors.New("history manager not initialized")

// HistoryMsg is a Bubble Tea message that signals history entries have been loaded.
type HistoryMsg struct {
	Entries []*history.HistoryEntry
	Error   error

	// quiet marks a background refresh, such as the one after an operation, which
	// must not replace the status the operation reported.
	quiet bool
}

// Model encapsulates the state of the TUI.
//
// context has to travel on the model for the handlers to reach the work they
// start.
//
//nolint:containedctx // Bubble Tea's Update receives no context, so the run's
type Model struct {
	// ctx governs the work the model starts, so an interrupt reaches it.
	ctx context.Context

	// busy names the operation currently running outside Update, or is empty
	// when the model is idle.
	busy string

	// interrupted records that the running operation was stopped by the user, so
	// its result cannot claim the work finished.
	interrupted bool

	// cancelOp stops the operation in flight, and opDone is closed when it has
	// finished. Shutdown waits on opDone so the history manager is never closed
	// under a recovery that is still running.
	cancelOp context.CancelFunc
	opDone   chan struct{}

	// Mode and view state
	mode         string // Current mode: "binaries" or "history"
	confirmation string // Pending confirmation for destructive operations

	// Binary selection state
	choices []string // List of available binaries
	cursorX int      // Horizontal cursor position (column)
	cursorY int      // Vertical cursor position (row)
	cols    int      // Number of columns in the grid
	rows    int      // Number of rows in the grid

	// History state
	historyEntries []*history.HistoryEntry // History entries for display
	historyCursor  int                     // Cursor position in history view
	historyManager history.Manager         // History manager for operations
	historyLoading bool                    // Whether history is being loaded

	// General state
	dir           string             // Directory containing binaries
	config        Config             // CLI configuration
	logger        logger.Logger      // Logger instance
	fs            fs.FS              // Filesystem operations
	width         int                // Terminal width
	height        int                // Terminal height
	status        string             // Status message
	styles        render.StyleConfig // TUI appearance settings
	sortAscending bool               // True for ascending sort, false for descending
	logs          []string           // Captured log messages (circular buffer)
	showLogs      bool               // Toggle log panel visibility
	logChan       chan LogMsg        // Channel for receiving log messages from the logger
}

// Config holds the model settings the TUI needs from its caller.
//
// It is deliberately narrower than the command layer's configuration: these
// three fields are the only ones the model reads, so adding a command flag does
// not reach into the TUI.
type Config struct {
	// Verbose enables debug and info output.
	Verbose bool

	// LogLevel is the level logging returns to when the log panel closes.
	LogLevel string

	// RestoreMode starts in the history view instead of the binary grid.
	RestoreMode bool
}

// New builds the model the TUI runs on.
//
// Log capture is always installed, even when the panel starts closed, so the
// channel and callback are ready by the time the user presses the toggle.
//
// Parameters:
//   - ctx: Context governing the work the model starts.
//   - dir: Directory containing the binaries.
//   - config: Settings the model reads.
//   - log: Logger that feeds the log panel.
//   - filesystem: Filesystem operations.
//   - historyMgr: History manager for undo and restore.
//   - choices: Binaries found in dir.
//
// Returns:
//   - A model ready to hand to Bubble Tea.
func New(
	ctx context.Context,
	dir string,
	config Config,
	log logger.Logger,
	filesystem fs.FS,
	historyMgr history.Manager,
	choices []string,
) *Model {
	// Initialize the model with default styles.
	// Enable log visibility by default when verbose mode is active.
	m := &Model{
		ctx:            ctx,
		choices:        choices,
		dir:            dir,
		config:         config,
		logger:         log,
		fs:             filesystem,
		cursorX:        0,
		cursorY:        0,
		sortAscending:  true,
		styles:         render.DefaultStyleConfig(),
		logs:           make([]string, 0, maxLogLines),
		showLogs:       config.Verbose,
		mode:           modeBinaries,
		historyEntries: make([]*history.HistoryEntry, 0),
		historyCursor:  0,
		historyManager: historyMgr,
	}

	// Set up mode based on config
	if config.RestoreMode {
		m.mode = modeHistory
		m.historyLoading = true
	}

	// Always set up log capture infrastructure so verbose mode can be toggled at runtime.
	// This ensures the log channel and capture callback are ready when the user presses 'L'.
	m.logChan = make(chan LogMsg, maxLogLines)
	m.setupLogCapture(log)

	return m
}

// Wait blocks until any operation started outside Update has finished.
func (m *Model) Wait() bool { return m.waitForOperation() }

// context returns the context governing the model, falling back to a background
// context when the model was built without one.
//
// Returns:
//   - A context that is never nil.
func (m *Model) context() context.Context {
	if m.ctx != nil {
		return m.ctx
	}

	return context.Background()
}

// refreshChoices rescans the binary directory, keeping the current list when the
// directory cannot be read so a transient read failure does not empty the view.
func (m *Model) refreshChoices() {
	choices, err := m.fs.ListBinaries(m.dir)
	if err != nil {
		m.logger.Warn().
			Err(err).
			Str("dir", m.dir).
			Msg("Could not rescan the binary directory")

		// Surface it rather than only logging, so a stale list is not passed off
		// as current.
		m.status = fmt.Sprintf("Could not read %s: %v", m.dir, err)

		return
	}

	m.choices = choices
}

// Init prepares the TUI model for rendering.
//
// Returns:
//   - Initial command batch for history loading and log polling.
func (m *Model) Init() tea.Cmd {
	m.sortChoices()

	// Log polling runs for the whole session, not only in verbose mode, so the
	// log panel holds recent entries whenever the user opens it. Starting it
	// only under --verbose meant the capture callback filled a channel nobody
	// drained, so every message was discarded.
	polling := m.pollLogChannel()

	if m.mode == modeHistory {
		return tea.Batch(
			m.loadHistory(),
			polling,
		)
	}

	return polling
}

// sortChoices sorts the binary list according to the current sort order.
func (m *Model) sortChoices() {
	if len(m.choices) == 0 {
		return
	}

	slices.Sort(m.choices)

	if !m.sortAscending {
		slices.Reverse(m.choices)
	}
}

// updateGrid recalculates the grid layout from the current state and terminal size.
//
// The arithmetic lives in the render package. This applies the result, because
// the grid and the cursor are model state rather than a rendering decision.
func (m *Model) updateGrid() {
	grid := render.GridLayout(&render.GridInput{
		Choices:  m.choices,
		Width:    m.width,
		Height:   m.height,
		Status:   m.status,
		ShowLogs: m.showLogs,
		LogCount: len(m.logs),
		CursorX:  m.cursorX,
		CursorY:  m.cursorY,
	})

	m.rows = grid.Rows
	m.cols = grid.Cols
	m.cursorX = grid.CursorX
	m.cursorY = grid.CursorY
}

// View renders the TUI interface.
//
// Returns:
//   - Bubble Tea view for the current mode.
func (m *Model) View() tea.View {
	state := m.renderState()

	if m.mode == modeHistory {
		return render.History(state)
	}

	return render.Binaries(state)
}

// renderState captures the model as a snapshot for the view to draw.
//
// Returns:
//   - The state describing the current frame.
func (m *Model) renderState() *render.State {
	return &render.State{
		Mode:           render.ModeBinaries,
		Width:          m.width,
		Height:         m.height,
		Styles:         m.styles,
		Choices:        m.choices,
		Rows:           m.rows,
		Cols:           m.cols,
		CursorX:        m.cursorX,
		CursorY:        m.cursorY,
		HistoryEntries: m.historyEntries,
		HistoryCursor:  m.historyCursor,
		HistoryLoading: m.historyLoading,
		Status:         m.status,
		Busy:           m.busy,
		Confirmation:   m.confirmation,
		Logs:           m.logs,
		ShowLogs:       m.showLogs,
	}
}
