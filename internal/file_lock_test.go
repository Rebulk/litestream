//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || windows

package internal_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/benbjohnson/litestream/internal"
)

func TestTryLockFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	open := func() *os.File {
		t.Helper()
		f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = f.Close() })
		return f
	}

	first, second := open(), open()
	if err := internal.TryLockFile(first); err != nil {
		t.Fatalf("lock first handle: %v", err)
	}

	// A second handle is refused even within the same process.
	if err := internal.TryLockFile(second); !errors.Is(err, internal.ErrFileLocked) {
		t.Fatalf("lock second handle: err=%v, want ErrFileLocked", err)
	}

	if err := internal.ReleaseFileLock(first); err != nil {
		t.Fatalf("release first handle: %v", err)
	}
	if err := internal.TryLockFile(second); err != nil {
		t.Fatalf("lock second handle after release: %v", err)
	}
	if err := internal.ReleaseFileLock(second); err != nil {
		t.Fatalf("release second handle: %v", err)
	}
}
