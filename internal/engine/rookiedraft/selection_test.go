package rookiedraft

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func testSelectionBasis() SelectionBasis {
	return SelectionBasis{Season: 2031, Local: time.UTC, Players: []PlayerContract{
		{ID: "0002", Franchise: "b", Salary: 100000000, Status: domain.RosterActive},
		{ID: "0003", Franchise: "a", Salary: 100000000, Status: domain.RosterActive},
	}}
}

func previewRules() leaguerules.RuleSet {
	r := testRules()
	r.Roster.Limits = leaguerules.Known(leaguerules.Bounds{Min: leaguerules.Known(0), Max: leaguerules.Known(2)})
	r.Cap.Amount = leaguerules.Known[leaguerules.Money](400000000)
	r.Cap.Quantum = leaguerules.Known[leaguerules.Money](1)
	return r
}

func TestSelectionRefusesRosterAndCapOveragesWithoutMutation(t *testing.T) {
	r := previewRules()
	basis := testSelectionBasis()
	original := slices.Clone(basis.Players)
	d, err := Replay(r.Draft, testOrder(t, r.Draft), fixedStart(), fixedStart(), nil)
	if err != nil {
		t.Fatal(err)
	}
	candidate := Candidate{ID: "0001", EntrySeason: 2031}
	terms, err := PreviewSelection(r, d, "b", candidate, basis)
	if err != nil || terms.Salary != 300000000 {
		t.Fatalf("exactly fits: %+v, %v", terms, err)
	}
	r.Cap.Amount = leaguerules.Known[leaguerules.Money](399999999)
	if _, err := PreviewSelection(r, d, "b", candidate, basis); err == nil ||
		!strings.Contains(err.Error(), "salary cap") {
		t.Fatalf("one-cent cap overage accepted: %v", err)
	}
	r.Cap.Amount = leaguerules.Known[leaguerules.Money](400000000)
	r.Roster.Limits.Value.Max = leaguerules.Known(1)
	if _, err := PreviewSelection(r, d, "b", candidate, basis); err == nil ||
		!strings.Contains(err.Error(), "roster limits") {
		t.Fatalf("one-player roster overage accepted: %v", err)
	}
	if !reflect.DeepEqual(original, basis.Players) || d.Picks[0].Player != "" {
		t.Fatal("refused preview changed caller assets or the draft")
	}
}

