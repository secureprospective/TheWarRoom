package contracts

import (
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/contractacts"
	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// heldMirror is franchise 0001 holding 0042 at $10M through 2028 and 0046 expiring this season, 2026.
func heldMirror() state.MirrorSnapshot {
	return state.MirrorSnapshot{Season: 2026, Players: []state.MirrorPlayer{
		{MFLID: "0042", FranchiseID: "0001", Salary: 10 * contractacts.Million, ContractYear: 2028, RosterStatus: domain.RosterActive},
		{MFLID: "0046", FranchiseID: "0001", Salary: 10 * contractacts.Million, ContractYear: 2026, RosterStatus: domain.RosterActive, ContractStatus: domain.CStatusFT1},
		{MFLID: "0050", FranchiseID: "0002", Salary: 20 * contractacts.Million, ContractYear: 2027, RosterStatus: domain.RosterActive},
	}, Adjustments: []state.Adjustment{
		{ID: "a", FranchiseID: "0001", Amount: 300_000_000, Description: "Dead cap for 0042 release"},
		{ID: "b", FranchiseID: "0002", Amount: 100_000_000, Description: "Dead cap for 0050 release"},
	}}
}

func TestActValidateRefusesEachMalformedRequest(t *testing.T) {
	t.Parallel()
	for want, r := range map[string]Act{
		"contract Act scope":                      {Mine: "bad", Action: "read"},
		"franchise and known action required":     {Mine: "0001", Action: "delete"},
		"contract Act player":                     {Mine: "0001", Action: "options", Player: "x"},
		"known intent, tender level 1–6":          {Mine: "0001", Action: "draft", Player: "0042", Intent: "rfa.tender", Level: 7, Years: 1},
		"move must be $0–$3M":                     {Mine: "0001", Action: "preview", Player: "0042", Intent: "contract.restructure", Level: 1, Years: 1, Move: "4"},
		"plan id required":                        {Mine: "0001", Action: "rebase"},
		"contract Act player: playerid: \"y\" is": {Mine: "0001", Action: "preview", Player: "y", Intent: "rfa.tender", Level: 1, Years: 1},
	} {
		if err := r.Validate(); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%+v: %v, want %q", r, err, want)
		}
	}
	for _, r := range []Act{{Mine: "0001", Action: "read"}, {Mine: "0001", Action: "options", Player: "0042"},
		{Mine: "0001", Action: "draft", Player: "0042", Intent: "contract.restructure", Level: 1, Years: 1, Move: "3"},
		{Mine: "0001", Action: "withdraw", ID: "contract-1"}} {
		if err := r.Validate(); err != nil {
			t.Errorf("%+v refused: %v", r, err)
		}
	}
}

func TestEvidenceFlatFillsTheHeldContractThroughExpiration(t *testing.T) {
	t.Parallel()
	m := heldMirror()
	e := Evidence(m, "0001", "0042")
	if len(e.Cells) != 3 || e.Cells[2].Season != 2028 || e.Cells[0].Salary != 10*contractacts.Million || e.Observed || e.FranchiseID != "0001" {
		t.Fatalf("evidence %+v", e)
	}
	if e := Evidence(m, "0001", "0046"); len(e.Cells) != 1 || !e.Observed || e.Status != domain.CStatusFT1 {
		t.Fatalf("expiring evidence %+v", e)
	}
	if e := Evidence(m, "0002", "0042"); len(e.Cells) != 0 {
		t.Fatalf("another franchise's player gave evidence %+v", e)
	}
	m.Players = append(m.Players, state.MirrorPlayer{MFLID: "x", FranchiseID: "0001"})
	if e := Evidence(m, "0001", "x"); len(e.Cells) != 0 {
		t.Fatalf("a malformed id gave evidence %+v", e)
	}
}

