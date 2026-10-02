/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	tea "charm.land/bubbletea/v2"

	mockFS "github.com/nicholas-fedor/go-remove/internal/fs/mocks"
	"github.com/nicholas-fedor/go-remove/internal/history"
	mockHistory "github.com/nicholas-fedor/go-remove/internal/history/mocks"
	"github.com/nicholas-fedor/go-remove/internal/logger"
	"github.com/nicholas-fedor/go-remove/internal/tui/render"
)

// allowHistoryReload permits the history reload that draining an operation
// performs, which a strict mock otherwise rejects as unexpected.
func allowHistoryReload(m *mockHistory.MockManager) {
	m.On("GetHistory", mock.Anything, mock.Anything).
		Return([]*history.HistoryEntry(nil), nil).
		Maybe()
}

// Test_model_Update verifies the Update method's state changes and commands.
func Test_model_Update(t *testing.T) {
	type args struct {
		msg tea.Msg
	}

	tests := []struct {
		name    string
		m       *Model
		args    args
		want    Model
		wantCmd tea.Cmd
	}{
		{
			name:    "quit with q",
			m:       &Model{},
			args:    args{msg: keyPress('q')},
			want:    Model{},
			wantCmd: tea.Quit,
		},
		{
			name:    "move up",
			m:       &Model{cursorY: 1, rows: 2},
			args:    args{msg: keyPressString("up")},
			want:    Model{cursorY: 0, rows: 2},
			wantCmd: nil,
		},
		{
			name:    "move down within bounds",
			m:       &Model{cursorY: 0, rows: 2, cols: 2, choices: []string{"a", "b", "c", "d"}},
			args:    args{msg: keyPressString(keyDown)},
			want:    Model{cursorY: 1, rows: 2, cols: 2, choices: []string{"a", "b", "c", "d"}},
			wantCmd: nil,
		},
		{
			name: "move down at last item in last column",
			m: &Model{
				cursorY: 1,
				cursorX: 1,
				rows:    2,
				cols:    2,
				choices: []string{"a", "b", "c", "d"},
			},
			args: args{msg: keyPressString(keyDown)},
			want: Model{
				cursorY: 1,
				cursorX: 1,
				rows:    2,
				cols:    2,
				choices: []string{"a", "b", "c", "d"},
			},
			wantCmd: nil,
		},
		{
			name:    "move left",
			m:       &Model{cursorX: 1, cols: 2},
			args:    args{msg: keyPressString("left")},
			want:    Model{cursorX: 0, cols: 2},
			wantCmd: nil,
		},
		{
			name:    "move right within bounds",
			m:       &Model{cursorX: 0, cols: 2, choices: []string{"a", "b"}},
			args:    args{msg: keyPressString("right")},
			want:    Model{cursorX: 1, cols: 2, choices: []string{"a", "b"}},
			wantCmd: nil,
		},
		{
			name: "toggle sort to descending",
			m: &Model{
				choices:       []string{"age", "vhs"},
				sortAscending: true,
				cols:          1,
				rows:          2,
				width:         80,
				height:        24,
			},
			args: args{msg: keyPress('s')},
			want: Model{
				choices:       []string{"vhs", "age"},
				sortAscending: false,
				cols:          1,
				rows:          2,
				width:         80,
				height:        24,
			},
			wantCmd: nil,
		},
		{
			name: "toggle sort to ascending",
			m: &Model{
				choices:       []string{"vhs", "age"},
				sortAscending: false,
				cols:          1,
				rows:          2,
				width:         80,
				height:        24,
			},
			args: args{msg: keyPress('s')},
			want: Model{
				choices:       []string{"age", "vhs"},
				sortAscending: true,
				cols:          1,
				rows:          2,
				width:         80,
				height:        24,
			},
			wantCmd: nil,
		},
		{
			name: "enter removes binary",
			m: &Model{
				choices:       []string{"age", "vhs"},
				cols:          1,
				rows:          2,
				dir:           "/bin",
				config:        Config{Verbose: false},
				sortAscending: true,
				width:         80,
				height:        24,
				logger:        logger.NopLogger(),
				fs: func() *mockFS.MockFS {
					m := mockFS.NewMockFS(t)
					m.On("AdjustBinaryPath", "/bin", "age").Return("/bin/age")
					m.On("RemoveBinary", "/bin/age", "age", false, mock.Anything).Return(nil)
					m.On("ListBinaries", "/bin").Return([]string{"vhs"}, nil)

					return m
				}(),
			},
			args: args{msg: keyPressString(keyEnter)},
			want: Model{
				choices:       []string{"vhs"},
				cols:          1,
				rows:          1,
				dir:           "/bin",
				config:        Config{Verbose: false},
				sortAscending: true,
				status:        "Removed age",
				width:         80,
				height:        24,
			},
			wantCmd: nil,
		},
		{
			name: "enter with error",
			m: &Model{
				choices:       []string{"age"},
				cols:          1,
				rows:          1,
				dir:           "/bin",
				sortAscending: true,
				width:         80,
				height:        24,
				logger:        logger.NopLogger(),
				fs: func() *mockFS.MockFS {
					m := mockFS.NewMockFS(t)
					m.On("AdjustBinaryPath", "/bin", "age").Return("/bin/age")
					m.On("RemoveBinary", "/bin/age", "age", false, mock.Anything).
						Return(errors.New("remove failed"))

					return m
				}(),
			},
			args: args{msg: keyPressString(keyEnter)},
			want: Model{
				choices:       []string{"age"},
				cols:          1,
				rows:          1,
				dir:           "/bin",
				sortAscending: true,
				status:        "Error removing age: remove failed",
				width:         80,
				height:        24,
			},
			wantCmd: nil,
		},
		{
			name: "window size update",
			m:    &Model{choices: []string{"a", "b"}},
			args: args{msg: tea.WindowSizeMsg{Width: 80, Height: 24}},
			want: Model{
				choices: []string{"a", "b"},
				cols:    1,
				rows:    2,
				width:   80,
				height:  24,
			},
			wantCmd: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set default mocks if not provided.
			if tt.m.logger == nil {
				tt.m.logger = logger.NopLogger()
			}

			if tt.m.fs == nil {
				tt.m.fs = mockFS.NewMockFS(t)
			}

			got, gotCmd := tt.m.Update(tt.args.msg)
			gotModel, _ := got.(*Model)

			// An operation now runs outside Update, so the result is applied
			// before the outcome is asserted.
			gotModel = drainOperation(t, gotModel, gotCmd)

			if !reflect.DeepEqual(gotModel.choices, tt.want.choices) ||
				gotModel.cursorX != tt.want.cursorX ||
				gotModel.cursorY != tt.want.cursorY ||
				gotModel.cols != tt.want.cols ||
				gotModel.rows != tt.want.rows ||
				gotModel.dir != tt.want.dir ||
				gotModel.config != tt.want.config ||
				gotModel.width != tt.want.width ||
				gotModel.height != tt.want.height ||
				gotModel.status != tt.want.status ||
				gotModel.sortAscending != tt.want.sortAscending {
				t.Errorf("model.Update() got = %+v, want %+v", gotModel, tt.want)
			}

			gotCmdType := reflect.TypeFor[tea.Cmd]()

			cmdType := reflect.TypeFor[tea.Cmd]()
			if gotCmdType != nil && gotCmdType != cmdType {
				t.Errorf("model.Update() gotCmd = %T, want tea.Cmd", gotCmd)
			}

			tt.m.fs.(*mockFS.MockFS).AssertExpectations(t)
		})
	}
}

