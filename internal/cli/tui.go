/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package cli provides core logic for the go-remove command-line interface.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/rs/zerolog"
	"golang.org/x/term"

	tea "charm.land/bubbletea/v2"

	"github.com/nicholas-fedor/go-remove/internal/fs"
	"github.com/nicholas-fedor/go-remove/internal/history"
	"github.com/nicholas-fedor/go-remove/internal/logger"
)

// Layout constants for TUI rendering.
// These constants must be kept consistent between updateGrid() and the view functions.
const (
	// keyCtrlC is the key that interrupts, and the one a dialog must honour.
	keyCtrlC = "ctrl+c"

	colWidthPadding          = 3                  // Padding added to column width for spacing
	availWidthAdjustment     = 4                  // Adjustment to width for border and padding
	minAvailHeightAdjustment = 8                  // Minimum height adjustment for UI elements (title + footer + padding)
	visibleLenPrefix         = 2                  // Prefix length for cursor visibility
	footerHeight             = 1                  // Height reserved for footer/instructions
	leftPadding              = 2                  // Left padding for the entire TUI
	maxLogLines              = 50                 // Maximum number of log lines to retain
	maxVisibleLogLines       = 5                  // Maximum number of log lines to display
	logPanelSeparatorLines   = 2                  // Number of separator lines for log panel
	maxHistoryEntries        = 100                // Maximum number of history entries to display
	dateTimeFormat           = "2006-01-02 15:04" // Format for displaying timestamps
	separatorAdjustment      = 2                  // Extra width for column separator
	historyTableHeaderLines  = 2                  // Number of lines for history table header (header + separator)
)

// Mode constants for TUI state.
const (
	modeBinaries = "binaries" // Mode for binary selection view
	modeHistory  = "history"  // Mode for history view
)

// Confirmation constants for destructive operations.
const (
	confirmNone       = ""                 // No confirmation pending
	confirmClearAll   = "clear_all"        // Confirm clearing all history
	confirmDeletePerm = "delete_permanent" // Confirm permanent deletion
)

// ErrNoBinariesFound signals that no binaries were found in the target directory.
var ErrNoBinariesFound = errors.New("no binaries found in directory")

// ErrHistoryNotInitialized indicates the history manager was not initialized.
var ErrHistoryNotInitialized = errors.New("history manager not initialized")

// ErrNotATerminal indicates the interactive interface was started without a terminal.
var ErrNotATerminal = errors.New("no interactive terminal available")

// stdinIsTerminal reports whether standard input is an interactive terminal.
//
// term.IsTerminal is used rather than an os.ModeCharDevice check because
// /dev/null is itself a character device, so a redirected stdin would otherwise
// be mistaken for a terminal and fail later inside the TUI instead of here. It
// is a variable so the TUI can be exercised in tests, which have no terminal.
var stdinIsTerminal = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// LogMsg is a Bubble Tea message that carries a log entry to be displayed in the TUI.
type LogMsg struct {
	Level   string // Log level (e.g., "DBG", "INF", "WRN", "ERR")
	Message string // Log message content
}

// HistoryMsg is a Bubble Tea message that signals history entries have been loaded.
type HistoryMsg struct {
	Entries []*history.HistoryEntry
	Error   error

	// quiet marks a background refresh, such as the one after an operation, which
	// must not replace the status the operation reported.
	quiet bool
}

// ProgramRunner defines an interface for running Bubbletea programs.
type ProgramRunner interface {
	// RunProgram launches a Bubble Tea program.
	//
	// Parameters:
	//   - m: Initial TUI model.
	//   - opts: Optional Bubble Tea program options.
	//
	// Returns:
	//   - Started program.
	//   - An error if the program cannot be created.
	RunProgram(m tea.Model, opts ...tea.ProgramOption) (*tea.Program, error)
}

// styleConfig holds TUI appearance settings.
type styleConfig struct {
	TitleColor    string // ANSI 256-color code for title
	CursorColor   string // ANSI 256-color code for cursor
	FooterColor   string // ANSI 256-color code for footer
	StatusColor   string // ANSI 256-color code for status
	LogColor      string // ANSI 256-color code for log messages
	HistoryColor  string // ANSI 256-color code for history table header
	TrashYesColor string // ANSI 256-color code for "Yes" in trash available column
	TrashNoColor  string // ANSI 256-color code for "No" in trash available column
	Cursor        string // Symbol used for the cursor
}

// model encapsulates the state of the TUI.
//
// context has to travel on the model for the handlers to reach the work they
// start.
//
//nolint:containedctx // Bubble Tea's Update receives no context, so the run's
type model struct {
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
	dir           string        // Directory containing binaries
	config        Config        // CLI configuration
	logger        logger.Logger // Logger instance
	fs            fs.FS         // Filesystem operations
	width         int           // Terminal width
	height        int           // Terminal height
	status        string        // Status message
	styles        styleConfig   // TUI appearance settings
	sortAscending bool          // True for ascending sort, false for descending
	logs          []string      // Captured log messages (circular buffer)
	showLogs      bool          // Toggle log panel visibility
	logChan       chan LogMsg   // Channel for receiving log messages from the logger
}

// DefaultRunner provides the default Bubbletea program runner.
type DefaultRunner struct{}

var _ ProgramRunner = DefaultRunner{}

