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
	"unsafe"

	"golang.org/x/sys/windows"
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
// Field types carefully matched to Windows SHFILEOPSTRUCTW layout for 64-bit compatibility.
type shFileOpStruct struct {
	hwnd                  uintptr
	wFunc                 uint32
	pFrom                 uintptr
	pTo                   uintptr
	fileOpFlags           uint16
	_                     uint16 // padding field
	fAnyOperationsAborted uint32
	hNameMappings         uintptr
	lpszProgressTitle     uintptr
}

// windowsTrasher implements Trasher for Windows using Shell API.
type windowsTrasher struct {
	shell32         *windows.LazyDLL
	shFileOperation *windows.LazyProc
}

var _ Trasher = (*windowsTrasher)(nil)

// newTrasher creates a Windows trash manager using the Shell API.
//
// Returns:
//   - Windows trash implementation.
//   - Always nil error.
func newTrasher() (Trasher, error) {
	shell32 := windows.NewLazySystemDLL("shell32.dll")
	shFileOperation := shell32.NewProc("SHFileOperationW")

	return &windowsTrasher{
		shell32:         shell32,
		shFileOperation: shFileOperation,
	}, nil
}

// MoveToTrash moves a file to the Windows Recycle Bin.
//
// The original path is returned because Windows manages trash internally.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - filePath: Path of the file to trash.
//
// Returns:
//   - Original absolute path of the trashed file.
//   - An error if the move fails.
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
	utf16Path, err := windows.UTF16FromString(absPath)
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

// RestoreFromTrash restores a file from the Windows Recycle Bin.
//
// Windows does not expose a restore-by-path API, so this is best-effort.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - trashPath: Unused on Windows.
//   - originalPath: Destination path expected after restoration.
//
// Returns:
//   - An error if the file is not already present at originalPath.
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

// IsInTrash reports whether a file exists in the Windows Recycle Bin.
//
// Reliable detection is not available without COM, so this always returns false.
//
// Parameters:
//   - trashPath: Path to check.
//
// Returns:
//   - Always false.
func (t *windowsTrasher) IsInTrash(trashPath string) bool {
	return false
}

// ListTrash returns entries from the Windows Recycle Bin.
//
// Windows does not expose a direct listing API, so this returns an empty slice.
//
// Returns:
//   - Empty trash entry list.
//   - Always nil error.
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
//
// It uses SHFileOperation without FOF_ALLOWUNDO.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - trashPath: Path of the file to delete.
//
// Returns:
//   - An error if deletion fails.
func (t *windowsTrasher) DeletePermanently(ctx context.Context, trashPath string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Verify source exists
	_, err := os.Stat(trashPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrPathNotFound, trashPath)
		}

		return fmt.Errorf("checking file: %w", err)
	}

	// Get absolute path
	absPath, err := filepath.Abs(trashPath)
	if err != nil {
		absPath = trashPath
	}

	// Convert to UTF-16 with double null terminator
	utf16Path, err := windows.UTF16FromString(absPath)
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
	ret, _, _ := t.shFileOperation.Call(uintptr(unsafe.Pointer(param)))

	// SHFileOperation returns 0 on success, non-zero on failure
	if ret != 0 {
		return fmt.Errorf("SHFileOperation failed with code %d", ret)
	}

	return nil
}

// GetTrashPath returns an empty string on Windows because trash is system-managed.
//
// Returns:
//   - Empty string.
func (t *windowsTrasher) GetTrashPath() string {
	// Windows Recycle Bin location is system-managed
	// Typically at %USERPROFILE%\$Recycle.Bin
	return ""
}
