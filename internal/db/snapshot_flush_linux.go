//go:build linux

package db

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// writeOut pushes f's dirty pages to the device and waits for them, without the journal commit an
// fsync forces: on ext4 that commit is shared with the app writer's own fsync, so syncing a large
// snapshot file with fsync made every app write wait behind it.
func writeOut(f *os.File) error {
	const flags = unix.SYNC_FILE_RANGE_WAIT_BEFORE | unix.SYNC_FILE_RANGE_WRITE | unix.SYNC_FILE_RANGE_WAIT_AFTER
	if err := unix.SyncFileRange(int(f.Fd()), 0, 0, flags); err != nil {
		return fmt.Errorf("write out %s: %w", f.Name(), err)
	}
	return nil
}
