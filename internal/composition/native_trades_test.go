package composition

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func TestTradePartyRecusedAndThreeVotesDecide(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	offer := f.offer(t)
	f.accept(t, offer)
	r := f.only(t)
	if r.Status != "review" || len(r.Recusals) != 1 || r.Recusals[0].Who != "owner.two" || r.Recusals[0].Decision != "trade.recuse" {
		t.Fatalf("missing automatic recusal: %+v", r)
	}
	if _, err := f.action(t, "owner.two", commissioner.TradeReview, commissioner.RawTradeArgs{ID: offer.Entry, Decision: "veto"}); err == nil {
		t.Fatal("a party's DOT member counted")
	}
	f.vote(t, offer, "dot.a", "veto")
	f.vote(t, offer, "dot.b", "veto")
	if f.only(t).Status != "review" {
		t.Fatal("two vetoes decided a trade")
	}
	f.vote(t, offer, "dot.c", "veto")
	if f.only(t).Status != "vetoed" || f.ownerOf(t, "1000") != "franchise.02" {
		t.Fatal("three vetoes did not decide")
	}
	before := f.only(t)
	f.restart(t)
	if after := f.only(t); !reflect.DeepEqual(before, after) {
		t.Fatalf("replay: before %+v after %+v", before, after)
	}
}

