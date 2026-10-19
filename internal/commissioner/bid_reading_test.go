package commissioner

import (
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestBidCloseIsARulingUntilTheLeagueSetsHowLongABidStands(t *testing.T) {
	zone := time.FixedZone("PDT", -7*60*60)
	placed := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	if at, words := bidClose(bidStandFor(leaguerules.RuleSet{}), placed, zone); at != nil ||
		!strings.HasPrefix(words, "Needs a ruling") {
		t.Fatalf("a closing time was invented: %v %q", at, words)
	}
	at, words := bidClose(bidStandFor(leaguerules.LegacyNFL()), placed, zone)
	if at == nil || !at.Equal(placed.Add(24*time.Hour)) || at.Location() != zone ||
		words != "Closes Oct 14, 2026 at 5:00 AM PDT" {
		t.Fatalf("a set stand time did not date the close: %v %q", at, words)
	}
	if at, _ := bidClose(leaguerules.Known(time.Duration(0)), placed, zone); at != nil {
		t.Fatal("a zero stand time closed the auction")
	}
}

func TestLeaderIsTheMostPointsAndAmongEqualsTheFirstPlaced(t *testing.T) {
	early, late := time.Unix(100, 0), time.Unix(200, 0)
	lead, ok := leaderOf([]bidRecord{{ID: "a", Points: 50000, PlacedAt: late}, {ID: "b", Points: 60000, PlacedAt: late},
		{ID: "c", Points: 60000, PlacedAt: early}})
	if !ok || lead.ID != "c" {
		t.Fatalf("leader: %+v", lead)
	}
	if _, ok := leaderOf(nil); ok {
		t.Fatal("no bids had a leader")
	}
}

func TestExperienceNeedsARealDraftYear(t *testing.T) {
	held := func(year int) nativecheck.PlayerFacts {
		return nativecheck.PlayerFacts{DraftYear: year, DraftYearHeld: true}
	}
	for _, c := range []struct {
		facts nativecheck.PlayerFacts
		want  int
	}{
		{held(2020), 6}, {held(2026), 0}, {held(2027), 0}, {held(1970), 0},
		{nativecheck.PlayerFacts{DraftYear: 2020}, 0},
	} {
		if got := experienceYears(2026, c.facts); got != c.want {
			t.Errorf("%+v: %d years, want %d", c.facts, got, c.want)
		}
	}
}