func TestActFactsKeepTheReasonOrder(t *testing.T) {
	t.Parallel()
	m := heldMirror()
	rows := []snapshot.ContractRow{{ID: "0042", Name: "Example", Position: "QB"}}
	p, _ := Player(m, "0001", "0042")
	f, priced, err := ActFacts(m, p, Act{Mine: "0001", Player: "0042", Intent: "contract.extend", Years: 2}, "Window shut.", rows)
	if err != nil || !priced || f.Name != "Example" || f.Position != "QB" || f.Reasons[0] != "Window shut." || len(f.Terms.Cells) == 0 {
		t.Fatalf("extend %+v %v %v", f, priced, err)
	}
	f, priced, err = ActFacts(m, p, Act{Mine: "0001", Player: "0042", Intent: "rfa.tender", Level: 0, Years: 1}, "", nil)
	if err != nil || priced || f.Name != "0042" || !slices.Contains(f.Reasons, Ineligible) || !slices.ContainsFunc(f.Reasons, func(r string) bool { return strings.HasPrefix(r, "Terms: RFA price") }) {
		t.Fatalf("unpriced tender %+v %v %v", f, priced, err)
	}
	if _, _, err := ActFacts(m, p, Act{Mine: "0001", Player: "0042", Intent: "contract.restructure", Move: "x"}, "", rows); err == nil || !strings.HasPrefix(err.Error(), "contract Act move") {
		t.Fatalf("bad move %v", err)
	}
}

func TestEvidenceReasonsSayWhatWasNotChecked(t *testing.T) {
	t.Parallel()
	base := []contractacts.Cell{{Season: 2026, Salary: contractacts.Million}}
	for intent, want := range map[string]string{
		"rfa.tender":           "Priced from his 2026 salary, $1,000,000.",
		"fa.resign":            "competing offer",
		"contract.tag":         "playoff signings",
		"contract.extend":      "prior extension",
		"contract.restructure": "restructure allowance",
	} {
		if got := strings.Join(EvidenceReasons(envelope.ContractFacts{Intent: intent, Season: 2026, Baseline: base}), "\n"); !strings.Contains(got, want) || strings.Contains(got, "future salary") {
			t.Errorf("%s: %s", intent, got)
		}
	}
	two := envelope.ContractFacts{Intent: "contract.tag", Season: 2026, Status: domain.CStatusFT2, Baseline: append(base, contractacts.Cell{Season: 2027})}
	if got := strings.Join(EvidenceReasons(two), "\n"); !strings.Contains(got, "two consecutive franchise tags") || !strings.Contains(got, "future salary amounts") {
		t.Fatal(got)
	}
	two.Status = domain.CStatusFT1
	if got := strings.Join(EvidenceReasons(two), "\n"); !strings.Contains(got, "first tag price") {
		t.Fatal(got)
	}
}

