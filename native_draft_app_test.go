package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

const (
	standingsWords = "The draft order needs the final standings, which are not built yet."
	laterRound     = "Needs a ruling: later round draft pattern"
)

// governs is the survey's round trip for one setting of the draft: read, set through the real command,
// read the consequence, reopen the database for real, make the compensating revert and read again; then do
// it all again from a freshly reopened database.
func (f *draftFixture) governs(t *testing.T, read func() string, set func() commissioner.Receipt, before,
	after string) {
	t.Helper()
	for pass, name := range []string{"live", "after reopening the database"} {
		if pass == 1 {
			f.reopen(t)
		}
		if got := read(); got != before {
			t.Fatalf("%s: before the ruling: %q, want %q", name, got, before)
		}
		receipt := set()
		if got := read(); got != after {
			t.Fatalf("%s: after the ruling: %q, want %q", name, got, after)
		}
		f.reopen(t)
		if got := read(); got != after {
			t.Fatalf("%s: the ruling did not survive reopening: %q, want %q", name, got, after)
		}
		f.revert(t, receipt)
		if got := read(); got != before {
			t.Fatalf("%s: after the compensating revert: %q, want %q", name, got, before)
		}
	}
}

func deadlineOf(d commissioner.NativeDraft) string {
	if d.Draft.OnClock == nil {
		return d.Words
	}
	return d.Draft.OnClock.Deadline.Format(time.RFC3339)
}

func onClockOf(d commissioner.NativeDraft) string {
	if d.Draft.OnClock == nil {
		return d.Words
	}
	c := d.Draft.OnClock
	return fmt.Sprintf("round %d pick %d: %s (owned by %s)", c.Round, c.Slot, c.Original, c.Owner)
}

func draftRuleWords(d commissioner.NativeDraft, label string) string {
	for _, line := range d.Rules {
		if line.Label == label {
			return line.Words
		}
	}
	return "no such rule line: " + label
}

func TestDraftClockAndRoundsGovernTheDraft(t *testing.T) {
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.now = draftStartAt().Add(time.Hour)
	read := func() string { return deadlineOf(f.read(t, "franchise.03")) }
	f.governs(t, read, func() commissioner.Receipt { return f.setRule(t, "draft.clock", `7200000000000`) },
		"2027-07-14T15:00:00-07:00", "2027-07-14T11:00:00-07:00")
	words := func() string { return draftRuleWords(f.read(t, "franchise.03"), "Draft pick clock") }
	f.governs(t, words, func() commissioner.Receipt { return f.setRule(t, "draft.clock", `5400000000000`) },
		"6 hours", "1 hour 30 minutes")
	rounds := func() string {
		d := f.read(t, "franchise.03")
		return fmt.Sprint(len(d.Order))
	}
	f.governs(t, rounds, func() commissioner.Receipt { return f.setRule(t, "draft.rounds", `3`) }, "5", "3")
}

func TestDraftOrderCommandsGovernWhoIsOnTheClock(t *testing.T) {
	f := draftApp(t)
	f.draftCommand(t, commissioner.DraftStart, commissioner.RawDraftArgs{Season: draftSeason, At: draftStartAt()})
	round := func(n int) string {
		order := f.read(t, "franchise.01").Order
		if len(order[n-1].Picks) == 0 {
			return order[n-1].Words
		}
		return order[n-1].Words + ": " + order[n-1].Picks[0].Original
	}
	if got := round(1); got != standingsWords {
		t.Fatalf("round one invented an order: %q", got)
	}
	first := func() commissioner.Receipt {
		return f.draftCommand(t, commissioner.DraftOrderSet, commissioner.RawDraftArgs{Season: draftSeason,
			Round: 1, Order: draftOrder(3, 2, 1)})
	}
	f.governs(t, func() string { return round(1) }, first, standingsWords, "Draft order set: franchise.03")
	first()
	if got := round(2); got != laterRound {
		t.Fatalf("the later-round pattern was defaulted instead of needing a ruling: %q", got)
	}
	pattern := func(p leaguerules.OrderPattern) func() commissioner.Receipt {
		return func() commissioner.Receipt {
			return f.draftCommand(t, commissioner.DraftOrderPattern, commissioner.RawDraftArgs{Pattern: p})
		}
	}
	f.governs(t, func() string { return round(2) }, pattern(leaguerules.Linear), laterRound,
		"Draft order set: franchise.03")
	f.governs(t, func() string { return round(2) }, pattern(leaguerules.Snake), laterRound,
		"Draft order set: franchise.32")
	pattern(leaguerules.Linear)()
	f.now = draftStartAt().Add(time.Hour)
	onClock := func() string { return onClockOf(f.read(t, "franchise.01")) }
	f.governs(t, onClock, func() commissioner.Receipt {
		return f.draftCommand(t, commissioner.DraftOrderSet, commissioner.RawDraftArgs{Season: draftSeason,
			Order: draftOrder(5, 6), Running: true})
	}, "round 1 pick 1: franchise.03 (owned by franchise.03)",
		"round 1 pick 1: franchise.05 (owned by franchise.05)")
}

