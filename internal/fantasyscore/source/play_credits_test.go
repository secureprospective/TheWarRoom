package source

import (
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func creditPlay() RawPlay {
	c := make(RawPlayerCredits)
	for _, column := range creditColumns() {
		c[column] = ""
	}
	return RawPlay{GameID: "2026_01_BUF_NYJ", PlayID: "1", Season: "2026", Week: "1", SeasonType: "REG", Credits: c}
}

func TestMissingPlayWeeksDefensiveCredits(t *testing.T) {
	for _, c := range defensiveCredits() {
		t.Run(c.column, func(t *testing.T) {
			p := creditPlay()
			p.Credits[c.column] = "00-0030001"
			got, err := MissingPlayWeeks(nil, []RawPlay{p})
			if err != nil || len(got) != 1 {
				t.Fatalf("%+v %v", got, err)
			}
			if got[0].Line.Quantities[c.stat] != c.amount || got[0].Review != "nflverse missing row" {
				t.Fatalf("%+v", got[0])
			}
			held := SelectHeldWeek(RawStatScope{GSIS: "00-0030001", Season: 2026, Week: 1}, got, "digest", time.Now())
			if held.Authority != "nflverse play-by-play" || held.Review == "" {
				t.Fatalf("%+v", held)
			}
		})
	}
}

func TestMissingPlayWeeksSoloTackleScores(t *testing.T) {
	p := creditPlay()
	p.Credits["solo_tackle_1_player_id"] = "00-0030001"
	got, err := MissingPlayWeeks(nil, []RawPlay{p})
	if err != nil {
		t.Fatal(err)
	}
	points, _, err := fantasyscore.Score(leaguerules.Default32Scoring(), leaguerules.LB, got[0].Line)
	if err != nil || points.State != leaguerules.IsKnown || points.Value != leaguerules.Default32Scoring().Catalogue.Value["tackle"].Points.Value {
		t.Fatalf("%+v %v", points, err)
	}
}

func TestMissingPlayWeeksDoesNotInventOrOverwrite(t *testing.T) {
	p := creditPlay()
	p.Credits["solo_tackle_1_player_id"] = "00-0030001"
	official := PlayerWeek{GSIS: "00-0030001", Season: 2026, Week: 1, SeasonType: "REG", GameID: p.GameID,
		Line: fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"tackle": 30000}, Events: map[leaguerules.StatID][]leaguerules.Decimal{}}}
	got, err := MissingPlayWeeks([]PlayerWeek{official}, []RawPlay{p})
	if err != nil || len(got) != 1 || !reflect.DeepEqual(got[0], official) {
		t.Fatalf("%+v %v", got, err)
	}
	for _, mutate := range []func(*RawPlay){
		func(p *RawPlay) { p.Credits["solo_tackle_1_player_id"] = "" },
		func(p *RawPlay) { delete(p.Credits, "sack_player_id") },
		func(p *RawPlay) { p.Deleted = "1" },
		func(p *RawPlay) { p.PlayType = "no_play" },
		func(p *RawPlay) { p.TwoPoint = "1" },
	} {
		p = creditPlay()
		p.Credits["solo_tackle_1_player_id"] = "00-0030001"
		mutate(&p)
		got, err = MissingPlayWeeks(nil, []RawPlay{p})
		if err != nil || len(got) != 0 {
			t.Fatalf("absence must stay absent: %+v %v", got, err)
		}
	}
	p = creditPlay()
	p.Credits["sack_player_id"] = "bad"
	if _, err = MissingPlayWeeks([]PlayerWeek{official}, []RawPlay{p}); err == nil {
		t.Fatal("invalid identity accepted")
	}
	if official.Line.Quantities["tackle"] != 30000 {
		t.Fatal("input mutated on failure")
	}
}

func TestMissingPlayWeeksReturnsAndFumbles(t *testing.T) {
	p := creditPlay()
	p.Punt = "1"
	p.FumbleLost = "1"
	p.Credits["punt_returner_player_id"] = "00-0030001"
	p.Credits["return_yards"] = "17"
	p.Credits["fumbled_1_player_id"] = "00-0030002"
	p.Credits["fumbled_1_team"] = "NYJ"
	p.Credits["fumble_recovery_1_player_id"] = "00-0030001"
	p.Credits["fumble_recovery_1_team"] = "BUF"
	p.Credits["fumble_recovery_1_yards"] = "4"
	got, err := MissingPlayWeeks(nil, []RawPlay{p})
	if err != nil || len(got) != 2 {
		t.Fatalf("%+v %v", got, err)
	}
	if got[0].Line.Quantities["punt_return_yards"] != 170000 || got[0].Line.Quantities["opponent_fumble_recovery"] != 10000 || got[1].Line.Quantities["fumble_lost"] != 10000 {
		t.Fatalf("%+v", got)
	}
}

func TestMissingPlayWeeksKeepsZeroYardReturn(t *testing.T) {
	p := creditPlay()
	p.Punt = "1"
	p.Credits["punt_returner_player_id"] = "00-0030001"
	p.Credits["return_yards"] = "0"
	got, err := MissingPlayWeeks(nil, []RawPlay{p})
	if err != nil || len(got) != 1 {
		t.Fatalf("recorded zero return lost: %+v %v", got, err)
	}
	yards, ok := got[0].Line.Quantities["punt_return_yards"]
	if !ok || yards != 0 {
		t.Fatalf("%+v", got[0])
	}
}