// TestHistoryMsg_ClampsCursor verifies the cursor is brought back inside the
// list when entries disappear.
//
// Without this, clearing the selected entry leaves the cursor past the end, so
// no row renders as selected and the view looks frozen on a phantom row.
func TestHistoryMsg_ClampsCursor(t *testing.T) {
	t.Parallel()

	entry := func(name string) *history.HistoryEntry {
		return &history.HistoryEntry{
			ID:         name,
			BinaryName: name,
		}
	}

	m := &Model{
		historyEntries: []*history.HistoryEntry{entry("a"), entry("b"), entry("c")},
		historyCursor:  2,
		historyManager: &mockHistory.MockManager{},
		logger:         logger.NopLogger(),
		styles:         render.DefaultStyleConfig(),
		fs:             mockFS.NewMockFS(t),
	}

	// The selected entry is cleared, so the list shrinks under the cursor.
	updated, _ := m.Update(HistoryMsg{
		Entries: []*history.HistoryEntry{entry("a"), entry("b")},
	})

	got, ok := updated.(*Model)
	require.True(t, ok)

	assert.Less(t, got.historyCursor, len(got.historyEntries),
		"the cursor must stay inside the entry list")
	assert.Equal(t, 1, got.historyCursor,
		"the cursor should land on the new last entry")

	// An empty list must leave the cursor at zero rather than negative.
	empty, _ := got.Update(HistoryMsg{Entries: nil})

	emptyModel, ok := empty.(*Model)
	require.True(t, ok)
	assert.Equal(t, 0, emptyModel.historyCursor)
}

// TestHandleConfirmation_CtrlC verifies ctrl+c always leaves a dialog.
//
// A dialog that swallows it is a trap for the user who reaches for the one key
// every terminal program honours.
func TestHandleConfirmation_CtrlC(t *testing.T) {
	t.Parallel()

	m := &Model{
		confirmation:   confirmDeletePerm,
		historyEntries: []*history.HistoryEntry{{BinaryName: "vhs"}},
		historyManager: &mockHistory.MockManager{},
		logger:         logger.NopLogger(),
		styles:         render.DefaultStyleConfig(),
		fs:             mockFS.NewMockFS(t),
	}

	updated, cmd := m.handleConfirmation(keyPressString("ctrl+c"))

	got, ok := updated.(*Model)
	require.True(t, ok)

	assert.Equal(t, confirmNone, got.confirmation, "the dialog must close")
	require.NotNil(t, cmd, "ctrl+c must quit rather than be swallowed")
}

// Priority 1: Core History Integration Tests

// Test_model_Update_EnterWithHistoryManager verifies that RecordDeletion is called
// when a binary is removed and history manager is available.
func Test_model_Update_EnterWithHistoryManager(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)
	historyMock := mockHistory.NewMockManager(t)
	allowHistoryReload(historyMock)

	// Setup expectations
	fsMock.On("AdjustBinaryPath", "/bin", "test").Return("/bin/test")
	historyMock.On("RecordDeletion", mock.Anything, "/bin/test").
		Return(&history.HistoryEntry{ID: "123", BinaryName: "test"}, nil)
	fsMock.On("ListBinaries", "/bin").Return([]string{"other"}, nil)

	m := &Model{
		choices:        []string{"test"},
		dir:            "/bin",
		config:         Config{Verbose: false},
		fs:             fsMock,
		historyManager: historyMock,
		logger:         logger.NopLogger(),
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, opCmd := m.Update(keyPressString(keyEnter))
	gotModel, _ := got.(*Model)
	gotModel = drainOperation(t, gotModel, opCmd)

	assert.Equal(t, "Removed test", gotModel.status)
	assert.Contains(t, gotModel.choices, "other")
	fsMock.AssertExpectations(t)
	historyMock.AssertExpectations(t)
}

// Test_model_Update_HistoryRecordError verifies error handling when RecordDeletion fails.
func Test_model_Update_HistoryRecordError(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)
	historyMock := mockHistory.NewMockManager(t)
	allowHistoryReload(historyMock)

	fsMock.On("AdjustBinaryPath", "/bin", "test").Return("/bin/test")
	historyMock.On("RecordDeletion", mock.Anything, "/bin/test").
		Return(nil, errors.New("history storage full"))

	m := &Model{
		choices:        []string{"test"},
		dir:            "/bin",
		config:         Config{Verbose: false},
		fs:             fsMock,
		historyManager: historyMock,
		logger:         logger.NopLogger(),
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, opCmd := m.Update(keyPressString(keyEnter))
	gotModel, _ := got.(*Model)
	gotModel = drainOperation(t, gotModel, opCmd)

	assert.Equal(t, "Error recording test: history storage full", gotModel.status)
	assert.Equal(t, []string{"test"}, gotModel.choices) // Should not be removed
	fsMock.AssertExpectations(t)
	historyMock.AssertExpectations(t)
}

