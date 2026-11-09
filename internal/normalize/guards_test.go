package normalize

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

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

	rawPlayers, err := players.Fetch(ctx, c, liveSeason(ctx, t, c), ingestion.LeagueID)
	if err != nil {
		t.Fatalf("players.Fetch: %v", err)
	}
	if len(rawPlayers) == 0 {
		t.Fatal("players DB empty")
	}

	a := auditAggregateFilter(rawPlayers)

	t.Logf("players DB: %d records", len(rawPlayers))
	t.Logf("real position, id in [151,782]:   %d  (MUST be 0 — rosters would drop these)", len(a.realInRange))
	t.Logf("real position, id out of range:   %d", a.realOutOfRange)
	t.Logf("aggregate, id in [151,782]:       %d  (intended — rosters drops these)", a.aggInRange)
	t.Logf("aggregate, id out of range:       %d  (range too narrow if >0)", len(a.aggOutOfRange))
	t.Logf("unknown code (PosFlag) in range:  %d  (NewLookup flags for admin)", a.flagInRange)

	for _, rp := range a.realInRange {
		t.Errorf("AIRTIGHT VIOLATION: real player %s (%s, pos %q) has reserved-range id — rosters filter would silently drop it",
			rp.ID, rp.Name, rp.Position)
	}

	for _, rp := range a.aggOutOfRange {
		t.Logf("NOTE: aggregate %s (%q) sits OUTSIDE [151,782] — id-range filter alone would not drop it (join backstop covers it)", rp.ID, rp.Position)
	}
}

func liveSeason(ctx context.Context, t *testing.T, c *mfl.Client) string {
	t.Helper()
	cfg, err := league.Discover(ctx, c, ingestion.LeagueID, time.Now().Year())
	if err != nil {
		t.Fatalf("discover season: %v", err)
	}
	return strconv.Itoa(cfg.CurrentSeason)
}

type aggregateAudit struct {
	realInRange    []players.RawPlayer // FAILURE: real position, id in [151,782]
	realOutOfRange int                 // real position, id out of range — the norm
	aggInRange     int                 // aggregate, id in range — intended (rosters drops)
	aggOutOfRange  []players.RawPlayer // aggregate, id out of range — join backstop covers
	flagInRange    int                 // unknown code (→PosFlag) in range — admin-flagged
}

func auditAggregateFilter(raws []players.RawPlayer) aggregateAudit {
	posMap := newPositionMap()
	aggSet := newAggregateSet()

	var a aggregateAudit
	for _, rp := range raws {
		pos, isAgg := classifyPosition(rp.Position, posMap, aggSet)
		inRange := ingestion.IsTeamAggregateID(rp.ID)
		isReal := !isAgg && pos != domain.PosFlag

		switch {
		case isReal && inRange:
			a.realInRange = append(a.realInRange, rp)
		case isReal:
			a.realOutOfRange++
		case isAgg && inRange:
			a.aggInRange++
		case isAgg:
			a.aggOutOfRange = append(a.aggOutOfRange, rp)
		case inRange:
			a.flagInRange++
		}
	}
	return a
}
