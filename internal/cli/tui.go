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
	"sort"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/rs/zerolog"

	tea "charm.land/bubbletea/v2"

	"github.com/nicholas-fedor/go-remove/internal/fs"
	"github.com/nicholas-fedor/go-remove/internal/history"
	"github.com/nicholas-fedor/go-remove/internal/logger"
)

// Layout constants for TUI rendering.
// These constants must be kept consistent between updateGrid() and the view functions.
const (
	colWidthPadding          = 3                  // Padding added to column width for spacing
	availWidthAdjustment     = 4                  // Adjustment to width for border and padding
	minAvailHeightAdjustment = 8                  // Minimum height adjustment for UI elements (title + footer + padding)
	visibleLenPrefix         = 2                  // Prefix length for cursor visibility
	totalHeightBase          = 8                  // Base height for non-grid UI components (must match minAvailHeightAdjustment)
	footerHeight             = 1                  // Height reserved for footer/instructions
	leftPadding              = 2                  // Left padding for the entire TUI
	maxLogLines              = 50                 // Maximum number of log lines to retain
	maxVisibleLogLines       = 5                  // Maximum number of log lines to display
	logPanelSeparatorLines   = 2                  // Number of separator lines for log panel
	maxHistoryEntries        = 100                // Maximum number of history entries to display
	dateTimeFormat           = "2006-01-02 15:04" // Format for displaying timestamps
	separatorAdjustment      = 2                  // Extra width for column separator
	baseContentHeight        = 3                  // Base height for content area (title + empty lines)
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

// LogMsg is a Bubble Tea message that carries a log entry to be displayed in the TUI.
type LogMsg struct {
	Level   string // Log level (e.g., "DBG", "INF", "WRN", "ERR")
	Message string // Log message content
}

// HistoryMsg is a Bubble Tea message that signals history entries have been loaded.
type HistoryMsg struct {
	Entries []*history.HistoryEntry
	Error   error
}

// ProgramRunner defines an interface for running Bubbletea programs.
type ProgramRunner interface {
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
type model struct {
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

// NewLogMsg creates a new LogMsg for sending log entries to the TUI.
func NewLogMsg(level, message string) LogMsg {
	return LogMsg{Level: level, Message: message}
}

// RunTUI launches the interactive TUI mode for binary selection and removal.
func RunTUI(
	dir string,
	config Config,
	log logger.Logger,
	filesystem fs.FS,
	runner ProgramRunner,
	historyMgr history.Manager,
) error {
	// Fetch available binaries from the specified directory.
	choices := filesystem.ListBinaries(dir)
	if len(choices) == 0 && !config.RestoreMode {
		return fmt.Errorf("%w: %s", ErrNoBinariesFound, dir)
	}

	// Initialize the model with default styles.
	// Enable log visibility by default when verbose mode is active.
	m := &model{
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

	// Start the TUI program.
	program, err := runner.RunProgram(m)
	if err != nil {
		return fmt.Errorf("failed to start TUI program: %w", err)
	}

	// Allow mocked runners to return nil for testing purposes.
	if program == nil {
		return nil
	}

	// Run the program and capture any runtime errors.
	_, err = program.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI program: %w", err)
	}

	return nil
}

// setupLogCapture configures the logger's capture callback to send messages to the TUI.
// This method is idempotent and can be called multiple times safely.
func (m *model) setupLogCapture(log logger.Logger) {
	log.SetCaptureFunc(func(level, msg string) {
		// Send to channel without blocking.
		// If channel is full, the message is dropped to prevent blocking.
		select {
		case m.logChan <- NewLogMsg(level, msg):
		default:
			// Channel is full, message is dropped to prevent blocking.
		}
	})
}

// toggleVerboseLogging toggles verbose logging mode and log panel visibility.
// It updates both the logger level (Info <-> Debug) and the showLogs state.
// Returns a command to start polling for logs if verbose mode is being enabled.
func (m *model) toggleVerboseLogging() tea.Cmd {
	m.showLogs = !m.showLogs

	if m.showLogs {
		// Enable verbose logging by setting level to Debug
		m.logger.Level(zerolog.DebugLevel)
	} else {
		// Disable verbose logging by setting level to Info
		m.logger.Level(zerolog.InfoLevel)
	}

	// Ensure log capture is set up (lazy initialization for backwards compatibility)
	if m.logChan == nil {
		m.logChan = make(chan LogMsg, maxLogLines)
		m.setupLogCapture(m.logger)
	}

	// Start polling for logs if we're now in verbose mode
	if m.showLogs {
		return m.pollLogChannel()
	}

	// Return a no-op command instead of nil for consistency
	return tea.Batch()
}

// defaultStyleConfig provides default TUI style settings.
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

// RunProgram launches a Bubbletea program with the given model and options.
func (r DefaultRunner) RunProgram(m tea.Model, opts ...tea.ProgramOption) (*tea.Program, error) {
	program := tea.NewProgram(m, opts...)

	return program, nil
}

// Init prepares the TUI model for rendering.
func (m *model) Init() tea.Cmd {
	m.sortChoices()

	// Load history if in history mode
	if m.mode == modeHistory {
		cmd := m.loadHistory()

		// Start log polling if verbose mode is enabled
		if m.config.Verbose {
			return tea.Batch(
				cmd,
				m.pollLogChannel(),
			)
		}

		return cmd
	}

	// Start log polling if verbose mode is enabled.
	if m.config.Verbose {
		return m.pollLogChannel()
	}

	return nil
}

// loadHistory returns a command that loads deletion history.
func (m *model) loadHistory() tea.Cmd {
	return func() tea.Msg {
		// Check if history manager is available
		if m.historyManager == nil {
			return HistoryMsg{Entries: nil, Error: ErrHistoryNotInitialized}
		}

		ctx := context.Background()
		entries, err := m.historyManager.GetHistory(ctx, maxHistoryEntries)

		return HistoryMsg{Entries: entries, Error: err}
	}
}

// pollLogChannel returns a command that polls for log messages.
// This approach avoids deadlocks by having the TUI poll the channel
// rather than trying to Send() from within the capture callback.
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

// pollLogTickMsg is a message sent when it's time to poll for logs again.
type pollLogTickMsg struct{}

// Update processes TUI events and updates the model state.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle confirmation dialog first if active
		if m.confirmation != confirmNone {
			return m.handleConfirmation(msg)
		}

		// Handle mode-specific key bindings
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
		if msg.Error != nil {
			m.status = fmt.Sprintf("Error loading history: %v", msg.Error)
		} else {
			m.historyEntries = msg.Entries
			if len(m.historyEntries) == 0 {
				m.status = "No deletion history found"
			} else {
				m.status = fmt.Sprintf("Loaded %d history entries", len(m.historyEntries))
			}
		}

		return m, nil
	}

	return m, nil
}

// handleConfirmation processes key presses during confirmation dialogs.
func (m *model) handleConfirmation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// Confirmed - execute the operation
		return m.executeConfirmation()
	case "n", "N", "q", "esc":
		// Cancelled - clear confirmation
		m.confirmation = confirmNone
		m.status = "Operation cancelled"
	}

	return m, nil
}