// RunTUI launches the interactive TUI for binary selection, removal, and restore.
//
// Parameters:
//   - dir: Directory containing Go binaries.
//   - config: CLI configuration.
//   - log: Logger used by the TUI.
//   - filesystem: Filesystem implementation.
//   - runner: Bubble Tea program runner.
//   - historyMgr: History manager for undo and restore.
//
// Returns:
//   - An error if no binaries are found or the program fails to run.
func RunTUI(
	ctx context.Context,
	dir string,
	config Config,
	log logger.Logger,
	filesystem fs.FS,
	runner ProgramRunner,
	historyMgr history.Manager,
) error {
	// A TUI needs a terminal. Under a pipe or in CI the failure otherwise
	// surfaces as a nested bubbletea error with no hint that the non-interactive
	// form is a plain argument.
	if !stdinIsTerminal() {
		return fmt.Errorf(
			"%w: go-remove needs an interactive terminal, pass a binary name to remove it directly",
			ErrNotATerminal,
		)
	}

	// Fetch available binaries from the specified directory.
	choices, err := filesystem.ListBinaries(dir)
	if err != nil {
		return fmt.Errorf("listing binaries in %s: %w", dir, err)
	}

	if len(choices) == 0 && !config.RestoreMode {
		return fmt.Errorf("%w: %s", ErrNoBinariesFound, dir)
	}

	// Initialize the model with default styles.
	// Enable log visibility by default when verbose mode is active.
	m := &model{
		ctx:            ctx,
		choices:        choices,
		dir:            dir,
		config:         config,
		logger:         log,
		fs:             filesystem,
		cursorX:        0,
		cursorY:        0,
		sortAscending:  true,
		styles:         defaultStyleConfig(),
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

	// Start the TUI program with the caller's context. Bubbletea's own signal
	// handling is disabled so an interrupt reaches the context installed by
	// Execute rather than racing it, leaving a single interruption path.
	program, err := runner.RunProgram(
		m,
		tea.WithContext(ctx),
		tea.WithoutSignalHandler(),
	)
	if err != nil {
		return fmt.Errorf("failed to start TUI program: %w", err)
	}

	// Allow mocked runners to return nil for testing purposes.
	if program == nil {
		return nil
	}

	// Run the program and capture any runtime errors.
	_, runErr := program.Run()

	// An operation may still be running, possibly part way through a recovery.
	// The history manager is closed by the caller straight after this returns,
	// so wait rather than closing the store underneath it.
	m.waitForOperation()

	if runErr != nil {
		// An interrupt is the user asking to quit, so it is not a failure.
		if errors.Is(runErr, tea.ErrInterrupted) {
			return nil
		}

		// A cancelled context makes Bubbletea kill the program, which is still
		// the user's interrupt rather than a real failure. Anything else keeps
		// its own identity, so an unexpected kill is still reported.
		if errors.Is(runErr, tea.ErrProgramKilled) && errors.Is(ctx.Err(), context.Canceled) {
			return nil
		}

		return fmt.Errorf("failed to run TUI program: %w", runErr)
	}

	return nil
}

// context returns the context governing the model, falling back to a background
// context when the model was built without one.
//
// Parameters:
//   - None.
//
// Returns:
//   - A context that is never nil.
func (m *model) context() context.Context {
	if m.ctx != nil {
		return m.ctx
	}

	return context.Background()
}

// refreshChoices rescans the binary directory, keeping the current list when the
// directory cannot be read so a transient read failure does not empty the view.
//
// Parameters:
//   - None.
func (m *model) refreshChoices() {
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

// setupLogCapture configures the logger's capture callback to send messages to the TUI.
//
// This method is idempotent and can be called multiple times safely.
//
// Parameters:
//   - log: Logger that will receive the capture callback.
func (m *model) setupLogCapture(log logger.Logger) {
	log.SetCaptureFunc(func(level, msg string) {
		// Send to channel without blocking.
		// If channel is full, the message is dropped to prevent blocking.
		select {
		case m.logChan <- LogMsg{Level: level, Message: msg}:
		default:
			// Channel is full, message is dropped to prevent blocking.
		}
	})
}

// toggleVerboseLogging toggles log panel visibility and logging level.
//
// Polling already runs for the whole session, so this only changes what is
// shown and how much detail is emitted. It deliberately does not start a
// second poll chain, which previously left one running per press.
func (m *model) toggleVerboseLogging() {
	m.showLogs = !m.showLogs

	// Opening the panel raises the level so the extra detail is worth having.
	// Closing it returns to the configured level, except when the user asked
	// for verbose at startup: that request is independent of the panel, so
	// hiding the panel must not quietly reduce verbosity below what was asked.
	if m.showLogs || m.config.Verbose {
		m.logger.Level(zerolog.DebugLevel)
	} else {
		m.logger.Level(logger.ParseLevel(m.config.LogLevel))
	}

	// The panel changes the space available to the grid, so the reserved height
	// has to be recalculated.
	m.updateGrid()
}

// defaultStyleConfig provides default TUI style settings.
//
// Returns:
//   - Default color and cursor configuration.
func defaultStyleConfig() styleConfig {
	return styleConfig{
		TitleColor:    "39",  // Bright blue
		CursorColor:   "214", // Orange
		FooterColor:   "245", // Light gray
		StatusColor:   "46",  // Lime green
		LogColor:      "240", // Dark gray for subtle log display
		HistoryColor:  "141", // Purple for history header
		TrashYesColor: "46",  // Green for "Yes"
		TrashNoColor:  "196", // Red for "No"
		Cursor:        "❯ ",
	}
}

// RunProgram launches a Bubble Tea program with the given model and options.
//
// Parameters:
//   - m: Initial TUI model.
//   - opts: Optional Bubble Tea program options.
//
// Returns:
//   - Started program.
//   - Always nil error.
func (r DefaultRunner) RunProgram(m tea.Model, opts ...tea.ProgramOption) (*tea.Program, error) {
	program := tea.NewProgram(m, opts...)

	return program, nil
}

// Init prepares the TUI model for rendering.
//
// Returns:
//   - Initial command batch for history loading and log polling.
func (m *model) Init() tea.Cmd {
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

// loadHistory returns a command that loads deletion history.
//
// Returns:
//   - Command that produces a HistoryMsg.
func (m *model) loadHistory() tea.Cmd {
	return m.loadHistoryReporting(false)
}

// loadHistoryReporting loads the history, optionally as a background refresh.
//
// Parameters:
//   - quiet: When true the result leaves the status line alone.
//
// Returns:
//   - Command that produces a HistoryMsg.
func (m *model) loadHistoryReporting(quiet bool) tea.Cmd {
	return func() tea.Msg {
		// Check if history manager is available
		if m.historyManager == nil {
			return HistoryMsg{Entries: nil, Error: ErrHistoryNotInitialized, quiet: quiet}
		}

		ctx := m.context()
		entries, err := m.historyManager.GetHistory(ctx, maxHistoryEntries)

		return HistoryMsg{Entries: entries, Error: err, quiet: quiet}
	}
}

// pollLogChannel returns a command that polls for log messages.
//
// The TUI polls the channel instead of sending from the capture callback, which avoids deadlocks.
//
// Returns:
//   - Command that produces a LogMsg or poll tick.
func (m *model) pollLogChannel() tea.Cmd {
	return func() tea.Msg {
		// Wait for either a log message or a timeout.
		// Using time.After ensures we return a proper tea.Msg (not a tea.Cmd),
		// which maintains the continuous polling loop.
		select {
		case msg := <-m.logChan:
			return msg
		case <-time.After(pollInterval):
			// No message available after interval, return tick to schedule next poll.
			return pollLogTickMsg{}
		}
	}
}

// pollInterval is the duration between log channel polls.
const pollInterval = 50 * time.Millisecond

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
		switch {
		case errors.Is(err, history.ErrAlreadyRestored):
			return name + " has already been restored"
		case errors.Is(err, history.ErrNotInTrash):
			return name + " is no longer in trash"
		case errors.Is(err, history.ErrRestoreCollision):
			return fmt.Sprintf("Cannot restore %s: file already exists", name)
		default:
			// The error is already wrapped with the binary name.
			return "Error " + err.Error()
		}
	}
}

// pollLogTickMsg is a message sent when it's time to poll for logs again.
type pollLogTickMsg struct{}

// opResultMsg carries the outcome of work that must not block the render loop.
type opResultMsg struct {
	// operation names the work, for the status line.
	operation string

	// err is the outcome, nil on success.
	err error

	// errStatus turns a failure into a specific message. Without it the generic
	// "Error <op>" form is used, which loses the distinction the history layer
	// makes between an unrestorable entry and a real fault.
	errStatus func(error) string

	// okStatus is the operation's own success message. Only the operation knows
	// its result, so it supplies the wording.
	okStatus string

	// refresh updates the model once the operation has finished. It runs on the
	// update goroutine, never in the operation's own goroutine.
	refresh func(m *model)
}

// runAsync starts an operation outside the update goroutine and reports the
// outcome back as a message.
//
// The work runs with a context derived from the model's own, so an interrupt
// reaches it and the shutdown path can wait for it to finish.
//
// Parameters:
//   - op: Human-readable name of the operation, shown while it runs.
//   - work: The operation to perform.
//   - refresh: Applied to the model on the update goroutine after success.
//
// Returns:
//   - A command that performs the work and yields the result.
func (m *model) runAsync(
	operation string,
	work func(context.Context) error,
	refresh func(m *model),
) tea.Cmd {
	return m.runAsyncReporting(operation, work, refresh, nil, nil)
}

// runAsyncReporting starts an operation that maps its own failures to status
// messages.
//
// Parameters:
//   - op: Human-readable name of the operation, shown while it runs.
//   - work: The operation to perform.
//   - refresh: Applied to the model on the update goroutine after success.
//   - errStatus: Maps a failure to a status message, or nil for the generic form.
//   - okStatus: Supplies the success message from the operation's own result.
//
// Returns:
//   - A command that performs the work and yields the result.
func (m *model) runAsyncReporting(
	operation string,
	work func(context.Context) error,
	refresh func(m *model),
	errStatus func(error) string,
	okStatus func() string,
) tea.Cmd {
	// Only one operation at a time, so a second keypress cannot start another
	// while the first is still in flight.
	if m.busy != "" {
		return nil
	}

	ctx, cancel := context.WithCancel(m.context())
	done := make(chan struct{})

	m.busy = operation
	m.cancelOp = cancel
	m.opDone = done

	return func() tea.Msg {
		defer close(done)

		err := work(ctx)

		result := opResultMsg{
			operation: operation,
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
//
// Parameters:
//   - None.
func (m *model) cancelInFlight() {
	if m.cancelOp != nil {
		m.cancelOp()
	}

	m.busy = ""
	// The work may not observe the cancellation at all, so remember that the
	// user stopped it rather than trusting the result it eventually reports.
	m.interrupted = true
}

// waitForOperation blocks until the running operation has finished.
//
// It exists so shutdown does not close the history manager under a recovery
// that is still running.
//
// Returns:
//   - True if an operation had to be waited on.
func (m *model) waitForOperation() bool {
	if m.opDone == nil {
		return false
	}

	<-m.opDone
	m.opDone = nil

	return true
}

// Update processes TUI events and updates the model state.
//
// Parameters:
//   - msg: Incoming Bubble Tea message.
//
// Returns:
//   - Updated model.
//   - Follow-up command, if any.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// An interrupt always works, so a long operation can be stopped.
		if msg.String() == keyCtrlC {
			m.cancelInFlight()
			m.status = "Interrupted"
		}

		// While an operation is running the model is not in a state a keypress
		// can act on, so the key is ignored rather than acted on halfway.
		if m.busy != "" && msg.String() != keyCtrlC {
			return m, nil
		}

		// Ignore KeyReleaseMsg. KeyMsg matches both, so a single physical
		// keystroke would otherwise undo or restore twice.
		if m.confirmation != confirmNone {
			return m.handleConfirmation(msg)
		}

		if m.mode == modeHistory {
			return m.updateHistoryMode(msg)
		}

		return m.updateBinaryMode(msg)

	case tea.WindowSizeMsg:
		// Update dimensions and recalculate grid layout on resize.
		m.width = msg.Width
		m.height = msg.Height
		m.updateGrid()

		return m, nil

	case pollLogTickMsg:
		// Continue polling for log messages when a tick occurs.
		cmd := m.pollLogChannel()

		return m, cmd

	case opResultMsg:
		// The operation has finished, so the model is interactive again.
		m.busy = ""
		m.cancelOp = nil
		m.updateGrid()

		// An interrupted operation still applies its side effects, because the
		// work may not have observed the cancellation, but it must not then
		// report success the user has already been told they stopped.
		interrupted := m.interrupted
		m.interrupted = false

		if msg.err != nil {
			if !interrupted {
				if msg.errStatus != nil {
					m.status = msg.errStatus(msg.err)
				} else {
					// The error is already wrapped with the operation.
					m.status = "Error " + msg.err.Error()
				}
			}

			if m.historyManager == nil {
				return m, nil
			}

			reload := m.loadHistoryReporting(true)

			return m, reload
		}

		// The operation may report a more specific outcome than the generic
		// form, so its own status wins when it sets one. An interrupted
		// operation keeps the status the interrupt already showed, even though
		// the refresh below reports the work as done.
		preserved := m.status

		if !interrupted {
			m.status = msg.okStatus
		}

		if msg.refresh != nil {
			msg.refresh(m)
		}

		switch {
		case interrupted:
			m.status = preserved
		case m.status == "":
			m.status = "Done " + msg.operation
		}

		m.updateGrid()

		if m.historyManager == nil {
			return m, nil
		}

		reload := m.loadHistoryReporting(true)

		return m, reload

	case LogMsg:
		// Add log message to the circular buffer.
		m.addLogEntry(msg)
		m.updateGrid()

		// Continue polling for more log messages.
		// This ensures all pending logs are captured.
		cmd := m.pollLogChannel()

		return m, cmd

	case HistoryMsg:
		// Handle history loading result
		m.historyLoading = false

		if msg.quiet {
			// A background refresh keeps whatever the operation reported.
			if msg.Error == nil {
				m.historyEntries = msg.Entries
				m.clampHistoryCursor()
			}

			return m, nil
		}

		if msg.Error != nil {
			m.status = fmt.Sprintf("Error loading history: %v", msg.Error)
		} else {
			m.historyEntries = msg.Entries
			if len(m.historyEntries) == 0 {
				m.status = "No deletion history found"
			} else {
				m.status = fmt.Sprintf("Loaded %d history entries", len(m.historyEntries))
			}

			// The list may have shrunk, leaving the cursor past the end, in
			// which case no row renders as selected and the view looks frozen on
			// a phantom row until the user moves up.
			m.clampHistoryCursor()
		}

		return m, nil
	}

	return m, nil
}

// handleConfirmation processes key presses during confirmation dialogs.
//
// Parameters:
//   - msg: Key press from the user.
//
// Returns:
//   - Updated model.
//   - Follow-up command, if any.
func (m *model) handleConfirmation(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// Confirmed - execute the operation
		return m.executeConfirmation()
	case "ctrl+c":
		// The one key a dialog must never swallow, otherwise the prompt
		// becomes a trap for anyone who reaches for it.
		m.confirmation = confirmNone
		m.status = "Operation cancelled"

		return m, tea.Quit
	case "n", "N", "q", "esc":
		// Cancelled - clear confirmation
		m.confirmation = confirmNone
		m.status = "Operation cancelled"
	}

	return m, nil
}

// clampHistoryCursor keeps the history cursor within the loaded entries.
//
// Parameters:
//   - None.
func (m *model) clampHistoryCursor() {
	if len(m.historyEntries) == 0 {
		m.historyCursor = 0

		return
	}

	if m.historyCursor >= len(m.historyEntries) {
		m.historyCursor = len(m.historyEntries) - 1
	}

	if m.historyCursor < 0 {
		m.historyCursor = 0
	}
}

// executeConfirmation executes the pending confirmation operation.
//
// Returns:
//   - Updated model.
//   - Follow-up command, if any.
func (m *model) executeConfirmation() (tea.Model, tea.Cmd) {
	confirmation := m.confirmation

	// The user has confirmed, so the dialog closes as the work starts. Keys are
	// ignored while it runs.
	m.confirmation = confirmNone

	manager := m.historyManager

	var entry *history.HistoryEntry

	if m.historyCursor < len(m.historyEntries) {
		entry = m.historyEntries[m.historyCursor]
	}

	// Clearing the history iterates and permanently deletes every trashed
	// binary, the longest operation in the tool. All confirmations run outside
	// Update so the view keeps rendering and stays interruptible.
	switch confirmation {
	case confirmClearAll:
		if manager == nil {
			return m, nil
		}

		clearCmd := m.runAsyncReporting(
			"clearing history",
			func(ctx context.Context) error {
				if err := manager.ClearHistory(ctx, false); err != nil {
					return fmt.Errorf("clearing history: %w", err)
				}

				return nil
			},
			func(m *model) {
				m.historyEntries = make([]*history.HistoryEntry, 0)
				m.historyCursor = 0
			},
			nil,
			func() string { return "History cleared" },
		)

		return m, clearCmd

	case confirmDeletePerm:
		if manager == nil || entry == nil {
			return m, nil
		}

		entryID := entry.ID
		name := entry.BinaryName

		deleteCmd := m.runAsyncReporting(
			"deleting "+name+" permanently",
			func(ctx context.Context) error {
				if err := manager.DeletePermanently(ctx, entryID); err != nil {
					return fmt.Errorf("deleting permanently: %w", err)
				}

				return nil
			},
			nil,
			nil,
			func() string { return "Permanently deleted " + name },
		)

		return m, deleteCmd
	}

	return m, nil
}

// updateHistoryMode processes key events in history mode.
//
// Parameters:
//   - msg: Key press from the user.
//
// Returns:
//   - Updated model.
//   - Follow-up command, if any.
func (m *model) updateHistoryMode(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit // Exit the TUI

	case "b":
		// Back to binary mode
		m.mode = modeBinaries

		// Clear the history-view status first, so a failed rescan below can set
		// its own rather than being wiped here.
		m.status = ""
		m.refreshChoices()
		m.sortChoices()
		m.updateGrid()

	case "up", "k":
		// Move cursor up in history list
		if m.historyCursor > 0 {
			m.historyCursor--
		}

	case "down", "j":
		// Move cursor down in history list
		if m.historyCursor < len(m.historyEntries)-1 {
			m.historyCursor++
		}

	case "enter":
		// Restore selected entry
		return m.handleRestore()

	case "d":
		// Delete permanently (with confirmation)
		if m.historyCursor < len(m.historyEntries) {
			m.confirmation = confirmDeletePerm
		}

	case "c":
		// Clear this entry (keep in trash)
		return m.handleClearEntry(false)

	case "C":
		// Clear all history (with confirmation)
		if len(m.historyEntries) > 0 {
			m.confirmation = confirmClearAll
		}

	case "u":
		// Undo most recent deletion
		return m.handleUndo()

	case "L":
		// Toggle verbose logging and log panel visibility
		m.toggleVerboseLogging()

		return m, nil
	}

	return m, nil
}