func TestSelectionRequiresAnUnownedRookieAndCurrentOwner(t *testing.T) {
	r := previewRules()
	d, err := Replay(r.Draft, testOrder(t, r.Draft), fixedStart(), fixedStart(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		owner, player string
		entry         int
	}{
		{"a", "0001", 2031}, {"b", "0001", 0}, {"b", "0001", 2030},
		{"b", "0003", 2031}, {"b", "0002", 2031}, {"b", "", 2031},
	} {
		if _, err := PreviewSelection(r, d, row.owner, Candidate{ID: row.player, EntrySeason: row.entry},
			testSelectionBasis()); err == nil {
			t.Fatalf("ineligible selection accepted: %+v", row)
		}
	}
}

func TestOrderRejectsUnknownDuplicateAndMissingOwnership(t *testing.T) {
	r := testRules().Draft
	for _, owned := range [][]OwnedPick{
		{{Round: 1, Original: "a", Owner: "unknown"}},
		{{Round: 1, Original: "a", Owner: "a"}, {Round: 1, Original: "a", Owner: "b"}},
		{{Round: 1, Original: "unknown", Owner: "a"}},
		{{Round: 6, Original: "a", Owner: "a"}}, nil,
	} {
		if _, err := Order(r, []string{"a", "b"}, owned, heldStandings{[]string{"a", "b"}}); err == nil {
			t.Fatalf("bad ownership accepted: %+v", owned)
		}
	}
	r.Order = leaguerules.Known(map[int][]string{2: {"a", "a"}})
	if _, err := Order(r, []string{"a", "b"}, testPicks(5), heldStandings{[]string{"a", "b"}}); err == nil {
		t.Fatal("duplicate round override accepted")
	}
}

func TestReplayRejectsExpiredForeignDuplicateAndOutOfOrderSelections(t *testing.T) {
	r := testRules().Draft
	order := testOrder(t, r)
	start := fixedStart()
	good := Selection{Round: 1, Original: "a", Owner: "b", Player: "0001", At: start.Add(time.Hour)}
	second := Selection{Round: 1, Original: "b", Owner: "b", Player: "0001", At: start.Add(2 * time.Hour)}
	late := good
	late.At = start.Add(6 * time.Hour)
	foreign := good
	foreign.Original = "b"
	for _, selections := range [][]Selection{{late}, {foreign}, {good, second}, {second, good}} {
		if _, err := Replay(r, order, start, start.Add(12*time.Hour), selections); err == nil {
			t.Fatalf("bad selection history accepted: %+v", selections)
		}
	}
	if _, err := Replay(r, order, start, start.Add(-time.Second), []Selection{good}); err == nil {
		t.Fatal("selection accepted before start")
	}
}

func TestSlotTermsRefuseMissingOverlappingAndUnsetPrices(t *testing.T) {
	r := testRules().Draft
	if _, err := SlotTerms(r, 1, 33); err == nil {
		t.Fatal("invented a salary beyond the recorded slots")
	}
	r.Pricing.Value[1] = append(r.Pricing.Value[1], r.Pricing.Value[1][0])
	if _, err := SlotTerms(r, 1, 1); err == nil {
		t.Fatal("ambiguous overlapping salaries accepted")
	}
	r = testRules().Draft
	r.Lengths.Value[2] = leaguerules.Needs[int]("unset")
	if _, err := SlotTerms(r, 2, 1); err == nil {
		t.Fatal("invented missing rookie term length")
	}
	r = testRules().Draft
	r.Pricing.Value[2][0].Salary = leaguerules.Needs[leaguerules.Money]("unset")
	if _, err := SlotTerms(r, 2, 1); err == nil {
		t.Fatal("invented missing rookie salary")
	}
}

func TestDraftWaitsForStartAndDoesNotForfeitEarly(t *testing.T) {
	r := testRules().Draft
	r.MaximumPicks = leaguerules.Known(3)
	order := testOrder(t, r)
	start := fixedStart()
	waiting, err := Replay(r, order, start, start.Add(-time.Nanosecond), nil)
	if err != nil || waiting.Started || waiting.Exhausted || waiting.OnClock != nil {
		t.Fatalf("draft ran before start: %+v, %v", waiting, err)
	}
	for _, pick := range waiting.Picks {
		if pick.Forfeited || pick.Skipped || pick.Player != "" {
			t.Fatalf("draft spent a pick before the clock ran: %+v", pick)
		}
	}
	moved, err := Replay(r, order, start.Add(time.Hour), start, nil)
	if err != nil || moved.Started || moved.OnClock != nil {
		t.Fatalf("start change did not delay clock: %+v, %v", moved, err)
	}
}

func TestDraftCannotStartWithUnresolvedContractsOrIncompleteRounds(t *testing.T) {
	r := testRules().Draft
	order := testOrder(t, r)
	order[1].Picks = order[1].Picks[:1]
	if _, err := Replay(r, order, fixedStart(), fixedStart(), nil); err == nil {
		t.Fatal("incomplete round started")
	}
	order = testOrder(t, r)
	r.RFAAfter.Value[5] = leaguerules.Needs[bool]("unset")
	if _, err := Replay(r, order, fixedStart(), fixedStart(), nil); err == nil {
		t.Fatal("draft started with unknown rookie contract terms")
	}
}

func TestSoftCapPreviewKeepsTheCommissionersFlag(t *testing.T) {
	r := previewRules()
	r.Cap.Amount = leaguerules.Known[leaguerules.Money](399999999)
	r.Cap.Enforcement = leaguerules.Known(leaguerules.SoftCap)
	deadline := fixedStart().Add(24 * time.Hour)
	r.Cap.ComplianceDeadline = leaguerules.Known(deadline)
	d, err := Replay(r.Draft, testOrder(t, r.Draft), fixedStart(), fixedStart(), nil)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := PreviewSelection(r, d, "b", Candidate{ID: "0001", EntrySeason: 2031}, testSelectionBasis())
	if err != nil || !strings.Contains(preview.CapWords, "Over the salary cap: flagged; comply by 15 Jul 2031") {
		t.Fatalf("soft-cap flag lost or refused: %+v, %v", preview, err)
	}
}

func TestFrozenContractSurvivesARepricing(t *testing.T) {
	r := testRules().Draft
	start := fixedStart()
	order := testOrder(t, r)
	frozen := Terms{Salary: 777000000, Years: 2, RFAAfter: true}
	selection := Selection{Round: 1, Original: "a", Owner: "b", Player: "0001", At: start.Add(time.Hour), Terms: &frozen}
	d, err := Replay(r, order, start, selection.At, []Selection{selection})
	if err != nil || d.Picks[0].Contract == nil || *d.Picks[0].Contract != frozen {
		t.Fatalf("the contract fixed at the pick was repriced: %+v, %v", d.Picks[0].Contract, err)
	}
	selection.Terms = nil
	d, err = Replay(r, order, start, selection.At, []Selection{selection})
	if err != nil || d.Picks[0].Contract.Salary != 300000000 {
		t.Fatalf("a pick with no fixed contract takes the slot's terms: %+v, %v", d.Picks[0].Contract, err)
	}
}

func TestPickOnlyRefusesARosterItPushesOverALimit(t *testing.T) {
	r := previewRules()
	d, err := Replay(r.Draft, testOrder(t, r.Draft), fixedStart(), fixedStart(), nil)
	if err != nil {
		t.Fatal(err)
	}
	basis, candidate := testSelectionBasis(), Candidate{ID: "0001", EntrySeason: 2031}
	r.Roster.Limits = leaguerules.Known(leaguerules.Bounds{Min: leaguerules.Known(18), Max: leaguerules.Known(80)})
	if _, err := PreviewSelection(r, d, "b", candidate, basis); err != nil {
		t.Fatalf("a pick onto a roster short of its minimum was refused: %v", err)
	}
	r.Roster.PositionBounds = leaguerules.Known(map[leaguerules.Position]leaguerules.Bounds{
		leaguerules.WR: {Min: leaguerules.Known(0), Max: leaguerules.Known(0)}})
	basis.Facts = map[string]nativecheck.PlayerFacts{
		"0001": {Position: leaguerules.WR}, "0002": {Position: leaguerules.QB}}
	if _, err := PreviewSelection(r, d, "b", candidate, basis); err == nil || !strings.Contains(err.Error(), "WR 1 of 0") {
		t.Fatalf("a rookie past his position's limit was accepted: %v", err)
	}
	basis.Facts["0001"] = nativecheck.PlayerFacts{Position: leaguerules.QB}
	if _, err := PreviewSelection(r, d, "b", candidate, basis); err != nil {
		t.Fatalf("a rookie at another position was refused for an unrelated limit: %v", err)
	}
}

// A franchise's O-Line card never moves through the draft, and holding one does not block a rookie.
func TestDraftNeverMovesAnOLineCard(t *testing.T) {
	r := previewRules()
	r.Roster.Limits.Value.Max = leaguerules.Known(3)
	r.Roster.PositionBounds = leaguerules.Known(map[leaguerules.Position]leaguerules.Bounds{
		leaguerules.OL: {Min: leaguerules.Known(1), Max: leaguerules.Known(1)}})
	d, err := Replay(r.Draft, testOrder(t, r.Draft), fixedStart(), fixedStart(), nil)
	if err != nil {
		t.Fatal(err)
	}
	basis := testSelectionBasis()
	basis.Players = append(basis.Players, PlayerContract{ID: "990003", Franchise: "b", Status: domain.RosterActive})
	basis.Facts = nativecheck.Facts{"0001": {Position: leaguerules.WR}, "0002": {Position: leaguerules.QB}}
	if _, err := PreviewSelection(r, d, "b", Candidate{ID: "0001", EntrySeason: 2031}, basis); err != nil {
		t.Fatalf("a rookie onto a roster holding its card was refused: %v", err)
	}
	if _, err := PreviewSelection(r, d, "b", Candidate{ID: "990001", EntrySeason: 2031}, basis); err == nil ||
		!strings.Contains(err.Error(), "O-Line card") {
		t.Fatalf("a second card was drafted: %v", err)
	}
}
