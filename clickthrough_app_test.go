package main

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine/contractacts"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func TestContractPendingCanBeWithdrawn(t *testing.T) {
	a := releaseTestApp(t)
	raw := RawContractAct{Mine: "0001", Player: "0042", Action: "draft", Intent: "contract.restructure", Level: 1, Years: 1, Source: 2026, Destination: 2028, Move: "1"}
	kept := a.TargetContractActs(raw)
	if kept.Status != ReadingOK {
		t.Fatal(kept)
	}
	id := kept.Value[0].ID
	for _, action := range []string{"handoff", "posted"} {
		r := a.TargetContractActs(RawContractAct{Mine: "0001", Action: action, ID: id})
		if r.Status != ReadingOK {
			t.Fatal(r)
		}
	}
	plans := a.TargetContractActs(RawContractAct{Mine: "0001", Action: "read"})
	pending := plans.Value[0]
	if pending.Receipt.State != envelope.BidPending || !pending.CanWithdraw {
		t.Fatal("pending has no way out", pending)
	}
	gone := a.TargetContractActs(RawContractAct{Mine: "0001", Action: "withdraw", ID: id})
	if gone.Status != ReadingOK || gone.Value[0].Receipt.State != envelope.Withdrawn {
		t.Fatal(gone)
	}
	audit := gone.Value[0].Receipt.Audit
	if audit[len(audit)-1].Note != "Withdrawn: you said the commissioner did not process this request." {
		t.Fatal(audit)
	}
}

func TestEveryPlanViewOffersAnExit(t *testing.T) {
	for _, state := range []envelope.State{envelope.Draft, envelope.Blocked, envelope.Ready, envelope.HandedOff, envelope.NotYetDone, envelope.NotVerified, envelope.Stale, envelope.BidPending} {
		if !contractActView(envelope.Receipt{State: state}, envelope.ContractFacts{}).CanWithdraw {
			t.Fatal("contract", state)
		}
		actions := bidActions(state, "", nil, "")
		found := false
		for _, action := range actions {
			if action.ID == "withdraw" || action.ID == "unposted" {
				found = true
			}
		}
		if !found {
			t.Fatal("bid", state)
		}
		if state != envelope.BidPending {
			for _, cut := range []bool{false, true} {
				if releaseWithdrawLabel(state, cut) == "" {
					t.Fatal("release", cut, state)
				}
			}
		}
	}
}

func TestContractFigureWordsPerAct(t *testing.T) {
	for _, intent := range []string{"contract.restructure", "contract.extend", "contract.tag", "fa.resign", "rfa.tender"} {
		f := envelope.ContractFacts{Intent: intent, Source: 2026, Destination: 2028}
		want := "Price: $1,000,000"
		if intent == "contract.restructure" {
			want = "Moves: $1,000,000 from 2026 to 2028"
		}
		if got := contractFigureSentence(f, "$1,000,000"); got != want {
			t.Fatal(intent, got)
		}
	}
}

func TestRosterPlanWithdrawalIsLocalAndPersisted(t *testing.T) {
	for _, intent := range []string{"roster.ir", "roster.taxi"} {
		t.Run(intent, func(t *testing.T) {
			a := releaseTestApp(t)
			at := time.Now().UTC().Add(-time.Minute)
			spec := moveSpec(t)
			spec.Intent = intent
			e := constructedMove(t, at, spec, false)
			if err := a.moves.Save(a.ctx, e); err != nil {
				t.Fatal(err)
			}
			before, stamp, _ := a.league.ContractsView()
			opens := 0
			a.openURL = func(_ context.Context, _ string) { opens++ }
			gone, err := a.TargetWithdrawRoster(e.ID())
			if err != nil || gone.State != envelope.Withdrawn {
				t.Fatal(gone, err)
			}
			if !strings.Contains(gone.Audit[len(gone.Audit)-1].Note, "Removed by you before") {
				t.Fatal(gone.Audit)
			}
			held, err := a.moves.Get(a.ctx, e.ID())
			if err != nil || held.State() != envelope.Withdrawn {
				t.Fatal(held.State(), err)
			}
			after, afterStamp, _ := a.league.ContractsView()
			if opens != 0 || stamp != afterStamp || !reflect.DeepEqual(before, after) {
				t.Fatal("withdraw wrote MFL facts or opened browser")
			}
		})
	}
}

func TestRemovalWordsAgreeAcrossKinds(t *testing.T) {
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
	raw := RawContractAct{Mine: "0001", Player: "0042", Action: "draft", Intent: "contract.restructure", Level: 1, Years: 1, Source: 2026, Destination: 2028, Move: "1"}
	kept := a.TargetContractActs(raw)
	if kept.Status != ReadingOK {
		t.Fatal(kept)
	}
	gone := a.TargetContractActs(RawContractAct{Mine: "0001", Action: "withdraw", ID: kept.Value[0].ID}).Value[0].Receipt
	if gone.Audit[len(gone.Audit)-1].Note != want {
		t.Fatal(gone.Audit)
	}
	b := bidTestApp(t)
	p, err := b.TargetBidDraft(rawBid("5", 3))
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range p.Actions {
		if action.ID == "withdraw" && action.Label != "Remove plan" {
			t.Fatal(action)
		}
	}
	removed, err := b.TargetBidWithdraw(p.Receipt.CorrelationID)
	if err != nil || removed.Status != want {
		t.Fatal(removed.Status, err)
	}
}

func TestTenderPostUsesLeagueRoundAtEveryLevel(t *testing.T) {
	for i, level := range contractacts.TenderLevels() {
		facts := envelope.ContractFacts{Intent: "rfa.tender", Level: i + 1, Season: 2026,
			Terms: contractacts.Flat(2027, 1, level.Price, "")}
		got := contractActView(envelope.Receipt{}, facts).Post
		want := level.Round + "-round RFA tender for 2027: " + snapshot.ContractAmountOf(level.Price).Text
		if !strings.Contains(got, want) || strings.Contains(got, "Tender level") {
			t.Fatal(got, want)
		}
	}
}

func TestJoinedNotesDoNotDoublePunctuate(t *testing.T) {
	if got := joinNote("First.", "Second."); got != "First; Second" {
		t.Fatal(got)
	}
}
