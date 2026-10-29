package leaguemigration_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	ingestleague "github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func leagues(t *testing.T) map[string]mflsource.Export {
	t.Helper()
	out := map[string]mflsource.Export{}
	legacy, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	out["Legacy NFL 14432"] = legacy
	for name, spec := range migrationtest.Specs() {
		out[name] = migrationtest.Synthetic(name, spec)
	}
	return out
}

func TestEveryFixtureLeaguePreparesWhole(t *testing.T) {
	t.Parallel()
	for name, export := range leagues(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			plan, err := mflsource.Prepare(export)
			if err != nil {
				problems, _ := leaguemigration.AsProblems(err)
				t.Fatalf("%v: %q", err, problems)
			}
			if plan.Create.Preset != commissioner.PresetMFLLeague || plan.Create.Source != commissioner.SourceMFLImport {
				t.Fatalf("a migrated league fell back to a catalog preset: %+v", plan.Create)
			}
			if len(plan.Seed.Franchises) != len(export.Config.Franchises) || len(plan.Seed.Players) != len(export.Rosters) ||
				len(plan.Seed.Picks) != len(export.Draft)+len(export.Future) || len(plan.Seed.Adjustments) != len(export.Adjustments) {
				t.Fatalf("assets lost: %d/%d franchises, %d/%d players, %d picks", len(plan.Seed.Franchises),
					len(export.Config.Franchises), len(plan.Seed.Players), len(export.Rosters), len(plan.Seed.Picks))
			}
			if plan.Rules.Scoring.Rules.State != leaguerules.IsKnown || len(plan.Rules.Scoring.Rules.Value) == 0 {
				t.Fatal("the league's MFL scoring was not carried through FromMFLScoring")
			}
		})
	}
}

func TestLegacySettingsAreItsOwnNotThePreset(t *testing.T) {
	t.Parallel()
	legacy, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	plan, err := mflsource.Prepare(legacy)
	if err != nil {
		t.Fatal(err)
	}
	r := plan.Rules
	for name, ok := range map[string]bool{
		"cap $125M":       r.Cap.Amount.Value == 12500000000,
		"IR 12":           r.IR.Slots.Value == 12,
		"taxi 8":          r.Taxi.Slots.Value == 8,
		"roster 80":       r.Roster.Limits.Value.Max.Value == 80,
		"13 weeks":        r.Schedule.RegularWeeks.Value == 13,
		"21 starters":     r.Lineup.Groups.Value[leaguerules.Total].Max.Value == 21,
		"7 tie breakers":  len(r.Standings.Criteria.Value) == 7,
		"cited as MFL":    r.Cap.Amount.Citation == "Your MFL league",
		"no preset value": r.Contracts.MaximumYears.State == leaguerules.NeedsRuling,
	} {
		if !ok {
			t.Errorf("%s: not carried as MFL holds it", name)
		}
	}
}

