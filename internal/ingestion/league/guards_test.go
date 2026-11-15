package league

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

func TestLive_LeagueFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_MFL") != "1" {
		t.Skip("live MFL league fetch: set TWR_LIVE_MFL=1 to run (makes real network calls)")
	}

	c, err := mfl.New("api", 2)
	if err != nil {
		t.Fatalf("mfl.New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	cfg, err := Fetch(ctx, c, liveSeason(ctx, t, c), ingestion.LeagueID)
	if err != nil {
		t.Fatalf("Fetch against live MFL: %v", err)
	}

	if cfg.SalaryCapAmount == "" {
		t.Error("live cap amount empty")
	}
	if len(cfg.ScoringRules) == 0 {
		t.Fatal("live config has zero scoring rule blocks")
	}

	// The decoded scoring config must contain the missed-FG rule (MG, OQ-002) — a
	// concrete proof the rules export decoded, not just the league export.
	var sawMG bool
	for _, set := range cfg.ScoringRules {
		for _, r := range set.Rules {
			if strings.TrimSpace(r.Event) == "MG" {
				sawMG = true
			}
		}
	}
	if !sawMG {
		t.Error("missed-FG (MG) rule not found in live scoring config")
	}
	t.Logf("live rulebook: cap=%s, rosterSize=%s, %d scoring blocks",
		cfg.SalaryCapAmount, cfg.RosterSize, len(cfg.ScoringRules))
}

func liveSeason(ctx context.Context, t *testing.T, c *mfl.Client) string {
	t.Helper()
	cfg, err := Discover(ctx, c, ingestion.LeagueID, time.Now().Year())
	if err != nil {
		t.Fatalf("discover season: %v", err)
	}
	return strconv.Itoa(cfg.CurrentSeason)
}

func TestAssemble_RealLeagueCarriesTheRaceStructure(t *testing.T) {
	leagueRaw, err := os.ReadFile("testdata/league-2025.json")
	if err != nil {
		t.Fatal(err)
	}
	rulesRaw, err := os.ReadFile("testdata/rules-2025.json")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := Parse(leagueRaw, rulesRaw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.StandingsSort != "PCT,H2H,DIVPCT,CONFPCT,ALL_PLAY_PCT,PTS,PWR," {
		t.Fatalf("standings sort = %q", cfg.StandingsSort)
	}
	if len(cfg.Conferences) != 2 || cfg.Conferences[1] != (Conference{ID: "01", Name: "NFC"}) {
		t.Fatalf("conferences = %+v", cfg.Conferences)
	}
	if len(cfg.Divisions) != 8 || cfg.Divisions[3] != (Division{ID: "03", Name: "AFC North", Conference: "00"}) {
		t.Fatalf("divisions = %+v", cfg.Divisions)
	}
	if cfg.Franchises[0] != (Franchise{ID: "0001", Name: "Buffalo Bills", Abbrev: "BUF", Division: "00"}) {
		t.Fatalf("franchise = %+v", cfg.Franchises[0])
	}
}
