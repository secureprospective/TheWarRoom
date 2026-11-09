package playerscores

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

func TestLive_PlayerScoresFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_MFL") != "1" {
		t.Skip("live MFL playerScores fetch: set TWR_LIVE_MFL=1 to run (makes real network calls)")
	}

	c, err := mfl.New("api", 2)
	if err != nil {
		t.Fatalf("mfl.New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	batch, err := Fetch(ctx, c, liveSeason(ctx, t, c), ingestion.LeagueID, 2025)
	if err != nil {
		t.Fatalf("Fetch against live MFL: %v", err)
	}
	if len(batch.Facts) < 500 || len(batch.BodySHA256) != 64 {
		t.Fatalf("got %d facts (sha %q), want a populated completed-season export (recon saw 1650)",
			len(batch.Facts), batch.BodySHA256)
	}

	var top float64
	for _, r := range batch.Facts {
		if r.Season != 2025 || r.Week != 0 || r.Field != Field {
			t.Fatalf("fact %+v, want season 2025, week 0, field %s", r, Field)
		}
		v, perr := strconv.ParseFloat(r.Raw, 64)
		if perr != nil {
			t.Fatalf("record %s has non-numeric score %q despite Validate", r.ID, r.Raw)
		}
		if v > top {
			top = v
		}
	}
	if top < 100 || top > 1500 {
		t.Fatalf("top YTD score %v outside plausible fantasy band [100,1500]", top)
	}
	t.Logf("live playerScores: %d facts, top YTD %v", len(batch.Facts), top)
}

func liveSeason(ctx context.Context, t *testing.T, c *mfl.Client) string {
	t.Helper()
	cfg, err := league.Discover(ctx, c, ingestion.LeagueID, time.Now().Year())
	if err != nil {
		t.Fatalf("discover season: %v", err)
	}
	return strconv.Itoa(cfg.CurrentSeason)
}
