package schooltier

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/scouting"
)

// TestLive_SchoolTierFetch exercises Fetch against the REAL CFBD /teams endpoint,
// proving the bearer auth, the HTTP/1.1 transport (h2 fails from CT105), the lenient
// decode, and the classification on live data. Opt-in: set TWR_LIVE_CFBD=1 and provide
// the key in CFBD_API_KEY.
//
//	TWR_LIVE_CFBD=1 go test -run TestLive_SchoolTierFetch -v ./internal/ingestion/schooltier/...
func TestLive_SchoolTierFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_CFBD") != "1" {
		t.Skip("live schooltier fetch: set TWR_LIVE_CFBD=1 (and CFBD_API_KEY) to run")
	}
	// Trim: the CT105 env's CFBD_API_KEY carries trailing newlines (curl tolerates
	// them, Go's strict header validation rejects them). Callers must pass a clean key.
	key := strings.TrimSpace(os.Getenv("CFBD_API_KEY"))
	if key == "" {
		t.Fatal("CFBD_API_KEY is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	out, err := Fetch(ctx, NewClient(60*time.Second), TeamsURL, key, 2024)
	if err != nil {
		t.Fatalf("Fetch against live CFBD: %v", err)
	}

	// CFBD lists ~600+ classifiable schools across all divisions; far fewer means a
	// truncated decode or a schema drift.
	if len(out) < 400 {
		t.Errorf("classified only %d schools; expected >400", len(out))
	}

	// Spot-check known schools across every tier, including the two locked edge cases.
	for school, want := range map[string]scouting.SchoolTier{
		"Georgia":      scouting.SchoolPowerFour,   // SEC
		"Alabama":      scouting.SchoolPowerFour,   // SEC
		"Notre Dame":   scouting.SchoolPowerFour,   // independent special-case
		"Oregon State": scouting.SchoolGroupOfFive, // Pac-12 in 2024 (post-cutoff)
	} {
		if got := out[school]; got != want {
			t.Errorf("%s: tier = %q, want %q", school, got, want)
		}
	}

	var p4, g5, fcs, nonFCS int
	for _, tier := range out {
		switch tier {
		case scouting.SchoolPowerFour:
			p4++
		case scouting.SchoolGroupOfFive:
			g5++
		case scouting.SchoolFCS:
			fcs++
		case scouting.SchoolNonFCS:
			nonFCS++
		case scouting.SchoolUnset:
		}
	}
	t.Logf("tiers: P4=%d G5=%d FCS=%d nonFCS=%d total=%d", p4, g5, fcs, nonFCS, len(out))
}
