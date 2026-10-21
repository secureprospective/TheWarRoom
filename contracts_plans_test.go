package main

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	bidsapp "github.com/secureprospective/TheWarRoom/internal/app/offers/bids"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
)

func TestContractPendingCanBeWithdrawn(t *testing.T) {
	t.Parallel()
	a := releaseTestApp(t)
	raw := contractsapp.RawContractAct{Mine: "0001", Player: "0042", Action: "draft", Intent: "contract.restructure", Level: 1, Years: 1, Source: 2026, Destination: 2028, Move: "1"}
	kept := a.contracts().TargetContractActs(raw)
	if kept.Status != report.ReadingOK {
		t.Fatal(kept)
	}
	id := kept.Value[0].ID
	for _, action := range []string{"handoff", "posted"} {
		r := a.contracts().TargetContractActs(contractsapp.RawContractAct{Mine: "0001", Action: action, ID: id})
		if r.Status != report.ReadingOK {
			t.Fatal(r)
		}
	}
	plans := a.contracts().TargetContractActs(contractsapp.RawContractAct{Mine: "0001", Action: "read"})
	pending := plans.Value[0]
	if pending.Receipt.State != envelope.BidPending || !pending.CanWithdraw {
		t.Fatal("pending has no way out", pending)
	}
	gone := a.contracts().TargetContractActs(contractsapp.RawContractAct{Mine: "0001", Action: "withdraw", ID: id})
	if gone.Status != report.ReadingOK || gone.Value[0].Receipt.State != envelope.Withdrawn {
		t.Fatal(gone)
	}
	audit := gone.Value[0].Receipt.Audit
	if audit[len(audit)-1].Note != "Withdrawn: you said the commissioner did not process this request." {
		t.Fatal(audit)
	}
}

func TestEveryPlanViewOffersAnExit(t *testing.T) {
	t.Parallel()
	for _, state := range []envelope.State{envelope.Draft, envelope.Blocked, envelope.Ready, envelope.HandedOff, envelope.NotYetDone, envelope.NotVerified, envelope.Stale, envelope.BidPending} {
		actions := bidsapp.BidActions(state, "", nil, "")
		found := false
		for _, action := range actions {
			if action.ID == "withdraw" || action.ID == "unposted" {
				found = true
			}
		}
		if !found {
			t.Fatal("bid", state)
		}
	}
}

func TestRosterPlanWithdrawalIsLocalAndPersisted(t *testing.T) {
	t.Parallel()
	for _, intent := range []string{"roster.ir", "roster.taxi"} {
		t.Run(intent, func(t *testing.T) {
			t.Parallel()
			a := releaseTestApp(t)
			at := time.Now().UTC().Add(-time.Minute)
			spec := moveSpec(t)
			spec.Intent = intent
			e := constructedMove(t, at, spec, false)
			if err := a.core.Moves.Save(a.core.Ctx, e); err != nil {
				t.Fatal(err)
			}
			before, stamp, _ := a.core.League.ContractsView()
			opens := 0
			a.core.OpenURL = func(_ context.Context, _ string) { opens++ }
			gone, err := a.TargetWithdrawRoster(e.ID())
			if err != nil || gone.State != envelope.Withdrawn {
				t.Fatal(gone, err)
			}
			if !strings.Contains(gone.Audit[len(gone.Audit)-1].Note, "Removed by you before") {
				t.Fatal(gone.Audit)
			}
			held, err := a.core.Moves.Get(a.core.Ctx, e.ID())
			if err != nil || held.State() != envelope.Withdrawn {
				t.Fatal(held.State(), err)
			}
			after, afterStamp, _ := a.core.League.ContractsView()
			if opens != 0 || stamp != afterStamp || !reflect.DeepEqual(before, after) {
				t.Fatal("withdraw wrote MFL facts or opened browser")
			}
		})
	}
}

func TestRemovalWordsAgreeAcrossKinds(t *testing.T) {
	t.Parallel()
	const want = "Removed by you before it was handed off."
	for _, action := range []string{"cut", "buyout"} {
		a := releaseTestApp(t)
		p := planRelease(t, a, action, "0042")
		if p.WithdrawLabel != "Remove plan" {
			t.Fatal(p.WithdrawLabel)
		}
		gone := releaseCall(t, a, "withdraw", "", p.ID, false)[0].Receipt
		if gone.Audit[len(gone.Audit)-1].Note != want {
			t.Fatal(action, gone.Audit)
		}
	}
	a := releaseTestApp(t)
	raw := contractsapp.RawContractAct{Mine: "0001", Player: "0042", Action: "draft", Intent: "contract.restructure", Level: 1, Years: 1, Source: 2026, Destination: 2028, Move: "1"}
	kept := a.contracts().TargetContractActs(raw)
	if kept.Status != report.ReadingOK {
		t.Fatal(kept)
	}
	gone := a.contracts().TargetContractActs(contractsapp.RawContractAct{Mine: "0001", Action: "withdraw", ID: kept.Value[0].ID}).Value[0].Receipt
	if gone.Audit[len(gone.Audit)-1].Note != want {
		t.Fatal(gone.Audit)
	}
	b := bidTestApp(t)
	p, err := offersTestAPI(b).TargetBidDraft(rawBid("5", 3))
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range p.Actions {
		if action.ID == "withdraw" && action.Label != "Remove plan" {
			t.Fatal(action)
		}
	}
	removed, err := offersTestAPI(b).TargetBidWithdraw(p.Receipt.CorrelationID)
	if err != nil || removed.Status != want {
		t.Fatal(removed.Status, err)
	}
}

func TestJoinedNotesDoNotDoublePunctuate(t *testing.T) {
	t.Parallel()
	if got := report.JoinNote("First.", "Second."); got != "First; Second" {
		t.Fatal(got)
	}
}
