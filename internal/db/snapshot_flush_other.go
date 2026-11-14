//go:build !linux

package db

import "os"

// writeOut is a no-op off Linux: there is no range writeback without a journal commit, and the
// snapshot's final fsync still makes both files durable.
func writeOut(*os.File) error { return nil }
