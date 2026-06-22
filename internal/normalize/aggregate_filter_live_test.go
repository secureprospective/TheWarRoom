package normalize

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// TestLive_AggregateFilterAirtight retires the loose end carried B2→B3: prove
// against LIVE position data that the rosters fetcher's ID-range aggregate filter
// is airtight. The rosters endpoint carries NO position field, so rosters drops
// team-aggregate records by ID RANGE alone — ids in [151,782] (ingestion.
// IsTeamAggregateID). That filter is only safe if the range and the real set of
// aggregate POSITION codes coincide exactly on real data. The dangerous failure
// mode is a REAL playable-position player (QB…S) whose MFL id happens to fall in
// [151,782]: rosters would silently drop a real rostered player and no error would
// ever fire. This test asserts that count is ZERO on the full live players DB, and
// prints the full id-range × position-class cross-tab so the proof is visible, not
// asserted in the dark.
//
// Opt-in: set TWR_LIVE_MFL=1. Makes one outbound players call (MFL caps this
// endpoint at once/day — do not loop). Compiles and lints with the default suite;
// only the network body is gated.
//
//	TWR_LIVE_MFL=1 go test -run TestLive_AggregateFilterAirtight -v ./internal/normalize/...
func TestLive_AggregateFilterAirtight(t *testing.T) {
	if os.Getenv("TWR_LIVE_MFL") != "1" {
		t.Skip("live aggregate-filter check: set TWR_LIVE_MFL=1 to run (makes real network calls)")
	}

	c, err := mfl.New("api", 2)
	if err != nil {
		t.Fatalf("mfl.New: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	rawPlayers, err := players.Fetch(ctx, c, ingestion.SeasonYear, ingestion.LeagueID)
	if err != nil {
		t.Fatalf("players.Fetch: %v", err)
	}
	if len(rawPlayers) == 0 {
		t.Fatal("players DB empty")
	}

	// Cross-tab every live record's position class against the ID-range filter
	// rosters relies on. auditAggregateFilter is the pure core (offline-tested in
	// aggregate_filter_test.go, where a planted real-player-in-range is seen to
	// fail it — gate philosophy); here we feed it live data and assert.
	a := auditAggregateFilter(rawPlayers)

	t.Logf("players DB: %d records", len(rawPlayers))
	t.Logf("real position, id in [151,782]:   %d  (MUST be 0 — rosters would drop these)", len(a.realInRange))
	t.Logf("real position, id out of range:   %d", a.realOutOfRange)
	t.Logf("aggregate, id in [151,782]:       %d  (intended — rosters drops these)", a.aggInRange)
	t.Logf("aggregate, id out of range:       %d  (range too narrow if >0)", len(a.aggOutOfRange))
	t.Logf("unknown code (PosFlag) in range:  %d  (NewLookup flags for admin)", a.flagInRange)

	// HARD ASSERTION — the airtight property. A real player inside the reserved
	// range is a silent-data-loss bug in the rosters ID-range filter.
	for _, rp := range a.realInRange {
		t.Errorf("AIRTIGHT VIOLATION: real player %s (%s, pos %q) has reserved-range id — rosters filter would silently drop it",
			rp.ID, rp.Name, rp.Position)
	}

	// SOFT SIGNAL — an aggregate code sitting OUTSIDE [151,782] means the range
	// bound is too narrow; rosters would not drop it by ID. The normalize join's
	// isAggregate check (roster.go) is the backstop, so this is a flag, not a
	// failure — surface it so the range can be widened if MFL ever does this.
	for _, rp := range a.aggOutOfRange {
		t.Logf("NOTE: aggregate %s (%q) sits OUTSIDE [151,782] — id-range filter alone would not drop it (join backstop covers it)", rp.ID, rp.Position)
	}
}
