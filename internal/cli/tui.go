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

// Package cli provides core logic for the go-remove command-line interface.
package cli

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"

	"github.com/nicholas-fedor/go-remove/internal/fs"
	"github.com/nicholas-fedor/go-remove/internal/logger"
)

// Layout constants for TUI rendering.
const (
	colWidthPadding          = 3  // Padding added to column width for spacing
	availWidthAdjustment     = 4  // Adjustment to width for border and padding
	minAvailHeightAdjustment = 7  // Minimum height adjustment for UI elements
	visibleLenPrefix         = 2  // Prefix length for cursor visibility
	totalHeightBase          = 4  // Base height for non-grid UI components
	leftPadding              = 2  // Left padding for the entire TUI
	maxLogLines              = 50 // Maximum number of log lines to retain
	maxVisibleLogLines       = 5  // Maximum number of log lines to display
	logPanelSeparatorLines   = 2  // Number of separator lines for log panel
)

// ErrNoBinariesFound signals that no binaries were found in the target directory.
var ErrNoBinariesFound = errors.New("no binaries found in directory")

// LogMsg is a Bubble Tea message that carries a log entry to be displayed in the TUI.
type LogMsg struct {
	Level   string // Log level (e.g., "DBG", "INF", "WRN", "ERR")
	Message string // Log message content
}

// ProgramRunner defines an interface for running Bubbletea programs.
type ProgramRunner interface {
	RunProgram(m tea.Model, opts ...tea.ProgramOption) (*tea.Program, error)
}

// styleConfig holds TUI appearance settings.
type styleConfig struct {
	TitleColor  string // ANSI 256-color code for title
	CursorColor string // ANSI 256-color code for cursor
	FooterColor string // ANSI 256-color code for footer
	StatusColor string // ANSI 256-color code for status
	LogColor    string // ANSI 256-color code for log messages
	Cursor      string // Symbol used for the cursor
}

// model encapsulates the state of the TUI.
type model struct {
	choices       []string      // List of available binaries
	cursorX       int           // Horizontal cursor position (column)
	cursorY       int           // Vertical cursor position (row)
	cols          int           // Number of columns in the grid
	rows          int           // Number of rows in the grid
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
	program       *tea.Program  // Reference to the Bubble Tea program for sending messages
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
) error {
	// Fetch available binaries from the specified directory.
	choices := filesystem.ListBinaries(dir)
	if len(choices) == 0 {
		return fmt.Errorf("%w: %s", ErrNoBinariesFound, dir)
	}

	// Initialize the model with default styles.
	// Enable log visibility by default when verbose mode is active.
	m := &model{
		choices:       choices,
		dir:           dir,
		config:        config,
		logger:        log,
		fs:            filesystem,
		cursorX:       0,
		cursorY:       0,
		sortAscending: true,
		styles:        defaultStyleConfig(),
		logs:          make([]string, 0, maxLogLines),
		showLogs:      config.Verbose,
	}

	// Set up log capture channel if verbose mode is enabled.
	// This must be done before starting the program to ensure capture is ready.
	if config.Verbose {
		m.logChan = make(chan LogMsg, maxLogLines)

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

	// Start the TUI program.
	program, err := runner.RunProgram(m)
	if err != nil {
		return fmt.Errorf("failed to start TUI program: %w", err)
	}

	// Allow mocked runners to return nil for testing purposes.
	if program == nil {
		return nil
	}

	// Store program reference for log message sending.
	m.program = program

	// Run the program and capture any runtime errors.
	_, err = program.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI program: %w", err)
	}

	return nil
}

