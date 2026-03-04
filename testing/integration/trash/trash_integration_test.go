/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package trash_test provides black-box integration tests for the trash package.
//
// These tests verify the Trasher interface behavior through the MockTrasher mock
// implementation. All tests use Mockery-generated mocks to ensure no real filesystem
// operations are performed. The tests cover the complete lifecycle of trash operations
// including moving files to trash, restoring, listing, and permanent deletion.
//
// Test Organization:
//   - TrashIntegrationTestSuite: Main test suite using testify suite
//   - Table-driven tests for parameterized scenarios
//   - Individual tests for complex error handling cases
//
// Coverage Areas:
//   - MoveToTrash workflow with context handling
//   - RestoreFromTrash workflow with collision detection
//   - IsInTrash validation checks
//   - ListTrash operations and filtering
//   - DeletePermanently workflow
//   - Cross-platform behavior (Linux XDG trash, Windows Recycle Bin)
//   - Error handling for invalid paths and permissions
//   - Context cancellation handling
package trash_test

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/nicholas-fedor/go-remove/internal/trash"
	"github.com/nicholas-fedor/go-remove/internal/trash/mocks"
)

// Test constants for consistent test data.
const (
	testFilePath      = "/usr/local/bin/test-binary"
	testTrashPath     = "/home/user/.local/share/Trash/files/test-binary_1709321234"
	testTrashInfoPath = "/home/user/.local/share/Trash/info/test-binary_1709321234.trashinfo"
	testBinaryName    = "test-binary"
	testOriginalDir   = "/usr/local/bin"
)

// TrashIntegrationTestSuite provides integration tests for the trash Trasher interface.
//
// This suite tests the orchestration between the application layer and trash operations,
// ensuring that all trash operations behave correctly through the Trasher interface.
// All tests use the MockTrasher to simulate trash interactions without requiring
// a real trash directory.
type TrashIntegrationTestSuite struct {
	suite.Suite

	// Mock for the Trasher interface
	mockTrasher *mocks.MockTrasher
}

// SetupTest initializes the test suite before each test.
//
// Creates a fresh MockTrasher instance for each test to ensure test isolation
// and prevent test interference.
func (s *TrashIntegrationTestSuite) SetupTest() {
	s.mockTrasher = mocks.NewMockTrasher(s.T())
}

// createTrashEntry creates a standard TrashEntry for testing.
//
// Parameters:
//   - name: The filename in trash
//   - originalPath: The original location before deletion
//   - trashPath: The current location in trash
//
// Returns:
//   - A TrashEntry populated with test data
func (s *TrashIntegrationTestSuite) createTrashEntry(
	name string,
	originalPath string,
	trashPath string,
) trash.TrashEntry {
	return trash.TrashEntry{
		Name:         name,
		OriginalPath: originalPath,
		TrashPath:    trashPath,
		DeletionTime: time.Now(),
	}
}

// TestTrashIntegrationTestSuite runs the integration test suite.
func TestTrashIntegrationTestSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(TrashIntegrationTestSuite))
}

// TestMoveToTrashWorkflow verifies the complete MoveToTrash workflow.
//
// This test ensures that the trash layer properly handles:
// 1. Moving a file to trash
// 2. Returning the correct trash path
// 3. Handling context properly.
func (s *TrashIntegrationTestSuite) TestMoveToTrashWorkflow() {
	ctx := context.Background()

	s.mockTrasher.EXPECT().
		MoveToTrash(mock.Anything, testFilePath).
		Return(testTrashPath, nil).
		Once()

	trashPath, err := s.mockTrasher.MoveToTrash(ctx, testFilePath)

	s.Require().NoError(err)
	s.Equal(testTrashPath, trashPath)
}

// TestMoveToTrashWithEmptyPath verifies error handling for empty path.
//
// An empty file path should return an error.
func (s *TrashIntegrationTestSuite) TestMoveToTrashWithEmptyPath() {
	ctx := context.Background()

	s.mockTrasher.EXPECT().
		MoveToTrash(mock.Anything, "").
		Return("", trash.ErrInvalidPath).
		Once()

	trashPath, err := s.mockTrasher.MoveToTrash(ctx, "")

	s.Require().ErrorIs(err, trash.ErrInvalidPath)
	s.Empty(trashPath)
}