// Test_model_Update_BinaryRemovedFromChoicesAfterHistoryRecord verifies state consistency
// after recording deletion in history.
func Test_model_Update_BinaryRemovedFromChoicesAfterHistoryRecord(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)
	historyMock := mockHistory.NewMockManager(t)
	allowHistoryReload(historyMock)

	fsMock.On("AdjustBinaryPath", "/bin", "binary1").Return("/bin/binary1")
	historyMock.On("RecordDeletion", mock.Anything, "/bin/binary1").
		Return(&history.HistoryEntry{ID: "entry1", BinaryName: "binary1"}, nil)
	fsMock.On("ListBinaries", "/bin").Return([]string{"binary2", "binary3"}, nil)

	m := &Model{
		choices:        []string{"binary1", "binary2", "binary3"},
		dir:            "/bin",
		config:         Config{},
		fs:             fsMock,
		historyManager: historyMock,
		logger:         logger.NopLogger(),
		cols:           1,
		rows:           3,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, opCmd := m.Update(keyPressString(keyEnter))
	gotModel, _ := got.(*Model)
	gotModel = drainOperation(t, gotModel, opCmd)

	assert.Len(t, gotModel.choices, 2)
	assert.NotContains(t, gotModel.choices, "binary1")
	fsMock.AssertExpectations(t)
	historyMock.AssertExpectations(t)
}

// Priority 2: Restore/Undo Operations

// Test_handleRestore_RefreshesBinaryList verifies binary list is refreshed after restore.
func Test_handleRestore_RefreshesBinaryList(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)
	historyMock := mockHistory.NewMockManager(t)
	allowHistoryReload(historyMock)

	entry := &history.HistoryEntry{
		ID:         "entry1",
		BinaryName: "restored_binary",
		InTrash:    true,
	}

	historyMock.On("Restore", mock.Anything, "entry1").
		Return(&history.RestoreResult{BinaryName: "restored_binary", RestoredTo: "/bin/restored_binary"}, nil)
	fsMock.On("ListBinaries", "/bin").Return([]string{"restored_binary", "existing"}, nil)

	m := &Model{
		choices:        []string{"existing"},
		dir:            "/bin",
		fs:             fsMock,
		historyManager: historyMock,
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{entry},
		historyCursor:  0,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	_, cmd := m.handleRestore()
	gotModel := drainOperation(t, m, cmd)

	assert.Contains(t, gotModel.choices, "restored_binary")
	assert.NotNil(t, cmd)
	fsMock.AssertExpectations(t)
	historyMock.AssertExpectations(t)
}

// Test_handleRestore_BinaryAppearsInChoices verifies restored binary is visible in list.
func Test_handleRestore_BinaryAppearsInChoices(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)
	historyMock := mockHistory.NewMockManager(t)
	allowHistoryReload(historyMock)

	entry := &history.HistoryEntry{
		ID:         "entry1",
		BinaryName: "newbinary",
		InTrash:    true,
	}

	historyMock.On("Restore", mock.Anything, "entry1").
		Return(&history.RestoreResult{BinaryName: "newbinary", RestoredTo: "/bin/newbinary"}, nil)
	fsMock.On("ListBinaries", "/bin").Return([]string{"newbinary"}, nil)

	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		fs:             fsMock,
		historyManager: historyMock,
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{entry},
		historyCursor:  0,
		width:          80,
		height:         24,
	}

	_, opCmd := m.handleRestore()
	gotModel := drainOperation(t, m, opCmd)

	assert.Equal(t, []string{"newbinary"}, gotModel.choices)
	assert.Equal(t, "Restored newbinary to /bin/newbinary", gotModel.status)
	fsMock.AssertExpectations(t)
	historyMock.AssertExpectations(t)
}

// Test_handleRestore_HistoryRefreshed verifies history is updated after restore.
func Test_handleRestore_HistoryRefreshed(t *testing.T) {
	historyMock := mockHistory.NewMockManager(t)
	allowHistoryReload(historyMock)

	entry := &history.HistoryEntry{
		ID:         "entry1",
		BinaryName: "testbin",
		InTrash:    true,
	}

	historyMock.On("Restore", mock.Anything, "entry1").
		Return(&history.RestoreResult{BinaryName: "testbin", RestoredTo: "/bin/testbin"}, nil)

	fsMock := mockFS.NewMockFS(t)
	fsMock.On("ListBinaries", "/bin").Return([]string{"testbin"}, nil)

	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		fs:             fsMock,
		historyManager: historyMock,
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{entry},
		historyCursor:  0,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	_, cmd := m.handleRestore()
	assert.NotNil(t, cmd, "a restore must return an operation")

	gotModel := drainOperation(t, m, cmd)

	assert.Equal(t, "Restored testbin to /bin/testbin", gotModel.status)
	fsMock.AssertExpectations(t)
	historyMock.AssertExpectations(t)
}

