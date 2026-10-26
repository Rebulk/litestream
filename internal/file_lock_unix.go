//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package internal

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// TryLockFile takes an exclusive, non-blocking flock on f. It returns
// ErrFileLocked if another open file description already holds the lock,
// including one in the same process. The lock is released by ReleaseFileLock
// or when f is closed.
func TryLockFile(f *os.File) error {
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		if errors.Is(err, unix.EWOULDBLOCK) {
			return ErrFileLocked
		}
		return err
	}
	return nil
}

// ReleaseFileLock releases a lock taken by TryLockFile.
func ReleaseFileLock(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN)
}
