package native

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestNFLWeekInPlayComesFromTheHeldWeekDates(t *testing.T) {
	t.Parallel()
	kick := func(day int) time.Time { return time.Date(2026, 9, day, 17, 0, 0, 0, time.UTC) }
	week := func(n, day int, final bool) leagueweek.Week {
		return leagueweek.Week{Number: n, Games: []leagueweek.Game{{Teams: [2]string{"AAA", "BBB"}, Kickoff: kick(day), Final: final}}}
	}
	held := []leagueweek.Week{week(1, 10, true), week(2, 17, true), week(3, 24, false), {Number: 4}}
	for _, c := range []struct {
		name  string
		weeks []leagueweek.Week
		at    time.Time
		want  nativecheck.BidSeason
	}{
		{"nothing held", nil, kick(20), nativecheck.BidSeason{}},
		{"a week with no games is not held", []leagueweek.Week{{Number: 1}}, kick(20), nativecheck.BidSeason{}},
		{"before the first kickoff", held, kick(1), nativecheck.BidSeason{Offseason: true}},
		{"the first kickoff ends the offseason; a finished week is past", held, kick(10), nativecheck.BidSeason{Week: 2, WeekKnown: true}},
		{"between weeks", held, kick(20), nativecheck.BidSeason{Week: 3, WeekKnown: true}},
		{"after every held week", held[:2], kick(28), nativecheck.BidSeason{Week: 2, WeekKnown: true}},
		{"the earliest kickoff, whatever the order", []leagueweek.Week{held[1], held[0]}, kick(12),
			nativecheck.BidSeason{Week: 2, WeekKnown: true}},
	} {
		if got := WeekAt(c.weeks, c.at); got != c.want {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestPlayoffFieldIsUnknownUntilTheBracketHasOne(t *testing.T) {
	t.Parallel()
	directory := commissioner.NativeDirectory{Franchises: []commissioner.NativeDirectorySeat{
		{PeopleSeat: commissioner.PeopleSeat{Franchise: "franchise.01"}},
		{PeopleSeat: commissioner.PeopleSeat{Franchise: "franchise.02"}}}}
	if field := PlayoffField(commissioner.NativeBracket{}, directory); len(field) != 0 {
		t.Fatalf("an unsettled bracket invented a field: %v", field)
	}
	bracket := commissioner.NativeBracket{Fields: []commissioner.ConferenceField{
		{Seeds: []commissioner.FieldSeed{{Franchise: "franchise.02"}}}}}
	field := PlayoffField(bracket, directory)
	if field["franchise.02"] != nativecheck.PlayoffIn || field["franchise.01"] != nativecheck.PlayoffOut || len(field) != 2 {
		t.Fatalf("a seeded team is in and every other franchise out: %v", field)
	}
}
