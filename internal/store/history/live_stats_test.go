package history

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func liveLine(yards leaguerules.Decimal) source.PlayerWeek {
	return source.PlayerWeek{Position: "WR",
		Line: fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{
			"receiving_yards": yards}}}
}

func nflverseEvidence(yards leaguerules.Decimal) source.NativeEvidence {
	week := liveLine(yards)
	week.Season, week.Week, week.SeasonType = 2026, 8, "REG"
	held := source.HeldWeek{Authority: source.NFLVerse, SHA256: "final", Week: &week, Missing: []string{}}
	return source.NativeEvidence{Held: &held, Lines: source.DistinctLines([]source.HeldWeek{held}), Missing: []string{}}
}

func TestLiveStatsBecomeTheNewestRevisionWithoutMovingThePinnedOnes(t *testing.T) {
	t.Parallel()
	s, at := &Store{}, time.Date(2026, 11, 1, 18, 0, 0, 0, time.UTC)
	scope := source.RawNativeScope{Player: "0099", Season: 2026, Week: 8}
	// With nothing live held, the evidence is untouched.
	held := nflverseEvidence(500000)
	if got := s.withLiveStats(held, scope); len(got.Versions()) != 1 {
		t.Fatalf("versions %+v", got.Versions())
	}
	s.SetLiveStats(2026, 8, at, "ESPN live", []string{"the game is being played"},
		map[string]source.PlayerWeek{"0099": liveLine(300000)})
	got := s.withLiveStats(nflverseEvidence(500000), scope)
	versions := got.Versions()
	if len(versions) != 2 {
		t.Fatalf("versions %+v", versions)
	}
	// The final revision keeps sequence 0, so a commissioner's pin does not move.
	if versions[0].Seq != 0 || versions[0].Held.Authority != source.NFLVerse {
		t.Fatalf("the held final revision moved: %+v", versions[0])
	}
	if versions[1].Seq != 1 || versions[1].Held.Authority != "ESPN live" ||
		versions[1].Held.Week.Line.Quantities["receiving_yards"] != 300000 {
		t.Fatalf("live revision %+v", versions[1])
	}
	if got.Held == nil || got.Held.Authority != "ESPN live" || !got.Held.FetchedAt.Equal(at) {
		t.Fatalf("the newest held evidence is the live one: %+v", got.Held)
	}
	if len(got.Missing) != 1 || got.Missing[0] != "the game is being played" {
		t.Fatalf("missing %+v", got.Missing)
	}
}

// The ladder: the final feed is counted over any live one, and a live revision counts when it is
// all the league holds.
func TestTheFinalFeedIsCountedOverALiveOne(t *testing.T) {
	t.Parallel()
	s, at := &Store{}, time.Now().UTC()
	scope := source.RawNativeScope{Player: "0099", Season: 2026, Week: 8}
	s.SetLiveStats(2026, 8, at, "ESPN live", nil, map[string]source.PlayerWeek{"0099": liveLine(300000)})
	liveOnly := s.withLiveStats(source.NativeEvidence{Missing: []string{}}, scope)
	counted := countedLive(t, liveOnly)
	if counted.Held.Authority != "ESPN live" || counted.Held.Week.Line.Quantities["receiving_yards"] != 300000 {
		t.Fatalf("a live revision should count when it is all that is held: %+v", counted)
	}
	withFinal := s.withLiveStats(nflverseEvidence(500000), scope)
	counted = countedLive(t, withFinal)
	if counted.Held.Authority != source.NFLVerse || counted.Held.Week.Line.Quantities["receiving_yards"] != 500000 {
		t.Fatalf("the final feed should be counted: %+v", counted)
	}
}

// A live book for another week is not offered for this one, and an emptied book stops offering.
func TestLiveStatsAreScopedToTheirWeekAndCanBeCleared(t *testing.T) {
	t.Parallel()
	s, at := &Store{}, time.Now().UTC()
	s.SetLiveStats(2026, 7, at, "ESPN live", nil, map[string]source.PlayerWeek{"0099": liveLine(300000)})
	for _, scope := range []source.RawNativeScope{
		{Player: "0099", Season: 2026, Week: 8}, {Player: "0099", Season: 2025, Week: 7},
		{Player: "0100", Season: 2026, Week: 7},
	} {
		if got := s.withLiveStats(source.NativeEvidence{}, scope); got.Held != nil {
			t.Fatalf("%+v was offered a live revision it does not own", scope)
		}
	}
	s.SetLiveStats(2026, 7, at, "ESPN live", nil, nil)
	if got := s.withLiveStats(source.NativeEvidence{}, source.RawNativeScope{Player: "0099", Season: 2026, Week: 7}); got.Held != nil {
		t.Fatal("a cleared live book still offered a revision")
	}
}

// A poll that changed nothing refreshes the held revision's time rather than adding a version.
func TestAnUnchangedLivePollDoesNotAddAVersion(t *testing.T) {
	t.Parallel()
	s := &Store{}
	scope := source.RawNativeScope{Player: "0099", Season: 2026, Week: 8}
	first := time.Date(2026, 11, 1, 18, 0, 0, 0, time.UTC)
	s.SetLiveStats(2026, 8, first, "ESPN live", nil, map[string]source.PlayerWeek{"0099": liveLine(300000)})
	one := s.withLiveStats(source.NativeEvidence{}, scope)
	s.SetLiveStats(2026, 8, first.Add(time.Minute), "ESPN live", nil,
		map[string]source.PlayerWeek{"0099": liveLine(300000)})
	again := s.withLiveStats(one, scope)
	if len(again.Versions()) != 1 {
		t.Fatalf("an unchanged poll added a version: %+v", again.Versions())
	}
	if !again.Held.FetchedAt.Equal(first.Add(time.Minute)) {
		t.Fatalf("the held time did not move: %v", again.Held.FetchedAt)
	}
	s.SetLiveStats(2026, 8, first.Add(2*time.Minute), "ESPN live", nil,
		map[string]source.PlayerWeek{"0099": liveLine(420000)})
	moved := s.withLiveStats(again, scope)
	if len(moved.Versions()) != 2 {
		t.Fatalf("a changed line is a new version: %+v", moved.Versions())
	}
}

// countedLive is the revision the league's feed ladder counts, with no commissioner choice pinned.
// It calls the league's own ladder, so this test fails if that ladder changes.
func countedLive(t *testing.T, evidence source.NativeEvidence) source.LineRevision {
	t.Helper()
	selected := commissioner.FeedVersion(evidence.Versions())
	if selected == nil {
		t.Fatal("no revision is held")
	}
	return *selected
}
