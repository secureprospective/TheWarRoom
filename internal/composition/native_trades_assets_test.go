package composition

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestTradePickOwnershipGovernsDraftOrderAndCompensation(t *testing.T) {
	t.Parallel()
	f := tradePickApp(t)
	f.command(t, commissioner.DraftStart, commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{Season: 2027, At: draftStartAt()}}, "league")
	f.command(t, commissioner.DraftOrderSet, commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{Season: 2027, Order: draftOrder(4, 1, 2, 3)}}, "league")
	f.set(t, "draft.laterPattern", `"linear"`, "now")
	f.set(t, "trades.review", `"none"`, "now")
	read := func() string {
		t.Helper()
		d, err := newCommissionerAPI(f.app).TargetNativeDraft(commissioner.RawNativeDraftScope{League: legalityLeague, Season: 2027, Franchise: "franchise.01", At: f.now})
		if err != nil {
			t.Fatal(err)
		}
		if len(d.Order) == 0 || len(d.Order[0].Picks) == 0 {
			t.Fatalf("draft order unavailable: %+v", d)
		}
		return d.Order[0].Picks[0].Owner
	}
	if read() != "franchise.02" {
		t.Fatal("wrong original pick holder")
	}
	offer := f.mustAction(t, "owner.two", commissioner.TradeOffer, tradePickOffer())
	accept := f.accept(t, offer)
	if read() != "franchise.01" {
		t.Fatal("draft order ignored traded pick")
	}
	f.restart(t)
	if read() != "franchise.01" {
		t.Fatal("traded pick did not replay")
	}
	f.app.core.CommissionerSession = testCommissionerSession{"creator"}
	f.revert(t, accept)
	if read() != "franchise.02" {
		t.Fatal("draft ownership compensation failed")
	}
	f.accept(t, offer)
	f.now = draftStartAt().Add(time.Minute)
	f.app.core.CommissionerSession = testCommissionerSession{"owner.one"}
	records, err := f.app.leagueRecords(t.Context(), legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	_, err = newCommissionerAPI(f.app).TargetNativeDraftPick(commissioner.DraftPickInput{League: legalityLeague, Season: 2027, Franchise: "franchise.01", Player: "5001", Request: "traded-pick", Expected: int64(len(records))})
	if err != nil {
		t.Fatal(err)
	}
	assets, err := newCommissionerAPI(f.app).TargetNativeAssets(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range assets.Snapshot.Picks {
		if p.Season == 2027 && p.Round == 1 && p.Original == "franchise.04" && p.Player != "5001" {
			t.Fatal("draft did not spend the traded pick")
		}
	}
	if f.ownerOf(t, "5001") != "franchise.01" {
		t.Fatal("new owner did not receive the rookie")
	}
	f.app.core.CommissionerSession = testCommissionerSession{"creator"}
	// Undoing the new acceptance would make the made selection belong to the wrong team.
	records, err = f.app.leagueRecords(t.Context(), legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	var last commissioner.Receipt
	for _, r := range records {
		if r.Command.Command == commissioner.TradeAccept {
			last = r.Receipt
		}
	}
	raw := f.raw(t, commissioner.Revert, commissioner.RawArgs{Entry: last.Entry}, "league")
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newCommissionerAPI(f.app).TargetCommissionerExecute(string(body)); err == nil {
		t.Fatal("compensation stranded a made draft pick")
	}
	f.restart(t)
	if f.ownerOf(t, "5001") != "franchise.01" {
		t.Fatal("drafted traded pick did not replay")
	}
}

func TestTradeActionsIdentityRetryVersionAndTerminalStates(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	if _, err := f.action(t, "owner.one", commissioner.TradeOffer, tradePlayerOffer()); err == nil {
		t.Fatal("foreign owner offered assets")
	}
	offer := f.offer(t)
	f.app.core.CommissionerSession = testCommissionerSession{"owner.one"}
	raw := f.raw(t, commissioner.TradeAccept, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.01"}}, "league")
	first, err := newCommissionerAPI(f.app).TargetNativeTradeAction(raw)
	if err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(8 * 24 * time.Hour)
	retry, err := newCommissionerAPI(f.app).TargetNativeTradeAction(raw)
	if err != nil || retry != first {
		t.Fatalf("retry changed: %+v %v", retry, err)
	}
	raw.Request = "stale.version"
	if _, err := newCommissionerAPI(f.app).TargetNativeTradeAction(raw); err == nil {
		t.Fatal("stale version accepted")
	}
	raw = f.raw(t, commissioner.TradeWithdraw, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.02"}}, "league")
	f.app.core.CommissionerSession = testCommissionerSession{"owner.two"}
	if _, err := newCommissionerAPI(f.app).TargetNativeTradeAction(raw); err == nil {
		t.Fatal("accepted trade was withdrawn")
	}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newCommissionerAPI(f.app).TargetCommissionerExecute(string(body)); err == nil || !strings.Contains(err.Error(), "no longer waiting") {
		t.Fatalf("generic door lost the trade refusal: %v", err)
	}
	// Separate offers retain distinct decline and withdrawal outcomes.
	for _, command := range []string{commissioner.TradeDecline, commissioner.TradeWithdraw} {
		o := f.offer(t)
		who, team := "owner.one", "franchise.01"
		if command == commissioner.TradeWithdraw {
			who, team = "owner.two", "franchise.02"
		}
		f.mustAction(t, who, command, commissioner.RawTradeArgs{ID: o.Entry, Franchise: team})
		r := f.trades(t, "owner.one", "franchise.01").Offers
		if got := r[len(r)-1].Status; got != "declined" && got != "withdrawn" {
			t.Fatalf("%s outcome %s", command, got)
		}
	}
}

func TestTradeCommissionerRouteAndNoEligibleJudge(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	f.set(t, "trades.review", `"commissioner"`, "now")
	offer := f.offer(t)
	f.accept(t, offer)
	if _, err := f.action(t, "dot.a", commissioner.TradeReview, commissioner.RawTradeArgs{ID: offer.Entry, Decision: "approve"}); err == nil {
		t.Fatal("DOT decided commissioner route")
	}
	f.vote(t, offer, "creator", "approve")
	if f.only(t).Status != "approved" {
		t.Fatal("commissioner route did not move assets")
	}
	f = tradeApp(t)
	f.set(t, "trades.review", `"commissioner"`, "now")
	f.command(t, "commish.seat.replace", commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.02", Principal: "creator"}}, "league")
	f.command(t, "commish.role.revoke", commissioner.RawArgs{People: &commissioner.RawPeople{Principal: "cc", Role: "co-commissioner"}}, "private")
	offer = f.mustAction(t, "creator", commissioner.TradeOffer, tradePlayerOffer())
	f.accept(t, offer)
	if got := f.only(t); got.Status != "failed" || !strings.Contains(got.Words, "No eligible") {
		t.Fatalf("no judge did not fail closed: %+v", got)
	}
}

func TestTradeMovesContractSalaryIntoNativeCap(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	f.set(t, "trades.review", `"none"`, "now")
	committed := func(team string) int64 {
		t.Helper()
		r, err := newCommissionerAPI(f.app).TargetNativeCapCheck(legalityLeague, "America/Los_Angeles")
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range r.Franchises {
			if row.Franchise == team && row.CommittedCents != nil {
				return int64(*row.CommittedCents)
			}
		}
		t.Fatal("cap missing")
		return 0
	}
	before := committed("franchise.01")
	assets, err := newCommissionerAPI(f.app).TargetNativeAssets(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	delta := int64(0)
	for _, p := range assets.Snapshot.Players {
		if p.ID == "1000" {
			delta += int64(p.Salary)
		}
		if p.ID == "4000" {
			delta -= int64(p.Salary)
		}
	}
	if delta == 0 {
		t.Fatal("fixture salaries must differ")
	}
	o := f.offer(t)
	accept := f.accept(t, o)
	if got := committed("franchise.01"); got != before+delta {
		t.Fatalf("cap did not follow contracts: %d, expected %d", got, before+delta)
	}
	f.restart(t)
	if committed("franchise.01") != before+delta {
		t.Fatal("traded cap did not replay")
	}
	f.app.core.CommissionerSession = testCommissionerSession{"creator"}
	f.revert(t, accept)
	if committed("franchise.01") != before {
		t.Fatal("cap compensation failed")
	}
}

func TestTradeDecisionRechecksCurrentRoster(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	offer := f.offer(t)
	f.accept(t, offer)
	f.vote(t, offer, "dot.a", "approve")
	f.vote(t, offer, "dot.b", "approve")
	f.set(t, "roster.limits", `{"min":0,"max":40}`, "now")
	if _, err := f.action(t, "dot.c", commissioner.TradeReview, commissioner.RawTradeArgs{ID: offer.Entry, Decision: "approve"}); err == nil {
		t.Fatal("approval moved assets into an illegal roster")
	}
	if f.only(t).Status != "review" || len(f.only(t).Votes) != 2 || f.ownerOf(t, "1000") != "franchise.02" {
		t.Fatal("refused approval changed the trade")
	}
}

func TestTradeNewVetoHasOneNewAppeal(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	offer := f.offer(t)
	f.accept(t, offer)
	f.veto(t, offer)
	f.mustAction(t, "owner.one", commissioner.TradeAppeal, commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.01"})
	f.mustAction(t, "creator", commissioner.TradeAppealRule, commissioner.RawTradeArgs{ID: offer.Entry, Decision: "approve"})
	f.mustAction(t, "creator", commissioner.TradeOverride, commissioner.RawTradeArgs{ID: offer.Entry, Decision: "veto"})
	f.mustAction(t, "owner.two", commissioner.TradeAppeal, commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.02"})
	if got := f.only(t); got.Status != "appealed" || got.VetoNumber != 2 || got.AppealedVeto != 2 || len(got.Appeals) != 3 {
		t.Fatalf("new veto appeal: %+v", got)
	}
	f.mustAction(t, "creator", commissioner.TradeAppealRule, commissioner.RawTradeArgs{ID: offer.Entry, Decision: "veto"})
	if _, err := f.action(t, "owner.one", commissioner.TradeAppeal, commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.01"}); err == nil {
		t.Fatal("confirming a veto manufactured another appeal")
	}
}

func TestTradeThreePartiesMustAllAccept(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	args := tradePlayerOffer()
	args.Parties = append(args.Parties, "franchise.03")
	args.Assets[1].To = "franchise.03"
	offer := f.mustAction(t, "owner.two", commissioner.TradeOffer, args)
	f.accept(t, offer)
	if f.only(t).Status != "offered" {
		t.Fatal("two of three parties finalized the trade")
	}
	f.mustAction(t, "owner.three", commissioner.TradeAccept, commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.03"})
	for _, who := range []string{"dot.a", "dot.b", "dot.c"} {
		f.vote(t, offer, who, "approve")
	}
	if f.only(t).Status != "approved" || f.ownerOf(t, "1000") != "franchise.01" || f.ownerOf(t, "4000") != "franchise.03" {
		t.Fatal("three-party settlement failed")
	}
}

func TestTradeConcurrentAcceptanceAppendsExactlyOnce(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	offer := f.offer(t)
	f.app.core.CommissionerSession = testCommissionerSession{"owner.one"}
	raw := f.raw(t, commissioner.TradeAccept, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.01"}}, "league")
	outcomes := make(chan error, 2)
	var wait sync.WaitGroup
	for i := 0; i < 2; i++ {
		wait.Go(func() {
			copyRaw := raw
			if i == 0 {
				copyRaw.Request = "concurrent.accept.a"
			} else {
				copyRaw.Request = "concurrent.accept.b"
			}
			_, err := newCommissionerAPI(f.app).TargetNativeTradeAction(copyRaw)
			outcomes <- err
		})
	}
	wait.Wait()
	close(outcomes)
	wins := 0
	for err := range outcomes {
		if err == nil {
			wins++
		}
	}
	if wins != 1 || f.only(t).Status != "review" {
		t.Fatalf("concurrent acceptance wins: %d", wins)
	}
	f.app.core.CommissionerSession = testCommissionerSession{"owner.three"}
	if _, err := newCommissionerAPI(f.app).TargetNativeTrades(legalityLeague, "franchise.01"); err == nil {
		t.Fatal("viewer impersonated another seated owner")
	}
}

func TestTradeCompensationCannotEraseEarlierVoteOrOffer(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	offer := f.offer(t)
	f.accept(t, offer)
	first := f.vote(t, offer, "dot.a", "approve")
	f.vote(t, offer, "dot.b", "approve")
	for _, target := range []commissioner.Receipt{first, offer} {
		f.app.core.CommissionerSession = testCommissionerSession{"creator"}
		raw := f.raw(t, commissioner.Revert, commissioner.RawArgs{Entry: target.Entry}, "league")
		body, err := json.Marshal(raw)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := newCommissionerAPI(f.app).TargetCommissionerExecute(string(body)); err == nil {
			t.Fatal("compensation erased a trade action on which a later action depends")
		}
	}
	if len(f.only(t).Votes) != 2 {
		t.Fatal("refused compensation changed votes")
	}
	f.vote(t, offer, "dot.c", "approve")
	if _, err := f.action(t, "cc", commissioner.TradeOverride, commissioner.RawTradeArgs{ID: offer.Entry, Decision: "veto"}); err == nil {
		t.Fatal("co-commissioner overrode a trade")
	}
}

func TestMutationDispatchParityAndExactlyOnce(t *testing.T) {
	for _, family := range []string{"generic", "trade", "bid", "waiver"} {
		t.Run(family, func(t *testing.T) {
			var want commissioner.Record
			var wantAssets commissioner.NativeAssets
			for _, door := range []string{"legacy", "generic", "service"} {
				f := nativeMarketApp(t)
				raw, principal := mutationProbe(t, f, family)
				f.app.core.CommissionerSession = testCommissionerSession{principal}
				before := mutationRecords(t, f)
				first, err := runMutationDoor(t, f, door, raw)
				if err != nil {
					t.Fatalf("%s: %v", door, err)
				}
				if first.Revision != int64(len(before)+1) {
					t.Fatalf("%s receipt: %+v", door, first)
				}
				after := mutationRecords(t, f)
				if len(after) != len(before)+1 {
					t.Fatal("mutation did not append exactly once")
				}
				assets, err := commissioner.NativeAssetsReading(legalityLeague, after)
				if err != nil {
					t.Fatal(err)
				}
				// Each independent held-data fixture captures its roster at wall-clock time.
				assets.Snapshot.RosterAt = time.Time{}
				if door == "legacy" {
					want, wantAssets = after[len(after)-1], assets
				} else if !reflect.DeepEqual(want, after[len(after)-1]) || !reflect.DeepEqual(wantAssets, assets) {
					t.Fatalf("%s %s changed the receipt or replayed assets", family, door)
				}
				for _, retryDoor := range []string{"legacy", "generic", "service"} {
					retry, err := runMutationDoor(t, f, retryDoor, raw)
					if err != nil || !reflect.DeepEqual(retry, first) {
						t.Fatalf("%s retry: %+v %v", retryDoor, retry, err)
					}
				}
				if got := mutationRecords(t, f); !reflect.DeepEqual(after, got) {
					t.Fatal("retry appended an event")
				}
				raw.Request = "stale.mutation"
				if _, err := runMutationDoor(t, f, door, raw); err == nil {
					t.Fatal("stale revision accepted")
				}
				if got := mutationRecords(t, f); !reflect.DeepEqual(after, got) {
					t.Fatal("refusal changed state")
				}
			}
		})
	}
}

func mutationProbe(t *testing.T, f *bidFixture, family string) (commissioner.RawCommand, string) {
	t.Helper()
	switch family {
	case "trade":
		return f.raw(t, commissioner.TradeOffer, commissioner.RawArgs{Trade: new(tradePlayerOffer())}, "league"), "owner.two"
	case "bid":
		return f.raw(t, commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{
			Player: "5004", Franchise: "franchise.01", Salary: moneyOf(5 * million), Years: 1}}, "private"), "owner.one"
	case "waiver":
		return f.raw(t, commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
			Player: "1000", Franchise: "franchise.02"}}, "league"), "owner.two"
	default:
		return f.raw(t, commissioner.RulesSet, commissioner.RawArgs{Field: "roster.limits", Value: json.RawMessage(`{"min":0,"max":90}`)}, "league"), "creator"
	}
}

func runMutationDoor(t *testing.T, f *bidFixture, door string, raw commissioner.RawCommand) (commissioner.Receipt, error) {
	t.Helper()
	api := newCommissionerAPI(f.app)
	switch {
	case door == "service":
		return leagueservice.New(commissionerHost{State: f.app.core, app: f.app}).Execute(t.Context(), raw)
	case door == "generic":
		return api.TargetCommissionerExecute(encodedCommand(t, raw))
	case commissioner.OwnerTradeAction(raw.Command) != "":
		return api.TargetNativeTradeAction(raw)
	case commissioner.OwnerBidAction(raw.Command) != "":
		return api.TargetNativeBidAction(raw)
	case commissioner.OwnerWaiverAction(raw.Command) != "":
		return api.TargetNativeWaiverAction(raw)
	default:
		return api.TargetCommissionerExecute(encodedCommand(t, raw))
	}
}

func mutationRecords(t *testing.T, f *bidFixture) []commissioner.Record {
	t.Helper()
	records, err := f.app.leagueRecords(t.Context(), legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	return records
}

func TestMutationDispatchRefusalParityAndUnchangedState(t *testing.T) {
	for _, family := range []string{"generic", "trade", "bid", "waiver"} {
		t.Run(family, func(t *testing.T) {
			f := nativeMarketApp(t)
			raw, principal := mutationProbe(t, f, family)
			f.app.core.CommissionerSession = testCommissionerSession{"stranger"}
			assertMutationRefusals(t, f, raw)
			if family == "generic" {
				return
			}
			f.set(t, "calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["rookie_draft"]}}`, "now")
			raw, _ = mutationProbe(t, f, family)
			f.app.core.CommissionerSession = testCommissionerSession{principal}
			assertMutationRefusals(t, f, raw)
		})
	}
}

func assertMutationRefusals(t *testing.T, f *bidFixture, raw commissioner.RawCommand) {
	t.Helper()
	before := mutationRecords(t, f)
	var want string
	for _, door := range []string{"legacy", "generic", "service"} {
		receipt, err := runMutationDoor(t, f, door, raw)
		if err == nil || receipt.Entry != "" {
			t.Fatalf("%s accepted refused mutation", door)
		}
		if want == "" {
			want = err.Error()
		} else if err.Error() != want {
			t.Fatalf("%s refusal: %q, want %q", door, err, want)
		}
		if after := mutationRecords(t, f); !reflect.DeepEqual(before, after) {
			t.Fatal("refusal appended an event")
		}
	}
}

type missingMutationFacts struct{ leagueservice.Host }

func (missingMutationFacts) NativeFacts() nativecheck.Facts { return nil }

func TestMutationDispatchKeepsFactsAndCancellation(t *testing.T) {
	f := nativeMarketApp(t)
	raw, principal := mutationProbe(t, f, "bid")
	f.app.core.CommissionerSession = testCommissionerSession{principal}
	host := commissionerHost{State: f.app.core, app: f.app}
	before := mutationRecords(t, f)
	if _, err := leagueservice.New(missingMutationFacts{Host: host}).Execute(t.Context(), raw); err == nil {
		t.Fatal("bid accepted without held player facts")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := leagueservice.New(host).Execute(ctx, raw); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled request: %v", err)
	}
	if after := mutationRecords(t, f); !reflect.DeepEqual(before, after) {
		t.Fatal("missing facts or cancellation appended an event")
	}
}
