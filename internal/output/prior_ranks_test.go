package output

import (
	"context"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/engine"
)

// write is a terse helper: append one config's board as (mflID, adjustedScore) pairs.
func write(t *testing.T, s *Store, cfg int, scores map[string]float64) {
	t.Helper()
	recs := make([]ScoreRecord, 0, len(scores))
	for id, sc := range scores {
		recs = append(recs, ScoreRecord{MFLID: id, Result: result(sc, engine.CapTierNeutral, true)})
	}
	if err := s.Writer().Write(context.Background(), testSeason, cfg, recs); err != nil {
		t.Fatalf("Write cfg %d: %v", cfg, err)
	}
}

// A first-ever board has nothing to compare against. This must report ok=false and NOT an
// error — the UI renders absent-delta differently from zero-delta, and collapsing the two
// would claim every player "held position" on the very first run.
func TestPriorRanksNoEarlierConfig(t *testing.T) {
	s := newStore(t)
	write(t, s, 5, map[string]float64{"0001": 90, "0002": 50})

	ranks, ok, err := s.Reader().PriorRanks(context.Background(), testSeason, 5)
	if err != nil {
		t.Fatalf("PriorRanks: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false with no earlier config, got ok=true with %v", ranks)
	}
}

// The prior board must be ranked by the SAME order the live board uses (adjusted_score
// DESC, then mfl_id) — otherwise the delta is measured against a differently-sorted list
// and manufactures movement that never happened.
func TestPriorRanksUsesBoardOrder(t *testing.T) {
	s := newStore(t)
	write(t, s, 1, map[string]float64{"0001": 10, "0002": 90, "0003": 50})

	ranks, ok, err := s.Reader().PriorRanks(context.Background(), testSeason, 2)
	if err != nil {
		t.Fatalf("PriorRanks: %v", err)
	}
	if !ok {
		t.Fatal("expected a prior board to be found")
	}
	want := map[string]int{"0002": 1, "0003": 2, "0001": 3}
	for id, w := range want {
		if ranks[id] != w {
			t.Errorf("rank of %s = %d, want %d (full: %v)", id, ranks[id], w, ranks)
		}
	}
}

// Ties must break on mfl_id, matching the board's ORDER BY. Without the secondary key the
// prior ranking is nondeterministic and the delta flickers between reads on identical data.
func TestPriorRanksTieBreaksOnID(t *testing.T) {
	s := newStore(t)
	write(t, s, 1, map[string]float64{"0009": 50, "0002": 50, "0005": 50})

	ranks, ok, err := s.Reader().PriorRanks(context.Background(), testSeason, 2)
	if err != nil || !ok {
		t.Fatalf("PriorRanks: ok=%v err=%v", ok, err)
	}
	want := map[string]int{"0002": 1, "0005": 2, "0009": 3}
	for id, w := range want {
		if ranks[id] != w {
			t.Errorf("rank of %s = %d, want %d (full: %v)", id, ranks[id], w, ranks)
		}
	}
}

// Config versions are NOT dense — a version can exist without ever being scored. PriorRanks
// must find the highest SCORED config below the cursor, not simply decrement, or it lands on
// an empty config and reports every player as unchanged.
func TestPriorRanksSkipsUnscoredConfigs(t *testing.T) {
	s := newStore(t)
	write(t, s, 1, map[string]float64{"0001": 10, "0002": 90})
	// configs 2..6 were never scored; the live board is 7.
	write(t, s, 7, map[string]float64{"0001": 90, "0002": 10})

	ranks, ok, err := s.Reader().PriorRanks(context.Background(), testSeason, 7)
	if err != nil || !ok {
		t.Fatalf("PriorRanks: ok=%v err=%v", ok, err)
	}
	// Must have found config 1's board, where 0002 led.
	if ranks["0002"] != 1 || ranks["0001"] != 2 {
		t.Fatalf("expected config 1's board (0002 first), got %v", ranks)
	}
}

// The prior board is scoped to its season. A different season's rows must not leak in and
// produce deltas against a board the user never saw.
func TestPriorRanksIsSeasonScoped(t *testing.T) {
	s := newStore(t)
	if err := s.Writer().Write(context.Background(), testSeason+1, 1,
		[]ScoreRecord{{MFLID: "0001", Result: result(90, engine.CapTierNeutral, true)}}); err != nil {
		t.Fatalf("Write other season: %v", err)
	}

	_, ok, err := s.Reader().PriorRanks(context.Background(), testSeason, 2)
	if err != nil {
		t.Fatalf("PriorRanks: %v", err)
	}
	if ok {
		t.Fatal("a different season's board leaked into PriorRanks")
	}
}
