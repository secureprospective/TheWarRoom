package rookiedraft

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

type heldStandings struct{ teams []string }

func (h heldStandings) FirstRound(tie leaguerules.Value[leaguerules.TieRule]) ([]string, error) {
	if tie.State != leaguerules.IsKnown {
		return nil, fmt.Errorf("Needs a ruling: draft order tie breaker")
	}
	return slices.Clone(h.teams), nil
}

func testRules() leaguerules.RuleSet {
	r := leaguerules.LegacyNFL()
	r.Draft.TieRule = leaguerules.Known[leaguerules.TieRule]("commissioner recorded final rank")
	r.Draft.LaterPattern = leaguerules.Known(leaguerules.Linear)
	return r
}

func testPicks(rounds int) []OwnedPick {
	out := []OwnedPick{}
	for round := 1; round <= rounds; round++ {
		out = append(out, OwnedPick{Round: round, Original: "a", Owner: "b"},
			OwnedPick{Round: round, Original: "b", Owner: "b"})
	}
	return out
}

func testOrder(t *testing.T, r leaguerules.DraftRules) []RoundOrder {
	t.Helper()
	order, err := Order(r, []string{"a", "b"}, testPicks(r.Rounds.Value), heldStandings{[]string{"a", "b"}})
	if err != nil {
		t.Fatal(err)
	}
	return order
}

func TestSetupConsumesResolvedRulesWithoutInventingValues(t *testing.T) {
	r := leaguerules.LegacyNFL().Draft
	setup, err := ResolveSetup(r)
	if err != nil || setup.Mode != leaguerules.SlowDraft || setup.Clock != 6*time.Hour ||
		setup.OnExpire != leaguerules.SkipPick || !setup.UnlimitedTimeouts || setup.Rounds != 5 ||
		setup.MaximumPicks != 15 || !setup.ForfeitExtra {
		t.Fatalf("preset setup = %+v, %v", setup, err)
	}
	r.Mode = leaguerules.Known(leaguerules.LiveDraft)
	r.Clock = leaguerules.Known(17 * time.Minute)
	r.Rounds = leaguerules.Known(2)
	r.MaximumPicks = leaguerules.Known(3)
	r.ForfeitExtra = leaguerules.Known(false)
	r.Timeouts = leaguerules.Known(leaguerules.TimeoutPolicy{
		Unlimited: leaguerules.Known(false), Count: leaguerules.Known(2)})
	setup, err = ResolveSetup(r)
	if err != nil || setup.Mode != leaguerules.LiveDraft || setup.Clock != 17*time.Minute ||
		setup.Rounds != 2 || setup.MaximumPicks != 3 || setup.ForfeitExtra || setup.UnlimitedTimeouts ||
		setup.Timeouts != 2 {
		t.Fatalf("custom setup = %+v, %v", setup, err)
	}
	r.Clock = leaguerules.Needs[time.Duration]("commissioner has not set it")
	if _, err := ResolveSetup(r); err == nil || !strings.Contains(err.Error(), "needs a ruling") {
		t.Fatalf("unset clock silently defaulted: %v", err)
	}
}

func TestLaterRoundsAndTieRuleHaveNoDefault(t *testing.T) {
	r := leaguerules.LegacyNFL().Draft
	order := testOrder(t, r)
	if len(order[0].Picks) != 0 || order[0].Words != "Needs a ruling: draft order tie breaker" {
		t.Fatalf("invented tie rule: %+v", order[0])
	}
	r.TieRule = leaguerules.Known[leaguerules.TieRule]("recorded rank")
	order = testOrder(t, r)
	if len(order[0].Picks) != 2 {
		t.Fatalf("first round missing: %+v", order[0])
	}
	for _, round := range order[1:] {
		if len(round.Picks) != 0 || round.Words != "Needs a ruling: later round draft pattern" {
			t.Fatalf("later pattern defaulted: %+v", round)
		}
	}
}