func TestEdgeLeagueSettings(t *testing.T) {
	t.Parallel()
	plan := func(name string) leaguemigration.Plan {
		t.Helper()
		p, err := mflsource.Prepare(migrationtest.Synthetic(name, migrationtest.Specs()[name]))
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	superflex := plan("14 teams superflex").Rules.Lineup.Positions.Value[leaguerules.QB]
	if superflex.Min.Value != 1 || superflex.Max.Value != 2 {
		t.Errorf("superflex QB: %+v", superflex)
	}
	capOff := plan("20 teams cap off")
	if capOff.Rules.Cap.Amount.State != leaguerules.NeedsRuling || !contains(capOff.Notes, "runs without salaries") {
		t.Errorf("a league without salaries must leave the cap to a ruling and say so: %+v %q", capOff.Rules.Cap.Amount, capOff.Notes)
	}
	reserves := plan("taxi and IR off").Rules
	if reserves.Taxi.Slots != (leaguerules.Value[int]{State: leaguerules.IsKnown, Citation: "Your MFL league"}) ||
		reserves.IR.Slots.Value != 0 || reserves.IR.Slots.State != leaguerules.IsKnown {
		t.Errorf("taxi and IR off must be a known zero: %+v %+v", reserves.Taxi.Slots, reserves.IR.Slots)
	}
	odd := plan("kickers off odd scoring")
	if _, ok := odd.Rules.Lineup.Positions.Value[leaguerules.PK]; ok {
		t.Error("kickers off still starts a kicker")
	}
	if !contains(odd.Notes, `"ZZQ"`) {
		t.Errorf("an unknown scoring event must be listed: %q", odd.Notes)
	}
	if _, ok := odd.Rules.Scoring.Rules.Value["mfl_ZZQ"]; !ok {
		t.Error("an unknown scoring event was dropped instead of kept as MFL holds it")
	}
	if got := plan("12 teams no divisions").Seed.Conferences; got != nil {
		t.Errorf("no divisions grew conferences: %v", got)
	}
}

func contains(list []string, part string) bool {
	for _, s := range list {
		if strings.Contains(s, part) {
			return true
		}
	}
	return false
}

func TestFailurePathsEndInAPlainList(t *testing.T) {
	t.Parallel()
	base := func() mflsource.Export {
		return migrationtest.Synthetic("Base", migrationtest.Specs()["8 teams"])
	}
	tests := map[string]struct {
		mutate func(*mflsource.Export)
		want   string
	}{
		"duplicate player": {func(e *mflsource.Export) {
			dup := e.Rosters[0]
			dup.Franchise = "0002"
			e.Rosters = append(e.Rosters, dup)
		}, "is listed twice"},
		"unknown player": {func(e *mflsource.Export) { delete(e.Players, e.Rosters[0].Player) },
			"not in MFL's player list"},
		"leading zeros dropped": {func(e *mflsource.Export) {
			e.Players["100"] = "No zeros"
			e.Rosters[0].Player = "100"
		}, `"100"`},
		"partial export: no franchises": {func(e *mflsource.Export) { e.Config.Franchises = nil },
			"a league needs between 2 and 32"},
		"partial export: roster of an unlisted franchise": {func(e *mflsource.Export) {
			e.Rosters[0].Franchise = "0099"
		}, `names franchise "0099"`},
		"partial export: no rosters": {func(e *mflsource.Export) { e.Rosters = nil }, "no rostered players"},
		"unreadable salary":          {func(e *mflsource.Export) { e.Rosters[0].Salary = "lots" }, "not an amount"},
		"unknown roster status": {func(e *mflsource.Export) { e.Rosters[0].Status = "BENCH" },
			"roster status"},
		"unreadable scoring": {func(e *mflsource.Export) {
			e.Config.ScoringRules[0].Rules[0].Points = "*lots"
		}, "scoring could not be read"},
		"unknown position": {func(e *mflsource.Export) {
			e.Config.Starters.Positions = append(e.Config.Starters.Positions, league("Def"))
		}, `"Def"`},
		"nameless franchise": {func(e *mflsource.Export) { e.Config.Franchises[2].Name = "" }, "has no name"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			export := base()
			tc.mutate(&export)
			plan, err := mflsource.Prepare(export)
			problems, ok := leaguemigration.AsProblems(err)
			if !ok || len(plan.Seed.Franchises) != 0 || len(plan.Create.Seed) != 0 {
				t.Fatalf("a broken league produced a plan: %v", err)
			}
			if !contains(problems, tc.want) {
				t.Fatalf("problems %q do not say %q", problems, tc.want)
			}
		})
	}
}

func TestReadStopsAtTheFirstUnreachablePart(t *testing.T) {
	t.Parallel()
	for _, part := range []string{"league", "rosters", "adjustments", "draft", "players"} {
		t.Run(part, func(t *testing.T) {
			t.Parallel()
			src := &migrationtest.Source{Export: migrationtest.Synthetic("x", migrationtest.Specs()["8 teams"]), FailAt: part}
			export, err := mflsource.Read(t.Context(), src, migrationtest.ReadAt)
			var read *leaguemigration.ReadError
			if !errors.As(err, &read) || !errors.Is(err, migrationtest.ErrUnreachable) || export.Name != "" {
				t.Fatalf("a failed read must end with nothing held: %v %+v", err, export.Name)
			}
			if !strings.Contains(err.Error(), "Nothing was imported") {
				t.Fatalf("not plain words: %v", err)
			}
		})
	}
}

func league(name string) ingestleague.PositionLimit {
	return ingestleague.PositionLimit{Name: name, Limit: "1"}
}
