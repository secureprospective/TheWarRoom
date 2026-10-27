package composition

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/core/coretest"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func TestArchiveOpponentLockOrderAndFallback(t *testing.T) {
	s := coretest.New(t).Build()
	s.Week.Number = 5
	s.SeasonLineups.Value = leaguefeed.Lineups{Week: 5, Matchups: []leaguefeed.Matchup{{Home: "0001", Away: "0002"}}}
	s.ArchiveOpponentFallback = func(week int, team string) string {
		if week != 5 || team != "0001" {
			t.Error(week, team)
		}
		if s.SeasonMu.TryLock() {
			s.SeasonMu.Unlock()
			t.Error("fallback ran without SeasonMu")
		}
		if !s.WeekMu.TryLock() {
			t.Error("WeekMu retained while reading fallback")
		} else {
			s.WeekMu.Unlock()
		}
		return "0003"
	}
	if got := (archiveHost{s}).ArchiveOpponent("0001"); got != "0002" {
		t.Fatal(got)
	}
	if got := (archiveHost{s}).ArchiveOpponent("0002"); got != "0001" {
		t.Fatal(got)
	}
	s.SeasonLineups.Value.Week = 4
	if got := (archiveHost{s}).ArchiveOpponent("0001"); got != "0003" {
		t.Fatal(got)
	}
	s.ArchiveOpponentFallback = nil
	if got := (archiveHost{s}).ArchiveOpponent("0001"); got != "" {
		t.Fatal(got)
	}
}
