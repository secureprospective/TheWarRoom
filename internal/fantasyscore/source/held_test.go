package source_test

import (
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
