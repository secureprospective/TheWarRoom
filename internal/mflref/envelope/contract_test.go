package envelope

import (
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/contractacts"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func contractFixture(t *testing.T) (Spec, ContractFacts, time.Time) {
	t.Helper()
	id, err := playerid.New("0531")
	if err != nil {
		t.Fatal(err)
	}
	s := Spec{Intent: "contract.restructure", LeagueID: "1", FranchiseID: "0001", Subject: Subject{Players: []playerid.PlayerID{id}}, Expected: ExpectedChange{Player: id}, Gravity: G3, Undo: Irreversible, Target: Target{Kind: Manual}}
	baseline := []contractacts.Cell{{Season: 2026, Salary: 600_000_000}, {Season: 2027, Salary: 600_000_000}}
	terms, err := contractacts.Restructure(baseline, 2026, 2027, 100_000_000, false)
	if err != nil {
		t.Fatal(err)
	}
	f := ContractFacts{Player: id, Name: "Example", Season: 2026, Intent: s.Intent, Baseline: baseline, Terms: terms, Status: domain.CStatusUFA}
	return s, f, time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
}
func TestContractWindowsFollowPolicyAndOffseasonRFA(t *testing.T) {
	for _, intent := range []string{"rfa.tender", "fa.resign", "contract.tag", "contract.extend", "contract.restructure"} {
		for _, phase := range []domain.Phase{domain.PhaseOffseason, domain.PhaseRegularSeason, domain.PhasePlayoffs} {
			reason := ContractWindow(intent, phase)
			wantBlocked := (intent == "rfa.tender" || intent == "fa.resign") && phase != domain.PhaseOffseason
			if (reason != "") != wantBlocked {
				t.Fatalf("%s %s: %s", intent, phase, reason)
			}
			if wantBlocked && reason != "Offseason only: can't be done until the offseason." {
				t.Fatal(reason)
			}
			result := (ContractCheck{Reasons: []string{reason}}).Evaluate(Spec{Intent: intent}, snapshot.Snapshot{})
			if result.Blocked != wantBlocked {
				t.Fatal(result)
			}
		}
	}
}
func TestContractUncheckedHistoryDoesNotBlockHeldViolationDoes(t *testing.T) {
	s, f, at := contractFixture(t)
	f.Reasons = []string{"Not checked: earlier restructure", "Not checked: team allowance"}
	e, err := DraftContract(at, s, f, func() string { return "ready" })
	if err != nil || e.State() != Ready {
		t.Fatalf("%s %v", e.State(), err)
	}
	f.Reasons = append(f.Reasons, "He has already had two consecutive franchise tags.")
	e, err = DraftContract(at, s, f, func() string { return "blocked" })
	if err != nil || e.State() != Blocked {
		t.Fatalf("%s %v", e.State(), err)
	}
}
func TestContractManualLifecycleAndRebase(t *testing.T) {
	s, f, at := contractFixture(t)
	e, err := DraftContract(at, s, f, func() string { return "contract-1" })
	if err != nil || e.State() != Ready {
		t.Fatalf("draft: %s %v", e.State(), err)
	}
	handed, err := e.HandOffManual(at.Add(time.Second), "Copied; not posted")
	if err != nil {
		t.Fatal(err)
	}
	pending, err := handed.ContractPosted(at.Add(2 * time.Second))
	if err != nil || pending.State() != BidPending {
		t.Fatalf("posted: %s %v", pending.State(), err)
	}
	if _, err := e.ContractPosted(at); err == nil {
		t.Fatal("posted before hand-off")
	}
	withdrawn, err := handed.WithdrawContract(at.Add(2 * time.Second))
	if err != nil || withdrawn.State() != Withdrawn {
		t.Fatalf("withdraw: %s %v", withdrawn.State(), err)
	}
	stale, err := e.Invalidate(at, "contract changed")
	if err != nil || stale.State() != Stale {
		t.Fatal(err)
	}
	draft, err := stale.Rebase(at, "repriced")
	if err != nil || draft.State() != Draft {
		t.Fatal(err)
	}
	blocked, err := draft.Check(at, snapshot.Snapshot{}, ContractCheck{Reasons: []string{"Rule unclear: window"}})
	if err != nil || blocked.State() != Blocked {
		t.Fatal(err)
	}
	if _, err := blocked.HandOffManual(at, "copied"); err == nil {
		t.Fatal("blocked plan handed off")
	}
}
func TestContractObservationRequiresEveryObservedCell(t *testing.T) {
	s, f, at := contractFixture(t)
	e, err := DraftContract(at, s, f, func() string { return "contract-1" })
	if err != nil {
		t.Fatal(err)
	}
	e, err = e.HandOffManual(at, "copied")
	if err != nil {
		t.Fatal(err)
	}
	fresh := snapshot.Provenance{Freshness: domain.Freshness{State: domain.FreshLive, FetchedAt: at.Add(time.Second).Format(time.RFC3339)}}
	held := ContractEvidence{Player: f.Player, FranchiseID: s.FranchiseID, Cells: f.Terms.Cells, Observed: true, Status: f.Status}
	obs := Observation{LeagueID: "1", Rosters: snapshot.Sourced[[]snapshot.Roster]{Value: []snapshot.Roster{{FranchiseID: "0001", Players: []snapshot.RosterPlayer{{ID: f.Player}}}}, Provenance: fresh}, Contracts: snapshot.Sourced[[]ContractEvidence]{Value: []ContractEvidence{held}, Provenance: fresh}}
	landed, err := e.Observe(at.Add(time.Second), obs, ContractPredicate{Facts: f})
	if err != nil || landed.State() != Landed {
		t.Fatalf("landed: %s %v", landed.State(), err)
	}
	obs.Contracts.Value[0].Cells = []contractacts.Cell{{Season: 2026, Salary: 400_000_000}}
	mismatch, err := e.Observe(at.Add(time.Second), obs, ContractPredicate{Facts: f})
	if err != nil || mismatch.State() != NotVerified || !strings.Contains(mismatch.Receipt().Audit[2].Note, "planned 2026 at $5,000,000.00") || !strings.Contains(mismatch.Receipt().Audit[2].Note, "MFL now shows 2026 at $4,000,000.00") {
		t.Fatalf("mismatch: %+v %v", mismatch.Receipt(), err)
	}
	obs.Contracts.Value[0].Observed = false
	unknown, err := e.Observe(at.Add(time.Second), obs, ContractPredicate{Facts: f})
	if err != nil || unknown.State() != NotVerified {
		t.Fatalf("inferred: %s %v", unknown.State(), err)
	}
	obs.Contracts.Provenance.Freshness.FetchedAt = at.Format(time.RFC3339)
	if _, err := e.Observe(at.Add(time.Second), obs, ContractPredicate{Facts: f}); err == nil {
		t.Fatal("old contract evidence accepted")
	}
}
func TestContractInvalidationIndependentReasons(t *testing.T) {
	s, f, _ := contractFixture(t)
	held := ContractEvidence{Player: f.Player, FranchiseID: s.FranchiseID, Cells: f.Baseline, Status: f.Status}
	if got := ContractInvalidation(f, held, ""); got != "" {
		t.Fatal(got)
	}
	if got := ContractInvalidation(f, held, "Window closed"); got != "Window closed" {
		t.Fatal(got)
	}
	held.Info = "new contract"
	if got := ContractInvalidation(f, held, ""); !strings.HasPrefix(got, "Contract:") {
		t.Fatal(got)
	}
	held.FranchiseID = ""
	if got := ContractInvalidation(f, held, ""); !strings.HasPrefix(got, "Roster:") {
		t.Fatal(got)
	}
}

