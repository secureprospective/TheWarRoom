package history

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

type autoLineupEnv struct {
	facts nativecheck.Facts
	weeks []leagueweek.Week
	zone  *time.Location
}

func (e *autoLineupEnv) PlayerFacts() nativecheck.Facts { return e.facts }

func (e *autoLineupEnv) ScheduleDates(_ context.Context, _ int) ([]leagueweek.Week, error) {
	return e.weeks, nil
}

func (e *autoLineupEnv) LineupFacts(_ context.Context, _, week int) (commissioner.LineupFacts, error) {
	for _, w := range e.weeks {
		if w.Number == week {
			return commissioner.LineupFacts{
				Facts: e.facts, Locks: leagueweek.Locks(w), ScheduleHeld: true, Zone: e.zone,
			}, nil
		}
	}
	return commissioner.LineupFacts{}, fmt.Errorf("week %d not held", week)
}

func autoLineupFixture(t *testing.T) (*lineupFixture, *autoLineupEnv) {
	t.Helper()
	f := lineupTest(t)
	env := &autoLineupEnv{
		facts: f.env.facts,
		zone:  time.UTC,
		weeks: []leagueweek.Week{{
			Number: 1,
			Games: []leagueweek.Game{
				{Teams: [2]string{"DAL", "PHI"}, Kickoff: f.dallas},
				{Teams: [2]string{"NYG", "WAS"}, Kickoff: f.nyGiants},
			},
		}},
	}
	f.s.SetLineupEnvironment(env)
	return f, env
}

func TestProcessDueLineupsLeavesHumanUntouchedAndSetsForgetterAndVacant(t *testing.T) {
	f, _ := autoLineupFixture(t)
	humanStarters := f.legal(f.a)
	if _, err := f.set(t, f.a, humanStarters); err != nil {
		t.Fatalf("human lineup.set: %v", err)
	}
	// f.a has a human lineup; f.b is seated (owner-b) but forgot to set a lineup.
	f.at = f.dallas.Add(-time.Hour)
	if err := f.s.ProcessDueLineups(t.Context(), "native.test", f.at); err != nil {
		t.Fatalf("ProcessDueLineups: %v", err)
	}
	records := scoreRecords(t, f.s)
	gotA, foundA, err := commissioner.RecordedLineupFor(records, f.a, 2026, 1)
	if err != nil || !foundA || gotA.By != f.ownerOfA || !slices.Equal(gotA.Starters, humanStarters) {
		t.Fatalf("human lineup changed: found=%v by=%q starters=%v err=%v", foundA, gotA.By, gotA.Starters, err)
	}
	gotB, foundB, err := commissioner.RecordedLineupFor(records, f.b, 2026, 1)
	if err != nil || !foundB || gotB.By != commissioner.LineupClockPrincipal {
		t.Fatalf("forgetter did not get automatic lineup: found=%v by=%q err=%v", foundB, gotB.By, err)
	}
	if len(gotB.Starters) != 21 {
		t.Fatalf("forgetter starters = %d, want 21", len(gotB.Starters))
	}
	// Second pass at the same wave is idempotent.
	before := len(records)
	if err := f.s.ProcessDueLineups(t.Context(), "native.test", f.at); err != nil {
		t.Fatalf("second ProcessDueLineups: %v", err)
	}
	if after := len(scoreRecords(t, f.s)); after != before {
		t.Fatalf("idempotent pass appended %d events", after-before)
	}
}

func TestProcessDueLineupsSecondWaveKeepsLockedAndReplacesUnlocked(t *testing.T) {
	f, env := autoLineupFixture(t)
	// Wave 1 (DAL kickoff - 60 min): f.b gets an automatic lineup with DAL offense + NYG defense/PK.
	f.at = f.dallas.Add(-time.Hour)
	if err := f.s.ProcessDueLineups(t.Context(), "native.test", f.at); err != nil {
		t.Fatalf("wave 1: %v", err)
	}
	wave1, found, err := commissioner.RecordedLineupFor(scoreRecords(t, f.s), f.b, 2026, 1)
	if err != nil || !found {
		t.Fatalf("wave 1 lineup missing: found=%v err=%v", found, err)
	}
	// Pick one DAL starter (will be locked at wave 2) and one NYG starter (unlocked at wave 2).
	var lockedDAL, unlockedNYG string
	for _, id := range wave1.Starters {
		pf := env.facts[id]
		if pf.Team == "DAL" && lockedDAL == "" {
			lockedDAL = id
		}
		if pf.Team == "NYG" && pf.Position == leaguerules.LB && unlockedNYG == "" {
			unlockedNYG = id
		}
	}
	if lockedDAL == "" || unlockedNYG == "" {
		t.Fatalf("expected DAL and NYG LB starters in wave 1: dal=%q nyg=%q", lockedDAL, unlockedNYG)
	}
	// Mark BOTH Out before wave 2: the locked DAL player MUST stay; the unlocked NYG player MUST be replaced.
	dalFact := env.facts[lockedDAL]
	dalFact.Designation, dalFact.DesignationHeld = leaguerules.Out, true
	env.facts[lockedDAL] = dalFact
	nygFact := env.facts[unlockedNYG]
	nygFact.Designation, nygFact.DesignationHeld = leaguerules.Out, true
	env.facts[unlockedNYG] = nygFact

	f.at = f.nyGiants.Add(-time.Hour)
	if err := f.s.ProcessDueLineups(t.Context(), "native.test", f.at); err != nil {
		t.Fatalf("wave 2: %v", err)
	}
	wave2, found, err := commissioner.RecordedLineupFor(scoreRecords(t, f.s), f.b, 2026, 1)
	if err != nil || !found {
		t.Fatalf("wave 2 lineup missing: found=%v err=%v", found, err)
	}
	if !slices.Contains(wave2.Starters, lockedDAL) {
		t.Fatalf("locked DAL starter %s was removed in wave 2", lockedDAL)
	}
	if slices.Contains(wave2.Starters, unlockedNYG) {
		t.Fatalf("unlocked Out NYG starter %s was not replaced in wave 2", unlockedNYG)
	}
}

func TestClockLineupAuthorizerRefusesHumanLineup(t *testing.T) {
	f, env := autoLineupFixture(t)
	if _, err := f.set(t, f.a, f.legal(f.a)); err != nil {
		t.Fatalf("human lineup.set: %v", err)
	}
	records := scoreRecords(t, f.s)
	raw := commissioner.RawCommand{
		League:  "native.test",
		Command: commissioner.LineupSet,
		Args: commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{
			Season: 2026, Franchise: f.a, Week: 1, Starters: f.legal(f.a),
		}},
		Expected:   int64(len(records)),
		Request:    "lineup-clock.refuse-human",
		Effective:  "now",
		Visibility: "private",
	}
	_, err := f.s.ExecuteCommissionerWithFacts(t.Context(), raw, commissioner.LineupClockPrincipal,
		clockLineupAuthorizer{}, env.facts, time.UTC)
	if err == nil || !strings.Contains(err.Error(), "already has a human lineup") {
		t.Fatalf("expected human-lineup refusal, got: %v", err)
	}
}
