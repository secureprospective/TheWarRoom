package contracts

import (
	"strings"
	"testing"

	mflwhatif "github.com/secureprospective/TheWarRoom/internal/mflref/whatif"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestReleaseValidateRefusesEachMalformedRequest(t *testing.T) {
	t.Parallel()
	for want, r := range map[string]Release{
		"release scope":                       {Mine: "bad", Action: "read"},
		"franchise and known action required": {Mine: "0001", Action: "submit"},
		"what-if move required":               {Mine: "0001", Action: "draft"},
		"plan id required":                    {Mine: "0001", Action: "withdraw"},
	} {
		if err := r.Validate(); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%+v: %v, want %q", r, err, want)
		}
	}
	for _, r := range []Release{{Mine: "0001", Action: "read"}, {Mine: "0001", Action: "draft", Move: "1"}, {Mine: "0001", Action: "rebase", ID: "r"}} {
		if err := r.Validate(); err != nil {
			t.Errorf("%+v refused: %v", r, err)
		}
	}
}

func TestPlanWordsACutAndABuyout(t *testing.T) {
	t.Parallel()
	f := envelope.ReleaseFacts{Name: "Example", Charge: 600_000_000, Salary: 1_000_000_000, Season: 2026, Expiration: 2028}
	cut := Plan(envelope.Receipt{CorrelationID: "release-1", State: envelope.Ready, Spec: envelope.Spec{Intent: envelope.CutIntent}}, f)
	if !cut.Cut || cut.Label != "Cut Example" || !cut.CanHandOff || cut.CanRebase || cut.WithdrawLabel != "Remove plan" ||
		!strings.Contains(cut.Post, "2 paid years after 2026") || !strings.HasSuffix(cut.Post, "Cannot re-sign this player for the rest of this season.") ||
		cut.Confirmation != "This releases Example; $6,000,000 dead cap this season. It cannot be undone." {
		t.Fatalf("cut %+v", cut)
	}
	buyout := Plan(envelope.Receipt{State: envelope.Stale, Spec: envelope.Spec{Intent: envelope.BuyoutIntent}}, f)
	if buyout.Cut || buyout.Label != "Buy out Example" || !buyout.CanRebase || buyout.Channel != "Post this to the commissioner on ProBoards." ||
		!strings.Contains(buyout.Post, "Please process this Buy out request.\nCannot bid on this player for the rest of this season; eligible again the following offseason.") {
		t.Fatalf("buyout %+v", buyout)
	}
	for state, want := range map[envelope.State][2]string{
		envelope.HandedOff: {"I didn't drop him", "I didn't post it"}, envelope.NotVerified: {"I didn't drop him", "I didn't post it"},
		envelope.Landed: {"", ""}, envelope.Draft: {"Remove plan", "Remove plan"},
	} {
		if withdrawLabel(state, true) != want[0] || withdrawLabel(state, false) != want[1] {
			t.Errorf("%s: %q %q", state, withdrawLabel(state, true), withdrawLabel(state, false))
		}
	}
}

func TestWithdrawNoteUsesTheOwnersWordsAfterAHandOff(t *testing.T) {
	t.Parallel()
	r := envelope.Receipt{Spec: envelope.Spec{Intent: envelope.CutIntent}, Audit: []envelope.AuditEntry{{To: envelope.Ready}}}
	if got := WithdrawNote(r); got != envelope.RemovePlanNote {
		t.Fatal(got)
	}
	r.Audit = append(r.Audit, envelope.AuditEntry{To: envelope.HandedOff})
	if got := WithdrawNote(r); got != "You said you did not drop him, so the plan is withdrawn." {
		t.Fatal(got)
	}
	r.Spec.Intent = envelope.BuyoutIntent
	if got := WithdrawNote(r); got != "You said you did not post it, so the plan is withdrawn." {
		t.Fatal(got)
	}
}

