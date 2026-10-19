package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/lab"
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
	a.labData = func() (*lab.Data, error) { return nil, fmt.Errorf("seeded archive failure") }
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
	r := (&App{}).TargetOffers(RawOfferScope{From: 2013, To: 2026, Direction: "all"})
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
	r := (&App{}).TargetOffers(RawOfferScope{From: 2013, To: 2026, Direction: "all"})
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
