package commissioner

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// trialScores scores each franchise's starters; once a proposal is appended, the decided player gains bonus.
type trialScores struct {
	held    int
	decided string
	bonus   leaguerules.Decimal
	lineups map[string][]string
	points  map[string]leaguerules.Decimal
}

func (s trialScores) Lineup(_ context.Context, _ []Record, id string, _ int) (WeekLineup, error) {
	l := WeekLineup{Set: true}
	for _, p := range s.lineups[id] {
		l.Starters = append(l.Starters, Starter{Player: p})
	}
	return l, nil
}

func (s trialScores) PlayerScore(_ context.Context, records []Record, id string, _ int) (NativePlayerScore, error) {
	points, ok := s.points[id]
	if !ok {
		return NativePlayerScore{Status: "Incomplete", Words: "Not reported"}, nil
	}
	if id == s.decided && len(records) > s.held {
		points += s.bonus
	}
	return NativePlayerScore{Status: "Complete", Points: &points}, nil
}

// Re-scoring only the franchises that start the decided player must read exactly as scoring the whole week
// after the decision, for a flip, a game that cannot be called, a bye and a player nobody starts.
func TestRescoredFlipMatchesFullWeek(t *testing.T) {
	directory := NativeDirectory{}
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		directory.Franchises = append(directory.Franchises, NativeDirectorySeat{PeopleSeat: PeopleSeat{Franchise: id, Name: "Team " + id}})
	}
	base := &flipBase{directory: directory, before: map[int]flipWeek{},
		published: []ScheduleWeek{{Week: 1, Games: []ScheduleGame{{Home: "a", Away: "b"}, {Home: "c", Away: "d"}}, Bye: "e"}}}
	records := []Record{{Receipt: Receipt{Entry: "e1"}}}
	for _, tc := range []struct {
		player string
		bonus  leaguerules.Decimal
	}{{"0099", 50000}, {"0099", 1}, {"0101", 90000}, {"0103", 90000}, {"0104", 90000}} {
		src := trialScores{held: len(records), decided: tc.player, bonus: tc.bonus,
			lineups: map[string][]string{"a": {"0099", "0100"}, "b": {"0102"}, "c": {"0101", "0099"}, "d": {"0105"}, "e": {"0103"}},
			points:  map[string]leaguerules.Decimal{"0099": 10000, "0100": 10000, "0101": 10000, "0102": 40000, "0103": 1, "0104": 1}}
		item := StatItemID{Season: 2026, Week: 1, Player: tc.player, Seq: 1, Digest: "abcdef0123456789"}.String()
		raw := RawCommand{League: "native.test", Command: StatApprove, Request: "queue." + item,
			Args: RawArgs{Stat: &RawStatChange{Item: item}}}
		clear(base.before)
		got, err := base.check(t.Context(), "native.test", records, src, raw, 2026, 1)
		if err != nil {
			t.Fatal(err)
		}
		before, err := weekResults(t.Context(), "native.test", records, src, base.rules.Lineup, directory, base.published, 2026, 1)
		if err != nil {
			t.Fatal(err)
		}
		after, err := weekResults(t.Context(), "native.test", scoreTrial(raw, records), src, base.rules.Lineup, directory, base.published, 2026, 1)
		if err != nil {
			t.Fatal(err)
		}
		want := compareWeeks(before, after, franchisesTouched(raw, before, after))
		g, _ := json.Marshal(got)
		w, _ := json.Marshal(want)
		if string(g) != string(w) {
			t.Fatalf("%s +%d: rescored %s\nfull %s", tc.player, tc.bonus, g, w)
		}
	}
}