// Test_handleRestore_ErrorHandling verifies restore error scenarios.
func Test_handleRestore_ErrorHandling(t *testing.T) {
	tests := []struct {
		name         string
		setupMock    func(*mockHistory.MockManager)
		entry        *history.HistoryEntry
		wantStatus   string
		wantCmdIsNil bool
	}{
		{
			name: "already restored error",
			setupMock: func(m *mockHistory.MockManager) {
				m.On("Restore", mock.Anything, "entry1").
					Return(nil, history.ErrAlreadyRestored)
			},
			entry: &history.HistoryEntry{
				ID:         "entry1",
				BinaryName: "test",
				InTrash:    true,
			},
			wantStatus:   "test has already been restored",
			wantCmdIsNil: true,
		},
		{
			name: "not in trash error",
			setupMock: func(m *mockHistory.MockManager) {
				m.On("Restore", mock.Anything, "entry2").
					Return(nil, history.ErrNotInTrash)
			},
			entry: &history.HistoryEntry{
				ID:         "entry2",
				BinaryName: "test2",
				InTrash:    true,
			},
			wantStatus:   "test2 is no longer in trash",
			wantCmdIsNil: true,
		},
		{
			name: "restore collision error",
			setupMock: func(m *mockHistory.MockManager) {
				m.On("Restore", mock.Anything, "entry3").
					Return(nil, history.ErrRestoreCollision)
			},
			entry: &history.HistoryEntry{
				ID:         "entry3",
				BinaryName: "test3",
				InTrash:    true,
			},
			wantStatus:   "Cannot restore test3: file already exists",
			wantCmdIsNil: true,
		},
		{
			name: "generic error",
			setupMock: func(m *mockHistory.MockManager) {
				m.On("Restore", mock.Anything, "entry4").
					Return(nil, errors.New("disk error"))
			},
			entry: &history.HistoryEntry{
				ID:         "entry4",
				BinaryName: "test4",
				InTrash:    true,
			},
			wantStatus:   "Error restoring test4: disk error",
			wantCmdIsNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			historyMock := mockHistory.NewMockManager(t)
			allowHistoryReload(historyMock)
			tt.setupMock(historyMock)

			m := &Model{
				choices:        []string{},
				dir:            "/bin",
				fs:             mockFS.NewMockFS(t),
				historyManager: historyMock,
				logger:         logger.NopLogger(),
				mode:           modeHistory,
				historyEntries: []*history.HistoryEntry{tt.entry},
				historyCursor:  0,
			}

			_, cmd := m.handleRestore()
			gotModel := drainOperation(t, m, cmd)

			assert.Equal(t, tt.wantStatus, gotModel.status)

			historyMock.AssertExpectations(t)
		})
	}
}

// Test_handleUndo_BinaryModeRefresh verifies undo refreshes binary list in binary mode.
func Test_handleUndo_BinaryModeRefresh(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)
	historyMock := mockHistory.NewMockManager(t)
	allowHistoryReload(historyMock)

	historyMock.On("UndoMostRecent", mock.Anything).
		Return(&history.RestoreResult{BinaryName: "undone_binary", RestoredTo: "/bin/undone_binary"}, nil)
	fsMock.On("ListBinaries", "/bin").Return([]string{"undone_binary", "existing"}, nil)

	m := &Model{
		choices:        []string{"existing"},
		dir:            "/bin",
		fs:             fsMock,
		historyManager: historyMock,
		logger:         logger.NopLogger(),
		mode:           modeBinaries,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	_, opCmd := m.handleUndo()
	gotModel := drainOperation(t, m, opCmd)

	assert.Contains(t, gotModel.choices, "undone_binary")
	assert.Equal(t, "Restored undone_binary to /bin/undone_binary", gotModel.status)
	fsMock.AssertExpectations(t)
	historyMock.AssertExpectations(t)
}

// Priority 3: Log Polling

// Test_model_Update_LogMsgHandling verifies LogMsg adds entry to logs slice.
func Test_model_Update_LogMsgHandling(t *testing.T) {
	m := &Model{
		choices:       []string{"test"},
		cols:          1,
		rows:          1,
		width:         80,
		height:        24,
		logs:          []string{},
		sortAscending: true,
	}

	logMsg := LogMsg{Level: "DBG", Message: "debug info"}
	got, _ := m.Update(logMsg)
	gotModel := got.(*Model)

	assert.Len(t, gotModel.logs, 1)
	assert.Contains(t, gotModel.logs[0], "debug info")
}

// Test_model_Update_PollLogTickMsgHandling verifies tick triggers next poll.
func Test_model_Update_PollLogTickMsgHandling(t *testing.T) {
	logChan := make(chan LogMsg, 10)
	m := &Model{
		choices:       []string{"test"},
		cols:          1,
		rows:          1,
		width:         80,
		height:        24,
		logChan:       logChan,
		logs:          []string{},
		sortAscending: true,
	}

	tickMsg := pollLogTickMsg{}
	_, cmd := m.Update(tickMsg)

	assert.NotNil(t, cmd)
}

// Priority 4: History Mode Operations

// Test_model_Update_ModeSwitchToHistory verifies 'r' key switches to history mode.
func Test_model_Update_ModeSwitchToHistory(t *testing.T) {
	historyMock := mockHistory.NewMockManager(t)
	allowHistoryReload(historyMock)

	m := &Model{
		choices:        []string{"test"},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		historyManager: historyMock,
		logger:         logger.NopLogger(),
		mode:           modeBinaries,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, cmd := m.Update(keyPress('r'))
	gotModel := got.(*Model)

	assert.Equal(t, modeHistory, gotModel.mode)
	assert.True(t, gotModel.historyLoading)
	assert.Equal(t, 0, gotModel.historyCursor)
	assert.Equal(t, "Loading history...", gotModel.status)
	assert.NotNil(t, cmd)
}

// Test_model_Update_ModeSwitchToBinaries verifies 'b' returns to binary mode.
func Test_model_Update_ModeSwitchToBinaries(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)
	fsMock.On("ListBinaries", "/bin").Return([]string{"test"}, nil)

	m := &Model{
		choices:       []string{"test"},
		dir:           "/bin",
		config:        Config{},
		fs:            fsMock,
		logger:        logger.NopLogger(),
		mode:          modeHistory,
		cols:          1,
		rows:          1,
		width:         80,
		height:        24,
		sortAscending: true,
	}

	_, opCmd := m.Update(keyPress('b'))
	gotModel := drainOperation(t, m, opCmd)

	assert.Equal(t, modeBinaries, gotModel.mode)
	fsMock.AssertExpectations(t)
}

