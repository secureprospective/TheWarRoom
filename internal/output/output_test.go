package output

import (
	"context"
	"errors"
	"math"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/engine"
)

const (
	testSeason = 2026
	testConfig = 1
)

func newStore(t *testing.T) *Store {
	t.Helper()
	pools, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "output.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	s := New(pools)
	if err := s.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	return s
}

// result builds a finite, valid engine.Result with the given adjusted score and tier.
func result(adjusted float64, tier engine.CapTier, vet bool) engine.Result {
	return engine.Result{
		BasePoints: 100, AgePull: 1.0,
		Layer4Output: engine.Layer4Output{
			FilmEffective: 1.0, FilmRaw: 0.5, RASEffective: 1.0,
			BreakoutEffective: 1.0, Combined: 1.0,
		},
		ScoutingAdjusted: adjusted, CapMultiplier: 1.0, CapTier: tier,
		AdjustedScore: adjusted,
		Tiebreaker:    engine.TiebreakerKey{IsVeteran: vet, RAS: 9.5, ScarcityRank: 3},
	}
}

func TestWriteAndReadBackSorted(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	recs := []ScoreRecord{
		{MFLID: "0001", Result: result(50, engine.CapTierNeutral, true)},
		{MFLID: "0002", Result: result(90, engine.CapTierHot, false)},
		{MFLID: "0003", Result: result(70, engine.CapTierCold, true)},
	}
	if err := s.Writer().Write(ctx, testSeason, testConfig, recs); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := s.Reader().Scores(ctx, testSeason, testConfig)
	if err != nil {
		t.Fatalf("Scores: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d scores, want 3", len(got))
	}
	// AdjustedScore DESC: 90, 70, 50.
	wantOrder := []string{"0002", "0003", "0001"}
	for i, w := range wantOrder {
		if got[i].MFLID != w {
			t.Errorf("rank %d = %q, want %q", i, got[i].MFLID, w)
		}
	}
	// Field fidelity on the top row.
	top := got[0]
	if top.AdjustedScore != 90 || top.CapTier != engine.CapTierHot || top.Tiebreaker.RAS != 9.5 {
		t.Errorf("top row fidelity wrong: %+v", top)
	}
	if top.ScoringConfigID != testConfig || top.Season != testSeason {
		t.Errorf("top row tag wrong: season=%d cfg=%d", top.Season, top.ScoringConfigID)
	}
}

func TestScoreLookup(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	if err := s.Writer().Write(ctx, testSeason, testConfig,
		[]ScoreRecord{{MFLID: "0042", Result: result(33, engine.CapTierNeutral, false)}}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, ok, err := s.Reader().Score(ctx, testSeason, testConfig, "0042")
	if err != nil || !ok {
		t.Fatalf("Score: ok=%v err=%v", ok, err)
	}
	if got.AdjustedScore != 33 {
		t.Errorf("AdjustedScore = %v, want 33", got.AdjustedScore)
	}
	if _, ok, _ := s.Reader().Score(ctx, testSeason, testConfig, "9999"); ok {
		t.Error("Score for absent player returned ok=true")
	}
}

// DECISION-010: a new scoring config writes NEW records and never touches the old ones.
func TestDecision010NewConfigDoesNotRescore(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	if err := s.Writer().Write(ctx, testSeason, 1,
		[]ScoreRecord{{MFLID: "0001", Result: result(50, engine.CapTierNeutral, true)}}); err != nil {
		t.Fatalf("Write cfg1: %v", err)
	}
	if err := s.Writer().Write(ctx, testSeason, 2,
		[]ScoreRecord{{MFLID: "0001", Result: result(80, engine.CapTierHot, true)}}); err != nil {
		t.Fatalf("Write cfg2: %v", err)
	}
	old, _, _ := s.Reader().Score(ctx, testSeason, 1, "0001")
	if old.AdjustedScore != 50 {
		t.Errorf("cfg1 score = %v, want 50 (must be untouched by cfg2)", old.AdjustedScore)
	}
	fresh, _, _ := s.Reader().Score(ctx, testSeason, 2, "0001")
	if fresh.AdjustedScore != 80 {
		t.Errorf("cfg2 score = %v, want 80", fresh.AdjustedScore)
	}
}

// Duplicate-key policy: a re-write of the same (season, config, mfl_id) is drift.
func TestDuplicateKeyRejected(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	rec := []ScoreRecord{{MFLID: "0001", Result: result(50, engine.CapTierNeutral, true)}}
	if err := s.Writer().Write(ctx, testSeason, testConfig, rec); err != nil {
		t.Fatalf("first Write: %v", err)
	}
	err := s.Writer().Write(ctx, testSeason, testConfig, rec)
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate Write error = %v, want ErrDuplicate", err)
	}
	// The batch must roll back whole, not partially persist a duplicate.
	got, _ := s.Reader().Scores(ctx, testSeason, testConfig)
	if len(got) != 1 {
		t.Errorf("after rejected duplicate, %d rows persisted, want 1", len(got))
	}
}

// A duplicate id WITHIN one batch is self-contradictory input — rejected, but NOT as
// ErrDuplicate (nothing was previously persisted), and nothing lands.
func TestInBatchDuplicateRejected(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	err := s.Writer().Write(ctx, testSeason, testConfig, []ScoreRecord{
		{MFLID: "0001", Result: result(50, engine.CapTierNeutral, true)},
		{MFLID: "0001", Result: result(60, engine.CapTierHot, true)},
	})
	if err == nil {
		t.Fatal("in-batch duplicate Write succeeded, want fail-loud")
	}
	if errors.Is(err, ErrDuplicate) {
		t.Error("in-batch duplicate reported as ErrDuplicate (drift), want a distinct error")
	}
	if got, _ := s.Reader().Scores(ctx, testSeason, testConfig); len(got) != 0 {
		t.Errorf("in-batch duplicate persisted %d rows, want 0", len(got))
	}
}

