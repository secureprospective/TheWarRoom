package sources_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"
)

func TestGuardedRealArchive(t *testing.T) {
	root := os.Getenv("WARROOM_TEST_ARCHIVE")
	if root == "" {
		t.Skip("set WARROOM_TEST_ARCHIVE for read-only real archive test")
	}
	a := testHost(t)
	service := sourcesapp.ServiceFor(a)
	ctx := context.Background()
	first, err := service.ImportArchiveFrom(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if first.Files != 1720 || first.Recorded != 1720 {
		t.Fatalf("first import: %+v", first)
	}
	again, err := service.ImportArchiveFrom(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if again.Files != 1720 || again.Held != 1720 || again.Recorded != 0 {
		t.Fatalf("repeat: %+v", again)
	}
}

func TestCheckArchiveRefusesLinkedRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "2026"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "2026", "league.json"), []byte(`{"league":{"id":"1","baseURL":"https://example.test"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := sourcesapp.CheckArchiveDir(root); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(root, link); err != nil {
		t.Skip(err)
	}
	if err := sourcesapp.CheckArchiveDir(link); err == nil {
		t.Fatal("linked root accepted")
	}
}

func TestHeldIngressAndEgressDetached(t *testing.T) {
	h := &sourcesapp.Held{}
	cw := sourcesapp.CrosswalkReport{Rates: []sourcesapp.CrosswalkRate{{Position: "QB"}}, Unmatched: []sourcesapp.CrosswalkMiss{{MFLID: "0042"}}}
	signals := sourcesapp.SignalsReport{Loads: []sourcesapp.SignalLoad{{Feed: "stats", Missing: []string{"held"}}}}
	h.PublishCrosswalk(cw)
	h.PublishSignals(signals)
	cw.Rates[0].Position, cw.Unmatched[0].MFLID = "mutated", "mutated"
	signals.Loads[0].Feed, signals.Loads[0].Missing[0] = "mutated", "mutated"
	assertHeld := func() {
		t.Helper()
		c, s := h.SnapshotCrosswalk(), h.SnapshotSignals()
		if c.Rates[0].Position != "QB" || c.Unmatched[0].MFLID != "0042" || s.Loads[0].Feed != "stats" || s.Loads[0].Missing[0] != "held" {
			t.Fatal("holder aliases caller storage", c, s)
		}
	}
	assertHeld()
	c, s := h.SnapshotCrosswalk(), h.SnapshotSignals()
	c.Rates[0].Position, c.Unmatched[0].MFLID = "mutated", "mutated"
	s.Loads[0].Feed, s.Loads[0].Missing[0] = "mutated", "mutated"
	assertHeld()
}
