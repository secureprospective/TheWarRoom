package source

import (
	"fmt"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestEdgeCredits(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*RawPlay)
		want   map[leaguerules.StatID]leaguerules.Decimal
	}{
		{"third rush touchdown", func(p *RawPlay) { p.ThirdConversion = "1"; p.RushAttempt = "1"; p.Touchdown = "1"; p.Yardline = "20" }, map[leaguerules.StatID]leaguerules.Decimal{"third_down_conversion": 10000, "red_zone_carry": 10000}},
		{"fourth rush loss", func(p *RawPlay) { p.FourthConversion = "1"; p.RushAttempt = "1"; p.TackledForLoss = "1" }, map[leaguerules.StatID]leaguerules.Decimal{"fourth_down_conversion": 10000, "rushing_tfl": 10000}},
		{"third completion", func(p *RawPlay) {
			p.ThirdConversion = "1"
			p.CompletePass = "1"
			p.PassAttempt = "1"
			p.Yardline = "20"
		}, map[leaguerules.StatID]leaguerules.Decimal{"third_down_conversion": 10000, "third_down_passing_conversion": 10000, "red_zone_target": 10000, "red_zone_pass_attempt": 10000}},
		{"fourth completion", func(p *RawPlay) { p.FourthConversion = "1"; p.CompletePass = "1" }, map[leaguerules.StatID]leaguerules.Decimal{"fourth_down_conversion": 10000, "fourth_down_passing_conversion": 10000}},
		{"incomplete red zone", func(p *RawPlay) { p.PassAttempt = "1"; p.Yardline = "20" }, map[leaguerules.StatID]leaguerules.Decimal{"red_zone_target": 10000, "red_zone_pass_attempt": 10000}},
		{"red-zone sack is not an attempt or target", func(p *RawPlay) { p.PassAttempt = "1"; p.Sack = "1"; p.Yardline = "20" }, nil},
		{"play nullified by penalty credits nothing", func(p *RawPlay) { p.RushAttempt = "1"; p.ThirdConversion = "1"; p.PlayType = "no_play" }, nil},
		{"kneel is a carry, never a tackle for loss", func(p *RawPlay) { p.RushAttempt = "1"; p.PlayType = "qb_kneel"; p.Rushing = "-1"; p.Yardline = "20" }, map[leaguerules.StatID]leaguerules.Decimal{"red_zone_carry": 10000}},
		{"aborted snap is a carry, never a tackle for loss", func(p *RawPlay) { p.RushAttempt = "1"; p.Rushing = "0"; p.Yardline = "20" }, map[leaguerules.StatID]leaguerules.Decimal{"red_zone_carry": 10000}},
		{"scramble is a carry", func(p *RawPlay) { p.RushAttempt = "1"; p.Yardline = "20" }, map[leaguerules.StatID]leaguerules.Decimal{"red_zone_carry": 10000}},
		{"spike is an attempt with no receiver to target", func(p *RawPlay) { p.PassAttempt = "1"; p.PlayType = "qb_spike"; p.Receiver = ""; p.Yardline = "20" }, map[leaguerules.StatID]leaguerules.Decimal{"red_zone_pass_attempt": 10000}},
		{"third down run does not convert", func(p *RawPlay) { p.ThirdConversion = "0"; p.RushAttempt = "1" }, nil},
		{"nonconversion outside", func(p *RawPlay) { p.RushAttempt = "1"; p.Yardline = "21" }, nil},
		{"deleted", func(p *RawPlay) { p.RushAttempt = "1"; p.ThirdConversion = "1"; p.Deleted = "1" }, nil},
		{"two point", func(p *RawPlay) { p.CompletePass = "1"; p.FourthConversion = "1"; p.TwoPoint = "1" }, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, err := rawFixture(t).Convert()
			if err != nil {
				t.Fatal(err)
			}
			p := RawPlay{GameID: w.GameID, PlayID: "1", Season: "2025", Week: "1", SeasonType: "REG", Passer: w.GSIS, Receiver: w.GSIS, Rusher: w.GSIS, EdgeAvailable: true}
			tc.change(&p)
			weeks := []PlayerWeek{w}
			for range 2 {
				if err := AttachPlays(weeks, []RawPlay{p}); err != nil {
					t.Fatal(err)
				}
				for _, id := range edgeStatIDs() {
					if got := weeks[0].Line.Quantities[id]; got != tc.want[id] {
						t.Errorf("%s got %d want %d", id, got, tc.want[id])
					}
				}
			}
		})
	}
}

func TestReadPlaysEdgeCapability(t *testing.T) {
	body := "game_id,play_id,season,week,season_type,passer_player_id,receiver_player_id,rusher_player_id,passing_yards,receiving_yards,rushing_yards,touchdown,td_player_id,td_team,return_team,interception,punt_attempt,kickoff_attempt,play_deleted,two_point_attempt"
	row := "g,1,2025,1,REG,,,,,,,,,,,,,,,"
	header := ",third_down_converted,fourth_down_converted,rush_attempt,pass_attempt,complete_pass,yardline_100,tackled_for_loss"
	for _, tc := range []struct {
		name, h, r string
		fail       bool
	}{
		{"old", body, row, false}, {"full", body + header, row + ",1,0,1,0,0,20,1", false},
		{"NA", body + header, row + ",NA,0,0,0,0,NA,0", false},
		{"partial", body + ",rush_attempt", row + ",1", true},
		{"bad flag", body + header, row + ",2,0,1,0,0,20,1", true},
		{"bad yardline", body + header, row + ",1,0,1,0,0,101,1", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ReadPlays(strings.NewReader(fmt.Sprintf("%s\n%s\n", tc.h, tc.r)))
			if (err != nil) != tc.fail {
				t.Fatalf("error %v", err)
			}
		})
	}
}
