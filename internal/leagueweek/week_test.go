package leagueweek

import (
	"errors"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func TestPhaseEdges(t *testing.T) {
	b := Bounds{Start: 1, LastRegular: 13, End: 17}
	for _, tc := range []struct {
		week    int
		want    domain.Phase
		wantErr bool
	}{
		{0, "", true},
		{1, domain.PhaseRegularSeason, false},
		{13, domain.PhaseRegularSeason, false},
		{14, domain.PhasePlayoffs, false},
		{17, domain.PhasePlayoffs, false},
		{18, "", true},
	} {
		got, err := Phase(tc.week, b)
		if got != tc.want || (err != nil) != tc.wantErr {
			t.Fatalf("week %d: %s, %v", tc.week, got, err)
		}
		if tc.week == 18 && !errors.Is(err, ErrSeasonOver) {
			t.Fatal("season over sentinel lost", err)
		}
	}
	b.Start = 3
	for _, week := range []int{1, 2, 3} {
		got, err := Phase(week, b)
		want := domain.PhaseOffseason
		if week == b.Start {
			want = domain.PhaseRegularSeason
		}
		if err != nil || got != want {
			t.Fatalf("week %d: %s, %v", week, got, err)
		}
	}
	for _, bad := range []Bounds{
		{0, 13, 17}, {-1, 13, 17}, {14, 13, 17}, {1, 18, 17},
	} {
		if _, err := Phase(5, bad); err == nil {
			t.Fatalf("accepted bounds %+v", bad)
		}
	}
	if _, err := Phase(-1, b); err == nil {
		t.Fatal("negative week accepted")
	}
	if got, err := Phase(1, Bounds{1, 1, 1}); err != nil || got != domain.PhaseRegularSeason {
		t.Fatalf("equal bounds: %s, %v", got, err)
	}
}

func TestParseBounds(t *testing.T) {
	for _, tc := range []struct {
		values  [3]string
		wantErr bool
	}{
		{[3]string{"1", "13", "17"}, false},
		{[3]string{"", "13", "17"}, true},
		{[3]string{"1", "bad", "17"}, true},
		{[3]string{"1", "13", "bad"}, true},
		{[3]string{"0", "13", "17"}, true},
		{[3]string{"14", "13", "17"}, true},
		{[3]string{"1", "18", "17"}, true},
	} {
		b, err := ParseBounds(tc.values[0], tc.values[1], tc.values[2])
		if (err != nil) != tc.wantErr {
			t.Fatalf("%v: %+v, %v", tc.values, b, err)
		}
		if !tc.wantErr && b != (Bounds{Start: 1, LastRegular: 13, End: 17}) {
			t.Fatal(b)
		}
	}
}

func TestLineupWeek(t *testing.T) {
	at := time.Date(2026, 10, 7, 13, 38, 0, 0, time.UTC)
	finished := Week{Number: 4, Games: []Game{{Kickoff: at.Add(-time.Hour), Final: true}}}
	current := Week{Number: 5, Games: []Game{
		{Kickoff: at.Add(-time.Hour), Final: true},
		{Kickoff: at.Add(time.Hour)},
	}}
	future := Week{Number: 6, Games: []Game{{Kickoff: at.Add(7 * 24 * time.Hour)}}}
	for _, tc := range []struct {
		name  string
		weeks []Week
		want  int
		ok    bool
	}{
		{"empty", nil, 0, false},
		{"all final", []Week{finished}, 0, false},
		{"partial week", []Week{future, current, finished}, 5, true},
		{"during unfinished game", []Week{{Number: 5, Games: []Game{{Kickoff: at.Add(-time.Hour)}}}}, 5, true},
		{
			"future zero counter",
			[]Week{{Number: 5, Games: []Game{{Kickoff: at.Add(time.Hour), Final: true}}}},
			5, true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := LineupWeek(at, tc.weeks...)
			if ok != tc.ok || got.Number != tc.want {
				t.Fatalf("%+v, %v", got, ok)
			}
		})
	}
}

func TestLocksAndExtrema(t *testing.T) {
	at := time.Date(2026, 10, 9, 0, 15, 0, 0, time.UTC)
	w := Week{Number: 5, Games: []Game{
		{Teams: [2]string{"C", "D"}, Kickoff: at.Add(3 * time.Hour)},
		{Teams: [2]string{"A", "B"}, Kickoff: at},
		{Teams: [2]string{"E", "F"}, Kickoff: at.Add(time.Hour)},
	}}
	if !FirstLock(w).Equal(at) || !LastLock(w).Equal(at.Add(3*time.Hour)) {
		t.Fatal("wrong extrema", FirstLock(w), LastLock(w))
	}
	locks := Locks(w)
	if len(locks) != 6 {
		t.Fatal(locks)
	}
	for _, game := range w.Games {
		for _, team := range game.Teams {
			if !locks[team].Equal(game.Kickoff) {
				t.Fatalf("lock %s: %v", team, locks[team])
			}
		}
	}
	if _, ok := locks["bye"]; ok {
		t.Fatal("bye team must have no lock")
	}
	if len(Locks(Week{})) != 0 {
		t.Fatal("empty week has locks")
	}
	single := Week{Games: w.Games[:1]}
	if !FirstLock(single).Equal(LastLock(single)) {
		t.Fatal("single game extrema differ")
	}
}
