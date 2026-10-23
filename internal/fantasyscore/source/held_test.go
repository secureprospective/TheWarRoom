package source_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

// A postseason row that shares the regular-season week number is never the player's league week.
func TestSelectHeldWeekReadsOnlyTheRegularSeason(t *testing.T) {
	scope := source.RawStatScope{GSIS: "00-0023459", Season: 2025, Week: 1}
	post := source.PlayerWeek{GSIS: scope.GSIS, Season: 2025, Week: 1, SeasonType: "POST"}
	held := source.SelectHeldWeek(scope, []source.PlayerWeek{post}, "sha", time.Unix(0, 0))
	if held.Week != nil || len(held.Missing) != 2 {
		t.Fatalf("a postseason row was read as the regular-season week: %+v", held)
	}
	reg := post
	reg.SeasonType = "REG"
	held = source.SelectHeldWeek(scope, []source.PlayerWeek{post, reg}, "sha", time.Unix(0, 0))
	if held.Week == nil || held.Week.SeasonType != "REG" || len(held.Missing) != 1 {
		t.Fatalf("the regular-season row was not selected: %+v", held)
	}
}

// The index answers every scope exactly as the scan does: first row wins, preseason rows are not the week,
// and an unreported week is missing, never zero.
func TestHeldWeekIndexMatchesTheScan(t *testing.T) {
	at := time.Date(2025, 9, 9, 12, 0, 0, 0, time.UTC)
	weeks := []source.PlayerWeek{
		{GSIS: "00-0000001", Season: 2025, Week: 1, SeasonType: "PRE", GameID: "pre"},
		{GSIS: "00-0000001", Season: 2025, Week: 1, SeasonType: "REG", GameID: "first"},
		{GSIS: "00-0000001", Season: 2025, Week: 1, SeasonType: "REG", GameID: "second"},
		{GSIS: "00-0000002", Season: 2025, Week: 2, SeasonType: "REG", GameID: "other"},
		{GSIS: "00-0000002", Season: 2024, Week: 2, SeasonType: "REG", GameID: "old"},
	}
	index := source.IndexHeldWeeks(weeks)
	for _, gsis := range []string{"00-0000001", "00-0000002", "00-0000003"} {
		for _, season := range []int{2024, 2025} {
			for week := 1; week <= 3; week++ {
				scope := source.RawStatScope{GSIS: gsis, Season: season, Week: week}
				got, want := index.Select(scope, weeks, "sha", at), source.SelectHeldWeek(scope, weeks, "sha", at)
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("%+v: index %+v, scan %+v", scope, got, want)
				}
			}
		}
	}
}