// Test_model_Update_IgnoresKeyRelease verifies key-release events do not
// repeat navigation, restore, or undo actions.
func Test_model_Update_IgnoresKeyRelease(t *testing.T) {
	historyMock := mockHistory.NewMockManager(t)
	allowHistoryReload(historyMock)

	entries := []*history.HistoryEntry{
		{ID: "1", BinaryName: "bin1", InTrash: true},
		{ID: "2", BinaryName: "bin2", InTrash: true},
	}
	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		fs:             mockFS.NewMockFS(t),
		historyManager: historyMock,
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		historyEntries: entries,
		historyCursor:  0,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, cmd := m.Update(tea.KeyReleaseMsg{Text: "j", Code: 'j', ShiftedCode: 'j'})
	gotModel := got.(*Model)

	assert.Equal(t, 0, gotModel.historyCursor)
	assert.Nil(t, cmd)

	got, cmd = m.Update(tea.KeyReleaseMsg{Text: "u", Code: 'u', ShiftedCode: 'u'})
	gotModel = got.(*Model)

	assert.Empty(t, gotModel.status)
	assert.Nil(t, cmd)

	got, cmd = m.Update(tea.KeyReleaseMsg{Code: tea.KeyEnter})
	gotModel = got.(*Model)

	assert.Empty(t, gotModel.status)
	assert.Nil(t, cmd)
	historyMock.AssertExpectations(t)
}

// Test_model_Update_HistoryNavigation verifies up/down in history list.
func Test_model_Update_HistoryNavigation(t *testing.T) {
	entries := []*history.HistoryEntry{
		{ID: "1", BinaryName: "bin1"},
		{ID: "2", BinaryName: "bin2"},
		{ID: "3", BinaryName: "bin3"},
	}

	tests := []struct {
		name         string
		key          string
		initialCur   int
		expectedCur  int
		expectedMode string
	}{
		{name: "move up", key: "up", initialCur: 2, expectedCur: 1, expectedMode: modeHistory},
		{
			name:         "move up at top",
			key:          "up",
			initialCur:   0,
			expectedCur:  0,
			expectedMode: modeHistory,
		},
		{name: "move down", key: "down", initialCur: 0, expectedCur: 1, expectedMode: modeHistory},
		{
			name:         "move down at bottom",
			key:          "down",
			initialCur:   2,
			expectedCur:  2,
			expectedMode: modeHistory,
		},
		{name: "k key up", key: "k", initialCur: 1, expectedCur: 0, expectedMode: modeHistory},
		{name: "j key down", key: "j", initialCur: 0, expectedCur: 1, expectedMode: modeHistory},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{
				choices:        []string{},
				dir:            "/bin",
				fs:             mockFS.NewMockFS(t),
				logger:         logger.NopLogger(),
				mode:           modeHistory,
				historyEntries: entries,
				historyCursor:  tt.initialCur,
				cols:           1,
				rows:           1,
				width:          80,
				height:         24,
				sortAscending:  true,
			}

			_, opCmd := m.Update(keyPressString(tt.key))
			gotModel := drainOperation(t, m, opCmd)

			assert.Equal(t, tt.expectedCur, gotModel.historyCursor)
			assert.Equal(t, tt.expectedMode, gotModel.mode)
		})
	}
}

// Test_updateHistoryMode_EnterRestore verifies Enter key restores in history mode.
func Test_updateHistoryMode_EnterRestore(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)
	historyMock := mockHistory.NewMockManager(t)
	allowHistoryReload(historyMock)

	entry := &history.HistoryEntry{
		ID:         "entry1",
		BinaryName: "restoreme",
		InTrash:    true,
	}

	historyMock.On("Restore", mock.Anything, "entry1").
		Return(&history.RestoreResult{BinaryName: "restoreme", RestoredTo: "/bin/restoreme"}, nil)
	fsMock.On("ListBinaries", "/bin").Return([]string{"restoreme"}, nil)

	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		fs:             fsMock,
		historyManager: historyMock,
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{entry},
		historyCursor:  0,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, opCmd := m.Update(keyPressString(keyEnter))
	gotModel, _ := got.(*Model)
	gotModel = drainOperation(t, gotModel, opCmd)

	assert.Equal(t, "Restored restoreme to /bin/restoreme", gotModel.status)
	fsMock.AssertExpectations(t)
	historyMock.AssertExpectations(t)
}

// Test_updateHistoryMode_ClearEntry verifies clearing single entry.
func Test_updateHistoryMode_ClearEntry(t *testing.T) {
	historyMock := mockHistory.NewMockManager(t)
	allowHistoryReload(historyMock)

	entry := &history.HistoryEntry{
		ID:         "entry1",
		BinaryName: "testbin",
	}

	historyMock.On("ClearEntry", mock.Anything, "entry1", false).Return(nil)

	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		fs:             mockFS.NewMockFS(t),
		historyManager: historyMock,
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{entry},
		historyCursor:  0,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, cmd := m.Update(keyPress('c'))
	gotModel := got.(*Model)

	assert.Equal(t, "Cleared history entry for testbin", gotModel.status)
	assert.NotNil(t, cmd)
	historyMock.AssertExpectations(t)
}

// Test_updateHistoryMode_ClearAll verifies clear all entries with confirmation.
func Test_updateHistoryMode_ClearAll(t *testing.T) {
	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		fs:             mockFS.NewMockFS(t),
		historyManager: mockHistory.NewMockManager(t),
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{
			{ID: "1", BinaryName: "bin1"},
			{ID: "2", BinaryName: "bin2"},
		},
		historyCursor: 0,
		cols:          1,
		rows:          1,
		width:         80,
		height:        24,
		sortAscending: true,
	}

	_, opCmd := m.Update(keyPress('C'))
	gotModel := drainOperation(t, m, opCmd)

	assert.Equal(t, confirmClearAll, gotModel.confirmation)
}

// Priority 5: State Synchronization