func TestMissingStandingsSeamAndExplicitOrder(t *testing.T) {
	r := testRules().Draft
	order, err := Order(r, []string{"a", "b"}, testPicks(5), nil)
	if err != nil || order[0].Words != "The draft order needs the final standings, which are not built yet." ||
		len(order[0].Picks) != 0 {
		t.Fatalf("missing standings = %+v, %v", order, err)
	}
	r.Order = leaguerules.Known(map[int][]string{1: {"b", "a"}, 2: {"a", "b"}})
	r.TieRule = leaguerules.Needs[leaguerules.TieRule]("unset")
	r.LaterPattern = leaguerules.Needs[leaguerules.OrderPattern]("unset")
	order, err = Order(r, []string{"a", "b"}, testPicks(5), MissingFinalStandings{})
	if err != nil || order[0].Picks[0].Original != "b" || order[1].Picks[0].Original != "a" ||
		len(order[2].Picks) != 0 {
		t.Fatalf("explicit order = %+v, %v", order, err)
	}
}

func TestOrderKeepsTradedOwnerAndSnakeNeverChangesIdentity(t *testing.T) {
	r := testRules().Draft
	r.LaterPattern = leaguerules.Known(leaguerules.Snake)
	order := testOrder(t, r)
	for _, round := range order {
		want := "a"
		if round.Round%2 == 0 {
			want = "b"
		}
		if round.Picks[0].Original != want || round.Picks[0].Owner != "b" ||
			round.Picks[1].Owner != "b" {
			t.Fatalf("wrong original or traded owner: %+v", round)
		}
	}
}

func TestSlotContractsFollowEveryRulebookBoundary(t *testing.T) {
	rows := []struct {
		round, first, last, cents, years int
		rfa                              bool
	}{
		{1, 1, 10, 300000000, 5, false}, {1, 11, 15, 240000000, 5, false},
		{1, 16, 20, 200000000, 5, false}, {1, 21, 32, 150000000, 5, false},
		{2, 1, 12, 142000000, 3, true}, {2, 13, 24, 125000000, 3, true},
		{2, 25, 32, 108000000, 3, true}, {3, 1, 16, 83000000, 3, true},
		{3, 17, 32, 67000000, 3, true}, {4, 1, 32, 60000000, 3, true},
		{5, 1, 32, 50000000, 3, true},
	}
	for _, row := range rows {
		for slot := row.first; slot <= row.last; slot++ {
			terms, err := SlotTerms(leaguerules.LegacyNFL().Draft, row.round, slot)
			if err != nil || terms.Salary != domain.Money(row.cents) || terms.Years != row.years ||
				terms.RFAAfter != row.rfa {
				t.Fatalf("round %d slot %d = %+v, %v", row.round, slot, terms, err)
			}
		}
	}
}

func fixedStart() time.Time { return time.Date(2031, 7, 14, 10, 0, 0, 0, time.UTC) }

func TestExpiredClockSkipsAtDeadlineAndCatchesUp(t *testing.T) {
	r := testRules().Draft
	start := fixedStart()
	order := testOrder(t, r)
	before, err := Replay(r, order, start, start.Add(6*time.Hour-time.Nanosecond), nil)
	if err != nil || before.OnClock == nil || before.OnClock.Slot != 1 || before.Picks[0].Skipped {
		t.Fatalf("early skip: %+v, %v", before, err)
	}
	at, err := Replay(r, order, start, start.Add(6*time.Hour), nil)
	if err != nil || !at.Picks[0].Skipped || at.OnClock == nil || at.OnClock.Slot != 2 ||
		!at.OnClock.Deadline.Equal(start.Add(12*time.Hour)) {
		t.Fatalf("deadline failed to skip: %+v, %v", at, err)
	}
	late, err := Replay(r, order, start, start.Add(19*time.Hour), nil)
	if err != nil || !late.Picks[2].Skipped || late.OnClock == nil || late.OnClock.Round != 2 ||
		late.OnClock.Slot != 2 || !late.OnClock.Deadline.Equal(start.Add(24*time.Hour)) {
		t.Fatalf("catch-up shifted to read time: %+v, %v", late, err)
	}
}

