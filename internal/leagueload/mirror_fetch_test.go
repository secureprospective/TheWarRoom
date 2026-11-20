package leagueload

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
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
	if got := collectNFLScheduleWeeks(mflsource.Export{}); len(got) != 1 || got[0] != 1 {
		t.Fatalf("empty schedule weeks = %v; want [1]", got)
	}
	weeks := collectNFLScheduleWeeks(mflsource.Export{
		Schedule:    []leagueschedule.RawScheduleWeek{{Week: "3"}, {Week: "1"}},
		Results:     []leaguefeed.WeekResult{{Week: 2}, {Week: 1}},
		LiveLineups: &leaguefeed.Lineups{Week: 4},
	})
	if len(weeks) != 4 || weeks[0] != 1 || weeks[3] != 4 {
		t.Fatalf("collected weeks = %v; want [1 2 3 4]", weeks)
	}
	contracts := countRosterContracts([]mflsource.Rostered{
		{Player: "0001", Salary: "5.00", ContractYear: "2027", ContractStatus: "UFA"},
		{Player: "0002"},
		{Player: "0003", Salary: "1.25"},
	})
	if contracts != 2 {
		t.Fatalf("contracts = %d; want 2", contracts)
	}
	scored := countScoredWeeks([]leaguefeed.WeekResult{
		{Week: 1, Games: []leaguefeed.Game{{Home: leaguefeed.Side{Scored: true}}}},
		{Week: 2, Games: []leaguefeed.Game{{Home: leaguefeed.Side{Scored: false}}}},
		{Week: 3, Others: []leaguefeed.Side{{Scored: true}}},
	})
	if scored != 2 {
		t.Fatalf("scored weeks = %d; want 2", scored)
	}
}