// updateBinaryMode processes key events in binary selection mode.
//
// Parameters:
//   - msg: Key press from the user.
//
// Returns:
//   - Updated model.
//   - Follow-up command, if any.
func (m *model) updateBinaryMode(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit // Exit the TUI

	case "up", "k":
		// Move cursor up, stopping at the top row.
		if m.cursorY > 0 {
			m.cursorY--
		}

	case "down", "j":
		// Move cursor down, respecting grid bounds and item count.
		newY := m.cursorY + 1

		newIdx := newY + m.cursorX*m.rows // Column-major index (fill down columns)
		if newY < m.rows && newIdx < len(m.choices) {
			m.cursorY = newY
		}

	case "left", "h":
		// Move cursor left, stopping at the first column.
		if m.cursorX > 0 {
			m.cursorX--
		}

	case "right", "l":
		// Move cursor right, respecting column bounds and item count.
		newX := m.cursorX + 1

		newIdx := m.cursorY + newX*m.rows // Column-major index
		if newX < m.cols && newIdx < len(m.choices) {
			m.cursorX = newX
		}

	case "s":
		// Toggle sort order and re-sort the choices.
		m.sortAscending = !m.sortAscending
		m.sortChoices()
		m.updateGrid()

	case "L":
		// Toggle verbose logging and log panel visibility.
		m.toggleVerboseLogging()

		return m, nil

	case "r":
		// Switch to history view
		m.mode = modeHistory
		m.historyLoading = true
		m.historyCursor = 0
		m.status = "Loading history..."

		cmd := m.loadHistory()

		return m, cmd

	case "u":
		// Undo most recent deletion
		return m.handleUndo()

	case "enter":
		// Remove the selected binary and update the TUI state.
		if len(m.choices) > 0 {
			idx := m.cursorY + m.cursorX*m.rows // Column-major index
			if idx < len(m.choices) {
				binaryPath := m.fs.AdjustBinaryPath(m.dir, m.choices[idx])
				name := m.choices[idx]
				manager := m.historyManager
				filesystem := m.fs
				verbose := m.config.Verbose
				log := m.logger

				// Moving the binary can take a while on a large file, so it runs
				// outside Update to keep the view responsive and interruptible.
				removeCmd := m.runAsync(
					"removing "+name,
					func(ctx context.Context) error {
						// Use history manager if available (it handles trash
						// plus history).
						if manager != nil {
							if _, err := manager.RecordDeletion(ctx, binaryPath); err != nil {
								return fmt.Errorf("recording %s: %w", name, err)
							}

							return nil
						}

						// Fallback: permanent delete only without a manager.
						if err := filesystem.RemoveBinary(
							binaryPath,
							name,
							verbose,
							log,
						); err != nil {
							return fmt.Errorf("removing %s: %w", name, err)
						}

						return nil
					},
					func(m *model) {
						// Drop the deleted entry before rescanning, so a failed
						// rescan cannot leave the view showing a binary that is
						// already gone.
						m.choices = slices.DeleteFunc(m.choices, func(choice string) bool {
							return choice == name
						})
						m.refreshChoices()
						m.sortChoices()

						// Adjust cursor if it exceeds remaining choices.
						if m.cursorY+m.cursorX*m.rows >= len(m.choices) {
							lastIdx := len(m.choices) - 1
							m.cursorX = lastIdx / m.rows
							m.cursorY = lastIdx % m.rows
						}

						m.updateGrid()
						m.status = "Removed " + name
					},
				)

				return m, removeCmd
			}
		}
	}

	return m, nil
}

