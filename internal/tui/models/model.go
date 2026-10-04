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
	// keyCtrlC is the key that interrupts, and the one a dialog must honor.
	keyCtrlC = "ctrl+c"

	maxLogLines       = 50
	maxHistoryEntries = 100
)

// Mode constants for TUI state.
const (
	modeBinaries = "binaries"
	modeHistory  = "history"
)

// Confirmation constants for destructive operations.
const (
	confirmNone       = render.ConfirmNone
	confirmClearAll   = render.ConfirmClearAll
	confirmDeletePerm = render.ConfirmDeletePerm
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
// Context has to travel on the model for the handlers to reach the work they
// start.
//
//nolint:containedctx // Bubble Tea's Update receives no context; the run's context is carried here.
type Model struct {
	// ctx governs the work the model starts, so an interrupt reaches it.
	ctx context.Context

	// busy names the operation running outside Update, stopping a second from
	// starting and making Update ignore every key but the interrupt.
	busy string

	// interrupted records that the user stopped the operation. The result handler
	// reads it to withhold the success message for work already reported as stopped.
	interrupted bool

	// An interrupt clears busy without waiting for opDone, so this channel
	// rather than busy is what says whether the work is really over.
	cancelOp context.CancelFunc
	opDone   chan struct{}

	// mode selects the view and confirmation names the destructive action
	// awaiting acknowledgement.
	mode         string
	confirmation string

	// choices lists the available binaries, and cursorX and cursorY locate the
	// selection within the rows and cols grid.
	choices []string
	cursorX int
	cursorY int
	cols    int
	rows    int

	historyEntries []*history.HistoryEntry
	historyCursor  int
	historyManager history.Manager
	historyLoading bool

	dir           string
	config        Config
	logger        logger.Logger
	fs            fs.FS
	width         int
	height        int
	status        string
	styles        render.StyleConfig
	sortAscending bool

	// logs holds the captured log messages, showLogs reports whether the log
	// panel is visible, and logChan receives log messages from the logger.
	logs     []string
	showLogs bool
	logChan  chan LogMsg
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

	if config.RestoreMode {
		m.mode = modeHistory
		m.historyLoading = true
	}

	m.logChan = make(chan LogMsg, maxLogLines)
	m.setupLogCapture(log)

	return m
}

// Wait blocks until any operation started outside Update has finished.
//
// Returns:
//   - True if an operation had to be waited on.
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
		m.logger.Warn(
			"Could not rescan the binary directory",
			logger.Err(err),
			logger.Str("dir", m.dir),
		)

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

	// Polling runs for the whole session so the log panel holds recent entries;
	// polling only under --verbose would fill a channel nobody drains.
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
func (m *Model) updateGrid() {
	// The row count can change, so the selection is carried across as the
	// index it pointed at rather than as coordinates.
	selected := m.cursorY + m.cursorX*max(m.rows, 1)

	grid := render.GridLayout(&render.GridInput{
		Choices:  m.choices,
		Width:    m.width,
		Height:   m.height,
		Status:   m.status,
		Busy:     m.busy,
		ShowLogs: m.showLogs,
		LogCount: len(m.logs),
		CursorX:  m.cursorX,
		CursorY:  m.cursorY,
	})

	m.rows = grid.Rows
	m.cols = grid.Cols

	m.cursorX, m.cursorY = 0, 0

	switch {
	case len(m.choices) == 0 || m.rows == 0:
		return
	case selected >= len(m.choices):
		selected = len(m.choices) - 1
	}

	m.cursorX = selected / m.rows
	m.cursorY = selected % m.rows
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
	mode := render.ModeBinaries
	if m.mode == modeHistory {
		mode = render.ModeHistory
	}

	return &render.State{
		Mode:           mode,
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