func TestDraftTieRuleIsRecordedAndReverted(t *testing.T) {
	f := draftApp(t)
	tie := func() string { return draftRuleWords(f.read(t, "franchise.01"), "Draft order tie breaker") }
	f.governs(t, tie, func() commissioner.Receipt {
		return f.draftCommand(t, commissioner.DraftOrderTiebreak, commissioner.RawDraftArgs{Tie: "Better record"})
	}, "Needs a ruling", "Better record")
}

func TestDraftOrderAutoAsksForTheFinalStandings(t *testing.T) {
	f := draftApp(t)
	f.draftCommand(t, commissioner.DraftOrderSet, commissioner.RawDraftArgs{Season: draftSeason,
		Round: 1, Order: draftOrder(3, 2, 1)})
	read := func() string { return f.read(t, "franchise.01").Order[0].Words }
	f.governs(t, read, func() commissioner.Receipt {
		return f.draftCommand(t, commissioner.DraftOrderAuto, commissioner.RawDraftArgs{Season: draftSeason,
			Source: leaguerules.WorstStandingsFirst})
	}, "Draft order set", standingsWords)
}

// calendarDraftStart reads the draft window's opening as the league calendar shows it.
func calendarDraftStart(t *testing.T, f *draftFixture) string {
	t.Helper()
	reading, err := f.app.TargetLeagueCalendar(RawLeagueCalendarScope{League: draftLeague, Season: draftSeason,
		LocalZone: "America/Los_Angeles", Franchise: "franchise.01", Action: "draft_pick"})
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range reading.Clock.Windows {
		if w.ID == "rookie_draft" {
			return w.Start.Format(time.RFC3339)
		}
	}
	t.Fatal("the calendar has no rookie draft window")
	return ""
}

func TestDraftStartAndTheCalendarNeverDisagree(t *testing.T) {
	f := draftApp(t)
	f.draftCommand(t, commissioner.DraftOrderSet, commissioner.RawDraftArgs{Season: draftSeason,
		Order: draftOrder(3, 2, 1)})
	both := func() string {
		calendar, draft := calendarDraftStart(t, f), f.read(t, "franchise.01").Draft.Start.Format(time.RFC3339)
		if calendar != draft {
			t.Fatalf("the calendar opens the draft at %s but the draft starts at %s", calendar, draft)
		}
		return calendar
	}
	set := func(at time.Time) func() commissioner.Receipt {
		return func() commissioner.Receipt {
			return f.draftCommand(t, commissioner.DraftStart, commissioner.RawDraftArgs{Season: draftSeason, At: at})
		}
	}
	// With no start recorded the draft opens where the league's own window does: 14 July, midnight in New York.
	f.governs(t, both, set(draftStartAt()), "2027-07-13T21:00:00-07:00", "2027-07-14T09:00:00-07:00")
	first := set(draftStartAt())()
	moved := set(draftStartAt().Add(48 * time.Hour))()
	for pass := 0; pass < 2; pass++ {
		if got := both(); got != "2027-07-16T09:00:00-07:00" {
			t.Fatalf("the moved start: %s", got)
		}
		f.reopen(t)
	}
	f.revert(t, moved)
	if got := both(); got != "2027-07-14T09:00:00-07:00" {
		t.Fatalf("after reversing the move: %s", got)
	}
	f.revert(t, first)
	f.reopen(t)
	if got := both(); got != "2027-07-13T21:00:00-07:00" {
		t.Fatalf("after reversing every start: %s", got)
	}
}