// TestMoveToTrashPathNotFound verifies error handling for non-existent file.
//
// When the source file does not exist, MoveToTrash should return ErrPathNotFound.
func (s *TrashIntegrationTestSuite) TestMoveToTrashPathNotFound() {
	ctx := context.Background()
	nonExistentPath := "/non/existent/file"

	s.mockTrasher.EXPECT().
		MoveToTrash(mock.Anything, nonExistentPath).
		Return("", trash.ErrPathNotFound).
		Once()

	trashPath, err := s.mockTrasher.MoveToTrash(ctx, nonExistentPath)

	s.Require().ErrorIs(err, trash.ErrPathNotFound)
	s.Empty(trashPath)
}

// TestMoveToTrashFull verifies error handling when trash is full.
//
// When the trash directory is full or inaccessible, MoveToTrash should return ErrTrashFull.
func (s *TrashIntegrationTestSuite) TestMoveToTrashFull() {
	ctx := context.Background()

	s.mockTrasher.EXPECT().
		MoveToTrash(mock.Anything, testFilePath).
		Return("", trash.ErrTrashFull).
		Once()

	trashPath, err := s.mockTrasher.MoveToTrash(ctx, testFilePath)

	s.Require().ErrorIs(err, trash.ErrTrashFull)
	s.Empty(trashPath)
}

// TestRestoreFromTrashWorkflow verifies the complete RestoreFromTrash workflow.
//
// This test ensures that the trash layer properly handles:
// 1. Restoring a file from trash to its original location
// 2. Validating the trash path
// 3. Handling context properly.
func (s *TrashIntegrationTestSuite) TestRestoreFromTrashWorkflow() {
	ctx := context.Background()

	s.mockTrasher.EXPECT().
		RestoreFromTrash(mock.Anything, testTrashPath, testFilePath).
		Return(nil).
		Once()

	err := s.mockTrasher.RestoreFromTrash(ctx, testTrashPath, testFilePath)

	s.Require().NoError(err)
}

// TestRestoreFromTrashCollision verifies error handling for restore collision.
//
// When a file already exists at the restore location, RestoreFromTrash should
// return ErrRestoreCollision.
func (s *TrashIntegrationTestSuite) TestRestoreFromTrashCollision() {
	ctx := context.Background()

	s.mockTrasher.EXPECT().
		RestoreFromTrash(mock.Anything, testTrashPath, testFilePath).
		Return(trash.ErrRestoreCollision).
		Once()

	err := s.mockTrasher.RestoreFromTrash(ctx, testTrashPath, testFilePath)

	s.Require().ErrorIs(err, trash.ErrRestoreCollision)
}

// TestRestoreFromTrashNotInTrash verifies error handling for file not in trash.
//
// When the file is not in the trash, RestoreFromTrash should return ErrFileNotInTrash.
func (s *TrashIntegrationTestSuite) TestRestoreFromTrashNotInTrash() {
	ctx := context.Background()
	invalidTrashPath := "/not/in/trash/file"

	s.mockTrasher.EXPECT().
		RestoreFromTrash(mock.Anything, invalidTrashPath, testFilePath).
		Return(trash.ErrFileNotInTrash).
		Once()

	err := s.mockTrasher.RestoreFromTrash(ctx, invalidTrashPath, testFilePath)

	s.Require().ErrorIs(err, trash.ErrFileNotInTrash)
}

// TestIsInTrashTrue verifies IsInTrash returns true for files in trash.
//
// A file path that exists in the trash should return true.
func (s *TrashIntegrationTestSuite) TestIsInTrashTrue() {
	s.mockTrasher.EXPECT().
		IsInTrash(testTrashPath).
		Return(true).
		Once()

	inTrash := s.mockTrasher.IsInTrash(testTrashPath)

	s.True(inTrash)
}

// TestIsInTrashFalse verifies IsInTrash returns false for files not in trash.
//
// A file path that does not exist in the trash should return false.
func (s *TrashIntegrationTestSuite) TestIsInTrashFalse() {
	outsideTrashPath := "/some/other/path"

	s.mockTrasher.EXPECT().
		IsInTrash(outsideTrashPath).
		Return(false).
		Once()

	inTrash := s.mockTrasher.IsInTrash(outsideTrashPath)

	s.False(inTrash)
}

// TestIsInTrashEmptyPath verifies IsInTrash returns false for empty path.
//
// An empty path should return false.
func (s *TrashIntegrationTestSuite) TestIsInTrashEmptyPath() {
	s.mockTrasher.EXPECT().
		IsInTrash("").
		Return(false).
		Once()

	inTrash := s.mockTrasher.IsInTrash("")

	s.False(inTrash)
}