// Test_model_statusUpdates verifies status message updates correctly.
func Test_model_statusUpdates(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)

	fsMock.On("AdjustBinaryPath", "/bin", "test").Return("/bin/test")
	fsMock.On("RemoveBinary", "/bin/test", "test", false, mock.Anything).Return(nil)
	fsMock.On("ListBinaries", "/bin").Return([]string{}, nil)

	m := &Model{
		choices:       []string{"test"},
		dir:           "/bin",
		config:        Config{},
		fs:            fsMock,
		logger:        logger.NopLogger(),
		status:        "",
		cols:          1,
		rows:          1,
		width:         80,
		height:        24,
		sortAscending: true,
	}

	got, opCmd := m.Update(keyPressString(keyEnter))
	gotModel, _ := got.(*Model)
	gotModel = drainOperation(t, gotModel, opCmd)

	assert.Equal(t, "Removed test", gotModel.status)
	fsMock.AssertExpectations(t)
}

// Priority 6: Edge Cases

// Test_model_Update_EmptyBinaryList verifies empty list handling.
func Test_model_Update_EmptyBinaryList(t *testing.T) {
	m := &Model{
		choices:       []string{},
		dir:           "/bin",
		config:        Config{},
		fs:            mockFS.NewMockFS(t),
		logger:        logger.NopLogger(),
		status:        "",
		cols:          0,
		rows:          0,
		width:         80,
		height:        24,
		sortAscending: true,
	}

	// Try to remove with empty list - no action expected
	got, cmd := m.Update(keyPressString(keyEnter))
	gotModel := got.(*Model)

	// Should return no command when list is empty (nothing to remove)
	assert.NotNil(t, gotModel)
	assert.Nil(t, cmd) // No command when list is empty
}

// Test_model_Update_HistoryEmpty verifies empty history handling.
func Test_model_Update_HistoryEmpty(t *testing.T) {
	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{},
		historyCursor:  0,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	// Try to clear all with empty history - should not set confirmation
	_, opCmd := m.Update(keyPress('C'))
	gotModel := drainOperation(t, m, opCmd)

	assert.Equal(t, confirmNone, gotModel.confirmation)
}

// Test_model_Update_ConfirmationCancel verifies cancel confirmation dialog.
func Test_model_Update_ConfirmationCancel(t *testing.T) {
	tests := []struct {
		name string
		key  rune
	}{
		{name: "n key", key: 'n'},
		{name: "N key", key: 'N'},
		{name: "q key", key: 'q'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{
				choices:       []string{},
				dir:           "/bin",
				config:        Config{},
				fs:            mockFS.NewMockFS(t),
				logger:        logger.NopLogger(),
				mode:          modeHistory,
				confirmation:  confirmClearAll,
				cols:          1,
				rows:          1,
				width:         80,
				height:        24,
				sortAscending: true,
			}

			_, opCmd := m.Update(keyPress(tt.key))
			gotModel := drainOperation(t, m, opCmd)

			assert.Equal(t, confirmNone, gotModel.confirmation)
			assert.Equal(t, "Operation cancelled", gotModel.status)
		})
	}
}

// Test_model_Update_ConfirmationAccept verifies accept confirmation dialog.
func Test_model_Update_ConfirmationAccept(t *testing.T) {
	historyMock := mockHistory.NewMockManager(t)
	allowHistoryReload(historyMock)
	historyMock.On("ClearHistory", mock.Anything, false).Return(nil)

	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		historyManager: historyMock,
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		confirmation:   confirmClearAll,
		historyEntries: []*history.HistoryEntry{
			{ID: "1", BinaryName: "bin1"},
		},
		historyCursor: 0,
		cols:          1,
		rows:          1,
		width:         80,
		height:        24,
		sortAscending: true,
	}

	_, opCmd := m.Update(keyPress('y'))
	gotModel := drainOperation(t, m, opCmd)

	assert.Equal(t, confirmNone, gotModel.confirmation)
	assert.Equal(t, "History cleared", gotModel.status)
	historyMock.AssertExpectations(t)
}

// Additional tests for handleClearEntry

// Test_handleClearEntry_ErrorHandling verifies error handling when clearing entry fails.
func Test_handleClearEntry_ErrorHandling(t *testing.T) {
	historyMock := mockHistory.NewMockManager(t)
	allowHistoryReload(historyMock)

	entry := &history.HistoryEntry{
		ID:         "entry1",
		BinaryName: "testbin",
	}

	historyMock.On("ClearEntry", mock.Anything, "entry1", false).Return(errors.New("clear failed"))

	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		fs:             mockFS.NewMockFS(t),
		historyManager: historyMock,
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{entry},
		historyCursor:  0,
	}

	got, _ := m.handleClearEntry(false)
	gotModel := got.(*Model)

	assert.Equal(t, "Error clearing entry: clear failed", gotModel.status)
	historyMock.AssertExpectations(t)
}

// Test_handleClearEntry_NoHistoryManager verifies behavior when history manager is nil.
func Test_handleClearEntry_NoHistoryManager(t *testing.T) {
	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		fs:             mockFS.NewMockFS(t),
		historyManager: nil,
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{{ID: "1", BinaryName: "bin1"}},
		historyCursor:  0,
	}

	got, cmd := m.handleClearEntry(false)
	gotModel := got.(*Model)

	assert.Equal(t, "No history entry selected", gotModel.status)
	assert.Nil(t, cmd)
}

