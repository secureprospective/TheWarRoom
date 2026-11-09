package players

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

func TestLive_PlayersFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_MFL") != "1" {
		t.Skip("live MFL players fetch: set TWR_LIVE_MFL=1 to run (makes real network calls)")
	}

	c, err := mfl.New("api", 2)
	if err != nil {
		t.Fatalf("mfl.New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	all, err := Fetch(ctx, c, liveSeason(ctx, t, c), ingestion.LeagueID)
	if err != nil {
		t.Fatalf("Fetch against live MFL: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("Fetch returned 0 players")
	}

	if len(all) < 1000 {
		t.Errorf("fetched only %d players; expected >1000 for the full database", len(all))
	}

	// DETAILS=1 gate (M1): the MAJORITY of the database must carry a birthdate.
	// Recon 2026-07-02 saw 2021 of 2630 (~77%) — commissioner-created players and
	// deep-database rows legitimately lack it, so we assert >50%, not all. A drop
	// to near-zero means the DETAILS param regressed and M1's age input is gone.
	var withBirthdate int
	for _, p := range all {
		if p.Birthdate != "" {
			withBirthdate++
		}
	}
	if withBirthdate*2 < len(all) {
		t.Errorf("only %d of %d records carry a birthdate; DETAILS=1 appears broken", withBirthdate, len(all))
	}
	t.Logf("fetched %d player records (%d with birthdate)", len(all), withBirthdate)
}

func liveSeason(ctx context.Context, t *testing.T, c *mfl.Client) string {
	t.Helper()
	cfg, err := league.Discover(ctx, c, ingestion.LeagueID, time.Now().Year())
	if err != nil {
		t.Fatalf("discover season: %v", err)
	}
	return strconv.Itoa(cfg.CurrentSeason)
}