func TestReleaseFactsComeFromTheWhatIfBaseline(t *testing.T) {
	t.Parallel()
	s := mflwhatif.WhatIfSession{Mine: "0001", Snapshot: heldMirror(), Stamps: state.MirrorStamps{Rosters: "r", Contract: "c", DeadCap: "d"}}
	f, err := ReleaseFacts(s, mflwhatif.WhatIfMove{Player: "0042", Name: "Example", Charge: 7})
	if err != nil || f.Player.String() != "0042" || f.Charge != 7 || f.Expiration != 2028 || f.RostersStamp != "r" || f.ContractStamp != "c" || f.DeadCapStamp != "d" ||
		!strings.Contains(f.Adjustments, `"ID":"a"`) || strings.Contains(f.Adjustments, `"ID":"b"`) {
		t.Fatalf("facts %+v %v", f, err)
	}
	if _, err := ReleaseFacts(s, mflwhatif.WhatIfMove{Player: "x"}); err == nil || !strings.HasPrefix(err.Error(), "release player") {
		t.Fatal(err)
	}
	if _, err := ReleaseFacts(s, mflwhatif.WhatIfMove{Player: "0050"}); err == nil || err.Error() != "release: player not in what-if baseline" {
		t.Fatal(err)
	}
	if got, err := Adjustments(state.MirrorSnapshot{}, "0001"); err != nil || got != "[]" {
		t.Fatal(got, err)
	}
}

func TestObservedChargeNeedsExactlyOnePlayerLabelledDeadCap(t *testing.T) {
	t.Parallel()
	id, err := playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	f := envelope.ReleaseFacts{Player: id, Charge: 300_000_000}
	m := heldMirror()
	if amount, ok := ObservedCharge(f, "0001", m); !ok || amount != 300_000_000 {
		t.Fatal(amount, ok)
	}
	if got := ChargeNote(f, "0001", m); got != "Landed: planned dead cap matches $3,000,000 now" {
		t.Fatal(got)
	}
	f.Charge = 100_000_000
	if got := ChargeNote(f, "0001", m); got != "$1,000,000 planned vs $3,000,000 now (MFL player-labelled dead cap)" {
		t.Fatal(got)
	}
	m.Adjustments = append(m.Adjustments, state.Adjustment{FranchiseID: "0001", Description: "dead cap 0042 again"}, state.Adjustment{FranchiseID: "0001", Description: "credit 0042"})
	if _, ok := ObservedCharge(f, "0001", m); ok {
		t.Fatal("two charges read as one")
	}
	if got := ChargeNote(f, "0002", m); got != "Landed: cannot see a player-linked dead-cap charge in MFL salary adjustments." {
		t.Fatal(got)
	}
}

func TestReleaseStatesAndContractChanges(t *testing.T) {
	t.Parallel()
	for s, want := range map[envelope.State]bool{envelope.Ready: true, envelope.HandedOff: true, envelope.NotYetDone: true, envelope.NotVerified: true, envelope.Stale: false, envelope.Landed: false} {
		if NeedsValidation(s) != want {
			t.Errorf("%s", s)
		}
	}
	if StateLabel(envelope.HandedOff) != "Handed off" || StateLabel(envelope.Landed) != "" {
		t.Fatal("state labels")
	}
	p := state.MirrorPlayer{Salary: 5, ContractYear: 2028, ContractInfo: "i"}
	f := envelope.ReleaseFacts{Salary: 5, Expiration: 2028, ContractInfo: "i"}
	if ContractChanged(p, f) {
		t.Fatal("same contract changed")
	}
	for _, changed := range []state.MirrorPlayer{{Salary: 6, ContractYear: 2028, ContractInfo: "i"}, {Salary: 5, ContractYear: 2027, ContractInfo: "i"}, {Salary: 5, ContractYear: 2028}} {
		if !ContractChanged(changed, f) {
			t.Errorf("%+v unchanged", changed)
		}
	}
	if _, found := Player(heldMirror(), "0001", "0050"); found {
		t.Fatal("another franchise's player found")
	}
}
