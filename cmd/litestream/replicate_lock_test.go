package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// TestReplicateCommand_RefusesSecondInstance runs two `litestream replicate`
// processes against one database. The second must exit with an error instead
// of replicating alongside the first.
func TestReplicateCommand_RefusesSecondInstance(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses SIGTERM to stop the first process")
	}

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "db")
	replicaURL := "file://" + filepath.ToSlash(filepath.Join(dir, "replica"))

	sqldb, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqldb.Exec(`PRAGMA journal_mode = wal; CREATE TABLE t (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if err := sqldb.Close(); err != nil {
		t.Fatal(err)
	}

	first := litestreamMainCommand(t, "replicate", dbPath, replicaURL)
	stdout, err := first.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var firstStderr bytes.Buffer
	first.Stderr = &firstStderr
	if err := first.Start(); err != nil {
		t.Fatal(err)
	}
	firstDone := make(chan error, 1)
	t.Cleanup(func() {
		_ = first.Process.Kill()
		<-firstDone
	})

	// Wait until the first process reports the database open.
	ready := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "initialized db") {
				close(ready)
				break
			}
		}
		_, _ = io.Copy(io.Discard, stdout)
		firstDone <- first.Wait()
	}()
	select {
	case <-ready:
	case err := <-firstDone:
		firstDone <- err
		t.Fatalf("first replicate exited early: %v\nstderr:\n%s", err, firstStderr.String())
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for first replicate to open the database")
	}

	second := litestreamMainCommand(t, "replicate", dbPath, replicaURL)
	var secondOut bytes.Buffer
	second.Stdout = &secondOut
	second.Stderr = &secondOut
	secondDone := make(chan error, 1)
	if err := second.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { secondDone <- second.Wait() }()
	select {
	case err := <-secondDone:
		if err == nil {
			t.Fatalf("second replicate exited 0, want an error\noutput:\n%s", secondOut.String())
		}
	case <-time.After(30 * time.Second):
		_ = second.Process.Kill()
		<-secondDone
		t.Fatalf("second replicate is still running beside the first\noutput:\n%s", secondOut.String())
	}
	if !strings.Contains(secondOut.String(), "already being replicated by another litestream instance") {
		t.Fatalf("second replicate output does not name the lock:\n%s", secondOut.String())
	}

	// After the first process shuts down, a new one can start.
	if err := first.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-firstDone:
		firstDone <- err
		if err != nil {
			t.Fatalf("first replicate shutdown: %v\nstderr:\n%s", err, firstStderr.String())
		}
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for first replicate to shut down")
	}

	third := litestreamMainCommand(t, "replicate", "-once", dbPath, replicaURL)
	if out, err := third.CombinedOutput(); err != nil {
		t.Fatalf("replicate after first shutdown: %v\noutput:\n%s", err, out)
	}
}

// litestreamMainCommand returns a command that runs main() with args in a
// child copy of the test binary (see TestMainHarness).
func litestreamMainCommand(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestMainHarness$")
	cmd.Env = append(os.Environ(),
		"LITESTREAM_TEST_MAIN=1",
		"LITESTREAM_TEST_ARGS="+string(argsJSON),
	)
	return cmd
}
