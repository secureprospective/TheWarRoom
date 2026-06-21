package veteranfilm

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestLive_VeteranFilmFetch exercises Fetch against the REAL nflverse FTN charting +
// play-by-play for one season, proving the FTN->pbp join, the streaming pbp read (the
// file is ~95 MiB; it must stream, not buffer), the per-player rate aggregation, and
// the floor suppression end to end. Opt-in: set TWR_LIVE_NFLVERSE=1. Streams a large
// file, so it stays out of the default suite/CI but compiles + lints with everything.
//
//	TWR_LIVE_NFLVERSE=1 go test -run TestLive_VeteranFilmFetch -v ./internal/ingestion/veteranfilm/...
func TestLive_VeteranFilmFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_NFLVERSE") != "1" {
		t.Skip("live veteranfilm fetch: set TWR_LIVE_NFLVERSE=1 to run (streams a ~95 MiB file)")
	}

	// CT105's link to GitHub releases is ~310 KB/s, so the ~95 MiB pbp body alone
	// takes >5 min to transfer (the stream itself is cheap; the network is the cost).
	// The timeout must cover the whole slow body read, not just connect.
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	client := &http.Client{Timeout: 12 * time.Minute}

	out, err := Fetch(ctx, client, SeasonSources(2025), DefaultReceiverFloor, DefaultPasserFloor)
	if err != nil {
		t.Fatalf("Fetch against live nflverse 2025: %v", err)
	}

	var recvN, passN, topTargets, topAtt int
	var topWR, topQB string
	for gsis, rec := range out {
		if r := rec.Receiver; r != nil {
			recvN++
			assertRate(t, gsis, "contested", r.ContestedCatchRate)
			assertRate(t, gsis, "created", r.CreatedReceptionRate)
			assertRate(t, gsis, "drop", r.DropRate)
			if r.Targets > topTargets {
				topTargets, topWR = r.Targets, gsis
			}
		}
		if p := rec.Passer; p != nil {
			passN++
			assertRate(t, gsis, "int-worthy", p.InterceptionWorthyRate)
			assertRate(t, gsis, "throwaway", p.ThrowawayRate)
			if p.Attempts > topAtt {
				topAtt, topQB = p.Attempts, gsis
			}
		}
	}

	// A full season clears ~80+ receivers over a 30-target floor and ~25+ passers over
	// a 100-attempt floor; far fewer means a truncated stream or a join/key drift.
	if recvN < 50 {
		t.Errorf("only %d receivers above the floor; expected >50 (possible join/stream drift)", recvN)
	}
	if passN < 15 {
		t.Errorf("only %d passers above the floor; expected >15", passN)
	}
	t.Logf("resolved %d receivers / %d passers above floor", recvN, passN)
	t.Logf("top WR %s @ %d charted targets; top QB %s @ %d charted attempts", topWR, topTargets, topQB, topAtt)
}

// assertRate fails if a rate escapes [0,1] — a rate outside the unit interval means a
// numerator/denominator bug (the spot-check the close gate calls for, applied to every
// emitted player rather than one hand-picked name).
func assertRate(t *testing.T, gsis, name string, v float64) {
	t.Helper()
	if v < 0 || v > 1 {
		t.Errorf("%s %s rate = %v, outside [0,1]", gsis, name, v)
	}
}