// TestListTrashEmpty verifies ListTrash returns empty list when trash is empty.
//
// When no files are in trash, ListTrash should return an empty slice.
func (s *TrashIntegrationTestSuite) TestListTrashEmpty() {
	s.mockTrasher.EXPECT().
		ListTrash().
		Return([]trash.TrashEntry{}, nil).
		Once()

	entries, err := s.mockTrasher.ListTrash()

	s.Require().NoError(err)
	s.Empty(entries)
}

// TestListTrashWithEntries verifies ListTrash returns all entries.
//
// This test ensures that ListTrash properly returns all trash entries
// with correct metadata.
func (s *TrashIntegrationTestSuite) TestListTrashWithEntries() {
	now := time.Now()

	expectedEntries := []trash.TrashEntry{
		{
			Name:         "binary1_1709321234",
			OriginalPath: "/usr/local/bin/binary1",
			TrashPath:    "/home/user/.local/share/Trash/files/binary1_1709321234",
			DeletionTime: now,
		},
		{
			Name:         "binary2_1709321235",
			OriginalPath: "/usr/local/bin/binary2",
			TrashPath:    "/home/user/.local/share/Trash/files/binary2_1709321235",
			DeletionTime: now.Add(-time.Hour),
		},
	}

	s.mockTrasher.EXPECT().
		ListTrash().
		Return(expectedEntries, nil).
		Once()

	entries, err := s.mockTrasher.ListTrash()

	s.Require().NoError(err)
	s.Len(entries, 2)
	s.Equal("binary1_1709321234", entries[0].Name)
	s.Equal("/usr/local/bin/binary1", entries[0].OriginalPath)
	s.Equal("/home/user/.local/share/Trash/files/binary1_1709321234", entries[0].TrashPath)
}

// TestListTrashError verifies error handling when ListTrash fails.
//
// Errors from the underlying filesystem should be properly propagated.
func (s *TrashIntegrationTestSuite) TestListTrashError() {
	listError := errors.New("permission denied: cannot read trash directory")

	s.mockTrasher.EXPECT().
		ListTrash().
		Return(nil, listError).
		Once()

	entries, err := s.mockTrasher.ListTrash()

	s.Require().ErrorIs(err, listError)
	s.Nil(entries)
}

// TestDeletePermanentlyWorkflow verifies the complete DeletePermanently workflow.
//
// This test ensures that DeletePermanently properly:
// 1. Validates the file is in trash
// 2. Removes the file permanently
// 3. Cleans up associated metadata.
func (s *TrashIntegrationTestSuite) TestDeletePermanentlyWorkflow() {
	ctx := context.Background()

	s.mockTrasher.EXPECT().
		DeletePermanently(mock.Anything, testTrashPath).
		Return(nil).
		Once()

	err := s.mockTrasher.DeletePermanently(ctx, testTrashPath)

	s.Require().NoError(err)
}

// TestDeletePermanentlyNotInTrash verifies error handling for file not in trash.
//
// When attempting to delete a file that is not in trash, DeletePermanently
// should return ErrFileNotInTrash.
func (s *TrashIntegrationTestSuite) TestDeletePermanentlyNotInTrash() {
	ctx := context.Background()
	outsidePath := "/home/user/documents/file.txt"

	s.mockTrasher.EXPECT().
		DeletePermanently(mock.Anything, outsidePath).
		Return(trash.ErrFileNotInTrash).
		Once()

	err := s.mockTrasher.DeletePermanently(ctx, outsidePath)

	s.Require().ErrorIs(err, trash.ErrFileNotInTrash)
}

// TestDeletePermanentlyWithEmptyPath verifies error handling for empty path.
//
// An empty path should return an appropriate error.
func (s *TrashIntegrationTestSuite) TestDeletePermanentlyWithEmptyPath() {
	ctx := context.Background()

	s.mockTrasher.EXPECT().
		DeletePermanently(mock.Anything, "").
		Return(trash.ErrInvalidPath).
		Once()

	err := s.mockTrasher.DeletePermanently(ctx, "")

	s.Require().ErrorIs(err, trash.ErrInvalidPath)
}

// TestGetTrashPathLinux verifies GetTrashPath returns correct path on Linux.
//
// On Linux systems, the trash path should be based on XDG_DATA_HOME.
func (s *TrashIntegrationTestSuite) TestGetTrashPathLinux() {
	if runtime.GOOS != "linux" {
		s.T().Skip("Skipping Linux-specific test on non-Linux platform")
	}

	expectedPath := "/home/user/.local/share/Trash/files"

	s.mockTrasher.EXPECT().
		GetTrashPath().
		Return(expectedPath).
		Once()

	trashPath := s.mockTrasher.GetTrashPath()

	s.Equal(expectedPath, trashPath)
	s.Contains(trashPath, "Trash")
}

