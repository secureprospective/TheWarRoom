package kicking

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestLive_KickingFetch exercises Fetch against the REAL nflverse
// player_stats_kicking_season file. Opt-in: set TWR_LIVE_NFLVERSE=1. Makes one
// outbound network call, so it stays out of the default suite/CI but compiles +
// lints with everything else.
//
//	TWR_LIVE_NFLVERSE=1 go test -run TestLive_KickingFetch -v ./internal/ingestion/kicking/...
func TestLive_KickingFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_NFLVERSE") != "1" {
		t.Skip("live kicking fetch: set TWR_LIVE_NFLVERSE=1 to run (makes a real network call)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	m, err := Fetch(ctx, &http.Client{Timeout: 60 * time.Second}, SeasonURL("2024"))
	if err != nil {
		t.Fatalf("Fetch against live source: %v", err)
	}

	// A full NFL regular season fields ~32 starting kickers plus a handful of
	// in-season replacements. A low count means a truncated download or schema
	// change — surface it here rather than downstream.
	if len(m) < 25 {
		t.Errorf("resolved only %d REG kicking records; expected >=25 for a full season", len(m))
	}

	for gsis, k := range m {
		if !strings.HasPrefix(gsis, "00-") {
			t.Errorf("key %q is not a gsis id", gsis)
		}
		// Distance buckets must sum to no more than total made/missed (RAW counts,
		// internally consistent). A kicker with 0 attempts would not be in REG data.
		if k.FGAtt < 0 || k.FGMade < 0 {
			t.Errorf("%s negative counts: %+v", gsis, k)
		}
		var madeSum int
		for _, n := range k.FGMadeByDistance {
			madeSum += n
		}
		if madeSum > k.FGMade {
			t.Errorf("%s distance-bucket makes (%d) exceed fg_made (%d)", gsis, madeSum, k.FGMade)
		}
	}
	t.Logf("resolved %d regular-season kicking records", len(m))
}
