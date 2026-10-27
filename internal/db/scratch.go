package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// OpenScratch opens a throwaway database for one computation, so a projection never writes a real
// file. It lives in a private directory under the OS temp dir (tmpfs on most Linux systems) and
// Close deletes it. It is a normal WAL file rather than an in-memory one: SQLite's memdb has no
// WAL, and its readers block behind any open write, which the state store's reads inside its write
// transactions would deadlock on. A scratch file takes no pre-migration backup.
func OpenScratch(ctx context.Context) (*Pools, error) {
	dir, err := os.MkdirTemp("", "twr-scratch-")
	if err != nil {
		return nil, fmt.Errorf("db: scratch dir: %w", err)
	}
	pools, err := Open(ctx, filepath.Join(dir, "scratch.db"))
	if err != nil {
		return nil, errors.Join(err, os.RemoveAll(dir))
	}
	pools.scratchDir = dir
	return pools, nil
}
