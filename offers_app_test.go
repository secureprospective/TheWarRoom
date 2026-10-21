package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	offersapp "github.com/secureprospective/TheWarRoom/internal/app/offers"

	personalstore "github.com/secureprospective/TheWarRoom/internal/store/personal"
)

func TestOffersClusterDenialsGolden(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	denials := map[string]string{}
	for _, who := range []string{"stranger", "owner.two", "creator"} {
		denials[who] = errorWords(f.tryPlace(t, who, "franchise.01", "5002", 5*million, 1))
	}
	f.app.core.CommissionerSession = testCommissionerSession{"owner.one"}
	_, err := f.app.TargetCommissionerMarket(legalityLeague, "bids.award.override")
	denials["non-commissioner"] = errorWords(err)
	f.set(t, "calendar.windows", `{}`, "now")
	denials["closed"] = errorWords(f.tryPlace(t, "owner.one", "franchise.01", "5002", 5*million, 1))
	checkGolden(t, "offers", "denials", denials)
}

func TestOffersClusterWireGolden(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	checkGolden(t, "offers", "offers", offersTestAPI(a).TargetOffers(RawOfferScope{From: 2013, To: 2026, Direction: "all"}))
	personal, err := offersTestAPI(a).TargetPersonal("0001")
	checkGolden(t, "offers", "personal", personal)
	checkGolden(t, "offers", "personal-error", errorWords(err))
	checkGolden(t, "offers", "personal-change", errorWords(offersTestAPI(a).TargetPersonalChange(personalstore.RawChange{})))
	checkGolden(t, "offers", "personal-handoff", errorWords(offersTestAPI(a).TargetPersonalHandOff("invalid")))
	checkGolden(t, "offers", "pool", offersTestAPI(a).TargetPool(RawPoolScope{Sort: "name"}))
	a.core.StartupErr = fmt.Errorf("fixture startup failure")
	checkGolden(t, "offers", "pool-refresh", errorWords(offersTestAPI(a).TargetPoolRefresh(false)))
	checkGolden(t, "offers", "preview", offersTestAPI(a).TargetBidPreview(rawBid("5", 1)))
	plan, err := offersTestAPI(a).TargetBidDraft(rawBid("5", 1))
	checkGolden(t, "offers", "draft", plan)
	checkGolden(t, "offers", "draft-error", errorWords(err))
	for name, call := range map[string]func(string) (BidPlan, error){"handoff": offersTestAPI(a).TargetBidHandOff, "posted": offersTestAPI(a).TargetBidPosted, "withdraw": offersTestAPI(a).TargetBidWithdraw} {
		plan, err = call("missing")
		checkGolden(t, "offers", name, plan)
		checkGolden(t, "offers", name+"-error", errorWords(err))
	}
	plan, err = offersTestAPI(a).TargetBidRevise("missing", rawBid("5", 1))
	checkGolden(t, "offers", "revise", plan)
	checkGolden(t, "offers", "revise-error", errorWords(err))
	raw, err := json.Marshal(offersTestAPI(a).TargetBidPlans("0001"))
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "offers", "plans", json.RawMessage(normalizeWire(raw)))
}

func TestOffersWireTwinsMatch(t *testing.T) {
	t.Parallel()
	var full offersapp.View
	n := 0
	fillWire(reflect.ValueOf(&full).Elem(), &n)
	service, err := json.Marshal(full)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := json.Marshal(wireOffers(full))
	if err != nil {
		t.Fatal(err)
	}
	if string(service) != string(wire) {
		t.Fatalf("wire twin differs: %s != %s", wire, service)
	}
}

func TestOffersWireMarketPortDetachment(t *testing.T) {
	number := 12.5
	input := map[string][]MarketFact{
		"player": {{value: &number, Range: &MarketRange{Text: "held"}}},
		"nil":    nil,
		"empty":  {},
	}
	captured := offersMarketFacts(input)
	number = 99
	input["player"][0].Range.Text = "changed input"
	if *captured["player"][0].Value != 12.5 || captured["player"][0].Range.Text != "held" {
		t.Fatal("market capture aliases input")
	}
	*captured["player"][0].Value = 1
	captured["player"][0].Range.Text = "changed output"
	if number != 99 || input["player"][0].Range.Text != "changed input" {
		t.Fatal("market output aliases input")
	}
	if captured["nil"] != nil || captured["empty"] == nil {
		t.Fatal("nil/empty market cells changed")
	}
}
