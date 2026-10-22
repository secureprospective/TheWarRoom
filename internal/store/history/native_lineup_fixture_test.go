package history

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// lineupEnvFake holds the football facts a lineup is judged against: who each player is and when each
// NFL team's game starts in a week. A week with no entry is a week whose kickoff times are not held.
type lineupEnvFake struct {
	facts nativecheck.Facts
	locks map[int]map[string]time.Time
	zone  *time.Location
}

func (e *lineupEnvFake) PlayerFacts() nativecheck.Facts { return e.facts }

func (e *lineupEnvFake) LineupFacts(_ context.Context, _, week int) (commissioner.LineupFacts, error) {
	locks, held := e.locks[week]
	return commissioner.LineupFacts{Facts: e.facts, Locks: locks, ScheduleHeld: held, Zone: e.zone}, nil
}

// lineupSlots is a franchise's whole active roster: more than a lineup needs, at every position.
func lineupSlots() []struct {
	pos leaguerules.Position
	n   int
} {
	return []struct {
		pos leaguerules.Position
		n   int
	}{{leaguerules.QB, 2}, {leaguerules.RB, 3}, {leaguerules.WR, 4}, {leaguerules.TE, 2}, {leaguerules.PK, 1},
		{leaguerules.DT, 3}, {leaguerules.DE, 3}, {leaguerules.LB, 4}, {leaguerules.CB, 3}, {leaguerules.S, 4}}
}

// legalStarters is how many of each position start in a legal Legacy lineup: 21 in all.
func legalStarters() map[leaguerules.Position]int {
	return map[leaguerules.Position]int{leaguerules.QB: 1, leaguerules.RB: 2, leaguerules.WR: 4, leaguerules.TE: 1,
		leaguerules.PK: 1, leaguerules.DT: 2, leaguerules.DE: 2, leaguerules.LB: 3, leaguerules.CB: 2, leaguerules.S: 3}
}

type lineupFixture struct {
	*resultsFixture
	env       *lineupEnvFake
	at        time.Time
	points    map[string]leaguerules.Decimal
	roster    map[string]map[leaguerules.Position][]string
	requests  int
	dallas    time.Time
	nyGiants  time.Time
	ownerOfA  string
	ownerOfB  string
	cc        string
	creatorID string
}

func (f *lineupFixture) clock() time.Time { return f.at }

// lineupTest is a seeded Legacy league with a published schedule, real rosters for the two franchises of the
// first game, owners seated for both, and the league's own ledger as the only source of lineups.
func lineupTest(t *testing.T) *lineupFixture {
	t.Helper()
	path := filepath.Join(t.TempDir(), "lineups.db")
	s, pools := statStore(t, path)
	f := &lineupFixture{resultsFixture: &resultsFixture{s: s, pools: pools, path: path, lineups: &resultLineups{}},
		points: map[string]leaguerules.Decimal{}, roster: map[string]map[leaguerules.Position][]string{},
		dallas: time.Date(2026, 10, 18, 17, 0, 0, 0, time.UTC), nyGiants: time.Date(2026, 10, 19, 0, 20, 0, 0, time.UTC),
		ownerOfA: "owner-a", ownerOfB: "owner-b", cc: "deputy", creatorID: "creator"}
	f.at = time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	f.a, f.b = "franchise.01", "franchise.32"
	runScoreCommand(t, s, createWith(commissioner.PresetLegacy, 0), "creator")
	runScoreCommand(t, s, f.rosterSeed(t), "creator")
	runScoreCommand(t, s, ruleCommand("schedule.regularWeeks", "3", "now", "Choose the season length", 2), "creator")
	raw := commissioner.RawCommand{League: "native.test", Command: commissioner.ScheduleGenerate,
		Args: commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 3}}, Expected: 3, Request: "generate",
		Effective: "now", Visibility: "league"}
	runScoreCommand(t, s, raw, "creator")
	raw.Command, raw.Expected, raw.Request = commissioner.SchedulePublish, 4, "publish"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	runScoreCommand(t, s, raw, "creator")
	schedule, err := commissioner.ScheduleReading("native.test", scoreRecords(t, s))
	if err != nil {
		t.Fatal(err)
	}
	if game := schedule.Weeks[0].Games[0]; game.Home != f.a || game.Away != f.b {
		t.Fatalf("the first game is %s v %s", game.Home, game.Away)
	}
	f.install()
	f.seat(t, f.ownerOfA, f.a)
	f.seat(t, f.ownerOfB, f.b)
	return f
}

// rosterSeed is the held starting state with a full active roster for both franchises of the first game.
func (f *lineupFixture) rosterSeed(t *testing.T) commissioner.RawCommand {
	t.Helper()
	seed := heldSeedCommand(t)
	seed.Args.Native.Players = nil
	facts := nativecheck.Facts{}
	for _, franchise := range []string{f.a, f.b} {
		f.roster[franchise] = map[leaguerules.Position][]string{}
		number := strings.TrimPrefix(franchise, "franchise.")
		index := 0
		for _, slot := range lineupSlots() {
			for k := 1; k <= slot.n; k++ {
				index++
				id := fmt.Sprintf("1%s%02d", number, index)
				f.roster[franchise][slot.pos] = append(f.roster[franchise][slot.pos], id)
				facts[id] = nativecheck.PlayerFacts{Name: fmt.Sprintf("%s%d of team %s", slot.pos, k, number),
					Position: slot.pos, Team: lineupTeam(slot.pos, k)}
				seed.Args.Native.Players = append(seed.Args.Native.Players, commissioner.NativePlayer{ID: id,
					Franchise: franchise, Status: "ROSTER", Salary: 100000, ExpirationYear: 2028, ContractStatus: "UFA",
					ContractInfo: "Held contract"})
			}
		}
	}
	f.env = &lineupEnvFake{facts: facts, zone: time.FixedZone("PDT", -7*3600), locks: map[int]map[string]time.Time{
		1: {"DAL": f.dallas, "NYG": f.nyGiants}}}
	return seed
}

