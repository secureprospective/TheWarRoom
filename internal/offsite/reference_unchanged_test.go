package offsite

import (
	"testing"
	"time"
)

func TestUnchangedReferenceRenewsWithoutUpload(t *testing.T) {
	r2, fake := testR2(t)
	managers, order := testManagers(t, r2, time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC), nil)
	manager := managers[ReferenceSet]
	if err := manager.Tick(t.Context()); err != nil {
		t.Fatal(err)
	}
	fake.failPut = true
	if err := manager.Tick(t.Context()); err != nil {
		t.Fatalf("unchanged reference attempted an upload: %v", err)
	}
	if len(*order) != 1 || manager.CheckWrite() != nil {
		t.Fatalf("unchanged reference was captured again or not renewed: %v", *order)
	}
}