func TestContractRemovalWordsBeforeAndAfterHandOff(t *testing.T) {
	s, f, at := contractFixture(t)
	ready, err := DraftContract(at, s, f, func() string { return "remove-words" })
	if err != nil {
		t.Fatal(err)
	}
	handed, err := ready.HandOffManual(at, "Copied")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		e    Envelope
		want string
	}{
		{ready, "Removed by you before it was handed off."},
		{handed, "Withdrawn: you said you did not post this contract request"},
	} {
		removed, err := tc.e.WithdrawContract(at)
		if err != nil {
			t.Fatal(err)
		}
		audit := removed.Receipt().Audit
		if audit[len(audit)-1].Note != tc.want {
			t.Fatal(audit)
		}
	}
}
func TestContractTenderLandsOnlyInTheSeasonAfterExpiry(t *testing.T) {
	s, f, at := contractFixture(t)
	s.Intent, f.Intent, f.Status = "rfa.tender", "rfa.tender", domain.CStatusRFA
	f.Baseline = []contractacts.Cell{{Season: 2026, Salary: 33_000_000}}
	f.Terms = contractacts.Flat(contractacts.NextSeason(f.Baseline, f.Season), 1, 60_000_000, "tender")
	e, err := DraftContract(at, s, f, func() string { return "tender-1" })
	if err != nil {
		t.Fatal(err)
	}
	e, err = e.HandOffManual(at, "copied")
	if err != nil {
		t.Fatal(err)
	}
	fresh := snapshot.Provenance{Freshness: domain.Freshness{State: domain.FreshLive, FetchedAt: at.Add(time.Second).Format(time.RFC3339)}}
	observe := func(cells []contractacts.Cell) Envelope {
		t.Helper()
		held := ContractEvidence{Player: f.Player, FranchiseID: s.FranchiseID, Cells: cells, Observed: true, Status: domain.CStatusUFA}
		obs := Observation{LeagueID: "1", Rosters: snapshot.Sourced[[]snapshot.Roster]{Value: []snapshot.Roster{{FranchiseID: "0001", Players: []snapshot.RosterPlayer{{ID: f.Player}}}}, Provenance: fresh}, Contracts: snapshot.Sourced[[]ContractEvidence]{Value: []ContractEvidence{held}, Provenance: fresh}}
		next, err := e.Observe(at.Add(time.Second), obs, ContractPredicate{Facts: f})
		if err != nil {
			t.Fatal(err)
		}
		return next
	}
	if landed := observe([]contractacts.Cell{{Season: 2027, Salary: 60_000_000}}); landed.State() != Landed {
		t.Fatalf("new cell in 2027: %s", landed.State())
	}
	wrong := observe([]contractacts.Cell{{Season: 2026, Salary: 60_000_000}})
	if note := wrong.Receipt().Audit[2].Note; wrong.State() != NotVerified || !strings.Contains(note, "planned 2027 at $600,000.00") || !strings.Contains(note, "MFL now shows 2026 at $600,000.00") {
		t.Fatalf("new cell in 2026: %s %s", wrong.State(), note)
	}
}