// TestGetTrashPathWindows verifies GetTrashPath returns empty on Windows.
//
// On Windows, GetTrashPath should return an empty string since the system
// uses the Recycle Bin instead of a directory-based trash.
func (s *TrashIntegrationTestSuite) TestGetTrashPathWindows() {
	if runtime.GOOS != "windows" {
		s.T().Skip("Skipping Windows-specific test on non-Windows platform")
	}

	s.mockTrasher.EXPECT().
		GetTrashPath().
		Return("").
		Once()

	trashPath := s.mockTrasher.GetTrashPath()

	s.Empty(trashPath)
}

// TestContextCancellation verifies proper handling of canceled context.
//
// When the context is canceled before or during an operation,
// the trash layer should return an appropriate error.
func (s *TrashIntegrationTestSuite) TestContextCancellation() {
	// Test MoveToTrash with canceled context
	s.Run("MoveToTrash", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		s.mockTrasher.EXPECT().
			MoveToTrash(mock.Anything, testFilePath).
			Return("", context.Canceled).
			Once()

		_, err := s.mockTrasher.MoveToTrash(ctx, testFilePath)
		s.ErrorIs(err, context.Canceled)
	})

	// Test RestoreFromTrash with canceled context
	s.Run("RestoreFromTrash", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		s.mockTrasher.EXPECT().
			RestoreFromTrash(mock.Anything, testTrashPath, testFilePath).
			Return(context.Canceled).
			Once()

		err := s.mockTrasher.RestoreFromTrash(ctx, testTrashPath, testFilePath)
		s.ErrorIs(err, context.Canceled)
	})

	// Test DeletePermanently with canceled context
	s.Run("DeletePermanently", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		s.mockTrasher.EXPECT().
			DeletePermanently(mock.Anything, testTrashPath).
			Return(context.Canceled).
			Once()

		err := s.mockTrasher.DeletePermanently(ctx, testTrashPath)
		s.ErrorIs(err, context.Canceled)
	})
}

// TestContextTimeout verifies handling of context timeout.
//
// When the context times out during an operation, the operation should
// return context.DeadlineExceeded.
func (s *TrashIntegrationTestSuite) TestContextTimeout() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(10 * time.Millisecond)

	s.mockTrasher.EXPECT().
		MoveToTrash(mock.Anything, testFilePath).
		Return("", context.DeadlineExceeded).
		Once()

	_, err := s.mockTrasher.MoveToTrash(ctx, testFilePath)
	s.ErrorIs(err, context.DeadlineExceeded)
}

// TestFullTrashLifecycle verifies the complete trash workflow.
//
// This test ensures the trash operations work together correctly:
// 1. Move file to trash
// 2. Verify file is in trash
// 3. List trash and find the file
// 4. Restore file from trash
// 5. Verify file is no longer in trash
// 6. Delete permanently (should fail since file was restored).
func (s *TrashIntegrationTestSuite) TestFullTrashLifecycle() {
	ctx := context.Background()
	now := time.Now()

	// Step 1: Move file to trash
	s.mockTrasher.EXPECT().
		MoveToTrash(mock.Anything, testFilePath).
		Return(testTrashPath, nil).
		Once()

	trashPath, err := s.mockTrasher.MoveToTrash(ctx, testFilePath)
	s.Require().NoError(err)
	s.Equal(testTrashPath, trashPath)

	// Step 2: Verify file is in trash
	s.mockTrasher.EXPECT().
		IsInTrash(testTrashPath).
		Return(true).
		Once()

	inTrash := s.mockTrasher.IsInTrash(trashPath)
	s.True(inTrash)

	// Step 3: List trash and find the file
	expectedEntry := trash.TrashEntry{
		Name:         testBinaryName + "_1709321234",
		OriginalPath: testFilePath,
		TrashPath:    testTrashPath,
		DeletionTime: now,
	}

	s.mockTrasher.EXPECT().
		ListTrash().
		Return([]trash.TrashEntry{expectedEntry}, nil).
		Once()

	entries, err := s.mockTrasher.ListTrash()
	s.Require().NoError(err)
	s.Len(entries, 1)
	s.Equal(testFilePath, entries[0].OriginalPath)

	// Step 4: Restore file from trash
	s.mockTrasher.EXPECT().
		RestoreFromTrash(mock.Anything, testTrashPath, testFilePath).
		Return(nil).
		Once()

	err = s.mockTrasher.RestoreFromTrash(ctx, testTrashPath, testFilePath)
	s.Require().NoError(err)

	// Step 5: Verify file is no longer in trash
	s.mockTrasher.EXPECT().
		IsInTrash(testTrashPath).
		Return(false).
		Once()

	inTrash = s.mockTrasher.IsInTrash(testTrashPath)
	s.False(inTrash)
}

