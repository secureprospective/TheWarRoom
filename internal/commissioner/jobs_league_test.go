package commissioner

import (
	"testing"
	"time"
)

func TestTheRolloverJobFollowsTheRecordedRollover(t *testing.T) {
	b := rolloverLedger(t)
	b.set("calendar.rollover", `"2027-03-01T05:00:00Z"`, "now", seasonMoment(5))
	read := func(at time.Time) Job {
		t.Helper()
		jobs, err := JobsReading(rolloverLeague, b.records, at)
		if err != nil {
			t.Fatal(err)
		}
		return byID(t, jobs.Jobs, "rollover")
	}
	if j := read(seasonMoment(6)); j.State != JobUpcoming || j.Opens != "2027-03-01T05:00:00Z" {
		t.Fatalf("a dated rollover is upcoming: %+v", j)
	}
	if j := read(rolloverMoment()); j.State != JobLive {
		t.Fatalf("a rollover at its date is live: %+v", j)
	}
	// Run a day early: a rollover recorded after the date was set is the dated one, whenever it ran.
	early := rolloverMoment().Add(-24 * time.Hour)
	b.rollover(2027, early)
	after := rolloverMoment().Add(time.Hour)
	if j := read(early.Add(time.Hour)); j.State != JobBlocked || j.Words != rolloverRunWords {
		t.Fatalf("a rollover run before its date is done: %+v", j)
	}
	if j := read(after); j.State != JobBlocked || j.Words != rolloverRunWords {
		t.Fatalf("once run, the rollover waits for its next date: %+v", j)
	}
	b.set("calendar.rollover", `"2028-03-01T05:00:00Z"`, "now", after)
	if j := read(after.Add(time.Hour)); j.State != JobUpcoming || j.Opens != "2028-03-01T05:00:00Z" {
		t.Fatalf("a newly dated rollover is upcoming again: %+v", j)
	}
}

func TestTheRookieDraftJobFollowsTheNextDraft(t *testing.T) {
	b := rolloverLedger(t)
	read := func(at time.Time) Job {
		t.Helper()
		jobs, err := JobsReading(rolloverLeague, b.records, at)
		if err != nil {
			t.Fatal(err)
		}
		return byID(t, jobs.Jobs, "rookie_draft")
	}
	if j := read(seasonMoment(6)); j.State != JobBlocked || j.Words != "The 2028 rookie draft has no start time; the commissioner sets it." {
		t.Fatalf("an undated draft is blocked: %+v", j)
	}
	start := seasonMoment(20)
	b.add(RawCommand{League: rolloverLeague, Command: DraftStart, Args: RawArgs{Draft: &RawDraftArgs{Season: 2028, At: start}},
		Request: "start", Effective: "now", Visibility: "league"}, seasonMoment(6))
	if j := read(seasonMoment(7)); j.State != JobUpcoming || j.Opens != start.Format(time.RFC3339) {
		t.Fatalf("a dated draft is upcoming from its start: %+v", j)
	}
	if j := read(start); j.State != JobLive || j.Words != "The 2028 rookie draft is under way." {
		t.Fatalf("a started draft is live: %+v", j)
	}
	unseeded := newLedger(t, PresetLegacy)
	jobs, err := JobsReading(rolloverLeague, unseeded.records, seasonMoment(6))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"rookie_draft", "dot_review", "waivers", "bids", "stat_queue", "in_season"} {
		if j := byID(t, jobs.Jobs, id); j.State != JobBlocked || j.Words != startingAssetsWords {
			t.Fatalf("%s waits for the starting assets: %+v", id, j)
		}
	}
}

func TestTradeReviewCountsReviewsAndAppeals(t *testing.T) {
	seeded := NativeAssets{Seeded: true}
	trades := []TradeState{{Status: "review"}, {Status: "review"}, {Status: "appealed"}, {Status: "offered"}, {Status: "approved"}}
	j := tradeReviewJob(seeded, trades)
	if j.State != JobLive || j.Words != "Trades awaiting review: 2. Appeals awaiting a ruling: 1." {
		t.Fatalf("review counts only trades the DOT or a commissioner must decide: %+v", j)
	}
}

func TestPlayoffsAreUpcomingUntilThisSeasonsBracketIsBuilt(t *testing.T) {
	b := rolloverLedger(t)
	read := func() Job {
		t.Helper()
		jobs, err := JobsReading(rolloverLeague, b.records, seasonMoment(9))
		if err != nil {
			t.Fatal(err)
		}
		return byID(t, jobs.Jobs, "playoffs")
	}
	if j := read(); j.State != JobUpcoming {
		t.Fatalf("before the bracket the playoffs are upcoming: %+v", j)
	}
	b.add(RawCommand{League: rolloverLeague, Command: BracketBuild, Args: RawArgs{Bracket: &RawBracketChange{Season: 2025}},
		Request: "old", Effective: "now", Visibility: "league"}, seasonMoment(7))
	if j := read(); j.State != JobUpcoming {
		t.Fatalf("another season's bracket does not start these playoffs: %+v", j)
	}
	b.add(RawCommand{League: rolloverLeague, Command: BracketBuild, Args: RawArgs{Bracket: &RawBracketChange{Season: 2026}},
		Request: "build", Effective: "now", Visibility: "league"}, seasonMoment(8))
	if j := read(); j.State != JobLive {
		t.Fatalf("once this season's bracket is built the playoffs are live: %+v", j)
	}
}
