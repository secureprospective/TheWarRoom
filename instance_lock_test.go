package main

import (
	"path/filepath"
	"testing"
)

func TestInstanceLockRefusesASecondCopyUntilReleased(t *testing.T) {
	t.Parallel()
	db := filepath.Join(t.TempDir(), "thewarroom.db")
	first, err := acquireInstanceLock(db)
	if err != nil {
		t.Fatalf("first lock: %v", err)
	}
	if second, err := acquireInstanceLock(db); err == nil {
		releaseInstanceLock(second)
		t.Fatal("a second copy took the lock the first still holds")
	}
	releaseInstanceLock(first)
	again, err := acquireInstanceLock(db)
	if err != nil {
		t.Fatalf("lock not released: %v", err)
	}
	releaseInstanceLock(again)
}
