package lab

import (
	"encoding/json"
	"testing"
)

type offersGolden struct {
	Header struct {
		DataSHA   string `json:"dataSHA"`
		EngineSHA string `json:"engineSHA"`
		OffersSHA string `json:"offersSHA"`
		Node      string `json:"node"`
	} `json:"header"`
	Fixtures []struct {
		Scope OfferScope      `json:"scope"`
		Value json.RawMessage `json:"value"`
	} `json:"fixtures"`
}

func offerGoldens(t *testing.T) offersGolden {
	t.Helper()
	var g offersGolden
	if err := json.Unmarshal(readGoldenBytes(t, "testdata/goldens/offers.json.gz"), &g); err != nil {
		t.Fatal(err)
	}
	return g
}
func TestOffersParity(t *testing.T) {
	d := loadData(t)
	g := offerGoldens(t)
	if g.Header.DataSHA != DataSHA || g.Header.EngineSHA != "d7a914bcf5f12c22b004a513d0c1a443356fad2bffc3ef6080f788283d296cfd" || g.Header.OffersSHA != "22959dcc4fd77dc28a7d14328dfeb9added0fba19c20346829142f36dcba1b59" || g.Header.Node != "v24.21.0" {
		t.Fatal("offers provenance mismatch")
	}
	maxError := 0.0
	for _, f := range g.Fixtures {
		var want any
		if err := json.Unmarshal(f.Value, &want); err != nil {
			t.Fatal(err)
		}
		if err := parity(want, jsonValue(t, OfferStretches(d, f.Scope)), "offerStretches", &maxError); err != nil {
			t.Fatalf("scope %+v: %v", f.Scope, err)
		}
	}
	t.Logf("offerStretches: %d scopes; max absolute error %.17g", len(g.Fixtures), maxError)
	if len(d.Offers) != 470 || OfferOwner(d) != "0025" {
		t.Fatal("archive offer ownership changed")
	}
	for _, o := range d.Offers {
		if o.By != "0025" && o.To != "0025" {
			t.Fatal("offer outside Cardinals ownership")
		}
	}
}
func TestMutantsOffers(t *testing.T) {
	d := loadData(t)
	g := offerGoldens(t)
	killed := false
	for _, f := range g.Fixtures {
		var want any
		if err := json.Unmarshal(f.Value, &want); err != nil {
			t.Fatal(err)
		}
		maxError := 0.0
		if err := parity(want, jsonValue(t, offerStretches(d, f.Scope, 10, "rejected")), "offers.unknownAsRejected", &maxError); err != nil {
			killed = true
			t.Logf("unknown-ending denominator mutant killed: %v", err)
			break
		}
	}
	if !killed {
		t.Fatal("offers unknown-ending denominator mutant survived")
	}
}
func TestOfferBoundaries(t *testing.T) {
	d := &Data{Offers: []Offer{{By: "a", To: "b", Season: 2020, Phase: "early", Outcome: "unknown"}}}
	r := OfferStretches(d, OfferScope{From: 2020, To: 2020})
	if r.Owner != "a" || r.Rate != nil || r.Stretches[5].Rate != nil || r.Seasons != 1 {
		t.Fatal("unknown ending or stable owner tie failed")
	}
	for _, n := range []int{9, 10} {
		st := []OfferStretch{{Known: n, Rate: ptr(1.0), SE: ptr(0.00001)}, {Known: 10, Rate: ptr(0.0), SE: ptr(0.00001)}}
		offerExtremes(st, 10)
		if st[0].Best != (n == 10) || st[1].Worst != (n == 10) {
			t.Fatal("ten-ending threshold failed")
		}
	}
}

func TestRawOfferOutcomeValidation(t *testing.T) {
	d := loadData(t)
	raw := d.Raw.Offers
	raw.Outcome = append([]string{}, raw.Outcome...)
	raw.Outcome[0] = "invented"
	if err := raw.Validate(); err == nil {
		t.Fatal("unrecognized ending accepted")
	}
}
