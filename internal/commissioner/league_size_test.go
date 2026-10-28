package commissioner

import (
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// A league can have from 2 to maxFranchises franchises; only a size outside that is refused, in words
// that name the range.
func TestFranchiseCountRange(t *testing.T) {
	for _, c := range []struct {
		teams int
		ok    bool
	}{{1, false}, {2, true}, {32, true}, {33, true}, {100, true}, {maxFranchises, true}, {maxFranchises + 1, false}} {
		_, err := (RawCommand{League: "native.test", Command: Create, Args: RawArgs{Name: "Ours",
			Preset: PresetEmpty, FranchiseCount: c.teams, Season: 2026}, Request: "create", Effective: "now",
			Visibility: "league"}).Prepared()
		if (err == nil) != c.ok || (err != nil && !strings.Contains(err.Error(), "between 2 and 1000")) {
			t.Errorf("%d franchises: %v", c.teams, err)
		}
	}
	if got := ControlsReading().Teams; got != (TeamRange{Min: 2, Max: maxFranchises}) {
		t.Fatalf("the start form reads the range from Go: %+v", got)
	}
}

// Every native preset starts, seeds and reads a league at the largest size, and the count-dependent
// values wait for a ruling there rather than being guessed.
func TestEveryPresetStartsAtTheLargestSize(t *testing.T) {
	for _, p := range StartingPresets() {
		t.Run(p.ID, func(t *testing.T) {
			b := &ledgerBuilder{t: t}
			create, err := (RawCommand{League: "native.test", Command: Create, Args: RawArgs{Name: "Ours", Preset: p.ID,
				FranchiseCount: maxFranchises, Season: 2026}, Request: "create", Effective: "now",
				Visibility: "league"}).Prepared()
			if err != nil {
				t.Fatal(err)
			}
			b.add(create, seasonMoment(1))
			ids := LeagueFranchises(b.records)
			if len(ids) != maxFranchises || len(slices.Compact(slices.Sorted(slices.Values(ids)))) != maxFranchises ||
				ids[999] != "franchise.1000" || ids[98] != "franchise.99" {
				t.Fatalf("%d franchises, last %s", len(ids), ids[len(ids)-1])
			}
			if _, err := RulesReading("native.test", b.records, seasonMoment(2)); err != nil {
				t.Fatal(err)
			}
			people, err := PeopleReading("native.test", "creator", b.records)
			if err != nil || !slices.ContainsFunc(people.Seats, func(s PeopleSeat) bool { return s.Name == "Franchise 1000" }) {
				t.Fatalf("people: %v", err)
			}
			if _, err := JobsReading("native.test", b.records, seasonMoment(2)); err != nil {
				t.Fatal(err)
			}
			rules, err := RulesInForce(b.records, 2026, seasonMoment(2))
			if err != nil || (p.ID != PresetEmpty && rules.League.Teams.Value != maxFranchises) {
				t.Fatalf("rules: %v", err)
			}
			if p.ID == PresetEmpty {
				return
			}
			if rules.League.Alignment.State == leaguerules.IsKnown || rules.Playoffs.Graph.State == leaguerules.IsKnown {
				t.Fatal("a count-dependent value was guessed at the largest size")
			}
			seed, err := NativeStart("native.test", b.records, seasonMoment(2))
			if err != nil {
				t.Fatal(err)
			}
			b.add(RawCommand{League: "native.test", Command: NativeSeed, Args: RawArgs{Native: &seed}, Request: "seed",
				Effective: "now", Visibility: "league"}, seasonMoment(2))
			if _, err := JobsReading("native.test", b.records, seasonMoment(2)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// Franchises past 99 read in seat order and keep their default names.
func TestFranchiseNumbersPastNinetyNine(t *testing.T) {
	for id, want := range map[string]int{"franchise.01": 1, "franchise.99": 99, "franchise.100": 100,
		"franchise.1000": 1000, "franchise.1": 0, "franchise.007": 0, "team.3": 0} {
		if got := franchiseNumber(id); got != want {
			t.Errorf("%s: %d, want %d", id, got, want)
		}
	}
	if name, short := defaultFranchiseName("franchise.100"); name != "Franchise 100" || short != "F100" {
		t.Fatalf("%s %s", name, short)
	}
}

// The other limits widened with the franchise count: a bracket seed up to the largest league, a trade
// among as many parties as the league allows, and a bid as long as seasons are numbered.
func TestWidenedLimits(t *testing.T) {
	seed := bracketRaw(BracketSeed, RawBracketChange{Season: 2026, Slot: 40, Team: "franchise.40"})
	if err := seed.Validate(); err != nil {
		t.Fatalf("seed 40: %v", err)
	}
	seed.Args.Bracket.Slot = maxFranchises + 1
	if err := seed.Validate(); err == nil {
		t.Fatal("a seed past the largest league was accepted")
	}
	policy := TradePolicy{DecideAt: 1, MinimumRound: 1, MaximumRound: 1, Expiry: 1, MaximumParties: 6}
	if policy.invalidLimits() {
		t.Fatal("a six-team trade limit was refused")
	}
	policy.MaximumParties = 1
	if !policy.invalidLimits() {
		t.Fatal("a one-party trade was accepted")
	}
	d := bidDetails{contract: true}
	if d.incomplete(RawBidArgs{Salary: 1, Years: 120}) || !d.incomplete(RawBidArgs{Salary: 1, Years: 0}) {
		t.Fatal("bid years: a long contract is the league's call; zero years is not a contract")
	}
}
