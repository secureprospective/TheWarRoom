package rookiedraft

import (
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// Pool is who a draft may take. The zero value is the rookie draft's: the season's rookie class.
type Pool string

const (
	RookieClass Pool = ""
	// Unrostered is a startup draft's pool with the rookie class in it: every held player on no roster,
	// never an O-Line card.
	Unrostered Pool = "unrostered"
	// UnrosteredVeterans is a startup draft's pool without the incoming rookie class.
	UnrosteredVeterans Pool = "unrostered_veterans"
)

// Startup is a startup draft's resolved settings. It fills a new league's empty rosters with the same
// order, clock and replay as the rookie draft; only its rules, its picks and its pool differ.
type Startup struct {
	Rounds  int
	Order   leaguerules.StartupOrderSource
	List    []string
	Pattern leaguerules.OrderPattern
	Terms   Terms
	// Pool is who the startup draft may take: with or without the incoming rookie class.
	Pool Pool
}

// ResolveStartup reads the startup draft's settings. A rule set stored before they existed reads them
// unset, and each then takes its default: rounds to fill the active roster less any O-Line card, a random
// first round, snake order, one-year contracts at the lowest minimum salary, and the incoming rookie class
// left out of the pool.
func ResolveStartup(r leaguerules.RuleSet) (Startup, error) {
	d := r.Draft
	out := Startup{Order: leaguerules.RandomOrder, Pattern: leaguerules.Snake, Terms: Terms{Years: 1},
		Pool: UnrosteredVeterans}
	if d.StartupRookies.State == leaguerules.IsKnown && d.StartupRookies.Value {
		out.Pool = Unrostered
	}
	rounds, err := startupRounds(r)
	if err != nil {
		return Startup{}, err
	}
	out.Rounds = rounds
	if d.StartupOrder.State == leaguerules.IsKnown {
		out.Order = d.StartupOrder.Value
	}
	if d.StartupList.State == leaguerules.IsKnown {
		out.List = slices.Clone(d.StartupList.Value)
	}
	if d.StartupPattern.State == leaguerules.IsKnown {
		out.Pattern = d.StartupPattern.Value
	}
	if d.StartupYears.State == leaguerules.IsKnown {
		out.Terms.Years = d.StartupYears.Value
	}
	salary, err := startupSalary(r)
	if err != nil {
		return Startup{}, err
	}
	out.Terms.Salary = salary
	if out.Terms.Years < 1 {
		return Startup{}, fmt.Errorf("set at least one contract year for startup draft picks")
	}
	if out.Order != leaguerules.RandomOrder && out.Order != leaguerules.CommissionerOrder {
		return Startup{}, fmt.Errorf("choose a random or commissioner-set startup draft order")
	}
	if out.Pattern != leaguerules.Snake && out.Pattern != leaguerules.Linear {
		return Startup{}, fmt.Errorf("choose a snake or linear startup draft")
	}
	return out, nil
}

func startupRounds(r leaguerules.RuleSet) (int, error) {
	if v := r.Draft.StartupRounds; v.State == leaguerules.IsKnown {
		if v.Value < 1 {
			return 0, fmt.Errorf("set at least one startup draft round")
		}
		return v.Value, nil
	}
	rounds, ok := leaguerules.StartupRoundsFor(r)
	if !ok {
		return 0, fmt.Errorf("needs a ruling: startup draft rounds or the active roster maximum")
	}
	return rounds, nil
}

func startupSalary(r leaguerules.RuleSet) (domain.Money, error) {
	if v := r.Draft.StartupSalary; v.State == leaguerules.IsKnown {
		if v.Value < 0 {
			return 0, fmt.Errorf("a startup draft salary cannot be negative")
		}
		return domain.Money(v.Value), nil
	}
	lowest, ok := leaguerules.LowestMinimumSalary(r)
	if !ok {
		return 0, fmt.Errorf("needs a ruling: startup draft salary or minimum salaries")
	}
	return domain.Money(lowest), nil
}

// StartupFirstRound is the startup draft's first round: the commissioner's list, or the franchises
// shuffled by the seed recorded when the draft opened, so a replay draws the same order.
func StartupFirstRound(s Startup, franchises []string, seed uint64) ([]string, error) {
	if err := validateFranchises(franchises); err != nil {
		return nil, err
	}
	if s.Order == leaguerules.CommissionerOrder {
		if err := validateOrder(s.List, franchises); err != nil {
			return nil, fmt.Errorf("startup draft order: %w", err)
		}
		return slices.Clone(s.List), nil
	}
	out := slices.Clone(franchises)
	//nolint:gosec // a replay must draw the same order from the recorded seed; the seed itself is drawn unpredictably.
	rand.New(rand.NewPCG(seed, seed)).Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out, nil
}

// StartupRules turns the league's draft rules into the startup draft's: the pace, clock and timeouts stay
// the draft's own; the rounds, first round, pattern and contract are the startup draft's. Every slot of
// every round signs the same contract, and no franchise can hold an extra pick.
func StartupRules(r leaguerules.DraftRules, s Startup, first []string) leaguerules.DraftRules {
	out := leaguerules.DraftRules{Mode: r.Mode, Clock: r.Clock, OnExpire: r.OnExpire, Timeouts: r.Timeouts,
		Rounds: leaguerules.Known(s.Rounds), MaximumPicks: leaguerules.Known(s.Rounds),
		ForfeitExtra: leaguerules.Known(false), OrderSource: leaguerules.Known(leaguerules.WorstStandingsFirst),
		Order:        leaguerules.Known(map[int][]string{1: slices.Clone(first)}),
		LaterPattern: leaguerules.Known(s.Pattern), TieRule: r.TieRule}
	pricing, lengths, rfa := map[int][]leaguerules.SlotPrice{}, map[int]leaguerules.Value[int]{}, map[int]leaguerules.Value[bool]{}
	slots := leaguerules.Bounds{Min: leaguerules.Known(1), Max: leaguerules.Known(len(first))}
	for round := 1; round <= s.Rounds; round++ {
		pricing[round] = []leaguerules.SlotPrice{{Slots: slots, Salary: leaguerules.Known(leaguerules.Money(s.Terms.Salary))}}
		lengths[round], rfa[round] = leaguerules.Known(s.Terms.Years), leaguerules.Known(false)
	}
	out.Pricing, out.Lengths, out.RFAAfter = leaguerules.Known(pricing), leaguerules.Known(lengths), leaguerules.Known(rfa)
	return out
}

// StartupPicks are the picks a startup draft issues itself: one a round for every franchise, its own.
// They are never carried, traded or issued for seasons ahead.
func StartupPicks(franchises []string, rounds int) []OwnedPick {
	out := make([]OwnedPick, 0, len(franchises)*rounds)
	for round := 1; round <= rounds; round++ {
		for _, f := range franchises {
			out = append(out, OwnedPick{Round: round, Original: f, Owner: f})
		}
	}
	return out
}

// Available is a draft's pool as it stands: the candidates it admits that are on no roster and not yet
// drafted, in the order given.
func Available(pool Pool, season int, candidates []Candidate, players []PlayerContract, d Draft) []string {
	out := []string{}
	for _, c := range candidates {
		if admits(pool, season, c) != nil ||
			slices.ContainsFunc(players, func(p PlayerContract) bool { return p.ID == c.ID }) ||
			slices.ContainsFunc(d.Picks, func(p Result) bool { return p.Player == c.ID }) {
			continue
		}
		out = append(out, c.ID)
	}
	return out
}

// admits holds a candidate to the draft's pool. An O-Line card is assigned, never drafted.
func admits(pool Pool, season int, c Candidate) error {
	id, err := playerid.New(c.ID)
	valid := err == nil && id.String() == c.ID && season >= 1
	switch {
	case pool == RookieClass && (!valid || c.EntrySeason != season):
		return fmt.Errorf("choose a rookie from this season's player pool")
	case (pool == Unrostered || pool == UnrosteredVeterans) && !valid:
		return fmt.Errorf("choose a player from this draft's pool")
	case pool == UnrosteredVeterans && c.EntrySeason == season:
		return fmt.Errorf("the incoming rookie class is not in this startup draft's pool")
	case pool != RookieClass && pool != Unrostered && pool != UnrosteredVeterans:
		return fmt.Errorf("choose a player from this draft's pool")
	}
	if _, card := nativecheck.LineCardTeam(c.ID); card {
		return fmt.Errorf("an O-Line card is assigned, never drafted")
	}
	return nil
}