func TestSelectionUsesCurrentOwnerAndSlotContract(t *testing.T) {
	r := testRules().Draft
	start := fixedStart()
	order := testOrder(t, r)
	selection := Selection{Round: 1, Original: "a", Owner: "b", Player: "0001", At: start.Add(time.Hour)}
	d, err := Replay(r, order, start, selection.At, []Selection{selection})
	if err != nil || d.Picks[0].Player != "0001" || d.Picks[0].Owner != "b" ||
		d.Picks[0].Contract == nil || d.Picks[0].Contract.Salary != 300000000 ||
		d.OnClock == nil || !d.OnClock.Deadline.Equal(start.Add(7*time.Hour)) {
		t.Fatalf("selection did not award traded pick at slot terms: %+v, %v", d, err)
	}
	selection.Owner = "a"
	if _, err := Replay(r, order, start, selection.At, []Selection{selection}); err == nil {
		t.Fatal("original team drafted a pick it no longer owns")
	}
	if pick, ok := d.NextPick("b"); !ok || pick.Slot != 2 {
		t.Fatalf("next owned pick = %+v, %v", pick, ok)
	}
	if _, ok := d.NextPick("a"); ok {
		t.Fatal("original franchise shown an owned next pick")
	}
}

func TestRuleChangesRedateClockAndOrderWithoutMutatingInputs(t *testing.T) {
	r := testRules().Draft
	order := testOrder(t, r)
	original := slices.Clone(order[0].Picks)
	start := fixedStart()
	base, err := Replay(r, order, start, start, nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Clock = leaguerules.Known(2 * time.Hour)
	changed, err := Replay(r, order, start, start, nil)
	if err != nil || changed.OnClock == nil || !changed.OnClock.Deadline.Equal(start.Add(2*time.Hour)) {
		t.Fatalf("clock did not redate: %+v, %v", changed, err)
	}
	r.Clock = leaguerules.Known(6 * time.Hour)
	restored, err := Replay(r, order, start, start, nil)
	if err != nil || !reflect.DeepEqual(base, restored) || !reflect.DeepEqual(original, order[0].Picks) {
		t.Fatalf("revert or isolation failed: %+v, %v", restored, err)
	}
	r.Order = leaguerules.Known(map[int][]string{1: {"b", "a"}})
	changedOrder := testOrder(t, r)
	moved, err := Replay(r, changedOrder, start, start, nil)
	if err != nil || moved.OnClock == nil || moved.OnClock.Original != "b" {
		t.Fatalf("order did not move the clock: %+v, %v", moved, err)
	}
}

func TestExtraPicksForfeitWithoutConsumingAClock(t *testing.T) {
	r := testRules().Draft
	r.MaximumPicks = leaguerules.Known(3)
	order := testOrder(t, r)
	start := fixedStart()
	d, err := Replay(r, order, start, start.Add(18*time.Hour), nil)
	if err != nil || !d.Exhausted || d.OnClock != nil || !d.Picks[3].Forfeited || d.Picks[3].Skipped {
		t.Fatalf("extra picks not forfeited: %+v, %v", d, err)
	}
	for _, p := range d.Picks[3:] {
		if !p.Forfeited || p.MadeAt != nil {
			t.Fatalf("forfeit consumed clock or selection: %+v", p)
		}
	}
}

func TestClockShowsTheContractTheSlotWouldSign(t *testing.T) {
	r := testRules().Draft
	order := testOrder(t, r)
	start := fixedStart()
	d, err := Replay(r, order, start, start, nil)
	want := Terms{Salary: 300000000, Years: 5}
	if err != nil || d.OnClock == nil || d.OnClock.Contract != want {
		t.Fatalf("the contract on the clock: %+v, %v", d.OnClock, err)
	}
	d, err = Replay(r, order, start, start.Add(6*time.Hour*2), nil)
	if err != nil || d.OnClock.Round != 2 || d.OnClock.Contract != (Terms{Salary: 142000000, Years: 3, RFAAfter: true}) {
		t.Fatalf("the contract in round two: %+v, %v", d.OnClock, err)
	}
}
