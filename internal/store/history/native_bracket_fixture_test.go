package history

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// bracketFixture is a 32-franchise league in two conferences of four divisions, with a one-week regular
// season played. Franchise n scores n tackles a week unless the test names another quantity, so a higher
// number always wins a regular-season game and the standings are known without being restated.
type bracketFixture struct {
	s     *Store
	pools *db.Pools
	path  string
	bump  map[string]map[int]leaguerules.Decimal
	env   *lineupEnvFake
}

const bracketLeague = "native.test"

func franchiseID(n int) string { return fmt.Sprintf("franchise.%02d", n) }

// bracketSeedCommand is the held starting snapshot with eight divisions of four, four in each conference.
func bracketSeedCommand(t *testing.T) commissioner.RawCommand {
	t.Helper()
	raw := heldSeedCommand(t)
	snap := *raw.Args.Native
	snap.Franchises = slices.Clone(snap.Franchises)
	snap.Conferences = map[string]string{}
	for i := range snap.Franchises {
		division := i/4 + 1
		snap.Franchises[i].Division = fmt.Sprintf("division.%d", division)
		snap.Conferences[snap.Franchises[i].Division] = map[bool]string{true: "AFC", false: "NFC"}[division <= 4]
	}
	snap.Champions = []commissioner.NativeChampion{{Season: 2025, Franchise: franchiseID(5)}}
	raw.Args.Native = &snap
	return raw
}

func bracketTest(t *testing.T) *bracketFixture {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bracket.db")
	s, pools := statStore(t, path)
	runScoreCommand(t, s, createWith(commissioner.PresetLegacy, 0), "creator")
	runScoreCommand(t, s, bracketSeedCommand(t), "creator")
	runScoreCommand(t, s, ruleCommand("schedule.regularWeeks", "1", "now", "Choose the season length", 2), "creator")
	runScoreCommand(t, s, ruleCommand("lineup.invalidPenalty", `"none"`, "now", "Score one-player lineups as they are", 3), "creator")
	raw := commissioner.RawCommand{League: bracketLeague, Command: commissioner.ScheduleGenerate,
		Args: commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 3}}, Expected: 4, Request: "generate",
		Effective: "now", Visibility: "league"}
	runScoreCommand(t, s, raw, "creator")
	raw.Command, raw.Expected, raw.Request = commissioner.SchedulePublish, 5, "publish"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	runScoreCommand(t, s, raw, "creator")
	f := &bracketFixture{s: s, pools: pools, path: path, bump: map[string]map[int]leaguerules.Decimal{}}
	f.install()
	return f
}

// install gives the store the lineups, the held scores and, when the test names one, the NFL calendar.
func (f *bracketFixture) install() {
	f.s.SetNativeLineups(&resultLineups{})
	if f.env != nil {
		f.s.SetLineupEnvironment(f.env)
	}
	f.s.evidence = func(_ context.Context, scope source.RawNativeScope) (source.NativeEvidence, error) {
		franchise := "franchise." + strings.TrimPrefix(scope.Player, "10")
		n, _ := strconv.Atoi(strings.TrimPrefix(franchise, "franchise."))
		quantity := leaguerules.Decimal(n * 10000)
		if q, ok := f.bump[franchise][scope.Week]; ok {
			quantity = q
		}
		return source.NativeEvidence{GSIS: "held-" + scope.Player, Held: &source.HeldWeek{SHA256: "held-complete",
			FetchedAt: t0(), Week: &source.PlayerWeek{Position: "LB", Line: fantasyscore.StatLine{
				Quantities: map[leaguerules.StatID]leaguerules.Decimal{"tackle": quantity}}}}}, nil
	}
}

func (f *bracketFixture) reopen(t *testing.T) {
	t.Helper()
	if err := f.pools.Close(); err != nil {
		t.Fatal(err)
	}
	f.s, f.pools = statStore(t, f.path)
	f.install()
}

// kickoff holds the NFL calendar: the first playoff week's games start at the given time.
func (f *bracketFixture) kickoff(at time.Time) {
	f.env = &lineupEnvFake{locks: map[int]map[string]time.Time{14: {"KC": at, "BUF": at.Add(3 * time.Hour)}}}
	f.s.SetLineupEnvironment(f.env)
}

func (f *bracketFixture) bracket(t *testing.T) commissioner.NativeBracket {
	t.Helper()
	out, err := f.s.NativeBracket(t.Context(), bracketLeague, 2026)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// rev is the league's version, which is the number of changes recorded in it.
func (f *bracketFixture) rev(t *testing.T) int64 { return int64(len(scoreRecords(t, f.s))) }

// bracketCommand is a bracket change as a commissioner would send it.
func (f *bracketFixture) bracketCommand(t *testing.T, command string, change commissioner.RawBracketChange,
	reason string) commissioner.RawCommand {
	t.Helper()
	change.Season = 2026
	rev := f.rev(t)
	return commissioner.RawCommand{League: bracketLeague, Command: command, Args: commissioner.RawArgs{Bracket: &change},
		Reason: reason, Expected: rev, Request: fmt.Sprintf("bracket.%d", rev), Effective: "now", Visibility: "league"}
}

func (f *bracketFixture) run(t *testing.T, raw commissioner.RawCommand) commissioner.Receipt {
	t.Helper()
	return runScoreCommand(t, f.s, raw, "creator")
}

func (f *bracketFixture) do(t *testing.T, command string, change commissioner.RawBracketChange) commissioner.Receipt {
	t.Helper()
	return f.run(t, f.bracketCommand(t, command, change, "Set the playoffs"))
}

func (f *bracketFixture) undo(t *testing.T, entry string) {
	t.Helper()
	rev := f.rev(t)
	runScoreCommand(t, f.s, reverseCommand(entry, rev, fmt.Sprint("undo-", rev)), "creator")
}

// refused runs a command that must be turned away and returns the sentence it gives, checking nothing was recorded.
func (f *bracketFixture) refused(t *testing.T, raw commissioner.RawCommand, principal string) string {
	t.Helper()
	before := f.rev(t)
	_, err := f.s.ExecuteCommissioner(t.Context(), raw, principal, nil)
	if err == nil {
		t.Fatalf("%s was accepted: %+v", raw.Command, raw.Args)
	}
	if f.rev(t) != before {
		t.Fatalf("a refused %s was recorded", raw.Command)
	}
	return err.Error()
}

// grantCoCommissioner gives a principal the co-commissioner role through the real command.
func (f *bracketFixture) grantCoCommissioner(t *testing.T, principal string) {
	t.Helper()
	executePeople(t, f.s, "creator", "commish.role.grant",
		commissioner.RawPeople{Principal: principal, Role: governance.CoCommissioner}, f.rev(t))
}

func gameIn(b commissioner.NativeBracket, id string) commissioner.BracketGame {
	for _, r := range b.Rounds {
		for _, g := range r.Games {
			if g.ID == id {
				return g
			}
		}
	}
	return commissioner.BracketGame{}
}

func seedsOf(b commissioner.NativeBracket, conference string) []commissioner.FieldSeed {
	for _, f := range b.Fields {
		if f.Conference == conference {
			return f.Seeds
		}
	}
	return nil
}
