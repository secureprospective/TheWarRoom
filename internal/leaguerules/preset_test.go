package leaguerules

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLegacyPresetGolden(t *testing.T) {
	preset := LegacyNFL()
	body, err := json.MarshalIndent(preset, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := "testdata/legacy.json"
	if os.Getenv("LEAGUERULES_WRITE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(expected) != string(body) {
		t.Fatal("Legacy preset differs from the reviewed census golden")
	}
	assertCitations(t, reflect.ValueOf(preset))
	for _, c := range SeedChanges(preset, "test-league", 2026, testTime()) {
		if err := c.Validate(); err != nil {
			t.Fatalf("preset %s: %v", c.Field, err)
		}
	}
}
func assertCitations(t *testing.T, v reflect.Value) {
	t.Helper()
	switch v.Kind() {
	case reflect.Struct:
		if strings.HasPrefix(v.Type().Name(), "Value[") {
			if v.FieldByName("Citation").String() == "" {
				t.Fatalf("missing citation for %s", v.Type())
			}
			if v.FieldByName("State").Uint() == uint64(NeedsRuling) {
				return
			}
			assertCitations(t, v.FieldByName("Value"))
			return
		}
		for i := range v.NumField() {
			if v.Field(i).CanInterface() {
				assertCitations(t, v.Field(i))
			}
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			assertCitations(t, iter.Value())
		}
	case reflect.Slice:
		for i := range v.Len() {
			assertCitations(t, v.Index(i))
		}
	case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Pointer, reflect.String, reflect.UnsafePointer:
		return

	}
}
func TestLegacyCensusDefaults(t *testing.T) {
	r := LegacyNFL()
	integerCases := []struct {
		name      string
		got, want int
	}{
		{"teams", r.League.Teams.Value, 32}, {"conferences", r.League.Conferences.Value, 2},
		{"divisions", r.League.Divisions.Value, 8}, {"roster minimum", r.Roster.Limits.Value.Min.Value, 18},
		{"roster maximum", r.Roster.Limits.Value.Max.Value, 80}, {"IR", r.IR.Slots.Value, 12},
		{"taxi", r.Taxi.Slots.Value, 8}, {"taxi years", r.Taxi.MaximumYears.Value, 3},
		{"active bids", r.Bids.MaximumActive.Value, 30}, {"playoff field", r.Playoffs.PerConference.Value, 7},
		{"wildcards", r.Playoffs.WildCards.Value, 3}, {"regular weeks", r.Schedule.RegularWeeks.Value, 13},
		{"draft rounds", r.Draft.Rounds.Value, 5}, {"draft picks", r.Draft.MaximumPicks.Value, 15},
		{"trade parties", r.Trades.MaximumParties.Value, 3}, {"trade votes", r.Trades.DecideAt.Value, 3},
		{"tag top", r.Contracts.Tag.Value.TopPlayers.Value, 5},
		{"extension years", r.Contracts.Extension.Value.TotalYears.Value, 6},
	}
	for _, tc := range integerCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("got %d want %d", tc.got, tc.want)
			}
		})
	}
	if r.Cap.Amount.Value != 12500000000 || r.Cap.Quantum.Value != 1000000 || r.Cap.Enforcement.Value != HardCap {
		t.Fatalf("cap preset: %+v", r.Cap)
	}
	if r.Contracts.DeadCap.Value != 3500 || r.Contracts.RestructuredDeadCap.Value != 5000 {
		t.Fatal("dead cap rates changed")
	}
	for i, want := range []Decimal{10000, 12000, 14000, 16000} {
		if r.Bids.YearMultipliers.Value[i].Value != want {
			t.Fatal("bid multipliers changed")
		}
	}
	if r.Waivers.Duration.Value != 24*time.Hour || r.Waivers.Method.Value != ReverseStandings {
		t.Fatal("waiver preset changed")
	}
	if r.Taxi.Basis.Value != CalendarYears {
		t.Fatal("taxi ruling not applied")
	}
	if !reflect.DeepEqual(r.IR.Eligible.Value, []NFLStatus{InjuredReserve, PUP, NFI, Out, Suspended}) {
		t.Fatal("IR ruling not applied")
	}
	if r.Draft.LaterPattern.State != NeedsRuling || r.Draft.TieRule.State != NeedsRuling {
		t.Fatal("draft pattern or tie rule invented")
	}
	if r.Calendar.TradeDeadline.State != NeedsRuling || r.Calendar.Timezone.Value != "America/New_York" {
		t.Fatal("calendar ruling not applied")
	}
	if r.Calendar.Windows.Value["contracts"].Opens.State != NeedsRuling {
		t.Fatal("contract window actions invented")
	}
	if r.Bids.UFA.Value != Open || r.Bids.RFA.Value != Open || r.Bids.Mode(UDFAProcess).Value != Open {
		t.Fatal("Legacy bids are open for UFA and RFA, and UDFA follows UFA")
	}
	if r.Trades.DOTSeats.State != NeedsRuling {
		t.Fatal("actual DOT membership invented")
	}
}
func TestCompleteScoring(t *testing.T) {
	catalogue := LegacyNFL().Scoring.Catalogue.Value
	if len(catalogue) != 45 {
		t.Fatalf("scoring catalogue has %d entries", len(catalogue))
	}
	for id, want := range map[StatID]Decimal{
		"passing_td": 50000, "passing_yards": 500, "rushing_attempt": 1500, "kickoff_return_td": 100000,
		"pass_defensed": 25000, "tackle": 15000, "tackle_line": 25000, "assist": 10000,
		"assist_line": 15000, "tackle_for_loss": 25000, "sack": 45000, "fg_missed_0_29": -30000,
		"fg_missed_30_99": -10000,
	} {
		stat, ok := catalogue[id]
		if !ok || stat.Points.State != IsKnown || stat.Points.Value != want {
			t.Fatalf("%s = %+v", id, stat)
		}
	}
	for _, id := range []StatID{"long_rush", "long_reception"} {
		bonuses := catalogue[id].Bonuses.Value
		if len(bonuses) != 2 || bonuses[0].At.Value != 200000 || bonuses[1].At.Value != 400000 ||
			bonuses[0].Points.Value+bonuses[1].Points.Value != 20000 {
			t.Fatalf("bonuses lost: %+v", bonuses)
		}
	}
	for _, id := range []StatID{"tackle_line", "assist_line"} {
		if !reflect.DeepEqual(catalogue[id].Positions.Value, []Position{DT, DE}) {
			t.Fatal("true position split lost")
		}
	}
	unused := UnusedStat("Passing attempts", "Passing")
	zero := scoringStat("Passing attempts", "Passing", 0)
	if unused.Used.Value || unused.Points.State != NeedsRuling || !zero.Used.Value || zero.Points.State != IsKnown {
		t.Fatal("unused collapsed into zero")
	}
}
func TestAllDiffLabelsAndTypedFields(t *testing.T) {
	fields := ruleFields()
	changes := Diff(RuleSet{}, LegacyNFL())
	if len(changes) != len(fields) {
		t.Fatalf("diff omitted fields: %d of %d", len(changes), len(fields))
	}
	seen := make(map[FieldPath]bool)
	for _, c := range changes {
		if c.Label == "" || strings.Contains(c.Label, ".") || strings.Contains(c.Label, "_") ||
			strings.Contains(c.Label, "Value[") {
			t.Fatalf("not plain words: %q", c.Label)
		}
		if seen[c.Field] {
			t.Fatalf("duplicate path %s", c.Field)
		}
		seen[c.Field] = true
	}
	seed := SeedChanges(LegacyNFL(), "test-league", 2026, testTime())
	got, err := Resolve(seed, 2026, testTime())
	if err != nil {
		t.Fatal(err)
	}
	if diff := Diff(LegacyNFL(), got); len(diff) != 0 {
		t.Fatalf("preset replay loses fields: %+v", diff)
	}
}

