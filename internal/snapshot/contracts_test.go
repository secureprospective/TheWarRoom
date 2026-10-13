package snapshot

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestContractRowsAndYearInference(t *testing.T) {
	id, err := playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	m := state.MirrorSnapshot{Season: 2026, Players: []state.MirrorPlayer{{MFLID: "0042", FranchiseID: "0001", Salary: 123456789, ContractYear: 2028, ContractInfo: "MFL text"}, {MFLID: "9999", FranchiseID: "0002", Salary: 45}}}
	directory := Snapshot{Players: Sourced[[]Player]{Value: []Player{{ID: id, Name: "A", Position: domain.PosQB}}}}
	rows := ContractRows(m, directory, map[string]string{"0001": "Mine"})
	if rows[0].Salary.Cents != 123456789 || rows[0].Salary.Text != "$1,234,567.89" || rows[0].Name != "A" || rows[0].Info != "MFL text" {
		t.Fatalf("row: %+v", rows[0])
	}
	if rows[1].Name != "Player 9999" || rows[1].Team != "Franchise 0002" {
		t.Fatalf("fallback: %+v", rows[1])
	}
	years := ContractYears(rows[0], 2026)
	if len(years) != 3 || years[0].Inferred || !years[1].Inferred || !years[2].Inferred || years[1].Reason == "" || years[2].Cost.Cents != 123456789 {
		t.Fatalf("years: %+v", years)
	}
	if len(ContractYears(rows[1], 2026)) != 1 {
		t.Fatal("unknown expiration must not invent future years")
	}
}

func TestContractsCentsCapAndAdjustments(t *testing.T) {
	rows := []ContractRow{{FranchiseID: "0001", Salary: ContractAmountOf(101), Expiration: 2028}, {FranchiseID: "0001", Salary: ContractAmountOf(202), Expiration: 2027}, {FranchiseID: "0002", Salary: ContractAmountOf(999), Expiration: 2028}}
	capAmount := domain.Money(1000)
	got := ContractsCap(rows, "0001", 2026, 1001, &capAmount, map[string]string{"0001": "Mine"})
	if got.Used.Cents != 1001 || got.Room.Cents != -1 || got.Room.Text != "-$0.01" || len(got.Future) != 2 || got.Future[0].Cost.Cents != 303 || got.Future[1].Cost.Cents != 101 || !got.Future[0].Inferred {
		t.Fatalf("cap: %+v", got)
	}
	absent := ContractsCap(rows, "0001", 2026, 1001, nil, nil)
	if absent.Cap != nil || absent.Room != nil {
		t.Fatal("missing cap must not become zero or guessed room")
	}
	adjs := ContractsAdjustments([]state.Adjustment{{ID: "7", FranchiseID: "0001", Amount: -123, Description: "BUYOUT: literal"}}, nil, "")
	if len(adjs) != 1 || adjs[0].Amount.Cents != -123 || adjs[0].Amount.Text != "-$1.23" || adjs[0].Description != "BUYOUT: literal" {
		t.Fatalf("adjustments: %+v", adjs)
	}
	if ContractsAdjustments(nil, nil, "") == nil {
		t.Fatal("empty adjustments must be array")
	}
}

func TestContractsSortFilterMineFirst(t *testing.T) {
	rows := []ContractRow{{ID: "3", FranchiseID: "0002", Name: "A", Position: "QB", Salary: ContractAmountOf(300)}, {ID: "2", FranchiseID: "0001", Name: "Z", Position: "WR", Salary: ContractAmountOf(200)}, {ID: "1", FranchiseID: "0001", Name: "B", Position: "QB", Salary: ContractAmountOf(100)}}
	scope := RawContractScope{Mine: "0001", Sort: "salary", Desc: true}
	got := ContractsTable(rows, nil, scope)
	if got.Rows[0].ID != "2" || got.Rows[1].ID != "1" || got.Rows[2].ID != "3" || len(got.Teams) != 2 || len(got.Positions) != 2 {
		t.Fatalf("sort: %+v", got)
	}
	scope.Team, scope.Position = "0001", "QB"
	got = ContractsTable(rows, nil, scope)
	if len(got.Rows) != 1 || got.Rows[0].ID != "1" {
		t.Fatalf("filter: %+v", got)
	}
	scope.Position = "TE"
	if len(ContractsTable(rows, nil, scope).Rows) != 0 {
		t.Fatal("empty filter")
	}
}

func TestContractScopeBoundary(t *testing.T) {
	for _, r := range []RawContractScope{{Sort: "bogus"}, {Sort: "name", Mine: "1"}, {Sort: "name", Team: "abcd"}, {Sort: "name", Position: "unknown"}} {
		if r.Validate() == nil {
			t.Fatalf("accepted %+v", r)
		}
	}
	if err := (RawContractScope{Mine: "0001", Sort: "expiration", Position: "WR"}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestContractAmountDisplay(t *testing.T) {
	for _, tc := range []struct {
		cents domain.Money
		text  string
	}{
		{12500000000, "$125,000,000"}, {67000000, "$670,000"}, {0, "$0"},
		{-67000000, "-$670,000"}, {12345, "$123.45"}, {-1, "-$0.01"}, {101, "$1.01"},
	} {
		got := ContractAmountOf(tc.cents)
		if got.Cents != tc.cents || got.Text != tc.text {
			t.Fatalf("amount %d: %+v", tc.cents, got)
		}
	}
	if domain.Money(100).String() != "$1.00" {
		t.Fatal("domain money formatting changed")
	}
}

func TestContractYearTagsAndReasons(t *testing.T) {
	years := ContractYears(ContractRow{Salary: ContractAmountOf(67000000), Expiration: 2029}, 2026)
	for i, year := range years {
		tag := "Inferred"
		if i == 0 {
			tag = "Observed"
		}
		if year.Tag != tag || strings.Contains(year.Reason, tag) || year.Reason == "" {
			t.Fatalf("year tag and reason: %+v", year)
		}
	}
	summary := ContractsCap([]ContractRow{{FranchiseID: "0001", Salary: ContractAmountOf(67000000), Expiration: 2029}}, "0001", 2026, 0, nil, nil)
	for _, year := range summary.Future {
		if year.Tag != "Inferred" || strings.Contains(year.Reason, year.Tag) || year.Cost.Text != "$670,000" {
			t.Fatalf("future: %+v", year)
		}
	}
}

func TestSalaryAdjustmentsMineFirstStable(t *testing.T) {
	rows := ContractsAdjustments([]state.Adjustment{
		{ID: "other", FranchiseID: "0002"}, {ID: "mine-a", FranchiseID: "0001"},
		{ID: "mine-b", FranchiseID: "0001"},
	}, nil, "0001")
	if rows[0].ID != "mine-a" || rows[1].ID != "mine-b" || rows[2].ID != "other" {
		t.Fatal(rows)
	}
}