func TestDraftStartRefusals(t *testing.T) {
	f := draftApp(t)
	f.draftCommand(t, commissioner.DraftOrderSet, commissioner.RawDraftArgs{Season: draftSeason,
		Order: draftOrder(3, 2, 1)})
	tryStart := func(at time.Time) error {
		_, err := f.tryCommand(commissioner.DraftStart, commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{
			Season: draftSeason, At: at}}, "")
		return err
	}
	if err := tryStart(f.now.Add(-time.Hour)); err == nil || !strings.Contains(err.Error(), "has not passed") {
		t.Fatalf("a start in the past: %v", err)
	}
	f.draftCommand(t, commissioner.DraftStart, commissioner.RawDraftArgs{Season: draftSeason, At: draftStartAt()})
	if _, err := f.tryCommand(commissioner.WindowMove, commissioner.RawArgs{Window: &commissioner.RawWindowArgs{
		ID: "rookie_draft", Start: date(8, 1)}}, ""); err == nil || !strings.Contains(err.Error(), "agree") {
		t.Fatalf("a window move beside the draft start: %v", err)
	}
	f.now = draftStartAt().Add(time.Minute)
	if err := tryStart(draftStartAt().Add(24 * time.Hour)); err == nil ||
		!strings.Contains(err.Error(), "can no longer move") {
		t.Fatalf("a start moved after the clock runs: %v", err)
	}
}

func onClockContract(d commissioner.NativeDraft) string {
	if d.Draft.OnClock == nil {
		return d.Words
	}
	c := d.Draft.OnClock.Contract
	return fmt.Sprintf("%d years at %d cents, restricted free agency %v", c.Years, c.Salary, c.RFAAfter)
}

func TestDraftPricingAndLengthsGovernTheContractOnTheClock(t *testing.T) {
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.now = draftStartAt().Add(time.Hour)
	read := func() string { return onClockContract(f.read(t, "franchise.03")) }
	const legacy = "5 years at 300000000 cents, restricted free agency false"
	f.governs(t, read, func() commissioner.Receipt {
		return f.setRule(t, "draft.lengths", `{"1":4,"2":3,"3":3,"4":3,"5":3}`)
	}, legacy, "4 years at 300000000 cents, restricted free agency false")
	f.governs(t, read, func() commissioner.Receipt {
		return f.setRule(t, "draft.rfaAfter", `{"1":true,"2":true,"3":true,"4":true,"5":true}`)
	}, legacy, "5 years at 300000000 cents, restricted free agency true")
	flat := `{"1":[{"Slots":{"Min":1,"Max":32},"Salary":123400000}],"2":[{"Slots":{"Min":1,"Max":32},"Salary":1}],` +
		`"3":[{"Slots":{"Min":1,"Max":32},"Salary":1}],"4":[{"Slots":{"Min":1,"Max":32},"Salary":1}],` +
		`"5":[{"Slots":{"Min":1,"Max":32},"Salary":1}]}`
	f.governs(t, read, func() commissioner.Receipt { return f.setRule(t, "draft.pricing", flat) }, legacy,
		"5 years at 123400000 cents, restricted free agency false")
}

func TestDraftMaximumPicksForfeitsTheExtrasWithoutAClock(t *testing.T) {
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.now = draftStartAt().Add(time.Hour)
	forfeited := func() string {
		count := 0
		for _, pick := range f.read(t, "franchise.01").Draft.Picks {
			if pick.Forfeited && pick.Owner == "franchise.01" {
				count++
			}
		}
		return fmt.Sprint(count)
	}
	// franchise.01 holds two first round picks and one in each later round: six, so one allowed forfeits five.
	f.governs(t, forfeited, func() commissioner.Receipt { return f.setRule(t, "draft.maximumPicks", `1`) }, "0", "5")
}
