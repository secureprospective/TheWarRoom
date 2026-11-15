package history

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/measures"
)

func liveRowCounts(t *testing.T, pools *db.Pools) (int, int) {
	t.Helper()
	var captures, versions int
	if err := pools.Read().QueryRowContext(t.Context(), `SELECT COUNT(*) FROM live_evidence`).Scan(&captures); err != nil {
		t.Fatal(err)
	}
	if err := pools.Read().QueryRowContext(t.Context(), `SELECT COUNT(*) FROM live_chronology`).Scan(&versions); err != nil {
		t.Fatal(err)
	}
	return captures, versions
}

func TestDurableLiveMergedChronologyRestartsIdentically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	s, pools := statStore(t, path)
	scope := source.RawNativeScope{Player: "0099", Season: 2026, Week: 8}
	line := liveLine(300000)
	line.GameID = "game-8"
	line.Line.Quantities["receiving_drop"] = 10000
	line.Provenance = source.ESPNLive + " + " + source.Sleeper
	setLiveStats(t, s, 2026, 8, t0(), source.ESPNLive, nil, map[string]source.PlayerWeek{"0099": line})
	first := liveEvidence(t, s, nflverseEvidence(500000), scope)
	pin := *commissioner.FeedVersion(first.Versions())
	line.Line.Quantities["receiving_drop"] = 20000
	setLiveStats(t, s, 2026, 8, t0().Add(time.Minute), source.ESPNLive, nil, map[string]source.PlayerWeek{"0099": line})
	before := liveEvidence(t, s, nflverseEvidence(600000), scope)
	captures, versions := liveRowCounts(t, pools)
	setLiveStats(t, s, 2026, 8, t0().Add(2*time.Minute), source.ESPNLive, nil, map[string]source.PlayerWeek{"0099": line})
	liveEvidence(t, s, nflverseEvidence(600000), scope)
	if a, b := liveRowCounts(t, pools); a != captures || b != versions {
		t.Fatalf("unchanged poll grew rows: %d/%d -> %d/%d", captures, versions, a, b)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	restored, reopened := statStore(t, path)
	if err := restored.RestoreLiveStats(t.Context()); err != nil {
		t.Fatal(err)
	}
	after := liveEvidence(t, restored, nflverseEvidence(600000), scope)
	if !reflect.DeepEqual(before.Versions(), after.Versions()) {
		t.Fatal("revision chronology changed after restart")
	}
	if after.Versions()[pin.Seq].Digest != pin.Digest || after.Versions()[pin.Seq].Held.Week.Line.Quantities["receiving_drop"] != 10000 {
		t.Fatal("old merged pin changed")
	}
	if a, b := liveRowCounts(t, reopened); a != captures || b != versions {
		t.Fatal("restart appended duplicate revisions")
	}
	setLiveStats(t, restored, 2026, 8, t0(), source.ESPNLive, nil, nil)
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
	cleared, _ := statStore(t, path)
	if err := cleared.RestoreLiveStats(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, ok := cleared.liveHeldWeek("0099", 2026, 8); ok {
		t.Fatal("cleared board resurrected")
	}
	got := countedLive(t, liveEvidence(t, cleared, nflverseEvidence(600000), scope))
	if got.Held.Week.Line.Quantities["receiving_drop"] != 20000 {
		t.Fatal("live-only field lost after clearing and restarting")
	}
}

func TestDurableLiveScoresAndLedgerPinsSurviveRestart(t *testing.T) {
	for _, manual := range []bool{false, true} {
		t.Run(fmt.Sprintf("manual=%t", manual), func(t *testing.T) {
			f := statResultsTest(t)
			player := f.starter(f.a)
			scope := source.RawNativeScope{Player: player, Season: 2026, Week: 2}
			setLiveStats(t, f.s, 2026, 2, t0(), source.ESPNLive, nil, map[string]source.PlayerWeek{player: tackleWeek(2, 30000)})
			evidence, err := f.s.nativeEvidence(t.Context(), f.s, scope)
			if err != nil {
				t.Fatal(err)
			}
			pin := *commissioner.FeedVersion(evidence.Versions())
			if manual {
				item := commissioner.StatItemID{Player: player, Season: 2026, Week: 2, Seq: pin.Seq, Digest: pin.Digest[:16]}.String()
				f.approve(t, item, "pin-live")
			} else if err := f.s.ProcessStatFeed(t.Context(), "native.test", 2026); err != nil {
				t.Fatal(err)
			}
			setLiveStats(t, f.s, 2026, 2, t0().Add(time.Minute), source.Sleeper, nil, map[string]source.PlayerWeek{player: tackleWeek(2, 50000)})
			if err := f.s.ProcessStatFeed(t.Context(), "native.test", 2026); err != nil {
				t.Fatal(err)
			}
			before := f.score(t, f.a, 2)
			want := int64(75000)
			if manual {
				want = 45000
			}
			if before.Points == nil || int64(*before.Points) != want {
				t.Fatalf("wrong score before restart: %+v", before)
			}
			revisionBefore := revision(t, f.s)
			f.reopen(t)
			after := f.score(t, f.a, 2)
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("score changed after restart: before %+v after %+v", before, after)
			}
			if err := f.s.ProcessStatFeed(t.Context(), "native.test", 2026); err != nil {
				t.Fatal(err)
			}
			if revision(t, f.s) != revisionBefore {
				t.Fatal("restart duplicated automatic approval")
			}
		})
	}
}