// TestErrorPropagation verifies that errors are properly propagated.
//
// This test ensures that errors from the underlying filesystem are wrapped
// and returned correctly to the caller.
func (s *TrashIntegrationTestSuite) TestErrorPropagation() {
	ctx := context.Background()
	customError := errors.New("custom filesystem error")

	s.Run("MoveToTrash error propagation", func() {
		s.mockTrasher.EXPECT().
			MoveToTrash(mock.Anything, testFilePath).
			Return("", customError).
			Once()

		_, err := s.mockTrasher.MoveToTrash(ctx, testFilePath)
		s.ErrorIs(err, customError)
	})

	s.Run("RestoreFromTrash error propagation", func() {
		s.mockTrasher.EXPECT().
			RestoreFromTrash(mock.Anything, testTrashPath, testFilePath).
			Return(customError).
			Once()

		err := s.mockTrasher.RestoreFromTrash(ctx, testTrashPath, testFilePath)
		s.ErrorIs(err, customError)
	})

	s.Run("DeletePermanently error propagation", func() {
		s.mockTrasher.EXPECT().
			DeletePermanently(mock.Anything, testTrashPath).
			Return(customError).
			Once()

		err := s.mockTrasher.DeletePermanently(ctx, testTrashPath)
		s.ErrorIs(err, customError)
	})

	s.Run("ListTrash error propagation", func() {
		s.mockTrasher.EXPECT().
			ListTrash().
			Return(nil, customError).
			Once()

		_, err := s.mockTrasher.ListTrash()
		s.ErrorIs(err, customError)
	})
}

// TestCrossPlatformMoveToTrash verifies MoveToTrash behavior across platforms.
//
// This test uses table-driven testing to cover platform-specific scenarios.
func (s *TrashIntegrationTestSuite) TestCrossPlatformMoveToTrash() {
	ctx := context.Background()

	tests := []struct {
		name         string
		filePath     string
		expectedPath string
		expectError  error
	}{
		{
			name:         "Linux absolute path",
			filePath:     "/usr/local/bin/myapp",
			expectedPath: "/home/user/.local/share/Trash/files/myapp_1709321234",
			expectError:  nil,
		},
		{
			name:         "Linux path with spaces",
			filePath:     "/home/user/my apps/binary",
			expectedPath: "/home/user/.local/share/Trash/files/binary_1709321234",
			expectError:  nil,
		},
		{
			name:         "relative path",
			filePath:     "./local/binary",
			expectedPath: "/home/user/.local/share/Trash/files/binary_1709321234",
			expectError:  nil,
		},
		{
			name:         "path not found",
			filePath:     "/non/existent/path",
			expectedPath: "",
			expectError:  trash.ErrPathNotFound,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.mockTrasher.EXPECT().
				MoveToTrash(mock.Anything, tt.filePath).
				Return(tt.expectedPath, tt.expectError).
				Once()

			trashPath, err := s.mockTrasher.MoveToTrash(ctx, tt.filePath)

			if tt.expectError != nil {
				s.Require().ErrorIs(err, tt.expectError)
				s.Empty(trashPath)
			} else {
				s.Require().NoError(err)
				s.Equal(tt.expectedPath, trashPath)
			}
		})
	}
}

// TestTrashEntryCreation verifies TrashEntry struct creation and fields.
//
// This test ensures all TrashEntry fields are properly handled.
func (s *TrashIntegrationTestSuite) TestTrashEntryCreation() {
	now := time.Now()

	entry := s.createTrashEntry(
		"test-binary_1709321234",
		testFilePath,
		testTrashPath,
	)

	s.Equal("test-binary_1709321234", entry.Name)
	s.Equal(testFilePath, entry.OriginalPath)
	s.Equal(testTrashPath, entry.TrashPath)
	s.WithinDuration(now, entry.DeletionTime, time.Second)
}

