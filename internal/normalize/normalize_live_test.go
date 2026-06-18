package normalize

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/rosters"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// TestLive_NormalizePipeline is the B3 functional-verification gate: the full
// Layer-1 pipeline against REAL MFL data — fetch players, build the cross-reference
// Lookup (which enforces the reserved-range invariant on the entire live player
// database, B2 review item #1), fetch rosters, and normalize them into typed
// domain.Roster records. Opt-in: set TWR_LIVE_MFL=1. Makes outbound network calls;
// stays out of the default suite/CI but compiles and lints with everything else.
//
//	TWR_LIVE_MFL=1 go test -run TestLive_NormalizePipeline -v ./internal/normalize/...
func TestLive_NormalizePipeline(t *testing.T) {
	if os.Getenv("TWR_LIVE_MFL") != "1" {
		t.Skip("live normalize pipeline: set TWR_LIVE_MFL=1 to run (makes real network calls)")
	}

	c, err := mfl.New("api", 2)
	if err != nil {
		t.Fatalf("mfl.New: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	// 1. Players DB → Lookup. NewLookup runs the reserved-range invariant against
	//    every live record; if any real player sits in [151,782] it fails here.
	rawPlayers, err := players.Fetch(ctx, c, ingestion.SeasonYear, ingestion.LeagueID)
	if err != nil {
		t.Fatalf("players.Fetch: %v", err)
	}
	lookup, err := NewLookup(rawPlayers)
	if err != nil {
		t.Fatalf("NewLookup (reserved-range invariant) against live players DB: %v", err)
	}
	t.Logf("players DB: %d records, lookup built clean", len(rawPlayers))

	// 2. Rosters → normalize to typed domain records.
	rawRosters, err := rosters.Fetch(ctx, c, ingestion.SeasonYear, ingestion.LeagueID)
	if err != nil {
		t.Fatalf("rosters.Fetch: %v", err)
	}
	domRosters, err := Rosters(rawRosters, lookup)
	if err != nil {
		t.Fatalf("normalize.Rosters against live data: %v", err)
	}

	const wantFranchises = 32
	if len(domRosters) != wantFranchises {
		t.Errorf("got %d normalized franchises, want %d", len(domRosters), wantFranchises)
	}

	// 3. Walk every typed record and assert the B3 invariants hold on live data.
	var total, flagged, rookies int
	posCounts := make(map[domain.Position]int)
	for _, r := range domRosters {
		var prevID string
		for _, p := range r.Players {
			total++
			posCounts[p.Position]++
			if p.NeedsAdminReview() {
				flagged++
			}
			if p.IsRookie {
				rookies++
			}
			if p.MFLID.IsZero() {
				t.Errorf("franchise %s: zero player id leaked through normalize", r.FranchiseID)
			}
			if p.Salary < 0 {
				t.Errorf("franchise %s player %s: negative salary %.2f", r.FranchiseID, p.MFLID, p.Salary)
			}
			if p.RosterStatus != domain.RosterActive && p.RosterStatus != domain.RosterTaxi {
				t.Errorf("franchise %s player %s: bad roster status %q", r.FranchiseID, p.MFLID, p.RosterStatus)
			}
			// Intra-roster determinism: ids are non-decreasing within a franchise.
			if id := p.MFLID.String(); id < prevID {
				t.Errorf("franchise %s: players not in canonical id order (%s after %s)", r.FranchiseID, id, prevID)
			} else {
				prevID = id
			}
		}
	}

	t.Logf("normalized %d roster records across %d franchises", total, len(domRosters))
	t.Logf("position distribution: %v", posCounts)
	t.Logf("flagged for admin review: %d · rookies: %d", flagged, rookies)
	if total == 0 {
		t.Fatal("normalized zero roster records")
	}
}
