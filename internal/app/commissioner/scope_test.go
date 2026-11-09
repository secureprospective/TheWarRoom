package commissioner

import (
	"testing"
)

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
