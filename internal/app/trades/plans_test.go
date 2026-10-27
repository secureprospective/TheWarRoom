package trades

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type memoryReceipts struct {
	value   envelope.Envelope
	saveErr error
	saves   int
}

func (m *memoryReceipts) Get(context.Context, string) (envelope.Envelope, error) { return m.value, nil }
func (m *memoryReceipts) Save(_ context.Context, e envelope.Envelope) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.value = e
	m.saves++
	return nil
}
func planRequest(mine string, offer leaguefeed.PendingTrade) envelope.TradeRequest {
	return envelope.TradeRequest{LeagueID: "14432", FranchiseID: mine,
		Target: envelope.Target{Kind: envelope.Mapped, URL: "https://fixture.myfantasyleague.com/options"},
		Trade: envelope.ExpectedTrade{TradeID: offer.ID, Offering: offer.Offering, Accepting: offer.OfferedTo,
			OfferingGives: leaguefeed.AssetTokens(offer.Gives), AcceptingGives: leaguefeed.AssetTokens(offer.Gets)}}
}
func TestPlanKeepSaveOrderAndFailure(t *testing.T) {
	ctx := context.Background()
	at := time.Now()
	for _, fail := range []bool{true, false} {
		store := &memoryReceipts{}
		if fail {
			store.saveErr = errors.New("disk full")
		}
		calls := []string{}
		service := PlanWorkflow{Receipts: store, Request: planRequest, Now: func() time.Time { return at },
			Lock: func() { calls = append(calls, "lock") }, Unlock: func() { calls = append(calls, "unlock") },
			SavePlan: func(context.Context, state.TradePlan) error { calls = append(calls, "plan"); return nil },
			Changed: func(context.Context) {
				if store.value.State() != envelope.Ready {
					t.Fatal("event before receipt save")
				}
				calls = append(calls, "changed")
			}}
		receipt, err := service.Keep(ctx, RawTradeBuilder{Mine: "0001", Other: "0002", Give: []string{"0099"}, Get: []string{"12345"}}, nil)
		want := []string{"lock", "plan", "unlock"}
		if fail {
			if !errors.Is(err, store.saveErr) || store.saves != 0 {
				t.Fatal(err, store.saves)
			}
		} else {
			want = []string{"lock", "plan", "changed", "unlock"}
			if err != nil || receipt.CorrelationID == "" || store.saves != 1 {
				t.Fatal(receipt, err)
			}
		}
		if !reflect.DeepEqual(calls, want) {
			t.Fatal(calls)
		}
	}
}
func TestPlanExitRefusesMissingAndWrongDirectionBeforeEffects(t *testing.T) {
	service := PlanWorkflow{}
	held := snapshot.Sourced[[]leaguefeed.PendingTrade]{Value: []leaguefeed.PendingTrade{{ID: "offer", Offering: "0001", OfferedTo: "0002"}}}
	for _, r := range []RawTradeBuilder{
		{Mine: "0001", ID: "missing", Action: "reject"},
		{Mine: "0001", ID: "offer", Action: "reject"},
		{Mine: "0002", ID: "offer", Action: "revoke"},
	} {
		if _, err := service.Exit(context.Background(), r, held); err == nil {
			t.Fatal("accepted", r)
		}
	}
}
func TestPlanHandoffRechecksAndSavesBeforeBrowser(t *testing.T) {
	ctx := context.Background()
	at := time.Now()
	store := &memoryReceipts{}
	e, err := envelope.DraftTradeAct(at, "trade.propose", envelope.TradeRequest{LeagueID: "14432", FranchiseID: "0001",
		Target: envelope.Target{Kind: envelope.Mapped, URL: "https://fixture.myfantasyleague.com/options"},
		Trade:  envelope.ExpectedTrade{TradeID: "plan", Offering: "0001", Accepting: "0002", OfferingGives: []string{"0099"}, AcceptingGives: []string{"12345"}}}, nil, func() string { return "plan" })
	if err != nil {
		t.Fatal(err)
	}
	effects := []string{}
	blocked := true
	service := PlanWorkflow{Receipts: store, Now: func() time.Time { return at }, Lock: func() {}, Unlock: func() {},
		Plans: func(context.Context, string) ([]state.TradePlan, error) {
			return []state.TradePlan{{ID: "plan", Mine: "0001", Other: "0002"}}, nil
		},
		Own: func(string) error { return nil },
		Project: func(context.Context, RawTradeBuilder, snapshot.Snapshot) (TradeBuilderPlan, error) {
			if blocked {
				return TradeBuilderPlan{Reasons: []string{"stale"}}, nil
			}
			return TradeBuilderPlan{}, nil
		},
		ValidateURL: func(string) error { return nil },
		OpenURL: func(context.Context, string) {
			if store.value.State() != envelope.HandedOff {
				t.Fatal("browser before save")
			}
			effects = append(effects, "browser")
		},
		Changed: func(context.Context) { effects = append(effects, "changed") }, Wake: func() { effects = append(effects, "wake") },
	}
	r := RawTradeBuilder{Mine: "0001", ID: "plan"}
	store.value = e
	if err := service.Handoff(ctx, r, snapshot.Snapshot{}); err == nil || !strings.Contains(err.Error(), "stale") || store.saves != 0 {
		t.Fatal(err)
	}
	blocked = false
	store.saveErr = errors.New("disk full")
	if err := service.Handoff(ctx, r, snapshot.Snapshot{}); !errors.Is(err, store.saveErr) || len(effects) != 0 || store.value.State() != envelope.Ready {
		t.Fatal(err, effects)
	}
	store.saveErr = nil
	if err := service.Handoff(ctx, r, snapshot.Snapshot{}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(effects, []string{"browser", "changed", "wake"}) {
		t.Fatal(effects)
	}
}
func TestBuilderOwnerRefusalAvoidsStateAndPreviewFailureRetainsTeams(t *testing.T) {
	refused := errors.New("not yours")
	service := BuilderWorkflow{RequireOwner: func(context.Context, RawTradeBuilder) error { return refused }}
	if _, err := service.Run(context.Background(), RawTradeBuilder{}, TradeBuilderView{}); !errors.Is(err, refused) {
		t.Fatal(err)
	}
	service.RequireOwner = func(context.Context, RawTradeBuilder) error { return nil }
	service.State = func(string) (snapshot.Snapshot, snapshot.Sourced[[]leaguefeed.PendingTrade], error) {
		return snapshot.Snapshot{}, snapshot.Sourced[[]leaguefeed.PendingTrade]{}, nil
	}
	service.Teams = func(context.Context, snapshot.Snapshot) []TradeBuilderTeam { return []TradeBuilderTeam{{ID: "0001"}} }
	service.Project = func(context.Context, RawTradeBuilder, snapshot.Snapshot) (TradeBuilderPlan, error) {
		return TradeBuilderPlan{}, refused
	}
	out, err := service.Run(context.Background(), RawTradeBuilder{Action: "preview"}, TradeBuilderView{})
	if !errors.Is(err, refused) || len(out.Teams) != 1 || out.Preview != nil {
		t.Fatal(out, err)
	}
}
func TestSimulationAndProjectionStopOnInputFailure(t *testing.T) {
	failure := errors.New("unavailable")
	simulation := SimulationWorkflow{Scratch: db.OpenScratch, Open: func(context.Context, *db.Pools, string) (state.WhatIfSession, *state.Store, error) {
		return state.WhatIfSession{}, nil, failure
	}}
	if out, err := simulation.Run(context.Background(), "0001", "0002"); !errors.Is(err, failure) || out.Store != nil || len(out.Sessions) != 0 {
		t.Fatal(out, err)
	}
	projection := ProjectionWorkflow{Simulate: func(context.Context, string, string) (Simulation, error) { return Simulation{}, failure }}
	if out, err := projection.Run(context.Background(), RawTradeBuilder{}, snapshot.Snapshot{}); !errors.Is(err, failure) || len(out.Teams) != 0 || len(out.Reasons) != 0 {
		t.Fatal(out, err)
	}
}