// lineupTeam puts offense on Dallas, everyone else on the Giants, the third running back on a bye and the
// second quarterback with no team.
func lineupTeam(pos leaguerules.Position, k int) string {
	switch {
	case pos == leaguerules.RB && k == 3:
		return "NOP"
	case pos == leaguerules.QB && k == 2:
		return "FA"
	case pos == leaguerules.QB || pos == leaguerules.RB || pos == leaguerules.WR || pos == leaguerules.TE:
		return "DAL"
	}
	return "NYG"
}

// install wires the store to this fixture's clock, evidence and facts after a start or a reopen.
func (f *lineupFixture) install() {
	f.s.SetClock(f.clock)
	f.s.SetLineupEnvironment(f.env)
	f.s.evidence = func(_ context.Context, scope source.RawNativeScope) (source.NativeEvidence, error) {
		quantity, held := f.points[scope.Player]
		if !held {
			quantity = 10000
		}
		position := ""
		if f.env != nil {
			position = string(f.env.facts[scope.Player].Position)
		}
		return source.NativeEvidence{GSIS: "held-" + scope.Player, Held: &source.HeldWeek{SHA256: "held-complete",
			FetchedAt: t0(), Week: &source.PlayerWeek{Position: position, Line: fantasyscore.StatLine{
				Quantities: map[leaguerules.StatID]leaguerules.Decimal{"reception": quantity}}}}}, nil
	}
}

func (f *lineupFixture) reopen(t *testing.T) {
	t.Helper()
	if err := f.pools.Close(); err != nil {
		t.Fatal(err)
	}
	f.s, f.pools = statStore(t, f.path)
	f.install()
}

// seat gives an owner the franchise's seat through the real invitation commands.
func (f *lineupFixture) seat(t *testing.T, owner, franchise string) {
	t.Helper()
	invite := "invite-" + owner
	executePeople(t, f.s, "creator", "commish.seat.invite",
		commissioner.RawPeople{ID: invite, Franchise: franchise, Principal: owner}, revision(t, f.s))
	executePeople(t, f.s, owner, "commish.seat.invite.redeem",
		commissioner.RawPeople{ID: invite, Principal: owner}, revision(t, f.s))
}

// lineup lists the first n players at each position of a franchise's roster, as that franchise's starters.
func (f *lineupFixture) lineup(franchise string, counts map[leaguerules.Position]int) []string {
	out := []string{}
	for _, slot := range lineupSlots() {
		out = append(out, f.roster[franchise][slot.pos][:counts[slot.pos]]...)
	}
	return out
}

func (f *lineupFixture) legal(franchise string) []string { return f.lineup(franchise, legalStarters()) }

// command builds a lineup command at the league's current version.
func (f *lineupFixture) command(t *testing.T, command, franchise string, week int, starters []string) commissioner.RawCommand {
	t.Helper()
	f.requests++
	raw := commissioner.RawCommand{League: "native.test", Command: command, Effective: "now", Visibility: "private",
		Args:     commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{Season: 2026, Franchise: franchise, Week: week, Starters: starters}},
		Expected: revision(t, f.s), Request: fmt.Sprintf("lineup.%d", f.requests)}
	if command == commissioner.LineupCorrect {
		raw.Visibility, raw.Reason = "league", "The owner's lineup was lost in the outage"
	}
	return raw
}

func (f *lineupFixture) owner(franchise string) string {
	if franchise == f.a {
		return f.ownerOfA
	}
	return f.ownerOfB
}

// set records an owner's lineup as that franchise's own owner.
func (f *lineupFixture) set(t *testing.T, franchise string, starters []string) (commissioner.Receipt, error) {
	t.Helper()
	return f.s.ExecuteCommissioner(t.Context(), f.command(t, commissioner.LineupSet, franchise, 1, starters), f.owner(franchise), nil)
}

func (f *lineupFixture) mustSet(t *testing.T, franchise string, starters []string) commissioner.Receipt {
	t.Helper()
	receipt, err := f.set(t, franchise, starters)
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func (f *lineupFixture) correct(t *testing.T, franchise string, starters []string) commissioner.RawCommand {
	t.Helper()
	return f.command(t, commissioner.LineupCorrect, franchise, 1, starters)
}

// refusal returns the sentence an owner reads for a refused change, or fails if it was not refused.
func refusal(t *testing.T, err error) string {
	t.Helper()
	var refused *commissioner.Refusal
	if !errors.As(err, &refused) {
		t.Fatalf("expected the league to turn this down, got %v", err)
	}
	return refused.Reason()
}

func (f *lineupFixture) results(t *testing.T) commissioner.NativeResults {
	t.Helper()
	out, err := f.s.NativeResults(t.Context(), "native.test", 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func (f *lineupFixture) reading(t *testing.T, franchise string) commissioner.NativeLineup {
	t.Helper()
	out, err := f.s.NativeLineup(t.Context(), "native.test", franchise, f.owner(franchise), 1, f.at)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func grantDeputy(t *testing.T, f *lineupFixture) {
	t.Helper()
	executePeople(t, f.s, "creator", "commish.role.grant",
		commissioner.RawPeople{Principal: f.cc, Role: governance.CoCommissioner}, revision(t, f.s))
}