func TestUnlimitedIsNotUnanswered(t *testing.T) {
	r := LegacyNFL()
	last := r.Contracts.MinimumSalaries.Value[6]
	if last.MaximumUnlimited.State != IsKnown || !last.MaximumUnlimited.Value {
		t.Fatal("10+ experience was treated as an unanswered maximum")
	}
	tiers := r.Bids.Tiers.Value
	if tiers[0].Salary.MaximumUnlimited.Value || !tiers[1].Salary.MaximumUnlimited.Value {
		t.Fatal("open salary ceiling was treated as an unanswered maximum")
	}
	if r.Draft.Timeouts.Value.Unlimited.State != IsKnown || !r.Draft.Timeouts.Value.Unlimited.Value {
		t.Fatal("unlimited draft timeouts were treated as unanswered")
	}
}
func TestUDFAAlwaysFollowsUFA(t *testing.T) {
	for _, value := range []Value[BidMode]{Needs[BidMode](""), Known(Blind), Known(Open)} {
		bids := BidsRules{UFA: value, RFA: Known(Open)}
		if !reflect.DeepEqual(bids.Mode(UDFAProcess), value) {
			t.Fatal("UDFA diverged from UFA")
		}
		if !reflect.DeepEqual(bids.Mode(RFAProcess), Known(Open)) {
			t.Fatal("RFA used UFA mode")
		}
	}
}