// handleRestore restores the selected history entry.
//
// Returns:
//   - Updated model.
//   - Command to refresh history after a successful restore.
func (m *model) handleRestore() (tea.Model, tea.Cmd) {
	if m.historyManager == nil || m.historyCursor >= len(m.historyEntries) {
		m.status = "No history entry selected"

		return m, nil
	}

	entry := m.historyEntries[m.historyCursor]

	// Check if entry can be restored
	if !entry.InTrash {
		m.status = fmt.Sprintf("Cannot restore %s: not available in trash", entry.BinaryName)

		return m, nil
	}

	name := entry.BinaryName
	manager := m.historyManager

	// The destination is only known to the operation, which reads it from the
	// result on its own goroutine.
	var restoredTo string

	// The restore touches the filesystem and the history store, so it runs
	// outside Update to keep the view responsive.
	restoreCmd := m.runAsyncReporting(
		"restoring "+name,
		func(ctx context.Context) error {
			result, err := manager.Restore(ctx, entry.ID)
			if err != nil {
				return fmt.Errorf("restoring %s: %w", name, err)
			}

			if result != nil {
				restoredTo = fmt.Sprintf("Restored %s to %s", result.BinaryName, result.RestoredTo)
			}

			return nil
		},
		func(m *model) {
			m.refreshChoices()
			m.sortChoices()
			m.updateGrid()
		},
		restoreErrorStatus(name),
		func() string { return restoredTo },
	)

	return m, restoreCmd
}

