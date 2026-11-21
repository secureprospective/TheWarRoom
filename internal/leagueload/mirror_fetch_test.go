package leagueload

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
)

func TestMirrorFetchHelpers(t *testing.T) {
	t.Parallel()
	if err := checkMirrorFetchRequest(MirrorFetchRequest{}); err == nil {
		t.Fatal("expected error when store is nil")
	}
	s := store(t, "mirror-check.db")
	if err := checkMirrorFetchRequest(MirrorFetchRequest{Store: s}); err == nil {
		t.Fatal("expected error when MFL league is empty")
	}
	if err := checkMirrorFetchRequest(MirrorFetchRequest{
		Store: s, MFLLeague: "14432", Year: 1800,
	}); err == nil {
		t.Fatal("expected error for invalid year")
	}
	if got := collectNFLScheduleWeeks(leaguemigration.Plan{}); len(got) != 1 || got[0] != 1 {
		t.Fatalf("empty schedule weeks = %v; want [1]", got)
	}
	plan := leaguemigration.Plan{
		Schedule: []commissioner.ScheduleWeek{{Week: 3}, {Week: 1}},
		Recorded: &commissioner.RawRecordedResults{
			CutoverWeek: 3,
			Teams: []commissioner.RecordedTeamResult{
				{Week: 1, Franchise: "franchise.01"},
				{Week: 1, Franchise: "franchise.02"},
				{Week: 2, Franchise: "franchise.01"},
			},
		},
		Lineups: []commissioner.RawLineupChange{{Week: 4}},
		Seed: commissioner.RawNativeSnapshot{
			Players: []commissioner.NativePlayer{
				{ID: "0001", ContractMissing: false},
				{ID: "0002", ContractMissing: true},
				{ID: "0003", ContractMissing: false},
			},
		},
	}
	weeks := collectNFLScheduleWeeks(plan)
	if len(weeks) != 4 || weeks[0] != 1 || weeks[3] != 4 {
		t.Fatalf("collected weeks = %v; want [1 2 3 4]", weeks)
	}
	if got := countPlanContracts(plan); got != 2 {
		t.Fatalf("contracts = %d; want 2", got)
	}
	if got := countPlanScoreWeeks(plan); got != 2 {
		t.Fatalf("score weeks = %d; want 2", got)
	}
}