// executeConfirmation executes the pending confirmation operation.
func (m *model) executeConfirmation() (tea.Model, tea.Cmd) {
	ctx := context.Background()

	switch m.confirmation {
	case confirmClearAll:
		if m.historyManager != nil {
			if err := m.historyManager.ClearHistory(ctx, false); err != nil {
				m.status = fmt.Sprintf("Error clearing history: %v", err)
			} else {
				m.status = "History cleared"
				m.historyEntries = make([]*history.HistoryEntry, 0)
				m.historyCursor = 0
			}
		}

	case confirmDeletePerm:
		if m.historyManager != nil && m.historyCursor < len(m.historyEntries) {
			entry := m.historyEntries[m.historyCursor]
			if err := m.historyManager.DeletePermanently(ctx, entry.ID); err != nil {
				m.status = fmt.Sprintf("Error deleting permanently: %v", err)
			} else {
				m.status = "Permanently deleted " + entry.BinaryName
				// Clear confirmation state before refreshing history
				m.confirmation = confirmNone
				// Refresh history
				cmd := m.loadHistory()

				return m, cmd
			}
		}
	}

	m.confirmation = confirmNone

	return m, nil
}

// updateHistoryMode processes key events in history mode.
func (m *model) updateHistoryMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit // Exit the TUI

	case "b":
		// Back to binary mode
		m.mode = modeBinaries
		m.choices = m.fs.ListBinaries(m.dir)
		m.sortChoices()
		m.updateGrid()
		m.status = ""

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
		cmd := m.toggleVerboseLogging()

		return m, cmd
	}

	return m, nil
}

// updateBinaryMode processes key events in binary selection mode.
func (m *model) updateBinaryMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		cmd := m.toggleVerboseLogging()

		return m, cmd

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

				// Use history manager if available (it handles trash + history)
				if m.historyManager != nil {
					ctx := context.Background()
					if _, err := m.historyManager.RecordDeletion(ctx, binaryPath); err != nil {
						m.status = fmt.Sprintf("Error recording %s: %v", name, err)

						return m, nil
					}
				} else {
					// Fallback: permanent delete only if no history manager
					if err := m.fs.RemoveBinary(
						binaryPath,
						name,
						m.config.Verbose,
						m.logger,
					); err != nil {
						m.status = fmt.Sprintf("Error removing %s: %v", name, err)

						return m, nil
					}
				}

				m.status = "Removed " + name
				m.choices = m.fs.ListBinaries(m.dir)
				m.sortChoices()

				// Exit if no binaries remain.
				if len(m.choices) == 0 {
					return m, tea.Quit
				}

				// Adjust cursor if it exceeds remaining choices.
				if m.cursorY+m.cursorX*m.rows >= len(m.choices) {
					lastIdx := len(m.choices) - 1
					m.cursorX = lastIdx / m.rows
					m.cursorY = lastIdx % m.rows
				}

				m.updateGrid()
			}
		}
	}

	return m, nil
}

