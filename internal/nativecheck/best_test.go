package nativecheck

import (
	"fmt"
	"math/rand"
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func bounds(low, high int) leaguerules.Bounds {
	return leaguerules.Bounds{Min: leaguerules.Known(low), Max: leaguerules.Known(high)}
}

// bruteForce tries every subset of a small pool and returns the best total that JudgeLineup accepts.
func bruteForce(rules leaguerules.LineupRules, pool []Starter) (leaguerules.Decimal, bool) {
	var best leaguerules.Decimal
	found := false
	for mask := 0; mask < 1<<len(pool); mask++ {
		var chosen []Starter
		for i := range pool {
			if mask&(1<<i) != 0 {
				chosen = append(chosen, pool[i])
			}
		}
		if JudgeLineup(rules, true, chosen).Status != LineupValid {
			continue
		}
		if total := Total(chosen); !found || total > best {
			best, found = total, true
		}
	}
	return best, found
}

func smallRules() leaguerules.LineupRules {
	return leaguerules.LineupRules{
		Positions: leaguerules.Known(map[leaguerules.Position]leaguerules.Bounds{
			leaguerules.QB: bounds(1, 1), leaguerules.RB: bounds(0, 2), leaguerules.WR: bounds(1, 3),
			leaguerules.PK: bounds(0, 1), leaguerules.LB: bounds(1, 2)}),
		Groups: leaguerules.Known(map[leaguerules.LineupGroup]leaguerules.Bounds{
			leaguerules.Offense: bounds(3, 4), leaguerules.Defense: bounds(1, 2), leaguerules.Total: bounds(4, 6)}),
		InvalidPenalty: leaguerules.Known(leaguerules.ZeroTopStarter), MissedFlag: leaguerules.Known(true)}
}

func TestBestLineupMatchesAnExhaustiveSearch(t *testing.T) {
	//nolint:gosec // A fixed seed makes this exhaustive-search comparison reproducible; no secrets.
	rng := rand.New(rand.NewSource(7))
	positions := []leaguerules.Position{leaguerules.QB, leaguerules.RB, leaguerules.WR, leaguerules.PK, leaguerules.LB}
	rules := smallRules()
	for round := 0; round < 40; round++ {
		var pool []Starter
		for i := 0; i < 11; i++ {
			pool = append(pool, Starter{ID: fmt.Sprintf("p%02d", i), Position: positions[rng.Intn(len(positions))],
				Points: leaguerules.Decimal(rng.Intn(400000) - 50000)})
		}
		want, wantFound := bruteForce(rules, pool)
		got := SelectBestLineup(rules, pool)
		if got.Found != wantFound || (wantFound && got.Points != want) {
			t.Fatalf("round %d: search %v/%d, exhaustive %v/%d", round, got.Found, got.Points, wantFound, want)
		}
		if got.Found {
			if v := JudgeLineup(rules, true, got.Starters); v.Status != LineupValid {
				t.Fatalf("round %d: the best lineup breaks the limits: %+v", round, v)
			}
			if Total(got.Starters) != got.Points {
				t.Fatalf("round %d: points do not add up", round)
			}
		}
	}
}

func rosterPool() []Starter {
	var pool []Starter
	counts := map[leaguerules.Position]int{leaguerules.QB: 3, leaguerules.RB: 6, leaguerules.WR: 8, leaguerules.TE: 4,
		leaguerules.PK: 2, leaguerules.DT: 4, leaguerules.DE: 4, leaguerules.LB: 7, leaguerules.CB: 5, leaguerules.S: 5}
	for _, position := range sortedPositions(counts) {
		for i := 0; i < counts[position]; i++ {
			pool = append(pool, Starter{ID: fmt.Sprintf("%s%d", position, i), Position: position,
				Points: leaguerules.Decimal(1000 * (100 - i*7 + int(position[0])%9))})
		}
	}
	return pool
}

func TestBestLegacyLineupKeepsEveryGroupLimit(t *testing.T) {
	rules := leaguerules.LegacyNFL().Lineup
	best := SelectBestLineup(rules, rosterPool())
	if !best.Found || len(best.Starters) != 21 {
		t.Fatalf("%+v", best)
	}
	if v := JudgeLineup(rules, true, best.Starters); v.Status != LineupValid {
		t.Fatalf("%+v", v)
	}
	// Without the group limits the same pool lets the best scorers crowd the offense past eight.
	loose := rules
	loose.Groups = leaguerules.Needs[map[leaguerules.LineupGroup]leaguerules.Bounds]("")
	free := SelectBestLineup(loose, rosterPool())
	if free.Points <= best.Points {
		t.Fatalf("the group limits cost nothing, so this test proves nothing: %d vs %d", free.Points, best.Points)
	}
	if v := JudgeLineup(rules, true, free.Starters); v.Status != LineupInvalid {
		t.Fatalf("expected the unlimited pick to break the group limits: %+v", v)
	}
}

func TestBestLineupIsDeterministicAndOrdered(t *testing.T) {
	rules := leaguerules.LegacyNFL().Lineup
	pool := rosterPool()
	first := SelectBestLineup(rules, pool)
	slices.Reverse(pool)
	second := SelectBestLineup(rules, pool)
	if fmt.Sprint(first.Starters) != fmt.Sprint(second.Starters) {
		t.Fatalf("the pick depends on input order:\n%v\n%v", first.Starters, second.Starters)
	}
	if first.Starters[0].Position != leaguerules.QB {
		t.Fatalf("starters not in lineup order: %v", first.Starters)
	}
}

func TestBestLineupSaysWhenNoneCanBeBuilt(t *testing.T) {
	rules := leaguerules.LegacyNFL().Lineup
	if got := SelectBestLineup(rules, rosterPool()[:5]); got.Found || got.Words != "No lineup can meet this league's limits with these players" {
		t.Fatalf("%+v", got)
	}
	rules.Positions = leaguerules.Needs[map[leaguerules.Position]leaguerules.Bounds]("")
	if got := SelectBestLineup(rules, rosterPool()); got.Found || got.Words != "Needs a ruling: starting position limits" {
		t.Fatalf("%+v", got)
	}
}
