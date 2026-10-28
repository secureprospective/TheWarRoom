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
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func startupFranchises(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%04d", i+1)
	}
	return out
}

// startupDraft is a 12-team, three-round startup draft with the commissioner's order, every pick made in
// turn from a pool of 50 players.
func startupDraft(t *testing.T, pattern leaguerules.OrderPattern) (leaguerules.RuleSet, Startup, []RoundOrder, Draft, []Candidate) {
	t.Helper()
	r := leaguerules.Default32()
	franchises := startupFranchises(12)
	r.Draft.StartupRounds = leaguerules.Known(3)
	r.Draft.StartupOrder = leaguerules.Known(leaguerules.CommissionerOrder)
	r.Draft.StartupList = leaguerules.Known(franchises)
	r.Draft.StartupPattern = leaguerules.Known(pattern)
	s, err := ResolveStartup(r)
	if err != nil {
		t.Fatal(err)
	}
	first, err := StartupFirstRound(s, franchises, 0)
	if err != nil {
		t.Fatal(err)
	}
	rules := StartupRules(r.Draft, s, first)
	order, err := Order(rules, franchises, StartupPicks(franchises, s.Rounds), nil)
	if err != nil {
		t.Fatal(err)
	}
	pool := []Candidate{}
	for i := range 50 {
		pool = append(pool, Candidate{ID: fmt.Sprintf("%05d", 20000+i), EntrySeason: 2020 + i%7})
	}
	acts, at := []Act{}, fixedStart()
	for i, round := range order {
		for _, pick := range round.Picks {
			at = at.Add(time.Minute)
			acts = append(acts, Act{Kind: Select, At: at, Owner: pick.Owner, Player: pool[len(acts)].ID,
				Pick: PickRef{Round: i + 1, Original: pick.Original}})
		}
	}
	d, err := ReplayActs(rules, order, fixedStart(), at, acts)
	if err != nil {
		t.Fatal(err)
	}
	return r, s, order, d, pool
}

func TestStartupDraftRunsInSnakeOrderAndSignsEveryPick(t *testing.T) {
	franchises := startupFranchises(12)
	reversed := slices.Clone(franchises)
	slices.Reverse(reversed)
	for _, row := range []struct {
		pattern leaguerules.OrderPattern
		rounds  [][]string
	}{
		{leaguerules.Snake, [][]string{franchises, reversed, franchises}},
		{leaguerules.Linear, [][]string{franchises, franchises, franchises}},
	} {
		_, s, order, d, pool := startupDraft(t, row.pattern)
		for i, round := range order {
			got := []string{}
			for _, pick := range round.Picks {
				got = append(got, pick.Owner)
			}
			if !slices.Equal(got, row.rounds[i]) {
				t.Fatalf("%s round %d order %v, want %v", row.pattern, i+1, got, row.rounds[i])
			}
		}
		want := Terms{Salary: 33000000, Years: 1}
		players := []PlayerContract{}
		for _, pick := range d.Picks {
			if pick.Player == "" || pick.Contract == nil || *pick.Contract != want || pick.How != HowOwner {
				t.Fatalf("pick not made with the startup contract: %+v", pick)
			}
			players = append(players, PlayerContract{ID: pick.Player, Franchise: pick.Owner,
				Salary: pick.Contract.Salary, Status: domain.RosterActive})
		}
		if !d.Exhausted || len(d.Picks) != 36 || s.Terms != want {
			t.Fatalf("draft not complete: %d picks, exhausted %v", len(d.Picks), d.Exhausted)
		}
		left := Available(Unrostered, 2031, pool, players, d)
		if len(left) != 14 || slices.ContainsFunc(left, func(id string) bool {
			return slices.ContainsFunc(d.Picks, func(p Result) bool { return p.Player == id })
		}) {
			t.Fatalf("drafted players stayed in the pool: %v", left)
		}
	}
}

