package collegedefense

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

// TestLive_CollegeDefenseFetch exercises Fetch against the REAL CFBD season-stats
// endpoint (defensive + interceptions categories), wiring the REAL espn->gsis bridge
// from the live crosswalk. It proves the shared CFBD client, the two-call long-format
// team-sum, the rookie-keying bridge end to end, and the hard invariant that no
// within-team component share exceeds 1.0. Opt-in: set TWR_LIVE_CFBD=1 (and
// CFBD_API_KEY).
//
//	TWR_LIVE_CFBD=1 go test -run TestLive_CollegeDefenseFetch -v ./internal/ingestion/collegedefense/...
func TestLive_CollegeDefenseFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_CFBD") != "1" {
		t.Skip("live collegedefense fetch: set TWR_LIVE_CFBD=1 (and CFBD_API_KEY) to run")
	}
	key := strings.TrimSpace(os.Getenv("CFBD_API_KEY"))
	if key == "" {
		t.Fatal("CFBD_API_KEY is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cw, err := crosswalk.Fetch(ctx, &http.Client{Timeout: 60 * time.Second}, crosswalk.SourceURL)
	if err != nil {
		t.Fatalf("crosswalk.Fetch (for the resolver): %v", err)
	}
	t.Logf("crosswalk espn->gsis bridge: %d entries", cw.LenESPN())

	out, err := Fetch(ctx, ingestion.NewCFBDClient(120*time.Second), SeasonStatsURL, key, 2023, cw.GSISForESPN)
	if err != nil {
		t.Fatalf("Fetch against live CFBD: %v", err)
	}

	// Only 2023 college defenders in the NFL/dynastyprocess id ecosystem (recent and
	// upcoming draftees) resolve to a gsis. A count well below this means a truncated
	// decode or a broken bridge; an exact match is not expected.
	if len(out) < 200 {
		t.Errorf("resolved only %d gsis-keyed college-defense records; expected >200", len(out))
	}

	// HARD INVARIANT: a within-team component share in [0,1] (a count ratio cannot
	// exceed the team total; a share >1 means a team-sum bug such as cross-team
	// contamination). All five component shares are count ratios.
	for gsis, r := range out {
		for name, v := range map[string]float64{
			"TackleShare": r.TackleShare, "SackShare": r.SackShare, "TFLShare": r.TFLShare,
			"PassDefShare": r.PassDefShare, "InterceptionShare": r.InterceptionShare,
		} {
			if v < 0 || v > 1.0000001 {
				t.Errorf("%s (%s) %s = %v, out of [0,1]", r.Player, gsis, name, v)
			}
		}
	}

	// Eyeball the top tackle-share defenders (a college team's leading tackler should top
	// out at a meaningful fraction of team tackles).
	type kv struct {
		name  string
		share float64
	}
	top := make([]kv, 0, len(out))
	for _, r := range out {
		top = append(top, kv{r.Player, r.TackleShare})
	}
	sort.Slice(top, func(i, j int) bool { return top[i].share > top[j].share })
	for i := 0; i < 3 && i < len(top); i++ {
		t.Logf("top tackle share #%d: %s = %.3f", i+1, top[i].name, top[i].share)
	}
	t.Logf("resolved %d gsis-keyed college-defense records for 2023", len(out))
}