// handleRestore handles restoring the selected history entry.
func (m *model) handleRestore() (tea.Model, tea.Cmd) {
	if m.historyManager == nil || m.historyCursor >= len(m.historyEntries) {
		m.status = "No history entry selected"

		return m, nil
	}

	entry := m.historyEntries[m.historyCursor]

	// Check if entry can be restored
	if !entry.CanRestore {
		m.status = fmt.Sprintf("Cannot restore %s: not available in trash", entry.BinaryName)

		return m, nil
	}

	ctx := context.Background()

	result, err := m.historyManager.Restore(ctx, entry.ID)
	if err != nil {
		switch {
		case errors.Is(err, history.ErrAlreadyRestored):
			m.status = entry.BinaryName + " has already been restored"
		case errors.Is(err, history.ErrNotInTrash):
			m.status = entry.BinaryName + " is no longer in trash"
		case errors.Is(err, history.ErrRestoreCollision):
			m.status = fmt.Sprintf("Cannot restore %s: file already exists", entry.BinaryName)
		default:
			m.status = fmt.Sprintf("Error restoring %s: %v", entry.BinaryName, err)
		}
	} else {
		m.status = fmt.Sprintf("Restored %s to %s", result.BinaryName, result.RestoredTo)
		// Refresh the binary list to include the restored binary
		m.choices = m.fs.ListBinaries(m.dir)
		m.sortChoices()
		m.updateGrid()
		// Refresh history to update trash status
		cmd := m.loadHistory()

		return m, cmd
	}

	return m, nil
}

// handleUndo handles undoing the most recent deletion.
func (m *model) handleUndo() (tea.Model, tea.Cmd) {
	if m.historyManager == nil {
		m.status = "History manager not available"

		return m, nil
	}

	ctx := context.Background()

	result, err := m.historyManager.UndoMostRecent(ctx)
	if err != nil {
		switch {
		case errors.Is(err, history.ErrNoHistory):
			m.status = "No deletion history found - nothing to undo"
		case errors.Is(err, history.ErrAlreadyRestored):
			m.status = "Binary has already been restored"
		case errors.Is(err, history.ErrNotInTrash):
			m.status = "Binary is no longer in trash - cannot restore"
		case errors.Is(err, history.ErrRestoreCollision):
			m.status = "A file already exists at the restore location"
		default:
			m.status = fmt.Sprintf("Undo failed: %v", err)
		}
	} else {
		m.status = fmt.Sprintf("Restored %s to %s", result.BinaryName, result.RestoredTo)
		// Refresh history and binaries if in binary mode
		if m.mode == modeBinaries {
			m.choices = m.fs.ListBinaries(m.dir)
			m.sortChoices()
			m.updateGrid()
		}
		// Refresh history view if in history mode
		if m.mode == modeHistory {
			cmd := m.loadHistory()

			return m, cmd
		}
	}

	return m, nil
}

