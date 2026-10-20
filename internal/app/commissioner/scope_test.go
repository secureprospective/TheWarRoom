package commissioner

import (
	"errors"
	"fmt"
	"testing"

	commish "github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestAppLeagueAdmitsOnlyANamedAppLeague(t *testing.T) {
	t.Parallel()
	for league, ok := range map[string]bool{"native.seed": true, "native.": false, "legacy": false,
		" native.seed": false, "native.seed ": false, "": false} {
		if got := AppLeague(league) == nil; got != ok {
			t.Fatalf("%q admitted %v, want %v", league, got, ok)
		}
		if got := (ScheduleScope{League: league}).Validate() == nil; got != ok {
			t.Fatalf("schedule scope %q admitted %v, want %v", league, got, ok)
		}
	}
}

func TestCalendarScopeNamesWhatIsMissing(t *testing.T) {
	t.Parallel()
	good := CalendarScope{League: "native.seed", Season: 2026, LocalZone: "America/Chicago",
		Franchise: "franchise.01", Action: "cut"}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		edit func(*CalendarScope)
		want string
	}{
		{func(s *CalendarScope) { s.League = "native." }, "choose an app league, season, franchise and action"},
		{func(s *CalendarScope) { s.Season = 1899 }, "choose an app league, season, franchise and action"},
		{func(s *CalendarScope) { s.Franchise = " " }, "choose an app league, season, franchise and action"},
		{func(s *CalendarScope) { s.Action = "" }, "choose an app league, season, franchise and action"},
		{func(s *CalendarScope) { s.LocalZone = "Local" }, "choose your local time zone"},
		{func(s *CalendarScope) { s.LocalZone = "Nowhere/Zone" }, "read owner time zone: unknown time zone Nowhere/Zone"},
	} {
		scope := good
		tc.edit(&scope)
		if err := scope.Validate(); err == nil || err.Error() != tc.want {
			t.Fatalf("%+v: %v, want %q", scope, err, tc.want)
		}
	}
}

func TestBlockedAndChangeErrorReadTheRefusalAlone(t *testing.T) {
	t.Parallel()
	refusal := fmt.Errorf("decode: %w", commish.Refused(errors.New("your league role does not permit this action")))
	preview := Blocked(refusal)
	if len(preview.Blockers) != 1 || preview.Blockers[0] != "Your league role does not permit this action" ||
		preview.Fields == nil || preview.Affects.Readings == nil || preview.Affects.Words != "Not known yet" {
		t.Fatalf("blocked preview: %+v", preview)
	}
	if err := ChangeError(refusal); err == nil || err.Error() != "Your league role does not permit this action" {
		t.Fatalf("refused change: %v", err)
	}
	if err := ChangeError(errors.New("disk")); err == nil || err.Error() != "league change: disk" {
		t.Fatalf("failed change: %v", err)
	}
	if err := ChangeError(nil); err != nil {
		t.Fatalf("recorded change: %v", err)
	}
}
