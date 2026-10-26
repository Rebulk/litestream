//go:build windows

package internal

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// TryLockFile takes an exclusive, non-blocking lock on the first byte of f.
// It returns ErrFileLocked if another handle already holds the lock,
// including one in the same process. The lock is released by ReleaseFileLock.
func TryLockFile(f *os.File) error {
	ol := windows.Overlapped{}
	if err := windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &ol); err != nil {
		if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			return ErrFileLocked
		}
		return err
	}
	return nil
}

// ReleaseFileLock releases a lock taken by TryLockFile. Release before closing:
// Windows does not always drop a lock as soon as its handle closes.
func ReleaseFileLock(f *os.File) error {
	ol := windows.Overlapped{}
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &ol)
}