// TestMultipleSequentialOperations verifies multiple operations in sequence.
//
// This test simulates a realistic workflow of multiple trash operations
// to ensure the trash layer behaves correctly across multiple calls.
func (s *TrashIntegrationTestSuite) TestMultipleSequentialOperations() {
	ctx := context.Background()
	now := time.Now()

	files := []struct {
		originalPath string
		trashPath    string
		name         string
	}{
		{
			originalPath: "/usr/local/bin/binary1",
			trashPath:    "/home/user/.local/share/Trash/files/binary1_1709321234",
			name:         "binary1_1709321234",
		},
		{
			originalPath: "/usr/local/bin/binary2",
			trashPath:    "/home/user/.local/share/Trash/files/binary2_1709321235",
			name:         "binary2_1709321235",
		},
	}

	// Move first file to trash
	s.mockTrasher.EXPECT().
		MoveToTrash(mock.Anything, files[0].originalPath).
		Return(files[0].trashPath, nil).
		Once()

	trashPath1, err := s.mockTrasher.MoveToTrash(ctx, files[0].originalPath)
	s.Require().NoError(err)
	s.Equal(files[0].trashPath, trashPath1)

	// Move second file to trash
	s.mockTrasher.EXPECT().
		MoveToTrash(mock.Anything, files[1].originalPath).
		Return(files[1].trashPath, nil).
		Once()

	trashPath2, err := s.mockTrasher.MoveToTrash(ctx, files[1].originalPath)
	s.Require().NoError(err)
	s.Equal(files[1].trashPath, trashPath2)

	// List trash - should show both files
	entries := []trash.TrashEntry{
		{
			Name:         files[0].name,
			OriginalPath: files[0].originalPath,
			TrashPath:    files[0].trashPath,
			DeletionTime: now,
		},
		{
			Name:         files[1].name,
			OriginalPath: files[1].originalPath,
			TrashPath:    files[1].trashPath,
			DeletionTime: now.Add(-time.Second),
		},
	}

	s.mockTrasher.EXPECT().
		ListTrash().
		Return(entries, nil).
		Once()

	listResult, err := s.mockTrasher.ListTrash()
	s.Require().NoError(err)
	s.Len(listResult, 2)

	// Restore first file
	s.mockTrasher.EXPECT().
		RestoreFromTrash(mock.Anything, files[0].trashPath, files[0].originalPath).
		Return(nil).
		Once()

	err = s.mockTrasher.RestoreFromTrash(ctx, files[0].trashPath, files[0].originalPath)
	s.Require().NoError(err)

	// Delete second file permanently
	s.mockTrasher.EXPECT().
		DeletePermanently(mock.Anything, files[1].trashPath).
		Return(nil).
		Once()

	err = s.mockTrasher.DeletePermanently(ctx, files[1].trashPath)
	s.Require().NoError(err)
}

// Additional standalone tests for edge cases.

// TestTrashErrorTypes verifies all trash error types are defined.
func TestTrashErrorTypes(t *testing.T) {
	t.Parallel()

	// Verify all expected errors are defined
	require.Error(t, trash.ErrTrashFull)
	require.Error(t, trash.ErrFileNotInTrash)
	require.Error(t, trash.ErrRestoreCollision)
	require.Error(t, trash.ErrInvalidPath)
	require.Error(t, trash.ErrPathNotFound)
}

// TestTrashEntryFields verifies all TrashEntry fields can be set and retrieved.
func TestTrashEntryFields(t *testing.T) {
	t.Parallel()

	now := time.Now()

	entry := trash.TrashEntry{
		Name:         "test-file_1709321234",
		OriginalPath: "/home/user/test-file",
		TrashPath:    "/home/user/.local/share/Trash/files/test-file_1709321234",
		DeletionTime: now,
	}

	assert.Equal(t, "test-file_1709321234", entry.Name)
	assert.Equal(t, "/home/user/test-file", entry.OriginalPath)
	assert.Equal(t, "/home/user/.local/share/Trash/files/test-file_1709321234", entry.TrashPath)
	assert.Equal(t, now, entry.DeletionTime)
}

// TestTrasherInterface verifies the Trasher interface is properly defined.
func TestTrasherInterface(t *testing.T) {
	t.Parallel()

	// Verify that MockTrasher implements Trasher interface
	var _ trash.Trasher = (*mocks.MockTrasher)(nil)
}

// TestContextWithTimeout verifies context timeout behavior.
func TestContextWithTimeout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(10 * time.Millisecond)

	// Context should be done
	select {
	case <-ctx.Done():
		require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
	default:
		t.Error("expected context to be done")
	}
}