// handleUndo restores the most recently deleted binary.
//
// Returns:
//   - Updated model.
//   - Command to refresh history when viewing history mode.
func (m *model) handleUndo() (tea.Model, tea.Cmd) {
	if m.historyManager == nil {
		m.status = "History manager not available"

		return m, nil
	}

	manager := m.historyManager

	// The destination is only known to the operation, which reads it from the
	// result on its own goroutine.
	var restoredTo string

	// Undo walks the history in pages and restores from the trash, so it runs
	// outside Update to keep the view responsive and interruptible.
	undoCmd := m.runAsyncReporting(
		"undoing the most recent deletion",
		func(ctx context.Context) error {
			result, err := manager.UndoMostRecent(ctx)
			if err != nil {
				return fmt.Errorf("undo failed: %w", err)
			}

			if result != nil {
				restoredTo = fmt.Sprintf("Restored %s to %s", result.BinaryName, result.RestoredTo)
			}

			return nil
		},
		func(m *model) {
			// Refresh history and binaries if in binary mode.
			if m.mode == modeBinaries {
				m.refreshChoices()
				m.sortChoices()
				m.updateGrid()
			}
		},
		func(err error) string {
			switch {
			case errors.Is(err, history.ErrNoHistory):
				return "No deletion history found - nothing to undo"
			case errors.Is(err, history.ErrAlreadyRestored):
				return "Binary has already been restored"
			case errors.Is(err, history.ErrNotInTrash):
				return "Binary is no longer in trash - cannot restore"
			case errors.Is(err, history.ErrRestoreCollision):
				return "A file already exists at the restore location"
			default:
				// The error is already wrapped with the operation, and a
				// status line reads better capitalised.
				msg := err.Error()
				if msg == "" {
					return "Undo failed"
				}

				return strings.ToUpper(msg[:1]) + msg[1:]
			}
		},
		func() string { return restoredTo },
	)

	return m, undoCmd
}

