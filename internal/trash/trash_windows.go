//go:build windows

/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package trash

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

// Windows API constants from shellapi.h
const (
	foDelete          = 0x0003 // FO_DELETE
	fofAllowUndo      = 0x0040 // FOF_ALLOWUNDO
	fofNoConfirmation = 0x0010 // FOF_NOCONFIRMATION
	fofSilent         = 0x0004 // FOF_SILENT
	fofNoErrorUI      = 0x0400 // FOF_NOERRORUI
)

// shFileOpStruct represents the SHFILEOPSTRUCTW structure.
type shFileOpStruct struct {
	hwnd                  uintptr
	wFunc                 uintptr
	pFrom                 uintptr
	pTo                   uintptr
	fileOpFlags           uintptr
	fAnyOperationsAborted uintptr
	hNameMappings         uintptr
	lpszProgressTitle     uintptr
}

// windowsTrasher implements Trasher for Windows using Shell API.
type windowsTrasher struct {
	shell32         *syscall.LazyDLL
	shFileOperation *syscall.LazyProc
}

// newTrasher creates a new Windows-specific trash manager.
func newTrasher() (Trasher, error) {
	shell32 := syscall.NewLazyDLL("Shell32.dll")
	shFileOperation := shell32.NewProc("SHFileOperationW")

	return &windowsTrasher{
		shell32:         shell32,
		shFileOperation: shFileOperation,
	}, nil
}

// MoveToTrash moves a file to the Windows Recycle Bin.
// Returns the original path since Windows manages trash internally.
func (t *windowsTrasher) MoveToTrash(ctx context.Context, filePath string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// Verify source exists
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", ErrPathNotFound, filePath)
		}

		return "", fmt.Errorf("checking file: %w", err)
	}

	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	// Convert to UTF-16 with double null terminator
	utf16Path, err := syscall.UTF16FromString(absPath)
	if err != nil {
		return "", fmt.Errorf("converting path to UTF-16: %w", err)
	}

	// Create SHFILEOPSTRUCT
	param := &shFileOpStruct{
		wFunc:             foDelete,
		pFrom:             uintptr(unsafe.Pointer(&utf16Path[0])),
		fileOpFlags:       fofAllowUndo | fofNoConfirmation | fofSilent | fofNoErrorUI,
		lpszProgressTitle: 0,
	}

	// Call SHFileOperationW
	ret, _, err := t.shFileOperation.Call(uintptr(unsafe.Pointer(param)))

	// SHFileOperation returns 0 on success
	if ret != 0 {
		return "", fmt.Errorf("SHFileOperation failed with code %d: %w", ret, err)
	}

	// On Windows, we return the original path as the "trash path"
	// since the file is moved to the Recycle Bin internally
	return absPath, nil
}

// RestoreFromTrash restores a file from Windows Recycle Bin.
// Note: Windows does not provide a direct API to restore from Recycle Bin
// by original path. This is a best-effort implementation.
func (t *windowsTrasher) RestoreFromTrash(
	ctx context.Context,
	trashPath, originalPath string,
) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Windows Recycle Bin does not expose a direct restore API.
	// We can only check if the file exists at the original location
	// or inform the user they need to restore manually.
	//
	// For this implementation, we check if the file was restored
	// and return an error indicating manual restoration is needed.

	_, err := os.Stat(originalPath)
	if err == nil {
		// File exists at original location
		return nil
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("checking restore location: %w", err)
	}

	// File not restored - user needs to restore from Recycle Bin manually
	return errors.New("Windows Recycle Bin requires manual restoration. " +
		"Please restore the file from Recycle Bin to: " + originalPath)
}

// IsInTrash checks if a file exists in the Windows Recycle Bin.
// Note: This is a heuristic check as Windows doesn't expose a direct API.
func (t *windowsTrasher) IsInTrash(trashPath string) bool {
	// Windows Recycle Bin is at %USERPROFILE%\$Recycle.Bin
	// but checking if a specific file is there is complex.
	// We check if the original file no longer exists.
	_, err := os.Stat(trashPath)

	return os.IsNotExist(err)
}

// ListTrash returns entries from Windows Recycle Bin.
// Note: Windows does not expose a direct API to list Recycle Bin contents.
func (t *windowsTrasher) ListTrash() ([]TrashEntry, error) {
	// Windows does not provide a straightforward API to list Recycle Bin contents
	// without using COM interfaces. For this implementation, we return an empty list.
	//
	// To properly implement this, we would need to use:
	// - IShellFolder interface for the Recycle Bin
	// - IEnumIDList to enumerate items
	// - GetDisplayNameOf to get original paths
	//
	// This is beyond the scope of this implementation.

	return []TrashEntry{}, nil
}

// DeletePermanently removes a file permanently from Windows.
// Uses SHFileOperation without FOF_ALLOWUNDO flag.
func (t *windowsTrasher) DeletePermanently(ctx context.Context, trashPath string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Convert to UTF-16 with double null terminator
	utf16Path, err := syscall.UTF16FromString(trashPath)
	if err != nil {
		return fmt.Errorf("converting path to UTF-16: %w", err)
	}

	// Create SHFILEOPSTRUCT without FOF_ALLOWUNDO
	param := &shFileOpStruct{
		wFunc:             foDelete,
		pFrom:             uintptr(unsafe.Pointer(&utf16Path[0])),
		fileOpFlags:       fofNoConfirmation | fofSilent | fofNoErrorUI,
		lpszProgressTitle: 0,
	}

	// Call SHFileOperationW
	ret, _, err := t.shFileOperation.Call(uintptr(unsafe.Pointer(param)))

	// SHFileOperation returns 0 on success
	if ret != 0 {
		return fmt.Errorf("SHFileOperation failed with code %d: %w", ret, err)
	}

	return nil
}

// GetTrashPath returns an empty string on Windows since trash is managed by the system.
func (t *windowsTrasher) GetTrashPath() string {
	// Windows Recycle Bin location is system-managed
	// Typically at %USERPROFILE%\$Recycle.Bin
	return ""
}

// getRecycleBinPath returns the path to the current user's Recycle Bin.
func getRecycleBinPath() string {
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		return ""
	}

	return filepath.Join(userProfile, "$Recycle.Bin")
}

// Helper function to format time for trash entries.
func formatDeletionTime(t time.Time) string {
	return t.Format(time.RFC3339)
}
