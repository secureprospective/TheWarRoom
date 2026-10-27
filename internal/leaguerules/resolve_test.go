package leaguerules

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func testTime() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
func testChange(path FieldPath, value Setting) Change {
	return Change{LeagueID: "test-league", Season: 2026, Field: path, New: value,
		RecordedAt: testTime(), Effective: Effective{Kind: Now}}
}
func TestEffectiveRules(t *testing.T) {
	recorded := testTime()
	rollover := recorded.Add(24 * time.Hour)
	seed := SeedChanges(LegacyNFL(), "test-league", 2026, recorded)
	cases := []struct {
		name string
		kind EffectiveKind
		at   time.Time
		want Money
	}{
		{"rollover before", NextRollover, rollover.Add(-time.Nanosecond), 12500000000},
		{"rollover at", NextRollover, rollover, 13000000000},
		{"rollover after", NextRollover, rollover.Add(time.Second), 13000000000},
		{"now before", Now, recorded.Add(time.Hour - time.Nanosecond), 12500000000},
		{"now at", Now, recorded.Add(time.Hour), 13000000000},
		{"instant before", Instant, rollover.Add(-time.Nanosecond), 12500000000},
		{"instant at", Instant, rollover, 13000000000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := testChange("cap.amount", Known(Money(13000000000)))
			c.RecordedAt = recorded.Add(time.Hour)
			c.Effective = Effective{Kind: tc.kind}
			if tc.kind != Now {
				c.Effective.At = rollover
			}
			changes := append(append([]Change{}, seed...), c)
			got, err := Resolve(changes, 2026, tc.at)
			if err != nil {
				t.Fatal(err)
			}
			if got.Cap.Amount.Value != tc.want || got.Cap.Amount.State != IsKnown {
				t.Fatalf("cap = %+v", got.Cap.Amount)
			}
		})
	}
}
func TestPastSeasonIsolation(t *testing.T) {
	changes := SeedChanges(LegacyNFL(), "test-league", 2025, testTime())
	c := testChange("cap.amount", Known(Money(9000000000)))
	changes = append(changes, c)
	got, err := Resolve(changes, 2025, testTime().Add(365*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if got.Cap.Amount.Value != 12500000000 || got.Season != 2025 {
		t.Fatalf("past season changed: %+v", got.Cap)
	}
}
func TestNeedsRulingAndZero(t *testing.T) {
	cases := []struct {
		path          FieldPath
		unknown, zero Setting
	}{
		{"league.teams", Needs[int](""), Known(0)},
		{"cap.amount", Needs[Money](""), Known(Money(0))},
		{"contracts.deadCap", Needs[Decimal](""), Known(Decimal(0))},
		{"waivers.duration", Needs[time.Duration](""), Known(time.Duration(0))},
	}
	for _, tc := range cases {
		t.Run(string(tc.path), func(t *testing.T) {
			before, err := Resolve([]Change{testChange(tc.path, tc.unknown)}, 2026, testTime())
			if err != nil {
				t.Fatal(err)
			}
			after, err := Resolve([]Change{testChange(tc.path, tc.zero)}, 2026, testTime())
			if err != nil {
				t.Fatal(err)
			}
			diff := Diff(before, after)
			if len(diff) != 1 || diff[0].Field != tc.path {
				t.Fatalf("unknown collapsed into zero: %+v", diff)
			}
			if !reflect.DeepEqual(diff[0].Before, tc.unknown) || !reflect.DeepEqual(diff[0].After, tc.zero) {
				t.Fatalf("states lost: %+v", diff)
			}
		})
	}
}
func TestInvalidChanges(t *testing.T) {
	cases := []struct {
		name   string
		change Change
		words  string
	}{
		{"roster bounds", testChange("roster.limits", Known(bounds(81, 80, ""))), "minimum"},
		{"lineup bounds", testChange("lineup.positions", Known(map[Position]Bounds{QB: bounds(2, 1, "")})), "minimum"},
		{"seats eleven", testChange("trades.dotSeats", Known(11)), "between 4 and 10"},
		{"seats three", testChange("trades.dotSeats", Known(3)), "between 4 and 10"},
		{"wrong type", testChange("cap.amount", Known(125)), "correct kind"},
		{"negative money", testChange("cap.amount", Known(Money(-1))), "negative"},
		{"negative count", testChange("roster.limits", Known(bounds(-1, 80, ""))), "negative"},
		{"zero quantum", testChange("cap.quantum", Known(Money(0))), "greater than zero"},
		{"bad share", testChange("cap.irShare", Known(Decimal(10001))), "100 percent"},
		{"bad menu", testChange("cap.enforcement", Known(CapMode("anything"))), "available option"},
		{"unknown path", testChange("no.such.rule", Known(3)), "recognised"},
		{"bad state", testChange("cap.amount", Value[Money]{State: 99}), "set or Needs"},
		{"hidden unknown value", testChange("cap.amount", Value[Money]{Value: 1}), "undecided"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.change.Validate()
			if err == nil || !strings.Contains(err.Error(), tc.words) {
				t.Fatalf("error = %v, want %q", err, tc.words)
			}
			if strings.Contains(err.Error(), string(tc.change.Field)) {
				t.Fatalf("internal path in error: %v", err)
			}
		})
	}
	for _, seats := range []int{4, 10} {
		if err := testChange("trades.dotSeats", Known(seats)).Validate(); err != nil {
			t.Fatal(err)
		}
	}
}
func TestNoInventedDefaultsAndTiming(t *testing.T) {
	got, err := Resolve(nil, 2026, testTime())
	if err != nil {
		t.Fatal(err)
	}
	if got.Cap.Amount.State != NeedsRuling || got.League.Teams.State != NeedsRuling {
		t.Fatal("unseeded league inherited policy")
	}
	cases := []Effective{
		{Kind: NextRollover}, {Kind: NextRollover, At: testTime()},
		{Kind: EffectiveKind(99)}, {Kind: Now, At: testTime().Add(time.Hour)},
	}
	for _, effective := range cases {
		c := testChange("league.teams", Known(32))
		c.Effective = effective
		if c.Validate() == nil {
			t.Fatalf("accepted timing %+v", effective)
		}
	}
	future := testChange("league.teams", Known(64))
	future.RecordedAt = testTime().Add(time.Hour)
	future.Effective = Effective{Kind: Instant, At: testTime().Add(-time.Hour)}
	got, err = Resolve([]Change{future}, 2026, testTime())
	if err != nil {
		t.Fatal(err)
	}
	if got.League.Teams.State != NeedsRuling {
		t.Fatal("change leaked before recording")
	}
}
func TestResolveAndDiffOwnTheirCollections(t *testing.T) {
	preset := LegacyNFL()
	changes := SeedChanges(preset, "test-league", 2026, testTime())
	preset.Lineup.Positions.Value[QB] = bounds(9, 9, "")
	first, err := Resolve(changes, 2026, testTime())
	if err != nil {
		t.Fatal(err)
	}
	if first.Lineup.Positions.Value[QB].Min.Value != 1 {
		t.Fatal("seed aliases preset")
	}
	first.Lineup.Positions.Value[QB] = bounds(8, 8, "")
	second, err := Resolve(changes, 2026, testTime())
	if err != nil {
		t.Fatal(err)
	}
	if second.Lineup.Positions.Value[QB].Min.Value != 1 {
		t.Fatal("resolve aliases ledger")
	}
	diff := Diff(second, first)
	first.Lineup.Positions.Value[QB] = bounds(7, 7, "")
	after, ok := diff[0].After.(Value[map[Position]Bounds])
	if !ok || after.Value[QB].Min.Value != 8 {
		t.Fatal("diff aliases input")
	}
	for _, change := range diff {
		if strings.Contains(change.Label, ".") || strings.Contains(change.Label, "dOT") {
			t.Fatalf("internal diff label: %s", change.Label)
		}
	}
}

func TestOLineOnReadsTheRosterBound(t *testing.T) {
	r := RuleSet{Roster: RosterRules{PositionBounds: Known(map[Position]Bounds{})}}
	if OLineOn(r) {
		t.Fatal("no OL bound must leave O-Line off")
	}
	r.Roster.PositionBounds.Value[OL] = bounds(1, 1, "")
	if !OLineOn(r) {
		t.Fatal("an OL bound must turn O-Line on")
	}
	if OLineOn(RuleSet{}) {
		t.Fatal("unknown roster must not enable O-Line")
	}
}
