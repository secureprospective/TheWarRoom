package commissioner

import (
	"context"
	"fmt"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

type throughScores struct {
	complete int
	partial  bool
}

func (s throughScores) Lineup(context.Context, []Record, string, int) (WeekLineup, error) {
	return WeekLineup{}, nil
}
func (s throughScores) PlayerScore(context.Context, []Record, string, int) (NativePlayerScore, error) {
	return NativePlayerScore{}, nil
}
func (s throughScores) MemoWeekScore(_ context.Context, _ []Record, _ leaguerules.LineupRules, seat NativeDirectorySeat,
	_, week int, _ func() (FranchiseWeekScore, error)) (FranchiseWeekScore, error) {
	score := FranchiseWeekScore{Franchise: seat.Franchise, Name: seat.Name}
	if week <= s.complete || (s.partial && week == s.complete+1 && seat.Franchise != "team.00") {
		points := leaguerules.Decimal(10000)
		score.Points = &points
	}
	return score, nil
}

func TestStandingsThroughCompleteWeeks(t *testing.T) {
	directory := NativeDirectory{}
	weeks := make([]ScheduleWeek, 18)
	for i := range 32 {
		id := fmt.Sprintf("team.%02d", i)
		directory.Franchises = append(directory.Franchises, NativeDirectorySeat{PeopleSeat: PeopleSeat{Franchise: id, Name: id}})
	}
	for i := range weeks {
		weeks[i].Week = i + 1
		for j := 0; j < 32; j += 2 {
			weeks[i].Games = append(weeks[i].Games, ScheduleGame{Home: fmt.Sprintf("team.%02d", j), Away: fmt.Sprintf("team.%02d", j+1)})
		}
	}
	for _, tc := range []struct {
		name             string
		complete         int
		partial          bool
		decided, pending int
	}{
		{"eight", 8, false, 128, 160},
		{"nine", 9, false, 144, 144},
		{"partial ninth", 8, true, 143, 145},
		{"no results", 0, false, 0, 288},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rules := leaguerules.LegacyNFL()
			rules.Schedule.RegularWeeks.Value = 18
			counted, err := countSeason(t.Context(), "league", nil, throughScores{tc.complete, tc.partial}, rules.Lineup, directory, weeks, 2026)
			if err != nil {
				t.Fatal(err)
			}
			out, err := (NativeStandings{scheduledWeeks: 18}).rank(rules, directory, counted, StandingsAdjustments{}, conferenceLayout{})
			if err != nil {
				t.Fatal(err)
			}
			prefix := "Standings: no week is complete"
			if tc.complete > 0 {
				prefix = fmt.Sprintf("Standings through week %d", tc.complete)
			}
			want := fmt.Sprintf("%s: %d regular-season games have a result, %d remain without a result", prefix, tc.decided, tc.pending)
			if out.Through != tc.complete || out.Words != want || out.Decided != tc.decided || out.Pending != tc.pending {
				t.Fatalf("got through=%d decided=%d pending=%d words=%q; want %q", out.Through, out.Decided, out.Pending, out.Words, want)
			}
		})
	}
}
