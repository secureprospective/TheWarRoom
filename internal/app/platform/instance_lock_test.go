package platform

import (
	"path/filepath"
	"testing"
)

// TestInstanceLock_SecondAcquireFails proves the single-instance guard: once the
// lock is held, a second acquire on the same DB path is refused; after release a
// fresh acquire succeeds.
func TestInstanceLock_SecondAcquireFails(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "thewarroom.db")

	first, err := AcquireInstanceLock(dbPath)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	if second, err := AcquireInstanceLock(dbPath); err == nil {
		ReleaseInstanceLock(second)
		t.Fatal("second acquire succeeded while the lock was held — instance guard failed")
	}

	ReleaseInstanceLock(first)

	third, err := AcquireInstanceLock(dbPath)
	if err != nil {
		t.Fatalf("acquire after release: %v", err)
	}
	ReleaseInstanceLock(third)
}

// TestInstanceLock_DevAndRealDoNotContend proves the lock is keyed to the DB path,
// so a dev build and a real build (different DB files) never falsely block.
func TestInstanceLock_DevAndRealDoNotContend(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	realLock, err := AcquireInstanceLock(filepath.Join(dir, "thewarroom.db"))
	if err != nil {
		t.Fatalf("acquire real: %v", err)
	}
	defer ReleaseInstanceLock(realLock)

	dev, err := AcquireInstanceLock(filepath.Join(dir, "thewarroom-dev.db"))
	if err != nil {
		t.Fatalf("dev build must not contend with the real ledger lock: %v", err)
	}
	ReleaseInstanceLock(dev)
}
