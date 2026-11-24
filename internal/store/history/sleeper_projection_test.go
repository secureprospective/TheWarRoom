package history

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

type sleeperFacts struct{ facts nativecheck.Facts }

func (f sleeperFacts) PlayerFacts() nativecheck.Facts { return f.facts }
func (sleeperFacts) LineupFacts(_ context.Context, _, _ int) (commissioner.LineupFacts, error) {
	return commissioner.LineupFacts{}, nil
}

func TestSleeperProjectionNoSnapshot(t *testing.T) {
	s, _, _ := newStore(t)
	got, stats, err := s.SleeperProjectedPoints(t.Context(), "native.missing", 2026, 12)
	if err != nil || len(got) != 0 || stats.Note != "no forecast held for week 12" {
		t.Fatalf("no snapshot: points=%v stats=%+v err=%v", got, stats, err)
	}
}

func TestSleeperProjectionUsesLeagueRulesAndCanonicalPosition(t *testing.T) {
	s, _, pools := newStore(t)
	for _, league := range []string{"native.first", "native.second"} {
		raw := createWith(commissioner.PresetLegacy, 0)
		raw.League = league
		if _, err := s.ExecuteCommissioner(t.Context(), raw, "creator", nil); err != nil {
			t.Fatal(err)
		}
	}
	identityTx, err := pools.ReferencePools().Write().BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"101", "102"} {
		if _, err := db.NewPlayerIdentity(t.Context(), identityTx, []playerid.RawOutsideID{{Provider: "mfl", Value: "20" + id}, {Provider: "sleeper", Value: id}}); err != nil {
			t.Fatal(err)
		}
	}
	if err := identityTx.Commit(); err != nil {
		t.Fatal(err)
	}
	// Rounding changes whether this sub-cent projection is rounded at each stat.
	records, err := s.nativeRecords(t.Context(), "native.second")
	if err != nil {
		t.Fatal(err)
	}
	change := commissioner.RawCommand{League: "native.second", Command: commissioner.RulesSet,
		Args: commissioner.RawArgs{Field: "scoring.roundStatPoints", Value: json.RawMessage(`false`)}, Reason: "projection test", Expected: int64(len(records)),
		Request: "sleeper.rules", Effective: "now", Visibility: "league"}
	if _, err := s.ExecuteCommissioner(t.Context(), change, "creator", nil); err != nil {
		t.Fatalf("set league scoring: %v", err)
	}
	identityIDs, err := s.SourceIDs(t.Context(), "sleeper")
	if err != nil {
		t.Fatal(err)
	}
	bySleeper := map[string]string{}
	for player, id := range identityIDs {
		bySleeper[id] = player
	}
	s.SetLineupEnvironment(sleeperFacts{facts: nativecheck.Facts{
		bySleeper["101"]: {Position: leaguerules.QB}, bySleeper["102"]: {Position: leaguerules.LB},
	}})
	url := "https://api.sleeper.app/projections/nfl/2026/12?season_type=regular"
	body := `[{"player_id":"101","team":"BUF","week":12,"sport":"nfl","season_type":"regular","stats":{"pass_yd":0.5},"player":{"position":"QB"}},{"player_id":"102","team":"BUF","week":12,"sport":"nfl","season_type":"regular","stats":{"idp_tkl_solo":1},"player":{"position":"DB"}}]`
	if err := s.Record(t.Context(), archiveFetch(url, body, time.Now())); err != nil {
		t.Fatal(err)
	}
	first, firstStats, err := s.SleeperProjectedPoints(t.Context(), "native.first", 2026, 12)
	if err != nil {
		t.Fatal(err)
	}
	second, secondStats, err := s.SleeperProjectedPoints(t.Context(), "native.second", 2026, 12)
	if err != nil {
		t.Fatal(err)
	}
	qb, dbPlayer := bySleeper["101"], bySleeper["102"]
	if first[qb] != 300 || second[qb] != 250 || first[dbPlayer] <= 0 || firstStats.Scored != 2 || secondStats.Scored != 2 {
		t.Fatalf("league scoring/DB position: first=%v (%+v), second=%v (%+v)", first, firstStats, second, secondStats)
	}
}

func TestSleeperProjectionOneUnscorableRowKeepsTheOthers(t *testing.T) {
	line := fantasyscore.StatLine{}
	rows := []source.SleeperProjection{
		{CanonicalID: "bad", Position: "XX", Line: line},
		{CanonicalID: "qb", Position: leaguerules.QB, Line: line},
	}
	points, stats, err := scoreSleeperRows(rows, nil, leaguerules.ScoringRules{})
	if err != nil {
		t.Fatalf("one unscorable row lost the week: %v", err)
	}
	if _, ok := points["qb"]; !ok && stats.Scored+stats.Unmapped != 2 {
		t.Fatalf("points=%v stats=%+v, want both rows accounted for", points, stats)
	}
	if stats.Unmapped < 1 {
		t.Fatalf("stats=%+v, want the bad row counted", stats)
	}
}