// Test_handleUndo_ErrorHandling verifies undo error handling.
func Test_handleUndo_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		setupMock  func(*mockHistory.MockManager)
		wantStatus string
	}{
		{
			name: "no history error",
			setupMock: func(m *mockHistory.MockManager) {
				m.On("UndoMostRecent", mock.Anything).Return(nil, history.ErrNoHistory)
			},
			wantStatus: "No deletion history found - nothing to undo",
		},
		{
			name: "already restored error",
			setupMock: func(m *mockHistory.MockManager) {
				m.On("UndoMostRecent", mock.Anything).Return(nil, history.ErrAlreadyRestored)
			},
			wantStatus: "Binary has already been restored",
		},
		{
			name: "not in trash error",
			setupMock: func(m *mockHistory.MockManager) {
				m.On("UndoMostRecent", mock.Anything).Return(nil, history.ErrNotInTrash)
			},
			wantStatus: "Binary is no longer in trash - cannot restore",
		},
		{
			name: "restore collision error",
			setupMock: func(m *mockHistory.MockManager) {
				m.On("UndoMostRecent", mock.Anything).Return(nil, history.ErrRestoreCollision)
			},
			wantStatus: "A file already exists at the restore location",
		},
		{
			name: "generic error",
			setupMock: func(m *mockHistory.MockManager) {
				m.On("UndoMostRecent", mock.Anything).Return(nil, errors.New("unexpected error"))
			},
			wantStatus: "Undo failed: unexpected error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			historyMock := mockHistory.NewMockManager(t)
			allowHistoryReload(historyMock)
			tt.setupMock(historyMock)

			m := &Model{
				choices:        []string{},
				dir:            "/bin",
				fs:             mockFS.NewMockFS(t),
				historyManager: historyMock,
				logger:         logger.NopLogger(),
				mode:           modeBinaries,
			}

			_, opCmd := m.handleUndo()
			gotModel := drainOperation(t, m, opCmd)

			assert.Equal(t, tt.wantStatus, gotModel.status)
			historyMock.AssertExpectations(t)
		})
	}
}

// Test_handleUndo_NoHistoryManager verifies undo behavior without history manager.
func Test_handleUndo_NoHistoryManager(t *testing.T) {
	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		fs:             mockFS.NewMockFS(t),
		historyManager: nil,
		logger:         logger.NopLogger(),
		mode:           modeBinaries,
	}

	updated, cmd := m.handleUndo()
	assert.Nil(t, cmd, "nothing to undo without a history manager")

	gotModel, _ := updated.(*Model)

	assert.Equal(t, "History manager not available", gotModel.status)
}

// Test_model_handleConfirmation_ExecuteClearAllError verifies error handling when ClearHistory fails.
func Test_model_handleConfirmation_ExecuteClearAllError(t *testing.T) {
	historyMock := mockHistory.NewMockManager(t)
	allowHistoryReload(historyMock)
	historyMock.On("ClearHistory", mock.Anything, false).Return(errors.New("storage error"))

	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		historyManager: historyMock,
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		confirmation:   confirmClearAll,
		historyEntries: []*history.HistoryEntry{{ID: "1", BinaryName: "bin1"}},
	}

	_, opCmd := m.executeConfirmation()
	gotModel := drainOperation(t, m, opCmd)

	assert.Equal(t, confirmNone, gotModel.confirmation)
	assert.Contains(t, gotModel.status, "Error clearing history")
	historyMock.AssertExpectations(t)
}

// Test_model_handleConfirmation_ExecuteDeletePermError verifies error handling when DeletePermanently fails.
func Test_model_handleConfirmation_ExecuteDeletePermError(t *testing.T) {
	historyMock := mockHistory.NewMockManager(t)
	allowHistoryReload(historyMock)
	historyMock.On("DeletePermanently", mock.Anything, "entry1").Return(errors.New("delete failed"))

	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		historyManager: historyMock,
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		confirmation:   confirmDeletePerm,
		historyEntries: []*history.HistoryEntry{{ID: "entry1", BinaryName: "testbin"}},
		historyCursor:  0,
	}

	_, opCmd := m.executeConfirmation()
	gotModel := drainOperation(t, m, opCmd)

	assert.Equal(t, confirmNone, gotModel.confirmation)
	assert.Contains(t, gotModel.status, "Error deleting permanently")
	historyMock.AssertExpectations(t)
}

// Test_handleRestore_NoHistoryManager verifies restore without history manager.
func Test_handleRestore_NoHistoryManager(t *testing.T) {
	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		historyManager: nil,
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{{ID: "1", BinaryName: "bin1"}},
		historyCursor:  0,
	}

	_, cmd := m.handleRestore()
	gotModel := drainOperation(t, m, cmd)

	assert.Equal(t, "No history entry selected", gotModel.status)
	assert.Nil(t, cmd)
}

// Test_handleRestore_CannotRestore verifies restore when entry cannot be restored.
func Test_handleRestore_CannotRestore(t *testing.T) {
	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		historyManager: mockHistory.NewMockManager(t),
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{{ID: "1", BinaryName: "bin1"}},
		historyCursor:  0,
	}

	_, cmd := m.handleRestore()
	gotModel := drainOperation(t, m, cmd)

	assert.Contains(t, gotModel.status, "Cannot restore")
	assert.Nil(t, cmd)
}

// Test_model_Update_CannotRestore verifies restore via Update when entry cannot be restored.
func Test_model_Update_CannotRestore(t *testing.T) {
	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		historyManager: mockHistory.NewMockManager(t),
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{{ID: "1", BinaryName: "bin1"}},
		historyCursor:  0,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, opCmd := m.Update(keyPressString(keyEnter))
	gotModel, _ := got.(*Model)
	gotModel = drainOperation(t, gotModel, opCmd)

	assert.Contains(t, gotModel.status, "Cannot restore")
}

// Test_model_Update_HistoryMsgError verifies HistoryMsg error handling.
func Test_model_Update_HistoryMsgError(t *testing.T) {
	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{},
		historyLoading: true,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	historyMsg := HistoryMsg{Entries: nil, Error: errors.New("load failed")}
	got, _ := m.Update(historyMsg)
	gotModel := got.(*Model)

	assert.False(t, gotModel.historyLoading)
	assert.Contains(t, gotModel.status, "Error loading history")
}

// Test_model_Update_HistoryMsgEmpty verifies HistoryMsg with empty entries.
func Test_model_Update_HistoryMsgEmpty(t *testing.T) {
	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{},
		historyLoading: true,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	historyMsg := HistoryMsg{Entries: []*history.HistoryEntry{}, Error: nil}
	got, _ := m.Update(historyMsg)
	gotModel := got.(*Model)

	assert.False(t, gotModel.historyLoading)
	assert.Equal(t, "No deletion history found", gotModel.status)
}