// handleClearEntry removes a history entry and optionally deletes it from trash.
//
// Parameters:
//   - deleteFromTrash: When true, also permanently delete the trashed binary.
//
// Returns:
//   - Updated model.
//   - Command to refresh history after a successful clear.
func (m *model) handleClearEntry(deleteFromTrash bool) (tea.Model, tea.Cmd) {
	if m.historyManager == nil || m.historyCursor >= len(m.historyEntries) {
		m.status = "No history entry selected"

		return m, nil
	}

	entry := m.historyEntries[m.historyCursor]
	ctx := m.context()

	if err := m.historyManager.ClearEntry(ctx, entry.ID, deleteFromTrash); err != nil {
		m.status = fmt.Sprintf("Error clearing entry: %v", err)
	} else {
		if deleteFromTrash {
			m.status = fmt.Sprintf("Permanently deleted %s and cleared history", entry.BinaryName)
		} else {
			m.status = "Cleared history entry for " + entry.BinaryName
		}
		// Refresh history
		cmd := m.loadHistory()

		return m, cmd
	}

	return m, nil
}

// addLogEntry adds a log message to the circular buffer.
//
// Parameters:
//   - msg: Log message to display.
func (m *model) addLogEntry(msg LogMsg) {
	// Format the log entry: "[LEVEL] message"
	entry := fmt.Sprintf("[%s] %s", msg.Level, msg.Message)

	// Add to logs slice
	m.logs = append(m.logs, entry)

	// Maintain circular buffer size
	if len(m.logs) > maxLogLines {
		m.logs = m.logs[len(m.logs)-maxLogLines:]
	}
}

// getVisibleLogs returns the last log lines that fit in the panel.
//
// Returns:
//   - Visible log lines, or nil when the log panel is hidden.
func (m *model) getVisibleLogs() []string {
	if !m.showLogs {
		return nil
	}

	// Return the last maxVisibleLogLines entries
	start := max(len(m.logs)-maxVisibleLogLines, 0)

	return m.logs[start:]
}

// sortChoices sorts the binary list according to the current sort order.
func (m *model) sortChoices() {
	if len(m.choices) == 0 {
		return
	}

	slices.Sort(m.choices)

	if !m.sortAscending {
		slices.Reverse(m.choices)
	}
}