func TestTermsPriceEachActFromTheRuleTable(t *testing.T) {
	t.Parallel()
	salary := domain.Money(10 * contractacts.Million)
	base := []contractacts.Cell{{Season: 2026, Salary: salary}}
	tender, err := contractacts.Tender(salary, 1)
	if err != nil {
		t.Fatal(err)
	}
	resign, err := contractacts.Resign(tender, 3)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		f     envelope.ContractFacts
		price domain.Money
		cells int
	}{
		{envelope.ContractFacts{Intent: "rfa.tender", Level: 1, Season: 2026, Baseline: base}, tender, 1},
		{envelope.ContractFacts{Intent: "fa.resign", Level: 1, Years: 3, Season: 2026, Baseline: base}, resign, 3},
	} {
		terms, err := Terms(tc.f, "QB", nil, salary)
		if err != nil || terms.Price != tc.price || len(terms.Cells) != tc.cells || terms.Cells[0].Season != 2027 {
			t.Fatalf("%s: %+v %v", tc.f.Intent, terms, err)
		}
	}
	rows, pool := []snapshot.ContractRow{{Position: "WR", Salary: snapshot.ContractAmountOf(90 * contractacts.Million)}}, []domain.Money{}
	for i := range 5 {
		amount := domain.Money(20+i) * contractacts.Million
		rows, pool = append(rows, snapshot.ContractRow{Position: "QB", Salary: snapshot.ContractAmountOf(amount)}), append(pool, amount)
	}
	tag, err := Terms(envelope.ContractFacts{Intent: "contract.tag", Season: 2026, Baseline: base}, "QB", rows, salary)
	if want, _ := contractacts.Tag(pool, salary, 0, 0); err != nil || tag.Price != want || want == 0 {
		t.Fatalf("tag %+v %v, want %d", tag, err, want)
	}
	for status, consecutive := range map[domain.ContractStatus]int{domain.CStatusFT1: 1, domain.CStatusFT2: 2} {
		first := salary
		if consecutive == 2 {
			first = 0
		}
		again, err := Terms(envelope.ContractFacts{Intent: "contract.tag", Season: 2026, Baseline: base, Status: status}, "QB", rows, salary)
		if want, wantErr := contractacts.Tag(pool, salary, first, consecutive); again.Price != want || (err == nil) != (wantErr == nil) {
			t.Fatalf("%s tag %+v %v, want %d %v", status, again, err, want, wantErr)
		}
	}
	if _, err := Terms(envelope.ContractFacts{Intent: "contract.extend", Season: 2026, Baseline: base, Years: 9}, "QB", nil, salary); err == nil || !strings.HasPrefix(err.Error(), "extension terms") {
		t.Fatal(err)
	}
	if _, err := Terms(envelope.ContractFacts{Intent: "contract.restructure", Baseline: base, Source: 2026, Destination: 2030}, "QB", nil, salary); err == nil || !strings.HasPrefix(err.Error(), "restructure terms") {
		t.Fatal(err)
	}
	if _, err := Terms(envelope.ContractFacts{Intent: "contract.void"}, "QB", nil, salary); err == nil || err.Error() != "unsupported contract Act" {
		t.Fatal(err)
	}
}

func TestTagPriceIsTheHeldSalaryOnlyForATaggedContract(t *testing.T) {
	t.Parallel()
	for status, want := range map[domain.ContractStatus]domain.Money{domain.CStatusFT1: 5, domain.CStatusFT2: 5, domain.CStatusUFA: 0} {
		if got := tagPrice(envelope.ContractFacts{Status: status}, 5); got != want {
			t.Errorf("%s: %d", status, got)
		}
	}
}

func TestNeedsInvalidationOnlyForAChangeOrAShutWindowOnAnUnsentPlan(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		state  envelope.State
		reason string
		want   bool
	}{
		{envelope.Ready, "", false}, {envelope.HandedOff, "Salary changed", true},
		{envelope.HandedOff, "Contract: window shut", false}, {envelope.Ready, "Contract: window shut", true},
		{envelope.Blocked, "Contract: window shut", true},
	} {
		if got := NeedsInvalidation(tc.state, tc.reason); got != tc.want {
			t.Errorf("%s %q: %v", tc.state, tc.reason, got)
		}
	}
}

func TestCapReasonJudgesOnlySeasonsTheActAddsMoneyTo(t *testing.T) {
	t.Parallel()
	f := envelope.ContractFacts{Baseline: []contractacts.Cell{{Season: 2026, Salary: 10}, {Season: 2027, Salary: 10}},
		Terms: contractacts.Terms{Cells: []contractacts.Cell{{Season: 2026, Salary: 5}, {Season: 2027, Salary: 15}}}}
	if got := CapReason(f, 100, map[int]domain.Money{2026: 200, 2027: 100}); got != "" {
		t.Fatal(got)
	}
	if got := CapReason(f, 100, map[int]domain.Money{2026: 200, 2027: 101}); got != "This move exceeds the team's 2027 salary cap." {
		t.Fatal(got)
	}
}