// handleClearEntry handles clearing a history entry.
func (m *model) handleClearEntry(deleteFromTrash bool) (tea.Model, tea.Cmd) {
	if m.historyManager == nil || m.historyCursor >= len(m.historyEntries) {
		m.status = "No history entry selected"

		return m, nil
	}

	entry := m.historyEntries[m.historyCursor]
	ctx := context.Background()

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

// addLogEntry adds a log message to the circular buffer, maintaining maxLogLines limit.
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

// getVisibleLogs returns the last N log lines that fit within the available height.
func (m *model) getVisibleLogs() []string {
	if !m.showLogs {
		return nil
	}

	// Return the last maxVisibleLogLines entries
	start := max(len(m.logs)-maxVisibleLogLines, 0)

	return m.logs[start:]
}

// sortChoices sorts the choices based on the current sort order.
func (m *model) sortChoices() {
	if len(m.choices) == 0 {
		return
	}

	if m.sortAscending {
		sort.Strings(m.choices)
	} else {
		sort.Sort(sort.Reverse(sort.StringSlice(m.choices)))
	}
}

// updateGrid recalculates the grid layout based on current state and terminal size.
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

	availHeight := maximum(m.height-minAvailHeightAdjustment-statusAdjustment, 1)

	// Adjust available height for log panel if visible
	if m.showLogs {
		// Reserve up to maxVisibleLogLines lines for log panel (plus separator lines)
		visibleLogCount := minimum(len(m.logs), maxVisibleLogLines)
		if visibleLogCount == 0 {
			// Empty log panel: header + placeholder
			visibleLogCount = 1
		}

		logPanelHeight := visibleLogCount + logPanelSeparatorLines
		availHeight = maximum(availHeight-logPanelHeight, 1)
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
	maxCols := maximum(availWidth/colWidth, 1)

	m.rows = minimum(availHeight, len(m.choices))
	if m.rows == 0 {
		m.rows = 1 // Ensure at least one row
	}

	m.cols = minimum(maxCols, (len(m.choices)+m.rows-1)/m.rows)

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

// View renders the TUI interface as a tea.View.
func (m *model) View() tea.View {
	if m.mode == modeHistory {
		return m.viewHistory()
	}

	return m.viewBinaries()
}

// viewBinaries renders the binary selection view.
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
			padding := maximum(colWidth-visibleLen, 0)
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

	if m.status != "" {
		s.WriteString(statusStyle.Render(m.status))
		s.WriteString("\n")
	}

	// Update footer to include new key bindings
	footerText := "↑/k: up  ↓/j: down  ←/h: left  →/l: right  Enter: remove  s: sort  r: history  u: undo  L: logs  q: quit"
	footer := footerStyle.Render(footerText)

	lenStatus := 0
	if m.status != "" {
		lenStatus = 1
	}

	// Account for log panel in height calculation
	logPanelLines := 0

	if m.showLogs {
		visibleLogs := m.getVisibleLogs()
		if len(visibleLogs) == 0 {
			// Empty log panel: header + placeholder + separator
			logPanelLines = 1 + logPanelSeparatorLines
		} else {
			// Log panel header + log lines + separator
			logPanelLines = len(visibleLogs) + logPanelSeparatorLines
		}
	}

	totalHeight := m.rows + totalHeightBase + lenStatus + logPanelLines

	// Add padding lines to fill the terminal height.
	for i := totalHeight; i < m.height; i++ {
		s.WriteString("\n")
	}

	s.WriteString(footer)

	content := lipgloss.NewStyle().
		PaddingLeft(leftPadding).
		Width(m.width - leftPadding).
		Render(s.String())

	view := tea.NewView(content)
	view.AltScreen = true

	return view
}

// viewHistory renders the history view.
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
			visibleLogCount := minimum(len(m.logs), maxVisibleLogLines)
			if visibleLogCount == 0 {
				visibleLogCount = 1 // Placeholder line
			}

			reservedHeight += visibleLogCount + logPanelSeparatorLines
		}

		maxVisibleEntries = maximum(m.height-reservedHeight, 1)
		entryCount = len(m.historyEntries)
		visibleCount = minimum(entryCount, maxVisibleEntries)

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
			visibleCount = minimum(entryCount-startIdx, maxVisibleEntries)
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

	// Calculate total height using actual displayed rows, not adjusted visibleCount
	// The visibleCount variable may have been decremented for the "show more" indicator,
	// so we calculate the actual displayed rows separately
	contentHeight := baseContentHeight

	if !m.historyLoading && len(m.historyEntries) > 0 {
		// Calculate actual displayed rows (before visibleCount was potentially decremented)
		actualVisibleCount := minimum(entryCount, maxVisibleEntries)
		contentHeight += actualVisibleCount + historyTableHeaderLines
		// Account for the "...and X more" indicator line if it will be displayed
		if entryCount > maxVisibleEntries {
			contentHeight++
		}
	}

	lenStatus := 0
	if m.status != "" || m.confirmation != confirmNone {
		lenStatus = 1
	}

	// Account for log panel in height calculation
	logPanelLines := 0

	if m.showLogs {
		visibleLogs := m.getVisibleLogs()
		if len(visibleLogs) == 0 {
			// Empty log panel: header + placeholder + separator
			logPanelLines = 1 + logPanelSeparatorLines
		} else {
			logPanelLines = len(visibleLogs) + logPanelSeparatorLines
		}
	}

	totalHeight := contentHeight + totalHeightBase + lenStatus + logPanelLines

	// Add padding lines to fill the terminal height.
	for i := totalHeight; i < m.height; i++ {
		s.WriteString("\n")
	}

	s.WriteString(footer)

	content := lipgloss.NewStyle().
		PaddingLeft(leftPadding).
		Width(m.width - leftPadding).
		Render(s.String())

	view := tea.NewView(content)
	view.AltScreen = true

	return view
}

// maximum returns the larger of two integers.
func maximum(a, b int) int {
	if a > b {
		return a
	}

	return b
}

// minimum returns the smaller of two integers.
func minimum(a, b int) int {
	if a < b {
		return a
	}

	return b
}
