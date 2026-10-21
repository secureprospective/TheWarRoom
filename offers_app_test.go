package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"

	"github.com/secureprospective/TheWarRoom/internal/app/core"

	offersapp "github.com/secureprospective/TheWarRoom/internal/app/offers"

	"github.com/secureprospective/TheWarRoom/internal/lab"
	personalstore "github.com/secureprospective/TheWarRoom/internal/store/personal"
)

func TestTargetOffers(t *testing.T) {
	t.Parallel()
	a := NewApp()
	raw := RawOfferScope{From: 2013, To: 2026, Direction: "all"}
	r := a.TargetOffers(raw)
	if r.Status != ReadingOK || r.Count != 470 || len(r.Value.Stretches) != 8 {
		t.Fatalf("bad full reading: %s %d", r.Status, r.Count)
	}
	for _, word := range []string{"Arizona Cardinals", "0025", "470", "2026-10-06", "not show other GMs"} {
		if !strings.Contains(r.Value.Disclosure, word) {
			t.Fatalf("missing disclosure %q", word)
		}
	}
	if r.HowCounted == "" || r.InputSHA != lab.DataSHA || r.Provenance.Kind != "archive" {
		t.Fatal("missing archive shell")
	}
	raw.Direction = "sent"
	sent := a.TargetOffers(raw)
	raw.Direction = "received"
	received := a.TargetOffers(raw)
	if sent.Count+received.Count != 470 || sent.Count == received.Count {
		t.Fatal("direction split does not requery engine")
	}
	raw.From = 2100
	raw.To = 2101
	thin := a.TargetOffers(raw)
	if thin.Status != ReadingThin || thin.Count != 0 || thin.AbsenceReason == nil || thin.Value.Disclosure != r.Value.Disclosure {
		t.Fatal("empty scope lost owner disclosure")
	}
	raw.Team = "invalid"
	failed := a.TargetOffers(raw)
	if failed.Status != ReadingFailed || failed.AbsenceReason == nil || failed.Value.Disclosure != r.Value.Disclosure {
		t.Fatal("invalid input not failed or header lost")
	}
	raw.Team = ""
	raw.Window = "bogus"
	if a.TargetOffers(raw).Status != ReadingFailed {
		t.Fatal("invalid window accepted")
	}
	raw.Window = ""
	raw.Direction = "bogus"
	if a.TargetOffers(raw).Status != ReadingFailed {
		t.Fatal("invalid direction accepted")
	}
	a.core.LabData = func() (*lab.Data, error) { return nil, fmt.Errorf("seeded archive failure") }
	failed = a.TargetOffers(RawOfferScope{From: 2013, To: 2026, Direction: "all"})
	if failed.Status != ReadingFailed || !strings.Contains(failed.Value.Disclosure, "cannot be verified") {
		t.Fatal("source failure invented owner")
	}
}
func TestOfferWords(t *testing.T) {
	t.Parallel()
	if offerWords().Words.Retry != "Try again" {
		t.Fatal("retry must name failure recovery")
	}
	st := lab.OfferStretch{Phase: "early", N: 1, Known: 1, PerSeason: 0, Counts: map[string]int{"unknown": 1}}
	v := offerStretchView(st)
	if v.CountText != "1 offer · 0.0 a season" || strings.Contains(v.Acceptance, "%") {
		t.Fatal("plural or thin rate failed")
	}
	if countLabel(12345, "offer") != "12,345 offers" {
		t.Fatal("count separator")
	}
	st.N = 5
	st.Known = 5
	rate := 0.0
	st.Rate = &rate
	v = offerStretchView(st)
	if !strings.Contains(v.Acceptance, "0.0%") || strings.Contains(v.Acceptance, "-0") {
		t.Fatal("unsigned zero rate failed")
	}
}
func TestOffersWireFixture(t *testing.T) {
	t.Parallel()
	if os.Getenv("TWR_WRITE_OFFERS_WIRE") != "1" {
		t.Skip("opt-in frontend IPC fixture generation")
	}
	r := (&App{raceHeld: &raceapp.Held{}, core: &core.State{}}).TargetOffers(RawOfferScope{From: 2013, To: 2026, Direction: "all"})
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("frontend/src/app/offers/wire.testdata.json", b, 0600); err != nil {
		t.Fatal(err)
	}
	b, err = json.MarshalIndent(offerWords(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("frontend/src/app/offers/fallback.gen.json", b, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestOfferStackWidthsAndCounts(t *testing.T) {
	t.Parallel()
	r := (&App{raceHeld: &raceapp.Held{}, core: &core.State{}}).TargetOffers(RawOfferScope{From: 2013, To: 2026, Direction: "all"})
	for _, st := range r.Value.Stretches {
		if len(st.Outcomes) != 6 {
			t.Fatal("stack requires five endings and open offers")
		}
		sum := 0.0
		for _, o := range st.Outcomes {
			sum += o.Width
			if o.Text == "" || o.Label == "" {
				t.Fatal("stack loses non-colour explanation")
			}
		}
		if sum < 99.97 || sum > 100.03 {
			t.Fatalf("stack widths total %f", sum)
		}
	}
}

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
	checkGolden(t, "offers", "offers", a.TargetOffers(RawOfferScope{From: 2013, To: 2026, Direction: "all"}))
	personal, err := a.TargetPersonal("0001")
	checkGolden(t, "offers", "personal", personal)
	checkGolden(t, "offers", "personal-error", errorWords(err))
	checkGolden(t, "offers", "personal-change", errorWords(a.TargetPersonalChange(personalstore.RawChange{})))
	checkGolden(t, "offers", "personal-handoff", errorWords(a.TargetPersonalHandOff("invalid")))
	checkGolden(t, "offers", "pool", a.TargetPool(RawPoolScope{Sort: "name"}))
	a.core.StartupErr = fmt.Errorf("fixture startup failure")
	checkGolden(t, "offers", "pool-refresh", errorWords(a.TargetPoolRefresh(false)))
	checkGolden(t, "offers", "preview", a.TargetBidPreview(rawBid("5", 1)))
	plan, err := a.TargetBidDraft(rawBid("5", 1))
	checkGolden(t, "offers", "draft", plan)
	checkGolden(t, "offers", "draft-error", errorWords(err))
	for name, call := range map[string]func(string) (BidPlan, error){"handoff": a.TargetBidHandOff, "posted": a.TargetBidPosted, "withdraw": a.TargetBidWithdraw} {
		plan, err = call("missing")
		checkGolden(t, "offers", name, plan)
		checkGolden(t, "offers", name+"-error", errorWords(err))
	}
	plan, err = a.TargetBidRevise("missing", rawBid("5", 1))
	checkGolden(t, "offers", "revise", plan)
	checkGolden(t, "offers", "revise-error", errorWords(err))
	raw, err := json.Marshal(a.TargetBidPlans("0001"))
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