func TestStartupPoolTakesAnyUnrosteredPlayerButNeverAnOLineCard(t *testing.T) {
	r := leaguerules.Default32()
	r.Draft.StartupRounds = leaguerules.Known(3)
	franchises := startupFranchises(12)
	s, err := ResolveStartup(r)
	if err != nil {
		t.Fatal(err)
	}
	first, err := StartupFirstRound(s, franchises, 7)
	if err != nil {
		t.Fatal(err)
	}
	rules := StartupRules(r.Draft, s, first)
	order, err := Order(rules, franchises, StartupPicks(franchises, s.Rounds), nil)
	if err != nil {
		t.Fatal(err)
	}
	d, err := ReplayActs(rules, order, fixedStart(), fixedStart(), nil)
	if err != nil {
		t.Fatal(err)
	}
	owner := d.OnClock.Owner
	card, _ := nativecheck.LineCardID(nativecheck.LineCardTeams()[0])
	veteran := Candidate{ID: "14001", EntrySeason: 2019}
	rostered := Candidate{ID: "14002", EntrySeason: 2025}
	basis := SelectionBasis{Pool: Unrostered, Season: 2031, Local: time.UTC,
		Players: []PlayerContract{{ID: rostered.ID, Franchise: "0002", Salary: 1, Status: domain.RosterActive}}}
	for _, row := range []struct {
		candidate Candidate
		words     string
	}{
		{veteran, ""},
		{Candidate{ID: card}, "O-Line"},
		{rostered, "already belongs"},
		{Candidate{ID: "x"}, "pool"},
	} {
		_, err := PreviewSelection(r, d, owner, row.candidate, basis)
		if row.words == "" && err != nil || row.words != "" && (err == nil || !strings.Contains(err.Error(), row.words)) {
			t.Fatalf("%+v: %v", row.candidate, err)
		}
	}
	rookie := basis
	rookie.Pool = RookieClass
	if _, err := PreviewSelection(r, d, owner, veteran, rookie); err == nil ||
		!strings.Contains(err.Error(), "rookie from this season") {
		t.Fatalf("the rookie draft took a veteran: %v", err)
	}
	if left := Available(Unrostered, 2031, []Candidate{veteran, {ID: card}, rostered}, basis.Players, d); !slices.Equal(left, []string{veteran.ID}) {
		t.Fatalf("pool %v", left)
	}
}

func TestStartupSettingsDefaultsAndRandomOrder(t *testing.T) {
	franchises := startupFranchises(12)
	unset := leaguerules.Default32()
	unset.Draft = leaguerules.DraftRules{Mode: unset.Draft.Mode, Clock: unset.Draft.Clock}
	for _, row := range []struct {
		name   string
		rules  leaguerules.RuleSet
		rounds int
	}{
		{"default32", leaguerules.Default32(), 43},
		{"stored before the settings", unset, 43},
		{"legacy", leaguerules.LegacyNFL(), 80},
	} {
		s, err := ResolveStartup(row.rules)
		if err != nil || s.Rounds != row.rounds || s.Pattern != leaguerules.Snake ||
			s.Order != leaguerules.RandomOrder || s.Terms != (Terms{Salary: 33000000, Years: 1}) {
			t.Fatalf("%s: %+v, %v", row.name, s, err)
		}
	}
	s, _ := ResolveStartup(leaguerules.Default32())
	a, _ := StartupFirstRound(s, franchises, 42)
	b, _ := StartupFirstRound(s, franchises, 42)
	c, _ := StartupFirstRound(s, franchises, 43)
	if !slices.Equal(a, b) || slices.Equal(a, c) || validateOrder(a, franchises) != nil {
		t.Fatalf("a recorded seed must draw one order: %v %v %v", a, b, c)
	}
	s.Order, s.List = leaguerules.CommissionerOrder, franchises[1:]
	if _, err := StartupFirstRound(s, franchises, 0); err == nil {
		t.Fatal("a commissioner order missing a franchise was accepted")
	}
}

// The startup rookies setting decides only whether the incoming rookie class is in the startup draft's
// pool: off by default and in every preset, and on puts the class in beside everyone else unrostered.
func TestStartupRookiesSettingDecidesOnlyThePool(t *testing.T) {
	veteran := Candidate{ID: "14001", EntrySeason: 2019}
	rookie := Candidate{ID: "14003", EntrySeason: 2031}
	undated := Candidate{ID: "14004"}
	for _, c := range []struct {
		name    string
		setting leaguerules.Value[bool]
		pool    Pool
		left    []string
	}{
		{"unset reads off", leaguerules.Value[bool]{}, UnrosteredVeterans, []string{veteran.ID, undated.ID}},
		{"off", leaguerules.Known(false), UnrosteredVeterans, []string{veteran.ID, undated.ID}},
		{"on", leaguerules.Known(true), Unrostered, []string{veteran.ID, rookie.ID, undated.ID}},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := leaguerules.Default32()
			r.Draft.StartupRookies = c.setting
			before := r.Draft
			s, err := ResolveStartup(r)
			if err != nil {
				t.Fatal(err)
			}
			if s.Pool != c.pool {
				t.Fatalf("pool %q, want %q", s.Pool, c.pool)
			}
			if left := Available(s.Pool, 2031, []Candidate{veteran, rookie, undated}, nil, Draft{}); !slices.Equal(left, c.left) {
				t.Fatalf("pool %v, want %v", left, c.left)
			}
			if !reflect.DeepEqual(r.Draft.Rounds, before.Rounds) || !reflect.DeepEqual(r.Draft.Pricing, before.Pricing) {
				t.Fatal("the startup rookies setting touched the rookie draft")
			}
		})
	}
	for name, preset := range map[string]leaguerules.RuleSet{"Legacy": leaguerules.LegacyNFL(),
		"32-Team Default": leaguerules.Default32(), "16-Team Mirror": leaguerules.Mirror16()} {
		if v := preset.Draft.StartupRookies; v.State != leaguerules.IsKnown || v.Value {
			t.Errorf("%s: startup rookies %+v, want off", name, v)
		}
	}
}
