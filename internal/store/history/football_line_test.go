package history

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

const linePlayHeader = "game_id,play_id,season,week,season_type,passer_player_id,receiver_player_id,rusher_player_id," +
	"passing_yards,receiving_yards,rushing_yards,touchdown,td_player_id,td_team,return_team,interception,punt_attempt," +
	"kickoff_attempt,play_deleted,two_point_attempt,posteam,play_type,rush_attempt,qb_kneel,lateral_rushing_yards," +
	"rush_touchdown,down,ydstogo,third_down_converted,fourth_down_converted,tackled_for_loss,sack,qb_hit_1_player_id,qb_hit_2_player_id,pass_attempt,complete_pass,yardline_100\n"

// lineImport is the scoring import with an RB and a WR carrying PFR ids, and a revision whose plays
// carry the line columns: PIT's 12-yard run, its 1-yard rushing TD, a kneel, a sack hit by two
// defenders; BAL's one pass. Advanced rushing is published for PIT_NYJ only.
func lineImport(t *testing.T) RawFootballEvidence {
	t.Helper()
	raw := scoringImport(t)
	for id, p := range map[string]struct {
		identity, name, pfr string
		position            leaguerules.Position
	}{"0101": {"00-0036223", "Running Back", "HarrNa00", leaguerules.RB}, "0102": {"00-0036900", "Receiver", "PickGe00", leaguerules.WR}} {
		player := raw.Players["0099"]
		player.Identity, player.Facts.Name, player.Facts.Position, player.PFR = p.identity, p.name, p.position, p.pfr
		raw.Players[id] = player
	}
	raw.Stats[0].Plays = linePlayHeader +
		"2025_01_PIT_NYJ,1,2026,1,REG,00-0023459,,,40,NA,NA,0,,,,0,0,0,0,0,PIT,pass,0,0,NA,0,1,10,0,0,0,0,NA,NA,1,1,NA\n" +
		"2025_01_PIT_NYJ,2,2026,1,REG,,,00-0036223,NA,NA,12,0,,,,0,0,0,0,0,PIT,run,1,0,NA,0,2,8,0,0,0,0,NA,NA,0,0,NA\n" +
		"2025_01_PIT_NYJ,3,2026,1,REG,,,00-0036223,NA,NA,1,1,00-0036223,PIT,,0,0,0,0,0,PIT,run,1,0,NA,1,1,1,0,0,0,0,NA,NA,0,0,NA\n" +
		"2025_01_PIT_NYJ,4,2026,1,REG,,,,NA,NA,-1,0,,,,0,0,0,0,0,PIT,qb_kneel,1,1,NA,0,1,10,0,0,1,0,NA,NA,0,0,NA\n" +
		"2025_01_PIT_NYJ,5,2026,1,REG,00-0023459,,,NA,NA,NA,0,,,,0,0,0,0,0,PIT,pass,0,0,NA,0,3,7,0,0,0,1,00-0031111,00-0032222,1,0,NA\n" +
		"2025_01_BAL_BUF,1,2026,1,REG,,,,NA,NA,NA,0,,,,0,0,0,0,0,BAL,pass,0,0,NA,0,1,10,0,0,0,0,NA,NA,1,0,NA\n"
	raw.Stats[0].Rushing = "game_id,season,week,game_type,team,pfr_player_id,rushing_yards_before_contact\n" +
		"2025_01_PIT_NYJ,2026,1,REG,PIT,HarrNa00,25\n" +
		"2025_01_PIT_NYJ,2026,1,REG,PIT,PickGe00,5\n" +
		"2025_01_PIT_NYJ,2026,1,REG,PIT,UnknXx00,7\n"
	return raw
}

func TestPinnedLineGamesCarryTheRBDesignation(t *testing.T) {
	s, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	pinned := installScoringImport(t, s, lineImport(t))
	held, ok, err := pinned.LineGames(t.Context(), 2026)
	if err != nil || !ok || held.SHA == "" {
		t.Fatalf("line games not served: %+v %v %v", held, ok, err)
	}
	pit := held.Games[source.LineGame{GameID: "2025_01_PIT_NYJ", Team: "PIT"}]
	for id, want := range map[leaguerules.StatID]leaguerules.Decimal{
		"line_rb_yards_before_contact": 250000, // the RB only: not the WR, not an unlisted PFR id
		"line_rushes_10_plus":          10000, "line_rushing_tds": 10000, "line_short_conversions": 0,
		"line_tackled_for_loss": 0, // the kneel is not a rush
		"line_sacks":            10000, "line_qb_hits": 20000,
	} {
		if got, held := pit.Quantities[id]; !held || got != want {
			t.Errorf("PIT %s: %d (held %v), want %d", id, got, held, want)
		}
	}
	bal := held.Games[source.LineGame{GameID: "2025_01_BAL_BUF", Team: "BAL"}]
	if _, published := bal.Quantities["line_rb_yards_before_contact"]; published || !bal.Played || len(held.Games) != 2 {
		t.Fatalf("a game with no advanced rushing must leave ybc absent: %+v (%d games)", bal, len(held.Games))
	}
	rules := leaguerules.Default32Scoring()
	if points, _, err := fantasyscore.ScoreUnit(rules, pit); err != nil || points.State != leaguerules.IsKnown {
		t.Fatalf("PIT line did not score: %+v %v", points, err)
	}
	if points, _, _ := fantasyscore.ScoreUnit(rules, bal); points.State != leaguerules.NeedsRuling {
		t.Fatalf("BAL scored without yards before contact: %+v", points)
	}
	held.Games[source.LineGame{GameID: "2025_01_PIT_NYJ", Team: "PIT"}].Quantities["line_sacks"] = 0
	again, _, _ := pinned.LineGames(t.Context(), 2026)
	if again.Games[source.LineGame{GameID: "2025_01_PIT_NYJ", Team: "PIT"}].Quantities["line_sacks"] != 10000 {
		t.Fatal("a reader changed the pinned line facts")
	}
	if _, _, err := pinned.LineGames(t.Context(), 2025); err == nil {
		t.Fatal("served line games for a season the evidence does not hold")
	}
}

