package archive

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type fakeHistory struct {
	attempts []history.SnapshotRecord
	latest   string
	err      error
}

func (f fakeHistory) Snapshots(context.Context, string) ([]history.SnapshotRecord, error) {
	return f.attempts, f.err
}

func (f fakeHistory) LatestManifest(context.Context, string) (history.ManifestRecord, bool, error) {
	return history.ManifestRecord{SHA256: f.latest}, f.latest != "", nil
}

func (f fakeHistory) ManifestBySHA(context.Context, string) (history.ManifestRecord, error) {
	return history.ManifestRecord{}, errors.New("no manifest held")
}

func TestReadKeepsTheHeldSnapshotAndMarksItStale(t *testing.T) {
	t.Parallel()
	rules := lab.CheckExceptions().SHA256
	good := history.SnapshotRecord{ManifestSHA: "m1", Compiler: "c", RulesHash: rules, ResultSHA: "r1"}
	held := Snapshot{SHA: "r1", Cutoff: "held"}
	for _, c := range []struct {
		name     string
		attempts []history.SnapshotRecord
		latest   string
		stale    bool
	}{
		{"newest attempt, current manifest", []history.SnapshotRecord{good}, "m1", false},
		{"newer manifest published", []history.SnapshotRecord{good}, "m2", true},
		{"no manifest published", []history.SnapshotRecord{good}, "", true},
		{"newer attempt failed", []history.SnapshotRecord{{ManifestSHA: "m2", Compiler: "c", Error: "check failed"}, good}, "m1", true},
		{"newer attempt by another compiler", []history.SnapshotRecord{{ManifestSHA: "m1", Compiler: "old", ResultSHA: "r0"}, good}, "m1", true},
	} {
		src := Source{History: fakeHistory{attempts: c.attempts, latest: c.latest}, Manifest: "frozen", Compiler: "c"}
		got, stale, err := Read(t.Context(), src, held)
		if err != nil || got.Cutoff != "held" || stale != c.stale {
			t.Errorf("%s: got %q stale %v err %v, want the held snapshot stale %v", c.name, got.Cutoff, stale, err, c.stale)
		}
	}
	changed := good
	changed.RulesHash = "older rules"
	if _, stale, _ := Read(t.Context(), Source{History: fakeHistory{attempts: []history.SnapshotRecord{changed}, latest: "m1"}, Compiler: "c"}, held); !stale {
		t.Error("a snapshot checked under older rules is not stale")
	}
}

func TestReadWithoutAVerifiedCompile(t *testing.T) {
	t.Parallel()
	src := Source{History: fakeHistory{attempts: []history.SnapshotRecord{{Compiler: "c", Error: "failed"}}}, Compiler: "c"}
	if got, stale, err := Read(t.Context(), src, Snapshot{SHA: "r1"}); err != nil || stale || got.SHA != "" {
		t.Fatal("with no verified compile Read returns the zero snapshot", got.SHA, stale, err)
	}
	src.History = fakeHistory{err: errors.New("disk")}
	if _, _, err := Read(t.Context(), src, Snapshot{}); err == nil || !strings.HasPrefix(err.Error(), "archive: snapshots") {
		t.Fatal(err)
	}
	src.History = fakeHistory{attempts: []history.SnapshotRecord{{ManifestSHA: "m1", Compiler: "c", ResultSHA: "r2"}}, latest: "m1"}
	if _, _, err := Read(t.Context(), src, Snapshot{SHA: "r1"}); err == nil || !strings.Contains(err.Error(), "archive: snapshot manifest") {
		t.Fatal("a snapshot that needs rebuilding reads its manifest", err)
	}
}
