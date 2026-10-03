package main

import (
	"fmt"
	"os"
	"syscall"
)

// acquireInstanceLock takes a non-blocking flock on a lockfile beside the database, so a
// second copy of the app cannot open the same ledger. Keep the file open for the life of the
// process: closing it releases the lock.
func acquireInstanceLock(dbPath string) (*os.File, error) {
	path := dbPath + ".lock"
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lockfile %q: %w", path, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf(
			"another instance of The War Room is already running (lock %q is held): %w", path, err)
	}
	return f, nil
}

// releaseInstanceLock is safe on nil (startup failed before locking).
func releaseInstanceLock(f *os.File) {
	if f == nil {
		return
	}
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	_ = f.Close()
}