// updateGrid recalculates the grid layout from the current state and terminal size.
func (m *model) updateGrid() {
	// Determine the maximum length of binary names for column sizing.
	maxNameLen := 0
	for _, choice := range m.choices {
		if len(choice) > maxNameLen {
			maxNameLen = len(choice)
		}
	}

	// Calculate column width and available space for the grid.
	colWidth := maxNameLen + colWidthPadding
	availWidth := m.width - availWidthAdjustment

	// Account for status line when calculating available height.
	// The status line takes 1 line when present, but minAvailHeightAdjustment
	// is a constant that doesn't account for dynamic status display.
	statusAdjustment := 0
	if m.status != "" {
		statusAdjustment = 1
	}

	availHeight := max(m.height-minAvailHeightAdjustment-statusAdjustment, 1)

	// Adjust available height for log panel if visible
	if m.showLogs {
		// Reserve up to maxVisibleLogLines lines for log panel (plus separator lines)
		visibleLogCount := min(len(m.logs), maxVisibleLogLines)
		if visibleLogCount == 0 {
			// Empty log panel: header + placeholder
			visibleLogCount = 1
		}

		logPanelHeight := visibleLogCount + logPanelSeparatorLines
		availHeight = max(availHeight-logPanelHeight, 1)
	}

	// Clear grid if no choices remain.
	if len(m.choices) == 0 {
		m.rows = 0
		m.cols = 0
		m.cursorX = 0
		m.cursorY = 0

		return
	}

	// Compute grid dimensions: maximize rows, limit columns by width.
	maxCols := max(availWidth/colWidth, 1)

	m.rows = min(availHeight, len(m.choices))
	if m.rows == 0 {
		m.rows = 1 // Ensure at least one row
	}

	m.cols = min(maxCols, (len(m.choices)+m.rows-1)/m.rows)

	// Clamp cursor position to valid bounds after resizing.
	if m.cursorX >= m.cols {
		m.cursorX = m.cols - 1
	}

	if m.cursorY >= m.rows {
		m.cursorY = m.rows - 1
	}

	currentIdx := m.cursorY + m.cursorX*m.rows
	if currentIdx >= len(m.choices) {
		lastIdx := len(m.choices) - 1
		m.cursorX = lastIdx / m.rows
		m.cursorY = lastIdx % m.rows
	}
}

// View renders the TUI interface.
//
// Returns:
//   - Bubble Tea view for the current mode.
func (m *model) View() tea.View {
	if m.mode == modeHistory {
		return m.viewHistory()
	}

	return m.viewBinaries()
}

// viewBinaries renders the binary selection view.
//
// Returns:
//   - Bubble Tea view listing binaries in a grid.
func (m *model) viewBinaries() tea.View {
	if len(m.choices) == 0 {
		view := tea.NewView("No binaries found.\n")
		view.AltScreen = true

		return view
	}

	// Apply configured styles for UI elements.
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.styles.TitleColor))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.CursorColor))
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.FooterColor))
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.StatusColor))
	logStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.LogColor))

	// Calculate column width based on the longest binary name.
	var maxNameLen int
	for _, choice := range m.choices {
		if len(choice) > maxNameLen {
			maxNameLen = len(choice)
		}
	}

	colWidth := maxNameLen + colWidthPadding

	// Build the grid of binary choices with cursor highlighting.
	var grid strings.Builder

	for row := range m.rows {
		for col := range m.cols {
			idx := row + col*m.rows // Column-major index (fill down columns)
			if idx >= len(m.choices) {
				break
			}

			prefix := "  "
			if row == m.cursorY && col == m.cursorX {
				prefix = cursorStyle.Render(m.styles.Cursor)
			}

			item := m.choices[idx]
			visibleLen := visibleLenPrefix + len([]rune(item))
			padding := max(colWidth-visibleLen, 0)
			cell := prefix + item + strings.Repeat(" ", padding)
			grid.WriteString(cell)
		}

		grid.WriteString("\n")
	}

	// Assemble the full TUI layout: title, grid, logs (if visible), status, and footer.
	var s strings.Builder

	s.WriteString(titleStyle.Render("Select a binary to remove:\n"))
	s.WriteString("\n")
	s.WriteString(grid.String())
	s.WriteString("\n")

	// Render log panel if enabled
	if m.showLogs {
		visibleLogs := m.getVisibleLogs()

		s.WriteString(logStyle.Render("─ Log Messages ─"))
		s.WriteString("\n")

		if len(visibleLogs) == 0 {
			s.WriteString(logStyle.Render("No log messages yet"))
			s.WriteString("\n")
		} else {
			for _, logEntry := range visibleLogs {
				s.WriteString(logStyle.Render(logEntry))
				s.WriteString("\n")
			}
		}

		s.WriteString("\n")
	}

	switch {
	case m.busy != "":
		s.WriteString(statusStyle.Render("Working: " + m.busy + " (ctrl+c to stop)"))
		s.WriteString("\n")
	case m.status != "":
		s.WriteString(statusStyle.Render(m.status))
		s.WriteString("\n")
	}

	// Update footer to include new key bindings
	footerText := "↑/k: up  ↓/j: down  ←/h: left  →/l: right  Enter: remove  s: sort  r: history  u: undo  L: logs  q: quit"
	footer := footerStyle.Render(footerText)

	// Pad between the content and the footer from what actually renders, rather
	// than from a hand-counted total that drifts as soon as the layout does.
	// The measurement uses the same style as the render, because left padding
	// and the width can change how many lines the result occupies, and the
	// padding goes inside the body so every line keeps its width.
	frame := lipgloss.NewStyle().
		PaddingLeft(leftPadding).
		Width(m.width - leftPadding)

	body := s.String()
	pad := max(m.height-lipgloss.Height(frame.Render(body+footer)), 0)

	if pad > 0 {
		body += strings.Repeat("\n", pad)
	}

	content := frame.Render(body + footer)

	view := tea.NewView(content)
	view.AltScreen = true

	return view
}