func TestDurableLiveCapturesDeduplicateButRetainReturns(t *testing.T) {
	s, _, pools := newStore(t)
	for _, yards := range []int{3, 3, 4, 3} {
		setLiveStats(t, s, 2026, 8, t0(), source.ESPNLive, nil, map[string]source.PlayerWeek{"0099": liveLine(10000 * leaguerules.Decimal(yards))})
	}
	if n, _ := liveRowCounts(t, pools); n != 3 {
		t.Fatalf("captures %d, want A/B/A", n)
	}
}

func TestDurableLiveCompactionKeepsReferencedAndMissingFields(t *testing.T) {
	for _, layout := range []struct {
		name string
		open func(*testing.T, string) (*Store, *db.Pools)
	}{{"desktop", statStore}, {"hosted", hostedStatStore}} {
		t.Run(layout.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "history.db")
			s, pools := layout.open(t, path)
			scope := source.RawNativeScope{Player: "0099", Season: 2026, Week: 8}
			setLiveStats(t, s, 2026, 8, t0(), source.ESPNLive, nil, map[string]source.PlayerWeek{"0099": liveLine(10000)})
			pinned := liveEvidence(t, s, source.NativeEvidence{}, scope).Versions()[0]
			setLiveStats(t, s, 2026, 8, t0(), source.ESPNLive, nil, map[string]source.PlayerWeek{"0099": liveLine(20000)})
			gap := liveLine(30000)
			gap.Line.Quantities["receiving_drop"] = 10000
			setLiveStats(t, s, 2026, 8, t0(), source.ESPNLive, nil, map[string]source.PlayerWeek{"0099": gap})
			setLiveStats(t, s, 2026, 8, t0(), source.ESPNLive, nil, map[string]source.PlayerWeek{"0099": liveLine(40000)})
			before := liveEvidence(t, s, nflverseEvidence(500000), scope)
			if err := s.compactLivePlayer(t.Context(), liveStatKey{"0099", 2026, 8}, *nflverseEvidence(500000).Held.Week); err != nil {
				t.Fatal(err)
			}
			if n, _ := liveRowCounts(t, pools); n != 3 {
				t.Fatalf("retention kept %d rows, want pinned + gap + latest", n)
			}
			tx, err := pools.ReferencePools().BeginWrite(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := tx.ExecContext(t.Context(), `DELETE FROM live_evidence`); err == nil || !strings.Contains(err.Error(), "append-only") {
				t.Fatalf("compaction left deletion unguarded: %v", err)
			}
			if err := tx.Rollback(); err != nil {
				t.Fatal(err)
			}
			if err := pools.Close(); err != nil {
				t.Fatal(err)
			}
			restored, _ := layout.open(t, path)
			after := liveEvidence(t, restored, nflverseEvidence(500000), scope)
			if !reflect.DeepEqual(before.Versions(), after.Versions()) || after.Versions()[pinned.Seq].Digest != pinned.Digest {
				t.Fatal("compaction lost replayable reference")
			}
		})
	}
}