// Evidence without advanced rushing is unchanged: same bytes (no new keys), same player weeks,
// and no line facts.
func TestPinnedEvidenceWithoutRushingHoldsNoLineFacts(t *testing.T) {
	raw := scoringImport(t)
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), `"rushing"`) || strings.Contains(string(body), `"pfr"`) {
		t.Fatal("evidence without line facts gained keys, so its digest would change")
	}
	// Compare line capability with equal edge capability, so recorded edge zeros agree.
	rows := strings.Split(strings.TrimSuffix(raw.Stats[0].Plays, "\n"), "\n")
	rows[0] += ",third_down_converted,fourth_down_converted,rush_attempt,pass_attempt,complete_pass,yardline_100,tackled_for_loss"
	for i := 1; i < len(rows); i++ {
		rows[i] += ",0,0,0,0,0,NA,0"
	}
	raw.Stats[0].Plays = strings.Join(rows, "\n") + "\n"
	s, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	pinned := installScoringImport(t, s, raw)
	if _, ok, err := pinned.LineGames(t.Context(), 2026); ok || err != nil {
		t.Fatalf("evidence without advanced rushing served line games: %v %v", ok, err)
	}
	withLines := installScoringImport(t, s, lineImport(t))
	scope := source.RawStatScope{GSIS: "00-0023459", Season: 2026, Week: 1}
	a, err := pinned.PlayerWeekStats(t.Context(), scope)
	if err != nil {
		t.Fatal(err)
	}
	b, err := withLines.PlayerWeekStats(t.Context(), scope)
	if err != nil || len(a) != 1 || len(b) != 1 || a[0].Week == nil || b[0].Week == nil ||
		source.LineDigest(*a[0].Week) != source.LineDigest(*b[0].Week) {
		t.Fatalf("the line columns changed the QB's player week: %+v %+v %v", a, b, err)
	}
}

// The newest revision holding line facts is served; a newer weekly-only revision does not retract it.
func TestPinnedLineGamesComeFromTheNewestRevisionHoldingThem(t *testing.T) {
	raw := lineImport(t)
	newer := raw.Stats[0]
	newer.AsOf, newer.Rushing = raw.AsOf.Add(-time.Minute), ""
	raw.Stats = []RawFootballStats{newer, raw.Stats[0]}
	s, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	held, ok, err := installScoringImport(t, s, raw).LineGames(t.Context(), 2026)
	if err != nil || !ok || !held.At.Equal(raw.Stats[1].AsOf) {
		t.Fatalf("served %v %v %v, want the older revision holding advanced rushing", held.At, ok, err)
	}
}

func TestPinnedLineFactsRejectInvalidImports(t *testing.T) {
	s, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	for name, change := range map[string]func(*RawFootballEvidence){
		"plays without line columns": func(r *RawFootballEvidence) { r.Stats[0].Plays = scoringImport(t).Stats[0].Plays },
		"advanced rushing for an unplayed game": func(r *RawFootballEvidence) {
			r.Stats[0].Rushing += "2025_01_PIT_NYJ,2026,1,REG,NYJ,HarrNa00,3\n"
		},
		"advanced rushing without yards before contact": func(r *RawFootballEvidence) {
			r.Stats[0].Rushing = "game_id,team,pfr_player_id\n2025_01_PIT_NYJ,PIT,HarrNa00\n"
		},
		"a PFR id on two players": func(r *RawFootballEvidence) {
			p := r.Players["0102"]
			p.PFR = "HarrNa00"
			r.Players["0102"] = p
		},
		"an untrimmed PFR id": func(r *RawFootballEvidence) { p := r.Players["0102"]; p.PFR = " PickGe00"; r.Players["0102"] = p },
	} {
		bad := lineImport(t)
		change(&bad)
		body, err := json.Marshal(bad)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.ImportFootball(t.Context(), body, footballDigest(body)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