// Test_model_Update_HistoryMsgSuccess verifies HistoryMsg with entries.
func Test_model_Update_HistoryMsgSuccess(t *testing.T) {
	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{},
		historyLoading: true,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	entries := []*history.HistoryEntry{
		{ID: "1", BinaryName: "bin1"},
		{ID: "2", BinaryName: "bin2"},
	}
	historyMsg := HistoryMsg{Entries: entries, Error: nil}
	got, _ := m.Update(historyMsg)
	gotModel := got.(*Model)

	assert.False(t, gotModel.historyLoading)
	assert.Len(t, gotModel.historyEntries, 2)
	assert.Equal(t, "Loaded 2 history entries", gotModel.status)
}

// Test_model_Update_ToggleLogs verifies L key toggles log panel.
func Test_model_Update_ToggleLogs(t *testing.T) {
	m := &Model{
		choices:       []string{"test"},
		dir:           "/bin",
		config:        Config{},
		fs:            mockFS.NewMockFS(t),
		logger:        logger.NopLogger(),
		mode:          modeBinaries,
		showLogs:      false,
		cols:          1,
		rows:          1,
		width:         80,
		height:        24,
		sortAscending: true,
	}

	// Toggle on
	_, opCmd := m.Update(keyPress('L'))
	gotModel := drainOperation(t, m, opCmd)
	assert.True(t, gotModel.showLogs)

	// Toggle off
	toggled, _ := gotModel.Update(keyPress('L'))
	assert.False(t, toggled.(*Model).showLogs)
}

// Test_model_Update_UndoKey verifies u key triggers undo.
func Test_model_Update_UndoKey(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)
	historyMock := mockHistory.NewMockManager(t)
	allowHistoryReload(historyMock)

	historyMock.On("UndoMostRecent", mock.Anything).
		Return(&history.RestoreResult{BinaryName: "undone", RestoredTo: "/bin/undone"}, nil)
	fsMock.On("ListBinaries", "/bin").Return([]string{"undone"}, nil)

	historyMock.On("GetHistory", mock.Anything, mock.Anything).
		Return([]*history.HistoryEntry(nil), nil).
		Maybe()

	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             fsMock,
		historyManager: historyMock,
		logger:         logger.NopLogger(),
		mode:           modeBinaries,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	_, opCmd := m.Update(keyPress('u'))
	gotModel := drainOperation(t, m, opCmd)

	assert.Equal(t, "Restored undone to /bin/undone", gotModel.status)
	fsMock.AssertExpectations(t)
	historyMock.AssertExpectations(t)
}

// Test_model_Update_DeletePermanentlyKey verifies d key triggers permanent delete with confirmation.
func Test_model_Update_DeletePermanentlyKey(t *testing.T) {
	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		historyManager: mockHistory.NewMockManager(t),
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{{ID: "1", BinaryName: "bin1"}},
		historyCursor:  0,
		confirmation:   confirmNone,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	_, opCmd := m.Update(keyPress('d'))
	gotModel := drainOperation(t, m, opCmd)

	assert.Equal(t, confirmDeletePerm, gotModel.confirmation)
}

// Test_model_Update_ClearEntryKey verifies c key triggers clear entry.
func Test_model_Update_ClearEntryKey(t *testing.T) {
	historyMock := mockHistory.NewMockManager(t)
	allowHistoryReload(historyMock)
	historyMock.On("ClearEntry", mock.Anything, "1", false).Return(nil)
	historyMock.On("GetHistory", mock.Anything, mock.Anything).
		Return([]*history.HistoryEntry(nil), nil).
		Maybe()

	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		historyManager: historyMock,
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{{ID: "1", BinaryName: "bin1"}},
		historyCursor:  0,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	_, opCmd := m.Update(keyPress('c'))
	gotModel := drainOperation(t, m, opCmd)

	assert.Equal(t, "Cleared history entry for bin1", gotModel.status)
	historyMock.AssertExpectations(t)
}

// Test_model_handleConfirmation_UnknownConfirmation verifies behavior with unknown confirmation type.
func Test_model_handleConfirmation_UnknownConfirmation(t *testing.T) {
	m := &Model{
		choices:       []string{},
		dir:           "/bin",
		config:        Config{},
		fs:            mockFS.NewMockFS(t),
		logger:        logger.NopLogger(),
		mode:          modeHistory,
		confirmation:  "unknown_type",
		cols:          1,
		rows:          1,
		width:         80,
		height:        24,
		sortAscending: true,
	}

	got, cmd := m.handleConfirmation(keyPress('y'))
	gotModel := got.(*Model)

	// Should clear confirmation but do nothing else
	assert.Equal(t, confirmNone, gotModel.confirmation)
	assert.Nil(t, cmd)
}

// Test_handleRestore_HistoryModeRefresh verifies history refresh after restore in history mode.
func Test_handleRestore_HistoryModeRefresh(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)
	historyMock := mockHistory.NewMockManager(t)
	allowHistoryReload(historyMock)

	entry := &history.HistoryEntry{
		ID:         "entry1",
		BinaryName: "restoreme",
		InTrash:    true,
	}

	historyMock.On("Restore", mock.Anything, "entry1").
		Return(&history.RestoreResult{BinaryName: "restoreme", RestoredTo: "/bin/restoreme"}, nil)
	fsMock.On("ListBinaries", "/bin").Return([]string{"restoreme"}, nil)

	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		fs:             fsMock,
		historyManager: historyMock,
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{entry},
		historyCursor:  0,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	_, cmd := m.handleRestore()
	gotModel := drainOperation(t, m, cmd)

	assert.Equal(t, "Restored restoreme to /bin/restoreme", gotModel.status)
	assert.NotNil(t, cmd) // Should return loadHistory command
	fsMock.AssertExpectations(t)
	historyMock.AssertExpectations(t)
}
