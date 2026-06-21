package crosswalk

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestLive_CrosswalkFetch exercises Fetch against the REAL dynastyprocess
// db_playerids.csv. Opt-in: set TWR_LIVE_NFLVERSE=1. It makes an outbound network
// call, so it stays out of the default suite/CI/pre-commit but still compiles and
// is linted with everything else. Mirrors the MFL live tests; proves the
// external-HTTP-CSV path and by-name column binding on the real, full file.
//
//	TWR_LIVE_NFLVERSE=1 go test -run TestLive_CrosswalkFetch -v ./internal/ingestion/crosswalk/...
func TestLive_CrosswalkFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_NFLVERSE") != "1" {
		t.Skip("live crosswalk fetch: set TWR_LIVE_NFLVERSE=1 to run (makes a real network call)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	m, err := Fetch(ctx, &http.Client{Timeout: 60 * time.Second}, SourceURL)
	if err != nil {
		t.Fatalf("Fetch against live source: %v", err)
	}

	// The real file lists tens of thousands of players across history; a healthy
	// crosswalk resolves several thousand gsis<->MFL pairs. A low count means a
	// truncated download or a silently changed schema — surface it here, not at
	// the downstream join.
	if m.Len() < 5000 {
		t.Errorf("resolved only %d gsis->MFL entries; expected >5000 from the full source", m.Len())
	}

	// The espn->gsis bridge (collegeshare's rookie keying path) must also resolve at
	// scale — the source carries ~8000 rows with both espn_id and gsis_id. A near-zero
	// count means the espn_id column silently vanished; catch it here, not when
	// collegeshare's CFBD join quietly matches nothing.
	if m.LenESPN() < 5000 {
		t.Errorf("resolved only %d espn->gsis bridge entries; expected >5000 from the full source", m.LenESPN())
	}

	// Spot-check the verified bridge: Caleb Williams' CFBD/espn id resolves to his gsis.
	if gsis, ok := m.GSISForESPN("4431611"); !ok || gsis != "00-0039918" {
		t.Errorf("GSISForESPN(4431611) = %q,%v; want 00-0039918,true (Caleb Williams)", gsis, ok)
	}

	t.Logf("resolved %d gsis->MFL and %d espn->gsis crosswalk entries", m.Len(), m.LenESPN())
}
