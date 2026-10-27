//go:build windows

package platform

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// AcquireInstanceLock takes a non-blocking exclusive LockFileEx on a lockfile beside the database,
// the Windows twin of the flock in instance_lock_unix.go. Keep the file open for the life of the
// process: closing it releases the lock.
func AcquireInstanceLock(dbPath string) (*os.File, error) {
	path := dbPath + ".lock"
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lockfile %q: %w", path, err)
	}
	ol := new(windows.Overlapped)
	if err := windows.LockFileEx(windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, ol); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf(
			"another instance of The War Room is already running (lock %q is held): %w", path, err)
	}
	return f, nil
}

// ReleaseInstanceLock is safe on nil (startup failed before locking).
func ReleaseInstanceLock(f *os.File) {
	if f == nil {
		return
	}
	_ = windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, new(windows.Overlapped))
	_ = f.Close()
}