// viewHistory renders the history view.
//
// Returns:
//   - Bubble Tea view listing deletion history.
func (m *model) viewHistory() tea.View {
	// Apply configured styles for UI elements.
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.styles.TitleColor))
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.styles.HistoryColor))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.CursorColor))
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.FooterColor))
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.StatusColor))
	logStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.LogColor))
	trashYesStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.TrashYesColor))
	trashNoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.TrashNoColor))

	var s strings.Builder

	s.WriteString(titleStyle.Render("Deletion History\n"))
	s.WriteString("\n")

	// Calculate visible count first for use in both rendering and height calculation
	var (
		visibleCount      int
		entryCount        int
		maxVisibleEntries int
	)

	// Show loading state or history entries

	switch {
	case m.historyLoading:
		s.WriteString("Loading history...\n")
	case len(m.historyEntries) == 0:
		s.WriteString("No deletion history found.\n")
	default:
		// Calculate column widths
		dateWidth := 16
		nameWidth := 20
		trashWidth := 12

		// Table header
		header := fmt.Sprintf("%-*s %-*s %-*s",
			dateWidth, "Date/Time",
			nameWidth, "Binary",
			trashWidth, "In Trash")
		s.WriteString(headerStyle.Render(header))
		s.WriteString("\n")
		s.WriteString(strings.Repeat("─", dateWidth+nameWidth+trashWidth+separatorAdjustment))
		s.WriteString("\n")

		// Calculate available height for history entries
		// Reserve space for: title(2) + header(2) + footer(1) + status(1) + padding(2)
		reservedHeight := 8

		if m.showLogs {
			// Reserve additional space for log panel (header + separator + lines)
			visibleLogCount := min(len(m.logs), maxVisibleLogLines)
			if visibleLogCount == 0 {
				visibleLogCount = 1 // Placeholder line
			}

			reservedHeight += visibleLogCount + logPanelSeparatorLines
		}

		maxVisibleEntries = max(m.height-reservedHeight, 1)
		entryCount = len(m.historyEntries)
		visibleCount = min(entryCount, maxVisibleEntries)

		// Adjust if we need to show "...and X more" message
		showMoreIndicator := entryCount > maxVisibleEntries
		if showMoreIndicator && visibleCount > 0 {
			visibleCount--
		}

		// Ensure cursor is within visible range
		// This will scroll the view as needed
		startIdx := 0
		if m.historyCursor >= visibleCount {
			// If cursor is below visible range, adjust start index
			startIdx = m.historyCursor - visibleCount + 1
			// Recalculate visible count based on new start
			visibleCount = min(entryCount-startIdx, maxVisibleEntries)
			if showMoreIndicator && visibleCount > 0 {
				visibleCount--
			}
		}

		// Table rows - display only visible entries
		for i := range visibleCount {
			entryIdx := startIdx + i
			if entryIdx >= entryCount {
				break
			}

			entry := m.historyEntries[entryIdx]

			prefix := "  "
			if entryIdx == m.historyCursor {
				prefix = cursorStyle.Render(m.styles.Cursor)
			}

			dateStr := entry.Timestamp.Format(dateTimeFormat)

			nameStr := entry.BinaryName
			if len(nameStr) > nameWidth {
				nameStr = nameStr[:nameWidth-3] + "..."
			}

			var trashStr string
			if entry.InTrash {
				trashStr = trashYesStyle.Render("Yes")
			} else {
				trashStr = trashNoStyle.Render("No")
			}

			row := fmt.Sprintf("%-*s %-*s %s",
				dateWidth, dateStr,
				nameWidth, nameStr,
				trashStr)
			s.WriteString(prefix + row)
			s.WriteString("\n")
		}

		// Show indicator if there are more entries
		if showMoreIndicator {
			remaining := entryCount - startIdx - visibleCount
			moreMsg := fmt.Sprintf("...and %d more", remaining)
			s.WriteString(footerStyle.Render(moreMsg))
			s.WriteString("\n")
		}
	}

	s.WriteString("\n")

	// Render log panel if enabled
	if m.showLogs {
		visibleLogs := m.getVisibleLogs()

		s.WriteString(logStyle.Render("─ Log Messages ─"))
		s.WriteString("\n")

		if len(visibleLogs) == 0 {
			s.WriteString(logStyle.Render("No log messages yet"))
			s.WriteString("\n")
		} else {
			for _, logEntry := range visibleLogs {
				s.WriteString(logStyle.Render(logEntry))
				s.WriteString("\n")
			}
		}

		s.WriteString("\n")
	}

	// Show confirmation dialog if active
	switch m.confirmation {
	case confirmClearAll:
		s.WriteString(statusStyle.Render("Clear all history? This cannot be undone. (y/n)"))
		s.WriteString("\n")
	case confirmDeletePerm:
		if m.historyCursor < len(m.historyEntries) {
			entry := m.historyEntries[m.historyCursor]
			s.WriteString(
				statusStyle.Render(fmt.Sprintf("Permanently delete %s? (y/n)", entry.BinaryName)),
			)
			s.WriteString("\n")
		}
	default:
		if m.status != "" {
			s.WriteString(statusStyle.Render(m.status))
			s.WriteString("\n")
		}
	}

	// Footer with history-specific key bindings
	var footerText string

	switch {
	case m.confirmation != confirmNone:
		footerText = "y: confirm  n: cancel"
	default:
		footerText = "↑/k: up  ↓/j: down  Enter: restore  d: delete  c: clear entry  C: clear all  b: back  u: undo  L: logs  q: quit"
	}

	footer := footerStyle.Render(footerText)

	// Pad between the content and the footer from what actually renders, rather
	// than from a hand-counted total that drifts as soon as the layout does.
	// The measurement uses the same style as the render, because left padding
	// and the width can change how many lines the result occupies, and the
	// padding goes inside the body so every line keeps its width.
	frame := lipgloss.NewStyle().
		PaddingLeft(leftPadding).
		Width(m.width - leftPadding)

	body := s.String()
	pad := max(m.height-lipgloss.Height(frame.Render(body+footer)), 0)

	if pad > 0 {
		body += strings.Repeat("\n", pad)
	}

	content := frame.Render(body + footer)

	view := tea.NewView(content)
	view.AltScreen = true

	return view
}
