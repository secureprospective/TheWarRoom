package main

import (
	"fmt"
	"os"
	"syscall"
)

// acquireInstanceLock takes an exclusive, non-blocking advisory lock on a sidecar
// lockfile next to the database so a SECOND running copy of the app cannot open the
// same ledger concurrently (Tier 3 self-protection). The lock is keyed to the DB
// path, so a dev build (its own -dev database) and a real build never falsely
// contend. The returned *os.File MUST stay open for the whole process lifetime —
// closing it, or process exit, releases the lock. Linux-only (D-V3): flock is the
// advisory lock every desktop-launcher path goes through.
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

// releaseInstanceLock unlocks and closes the lockfile handle. Safe on a nil handle
// (a startup that failed before the lock was taken).
func releaseInstanceLock(f *os.File) {
	if f == nil {
		return
	}
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	_ = f.Close()
}
