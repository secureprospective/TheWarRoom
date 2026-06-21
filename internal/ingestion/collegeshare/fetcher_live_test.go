package collegeshare

import (
	"context"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
)

// TestLive_CollegeShareFetch exercises Fetch against the REAL CFBD season-stats
// endpoint, wiring the REAL espn->gsis bridge from the live crosswalk. It proves the
// shared CFBD client, the long-format team-sum, the rookie-keying bridge end to end,
// and the hard invariant that no within-team share exceeds 1.0. Opt-in: set
// TWR_LIVE_CFBD=1 (and CFBD_API_KEY); it also fetches the live crosswalk.
//
//	TWR_LIVE_CFBD=1 go test -run TestLive_CollegeShareFetch -v ./internal/ingestion/collegeshare/...
func TestLive_CollegeShareFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_CFBD") != "1" {
		t.Skip("live collegeshare fetch: set TWR_LIVE_CFBD=1 (and CFBD_API_KEY) to run")
	}
	key := strings.TrimSpace(os.Getenv("CFBD_API_KEY"))
	if key == "" {
		t.Fatal("CFBD_API_KEY is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Build the REAL espn->gsis bridge from the live crosswalk.
	cw, err := crosswalk.Fetch(ctx, &http.Client{Timeout: 60 * time.Second}, crosswalk.SourceURL)
	if err != nil {
		t.Fatalf("crosswalk.Fetch (for the resolver): %v", err)
	}
	t.Logf("crosswalk espn->gsis bridge: %d entries", cw.LenESPN())

	out, err := Fetch(ctx, ingestion.NewCFBDClient(120*time.Second), SeasonStatsURL, key, 2023, cw.GSISForESPN)
	if err != nil {
		t.Fatalf("Fetch against live CFBD: %v", err)
	}

	// Only 2023 college players who are in the NFL/dynastyprocess id ecosystem (recent
	// and upcoming draftees) resolve to a gsis — ~400 skill players. A count well below
	// that means a truncated decode or a broken bridge; an exact match is not expected.
	if len(out) < 300 {
		t.Errorf("resolved only %d gsis-keyed college records; expected >300", len(out))
	}

	// HARD INVARIANTS:
	//  - No player owns more than 100% of any team total (a share >1 would mean a
	//    team-sum bug such as cross-team contamination). Holds for all three shares.
	//  - ReceptionShare is a count ratio: structurally in [0,1].
	// Yardage shares may be slightly negative (real negative-yardage plays), so they
	// get a generous lower sanity floor rather than a hard >=0 — but a large negative
	// would signal a degenerate near-zero denominator and is worth catching.
	for gsis, r := range out {
		if r.ReceptionShare < 0 || r.ReceptionShare > 1.0000001 {
			t.Errorf("%s (%s) ReceptionShare = %v, out of [0,1]", r.Player, gsis, r.ReceptionShare)
		}
		for name, v := range map[string]float64{
			"ReceivingYardShare": r.ReceivingYardShare, "RushingYardShare": r.RushingYardShare,
		} {
			if v > 1.0000001 || v < -0.5 {
				t.Errorf("%s (%s) %s = %v, out of the sane [-0.5,1] band", r.Player, gsis, name, v)
			}
		}
	}

	// Eyeball the top reception-share players (a dominant college WR should top out
	// near his team's full target share).
	type kv struct {
		name  string
		share float64
	}
	top := make([]kv, 0, len(out))
	for _, r := range out {
		top = append(top, kv{r.Player, r.ReceptionShare})
	}
	sort.Slice(top, func(i, j int) bool { return top[i].share > top[j].share })
	for i := 0; i < 3 && i < len(top); i++ {
		t.Logf("top reception share #%d: %s = %.3f", i+1, top[i].name, top[i].share)
	}
	t.Logf("resolved %d gsis-keyed college-share records for 2023", len(out))
}