func TestDurableLiveCaptureFailurePreservesBook(t *testing.T) {
	s, _, pools := newStore(t)
	scope := source.RawNativeScope{Player: "0099", Season: 2026, Week: 8}
	setLiveStats(t, s, 2026, 8, t0(), source.ESPNLive, nil, map[string]source.PlayerWeek{"0099": liveLine(30000)})
	before := liveEvidence(t, s, source.NativeEvidence{}, scope)
	tx, err := pools.BeginWrite(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(t.Context(), `CREATE TRIGGER fail_live_capture BEFORE INSERT ON live_evidence BEGIN SELECT RAISE(ABORT,'capture fault'); END;`); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := s.SetLiveStats(t.Context(), 2026, 8, t0(), source.ESPNLive, nil, map[string]source.PlayerWeek{"0099": liveLine(40000)}); err == nil {
		t.Fatal("failed capture reported success")
	}
	after := liveEvidence(t, s, source.NativeEvidence{}, scope)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("failed capture changed held evidence")
	}
	if n, _ := liveRowCounts(t, pools); n != 1 {
		t.Fatal("failed capture wrote rows")
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	fresh := New(pools, s.reg)
	if err := fresh.RestoreLiveStats(cancelled); err == nil {
		t.Fatal("cancelled restore succeeded")
	}
	if fresh.liveHeld != nil || fresh.liveRevisions != nil {
		t.Fatal("failed restore installed partial state")
	}
}

func TestDurableLiveDropsPartsTotalsAndDetachesEvidence(t *testing.T) {
	s, _, pools := newStore(t)
	line := liveLine(30000)
	for _, stat := range leaguerules.StatDictionary() {
		if len(stat.Parts) > 0 {
			line.Line.Quantities[stat.ID] = 90000
		}
	}
	setLiveStats(t, s, 2026, 8, t0(), source.ESPNLive, nil, map[string]source.PlayerWeek{"0099": line})
	scope := source.RawNativeScope{Player: "0099", Season: 2026, Week: 8}
	evidence := liveEvidence(t, s, source.NativeEvidence{}, scope)
	for id := range evidence.Held.Week.Line.Quantities {
		stat, _ := leaguerules.StatByID(id)
		if len(stat.Parts) > 0 {
			t.Fatalf("persisted parts total %s", id)
		}
	}
	evidence.Held.Week.Line.Quantities["receiving_yards"] = 990000
	evidence.Lines[0].Held.Week.Line.Quantities["receiving_yards"] = 990000
	restored := New(pools, s.reg)
	after := liveEvidence(t, restored, source.NativeEvidence{}, scope)
	if after.Held.Week.Line.Quantities["receiving_yards"] != 30000 || after.Versions()[0].Held.Week.Line.Quantities["receiving_yards"] != 30000 {
		t.Fatal("detached evidence mutation reached storage")
	}
	again := liveEvidence(t, s, source.NativeEvidence{}, scope)
	if again.Held.Week.Line.Quantities["receiving_yards"] != 30000 || again.Versions()[0].Held.Week.Line.Quantities["receiving_yards"] != 30000 {
		t.Fatal("detached evidence mutation reached held state")
	}
}

func TestDurableLiveWorkerRestoresEarlierLiveOnlyWeeks(t *testing.T) {
	s, _, pools := newStore(t)
	for _, week := range []int{7, 8} {
		setLiveStats(t, s, 2026, week, t0(), source.ESPNLive, nil, map[string]source.PlayerWeek{"0099": liveLine(30000)})
		liveEvidence(t, s, source.NativeEvidence{}, source.RawNativeScope{Player: "0099", Season: 2026, Week: week})
	}
	before := liveWeeks(t, s, nil, 2026)
	restored := New(pools, s.reg)
	after := liveWeeks(t, restored, nil, 2026)
	if len(after) != 2 || !reflect.DeepEqual(before, after) {
		t.Fatal("worker lost earlier live-only week on restart")
	}
}

func hostedStatStore(t *testing.T, path string) (*Store, *db.Pools) {
	t.Helper()
	ctx := t.Context()
	alone, err := db.Open(ctx, path, LeagueSchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	if err := PrepareNewLeague(ctx, alone); err != nil {
		t.Fatal(err)
	}
	reference, err := db.Open(ctx, filepath.Join(filepath.Dir(path), "reference.db"), ReferenceSchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reference.Close() })
	if err := PrepareReferenceFile(ctx, reference); err != nil {
		t.Fatal(err)
	}
	moved, err := db.HasTable(ctx, alone, ReferenceMoveTable)
	if err != nil {
		t.Fatal(err)
	}
	if !moved {
		if err := CommitReferenceMove(ctx, alone, ReferenceMove{File: "reference.db"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := alone.Close(); err != nil {
		t.Fatal(err)
	}
	pools, err := db.OpenWithReference(ctx, path, LeagueSchemaVersion(), reference)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	s := New(pools, reg)
	s.SetClock(t0)
	if err := s.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	return s, pools
}