func TestTradeAppealOnceAndPartyCommissionerSitsOut(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	f.command(t, "commish.seat.replace", commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.02", Principal: "creator"}}, "league")
	offer := f.mustAction(t, "creator", commissioner.TradeOffer, tradePlayerOffer())
	f.accept(t, offer)
	f.veto(t, offer)
	f.mustAction(t, "owner.one", commissioner.TradeAppeal, commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.01"})
	args := commissioner.RawTradeArgs{ID: offer.Entry, Decision: "veto"}
	if _, err := f.action(t, "creator", commissioner.TradeAppealRule, args); err == nil {
		t.Fatal("party commissioner ruled an appeal")
	}
	f.mustAction(t, "cc", commissioner.TradeAppealRule, args)
	if _, err := f.action(t, "owner.one", commissioner.TradeAppeal, commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.01"}); err == nil {
		t.Fatal("a second appeal was accepted")
	}
	if f.only(t).Status != "vetoed" || len(f.only(t).Appeals) != 2 {
		t.Fatal("appeal outcome missing")
	}
	f.restart(t)
	if _, err := f.action(t, "owner.one", commissioner.TradeAppeal, commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.01"}); err == nil {
		t.Fatal("reopen allowed a second appeal")
	}
}

func TestTradeOverridePublicMovesAssetsAndCompensates(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	offer := f.offer(t)
	f.accept(t, offer)
	f.veto(t, offer)
	for cycle := 0; cycle < 2; cycle++ {
		override := f.mustAction(t, "creator", commissioner.TradeOverride, commissioner.RawTradeArgs{ID: offer.Entry, Decision: "approve"})
		assert := func() {
			t.Helper()
			for _, v := range []struct{ who, team string }{{"owner.one", "franchise.01"}, {"owner.two", "franchise.02"}, {"owner.three", "franchise.03"}} {
				r := f.trades(t, v.who, v.team).Offers[0]
				if r.Status != "approved" || len(r.Overrides) != 1 || r.Overrides[0].Who != "creator" || r.Overrides[0].Reason == "" {
					t.Fatalf("override hidden from %s: %+v", v.who, r)
				}
			}
			if f.ownerOf(t, "1000") != "franchise.01" || f.ownerOf(t, "4000") != "franchise.02" {
				t.Fatal("approved assets did not move")
			}
		}
		assert()
		f.restart(t)
		assert()
		f.app.core.CommissionerSession = testCommissionerSession{"creator"}
		f.revert(t, override)
		if f.only(t).Status != "vetoed" || f.ownerOf(t, "1000") != "franchise.02" {
			t.Fatal("compensation did not restore assets and veto")
		}
		f.restart(t)
		if f.only(t).Status != "vetoed" || f.ownerOf(t, "4000") != "franchise.01" {
			t.Fatal("reopened compensation changed")
		}
	}
}

func TestTradeOfferAndAcceptRefuseIllegalRosterOrCap(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	f.set(t, "roster.limits", `{"min":0,"max":40}`, "now")
	if _, err := f.action(t, "owner.two", commissioner.TradeOffer, tradePlayerOffer()); err == nil || !strings.Contains(err.Error(), "roster") {
		t.Fatalf("illegal roster trade offered: %v", err)
	}
	f.set(t, "roster.limits", `{"min":0,"max":100}`, "now")
	offer := f.offer(t)
	f.set(t, "roster.limits", `{"min":0,"max":40}`, "now")
	if _, err := f.action(t, "owner.one", commissioner.TradeAccept, commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.01"}); err == nil {
		t.Fatal("a trade leaving a roster illegal was accepted")
	}
	f.set(t, "roster.limits", `{"min":0,"max":100}`, "now")
	f.set(t, "cap.amount", `100000000`, "now")
	if _, err := f.action(t, "owner.one", commissioner.TradeAccept, commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.01"}); err == nil {
		t.Fatal("a trade leaving a team over cap was accepted")
	}
	if f.only(t).Status != "offered" {
		t.Fatal("refusal wrote a decision")
	}
}

func TestTradeExpiryInstantAndCalendarDeadline(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	offer := f.offer(t)
	expires := f.only(t).ExpiresAt
	if expires.Sub(f.now) != 7*24*time.Hour || f.only(t).Expires != "Oct 20, 2026 at 5:00 AM PDT" {
		t.Fatal("offer lifetime or local time wrong")
	}
	f.now = expires.Add(-time.Nanosecond)
	if f.only(t).Status != "offered" {
		t.Fatal("offer expired early")
	}
	f.now = expires
	if _, err := f.action(t, "owner.one", commissioner.TradeAccept, commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.01"}); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("offer accepted at its expiry: %v", err)
	}
	if f.only(t).Status != "expired" {
		t.Fatal("reading did not lapse the offer")
	}
	f.now = time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	if _, err := f.action(t, "owner.two", commissioner.TradeOffer, tradePlayerOffer()); err == nil || !strings.Contains(err.Error(), "deadline") {
		t.Fatalf("trade offered after deadline: %v", err)
	}
}

func TestNativeTradeWire(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	offer := f.offer(t)
	f.accept(t, offer)
	f.vote(t, offer, "dot.a", "approve")
	reading := f.trades(t, "owner.one", "franchise.01")
	if d := reading.Offers[0].Details; len(d) != 2 || d[0].Label == "" || d[0].From == "" || d[0].To == "" || d[0].Salary == nil || d[0].Contract == "" || d[0].Ends != 2028 {
		t.Fatalf("trade asset details missing: %+v", d)
	}
	// Receipts are random identities; normalize only those identities, not any Go-authored content.
	reading.Offers[0].ID = "trade.wire"
	body, err := json.MarshalIndent(reading, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := filepath.Join("internal", "commissioner", "testdata", "native-trades-wire.json")
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(body) {
		t.Fatal("native trade wire differs; regenerate through Go")
	}
}

func TestTradeTheDOTCannotDecideGoesToTheCommissioner(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	f.set(t, "trades.dotSeats", `5`, "now")
	f.set(t, "trades.decideAt", `4`, "now")
	f.role(t, "owner.one", governance.DOT)
	offer := f.offer(t)
	f.accept(t, offer)
	if r := f.only(t); r.Status != "review" || !strings.Contains(r.Words, "the commissioner decides this trade") {
		t.Fatalf("a review the DOT cannot decide was not handed to the commissioner: %+v", r)
	}
	if _, err := f.action(t, "dot.a", commissioner.TradeOverride, commissioner.RawTradeArgs{ID: offer.Entry, Decision: "approve"}); err == nil {
		t.Fatal("a DOT member overrode the review")
	}
	f.mustAction(t, "creator", commissioner.TradeOverride, commissioner.RawTradeArgs{ID: offer.Entry, Decision: "approve"})
	if f.only(t).Status != "approved" || f.ownerOf(t, "1000") != "franchise.01" {
		t.Fatal("the commissioner's decision did not move the assets")
	}
}

func TestTradeCommissionerCannotOverrideADecidableReview(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	offer := f.offer(t)
	f.accept(t, offer)
	if _, err := f.action(t, "creator", commissioner.TradeOverride, commissioner.RawTradeArgs{ID: offer.Entry, Decision: "approve"}); err == nil {
		t.Fatal("the commissioner overrode a review the DOT can still decide")
	}
}

func TestCapCorrectionWaitsForAPendingTrade(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	offer := f.offer(t)
	salary, years := domain.Money(200000000), 2
	edit := commissioner.RawArgs{Override: &commissioner.RawNativeOverride{Player: "1000", Salary: &salary, Years: &years}}
	correct := func() error {
		body, err := json.Marshal(f.raw(t, commissioner.ContractEdit, edit, "league"))
		if err != nil {
			t.Fatal(err)
		}
		f.app.core.CommissionerSession = testCommissionerSession{"creator"}
		_, err = newCommissionerAPI(f.app).TargetCommissionerExecute(string(body))
		return err
	}
	if err := correct(); err == nil || !strings.Contains(err.Error(), "settle the trade first") {
		t.Fatalf("a contract in a pending trade was corrected underneath it: %v", err)
	}
	f.mustAction(t, "owner.one", commissioner.TradeDecline, commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.01"})
	if err := correct(); err != nil {
		t.Fatalf("a contract stayed locked after its trade was declined: %v", err)
	}
}