// defaultStyleConfig provides default TUI style settings.
func defaultStyleConfig() styleConfig {
	return styleConfig{
		TitleColor:  "39",  // Bright blue
		CursorColor: "214", // Orange
		FooterColor: "245", // Light gray
		StatusColor: "46",  // Lime green
		LogColor:    "240", // Dark gray for subtle log display
		Cursor:      "❯ ",
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

	// Start log polling if verbose mode is enabled.
	if m.config.Verbose && m.logChan != nil {
		return m.pollLogChannel()
	}

	return nil
}

// pollLogChannel returns a command that polls for log messages.
// This approach avoids deadlocks by having the TUI poll the channel
// rather than trying to Send() from within the capture callback.
func (m *model) pollLogChannel() tea.Cmd {
	return func() tea.Msg {
		// Check if there's a message on the channel.
		// Use a non-blocking receive to avoid hanging.
		select {
		case msg := <-m.logChan:
			return msg
		default:
			// No message available, schedule next poll.
			return nil
		}
	}
}

// Update processes TUI events and updates the model state.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle keyboard input for navigation and actions.
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
			// Toggle log panel visibility.
			m.showLogs = !m.showLogs
			m.updateGrid()

		case "enter":
			// Remove the selected binary and update the TUI state.
			if len(m.choices) > 0 {
				idx := m.cursorY + m.cursorX*m.rows // Column-major index
				if idx < len(m.choices) {
					binaryPath := m.fs.AdjustBinaryPath(m.dir, m.choices[idx])
					name := m.choices[idx]

					if err := m.fs.RemoveBinary(
						binaryPath,
						name,
						m.config.Verbose,
						m.logger,
					); err != nil {
						m.status = fmt.Sprintf("Error removing %s: %v", name, err)
					} else {
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
		}
	case tea.WindowSizeMsg:
		// Update dimensions and recalculate grid layout on resize.
		m.width = msg.Width
		m.height = msg.Height
		m.updateGrid()
	case LogMsg:
		// Add log message to the circular buffer.
		m.addLogEntry(msg)
		m.updateGrid()

		// Continue polling for more log messages.
		// This ensures all pending logs are captured.
		cmd := m.pollLogChannel()

		return m, cmd
	}

	// Continue polling for log messages if in verbose mode.
	// This non-blocking poll ensures logs are captured without deadlocks.
	if m.config.Verbose && m.logChan != nil {
		cmd := m.pollLogChannel()

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
func (m *model) getVisibleLogs(maxLines int) []string {
	if !m.showLogs || len(m.logs) == 0 || maxLines <= 0 {
		return nil
	}

	// Return the last maxLines entries
	start := len(m.logs) - maxLines
	if start < 0 {
		start = 0
	}

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
	availHeight := maximum(m.height-minAvailHeightAdjustment, 1)

	// Adjust available height for log panel if visible
	if m.showLogs && len(m.logs) > 0 {
		// Reserve up to maxVisibleLogLines lines for log panel (plus 1 for separator)
		logPanelHeight := minimum(len(m.logs), maxVisibleLogLines) + 1
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

	// Render log panel if enabled and logs exist
	if m.showLogs && len(m.logs) > 0 {
		visibleLogs := m.getVisibleLogs(
			maxVisibleLogLines,
		) // Show up to maxVisibleLogLines log lines
		if len(visibleLogs) > 0 {
			s.WriteString(logStyle.Render("─ Log Messages ─"))
			s.WriteString("\n")

			for _, logEntry := range visibleLogs {
				s.WriteString(logStyle.Render(logEntry))
				s.WriteString("\n")
			}

			s.WriteString("\n")
		}
	}

	if m.status != "" {
		s.WriteString(statusStyle.Render(m.status))
		s.WriteString("\n")
	}

	// Update footer to include L key for toggling logs
	footerText := "↑/k: up  ↓/j: down  ←/h: left  →/l: right  Enter: remove  s: toggle sort  L: logs  q: quit"
	footer := footerStyle.Render(footerText)

	lenStatus := 0
	if m.status != "" {
		lenStatus = 1
	}

	// Account for log panel in height calculation
	logPanelLines := 0

	if m.showLogs && len(m.logs) > 0 {
		visibleLogs := m.getVisibleLogs(maxVisibleLogLines)
		if len(visibleLogs) > 0 {
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
