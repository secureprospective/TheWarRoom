package main

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/lab"
)

func TestMarketSurfaceMethods(t *testing.T) {
	a := NewApp()
	full := RawMarketScope{From: 2017, To: 2026}
	methods := []struct {
		name string
		read func(RawMarketScope) Reading[MarketSurface]
	}{
		{"lists", a.TargetMarketLists}, {"clock", a.TargetPickClock}, {"stretches", a.TargetMarketStretches},
	}
	for _, method := range methods {
		t.Run(method.name, func(t *testing.T) {
			ok := method.read(full)
			if ok.Status != ReadingOK || ok.Count == 0 || ok.HowCounted == "" || ok.Cutoff == "" || ok.InputSHA != lab.DataSHA || len(ok.Value.Sections) == 0 {
				t.Fatalf("incomplete reading: %+v", ok)
			}
			wire, err := json.Marshal(ok)
			if err != nil {
				t.Fatal(err)
			}
			for _, grade := range []string{"\"solid\"", "\"rough\"", "\"evidence\""} {
				if strings.Contains(string(wire), grade) {
					t.Fatal("grade leaked")
				}
			}
			thin := method.read(RawMarketScope{From: 2100, To: 2101})
			if thin.Status != ReadingThin || thin.AbsenceReason == nil || thin.Count != 0 {
				t.Fatal("absent scope fabricated data")
			}
			failed := method.read(RawMarketScope{From: 2026, To: 2017})
			if failed.Status != ReadingFailed || failed.AbsenceReason == nil {
				t.Fatal("invalid IPC accepted")
			}
		})
	}
}
func TestMarketListsEngineParity(t *testing.T) {
	a := NewApp()
	d, err := a.lab()
	if err != nil {
		t.Fatal(err)
	}
	raw := RawMarketScope{From: 2017, To: 2026, Assets: []string{"LB"}}
	got := a.TargetMarketLists(raw)
	engine := lab.LeagueMarket(d, marketScope(raw))
	want := marketSections()
	for _, r := range engine.Rows {
		if lab.AssetOf(r.Kind) == "LB" {
			appendMarketRow(want, r)
		}
	}
	for i, section := range got.Value.Sections {
		if !reflect.DeepEqual(section.Rows, want[i].Rows) {
			t.Fatalf("section %s differs from engine", section.ID)
		}
		for _, r := range section.Rows {
			if r.Key != "LB" {
				t.Fatal("asset filter ignored")
			}
		}
	}
	all := a.TargetMarketLists(RawMarketScope{From: 2017, To: 2026})
	if got.Value.Summary != all.Value.Summary {
		t.Fatal("asset chip changed going rate")
	}
	age := a.TargetMarketLists(RawMarketScope{From: 2017, To: 2026, Assets: []string{"LB"}, ByAge: true})
	found := false
	for _, section := range age.Value.Sections {
		for _, r := range section.Rows {
			found = found || strings.HasPrefix(r.Key, "LB:")
		}
	}
	if !found {
		t.Fatal("age split ignored")
	}
	empty := a.TargetMarketLists(RawMarketScope{From: 2017, To: 2026, Phases: []string{}, Assets: []string{}})
	if !reflect.DeepEqual(empty.Value, all.Value) {
		t.Fatal("empty wire arrays narrowed scope")
	}
	winter := a.TargetMarketLists(RawMarketScope{From: 2017, To: 2026, Phases: []string{"winter"}})
	if winter.Status != ReadingThin || winter.Count == 0 {
		t.Fatal("stretch scope ignored")
	}
}
func TestMarketClockEngineParity(t *testing.T) {
	a := NewApp()
	d, err := a.lab()
	if err != nil {
		t.Fatal(err)
	}
	raw := RawMarketScope{From: 2017, To: 2026}
	clock := lab.PickClock(d, marketScope(raw))
	got := a.TargetPickClock(raw)
	if len(got.Value.Sections[0].Rows) != len(clock.Halves) {
		t.Fatal("missing clock half")
	}
	for i, h := range clock.Halves {
		line := got.Value.Sections[0].Rows[i]
		if h.Clock == nil {
			if line.Range != nil {
				t.Fatal("invented clock")
			}
			continue
		}
		if line.Count != h.Trades || line.Range.Value != h.Clock.Value || line.Range.Lo != h.Clock.Lo || line.Range.Hi != h.Clock.Hi {
			t.Fatal("clock differs from engine")
		}
	}
}
func TestMarketStretchEngineParity(t *testing.T) {
	a := NewApp()
	d, err := a.lab()
	if err != nil {
		t.Fatal(err)
	}
	raw := RawMarketScope{From: 2017, To: 2026, Assets: []string{"LB"}}
	scope := marketScope(raw)
	scope.To = d.LastFinished
	totals, err := lab.TradesByPhase(d, scope)
	if err != nil {
		t.Fatal(err)
	}
	got := a.TargetMarketStretches(raw)
	if got.Count != totals.Total || got.Threshold != 1 {
		t.Fatal("wrong completed count")
	}
	for _, line := range got.Value.Sections[0].Rows {
		if line.Count != totals.Phases[line.Key].N {
			t.Fatal("stretch differs from engine")
		}
	}
}
func TestAssetProfileEngineParityAndStates(t *testing.T) {
	a := NewApp()
	d, err := a.lab()
	if err != nil {
		t.Fatal(err)
	}
	raw := RawMarketAsset{Scope: RawMarketScope{From: 2017, To: 2026}, Kind: "LB"}
	got := a.TargetAssetProfile(raw)
	s := marketScope(raw.Scope)
	want := lab.AssetProfile(d, s, raw.Kind, lab.StretchPrices(d, s, marketPhases()))
	if got.Status != ReadingOK || !reflect.DeepEqual(got.Value.Sections, profileSections(want, profileDeliveryCount(d, s, raw.Kind))) {
		t.Fatal("profile differs from engine")
	}
	if len(got.Value.Sections) != 5 {
		t.Fatal("profile missing sections")
	}
	for i, w := range want.When {
		line := got.Value.Sections[1].Rows[i]
		if (line.Mark == "cheapest") != w.Cheapest || (line.Mark == "dearest") != w.Dearest {
			t.Fatal("chance mark changed")
		}
	}
	thin := a.TargetAssetProfile(RawMarketAsset{Scope: RawMarketScope{From: 2100, To: 2101}, Kind: "LB"})
	if thin.Status != ReadingThin || thin.AbsenceReason == nil || len(thin.Value.Sections) != 0 {
		t.Fatal("thin profile invented data")
	}
	for _, bad := range []RawMarketAsset{{Scope: raw.Scope, Kind: "unknown"}, {Scope: raw.Scope, Kind: "LB:prime"}, {Scope: raw.Scope, Kind: "pick4:next"}} {
		if a.TargetAssetProfile(bad).Status != ReadingFailed {
			t.Fatal("invalid asset accepted")
		}
	}
	raw.Scope.ByAge = true
	raw.Kind = "LB:prime"
	if a.TargetAssetProfile(raw).Status != ReadingOK {
		t.Fatal("valid age profile rejected")
	}
}
func TestMarketInputValidationAndLoadFailure(t *testing.T) {
	for _, raw := range []RawMarketScope{{From: 2017, To: 2026, Assets: []string{"bogus"}}, {From: 2017, To: 2026, Assets: []string{"LB", "LB"}}} {
		if raw.Validate() == nil {
			t.Fatal("invalid asset selection accepted")
		}
	}
	a := &App{labData: func() (*lab.Data, error) { return nil, errors.New("fingerprint mismatch") }}
	for _, r := range []Reading[MarketSurface]{a.TargetMarketLists(RawMarketScope{From: 2017, To: 2026}), a.TargetPickClock(RawMarketScope{From: 2017, To: 2026}), a.TargetMarketStretches(RawMarketScope{From: 2017, To: 2026}), a.TargetAssetProfile(RawMarketAsset{Scope: RawMarketScope{From: 2017, To: 2026}, Kind: "LB"})} {
		if r.Status != ReadingFailed || r.AbsenceReason == nil || !strings.Contains(*r.AbsenceReason, "fingerprint mismatch") {
			t.Fatal("load failure hidden")
		}
	}
}
func TestMarketRangeWire(t *testing.T) {
	r := marketRange(&lab.Interval{Value: 0.4, Lo: 0.2, Hi: 0.8})
	if math.Abs(r.Value-0.4) > 1e-12 || r.Text != "40.0%; likely 20.0% to 80.0%" {
		t.Fatal("wrong displayed range")
	}
	if marketRange(nil) != nil {
		t.Fatal("null range fabricated")
	}
}