// Reads encode the L6 tiebreaker: among equal AdjustedScores, veteran > non-veteran,
// then higher RAS, then higher scarcity, then mfl id ascending as the final stable key.
func TestTiebreakerOrdering(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	tie := func(vet bool, ras float64, scar int) engine.Result {
		r := result(50, engine.CapTierNeutral, vet) // identical AdjustedScore = 50
		r.Tiebreaker = engine.TiebreakerKey{IsVeteran: vet, RAS: ras, ScarcityRank: scar}
		return r
	}
	recs := []ScoreRecord{
		{MFLID: "0001", Result: tie(false, 9.9, 9)}, // non-vet — must sink despite high RAS
		{MFLID: "0002", Result: tie(true, 8.0, 1)},  // vet, lower RAS — wins on tenure
		{MFLID: "0003", Result: tie(true, 9.0, 1)},  // vet, higher RAS — beats 0002
		{MFLID: "0004", Result: tie(true, 9.0, 5)},  // vet, RAS ties 0003, higher scarcity — beats 0003
	}
	if err := s.Writer().Write(ctx, testSeason, testConfig, recs); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := s.Reader().Scores(ctx, testSeason, testConfig)
	if err != nil {
		t.Fatalf("Scores: %v", err)
	}
	want := []string{"0004", "0003", "0002", "0001"}
	for i, w := range want {
		if got[i].MFLID != w {
			t.Errorf("tiebreak rank %d = %q, want %q (full order %v)", i, got[i].MFLID, w, ids(got))
		}
	}
}

func ids(ss []SeasonScore) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = s.MFLID
	}
	return out
}

// AD-04 (half 1): the Go API is append-only — no Update/Delete exists. A Writer value
// must NOT be assertable to anything exposing mutation beyond Write; and a Reader must
// not be assertable back to Writer.
func TestReaderCannotMutate(t *testing.T) {
	s := newStore(t)
	if _, ok := s.Reader().(Writer); ok {
		t.Fatal("Reader is type-assertable to Writer — single-writer boundary breached")
	}
}

// AD-04 (half 2): the DB itself rejects mutation. A raw UPDATE/DELETE through the write
// pool must ABORT on the trigger even though the Go API never offers one.
func TestImmutabilityTriggerBlocksUpdateAndDelete(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	if err := s.Writer().Write(ctx, testSeason, testConfig,
		[]ScoreRecord{{MFLID: "0001", Result: result(50, engine.CapTierNeutral, true)}}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, err := s.pools.Write().ExecContext(ctx,
		`UPDATE season_scores SET adjusted_score = 999 WHERE mfl_id = '0001'`); err == nil {
		t.Error("raw UPDATE succeeded — immutability trigger did not fire")
	}
	if _, err := s.pools.Write().ExecContext(ctx,
		`DELETE FROM season_scores WHERE mfl_id = '0001'`); err == nil {
		t.Error("raw DELETE succeeded — immutability trigger did not fire")
	}
	// The row is unchanged.
	got, _, _ := s.Reader().Score(ctx, testSeason, testConfig, "0001")
	if got.AdjustedScore != 50 {
		t.Errorf("score after blocked mutation = %v, want 50", got.AdjustedScore)
	}
}

func TestWriteRejectsNonFinite(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	bad := result(math.NaN(), engine.CapTierNeutral, true)
	err := s.Writer().Write(ctx, testSeason, testConfig,
		[]ScoreRecord{{MFLID: "0001", Result: bad}})
	if err == nil {
		t.Fatal("Write accepted a NaN AdjustedScore")
	}
	got, _ := s.Reader().Scores(ctx, testSeason, testConfig)
	if len(got) != 0 {
		t.Errorf("non-finite batch persisted %d rows, want 0", len(got))
	}
}

func TestWriteRejectsBadID(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	for _, bad := range []string{"", "-5", "12a", "00 1"} {
		err := s.Writer().Write(ctx, testSeason, testConfig,
			[]ScoreRecord{{MFLID: bad, Result: result(50, engine.CapTierNeutral, true)}})
		if err == nil {
			t.Errorf("Write accepted malformed id %q", bad)
		}
	}
}

func TestWriteRejectsUnknownCapTier(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	err := s.Writer().Write(ctx, testSeason, testConfig,
		[]ScoreRecord{{MFLID: "0001", Result: result(50, engine.CapTier("Lukewarm"), true)}})
	if err == nil {
		t.Fatal("Write accepted an unknown cap tier")
	}
}

func TestWriteEmptyBatchFails(t *testing.T) {
	s := newStore(t)
	if err := s.Writer().Write(context.Background(), testSeason, testConfig, nil); err == nil {
		t.Fatal("empty-batch Write succeeded, want fail-loud")
	}
}

// AdjustedScore must round-trip at full float64 precision (L6 tiebreak needs exact
// equality).
func TestAdjustedScorePrecisionPreserved(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	want := 123.456789012345
	if err := s.Writer().Write(ctx, testSeason, testConfig,
		[]ScoreRecord{{MFLID: "0001", Result: result(want, engine.CapTierNeutral, true)}}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, _, _ := s.Reader().Score(ctx, testSeason, testConfig, "0001")
	if got.AdjustedScore != want {
		t.Errorf("AdjustedScore round-trip = %v, want %v (exact)", got.AdjustedScore, want)
	}
}
