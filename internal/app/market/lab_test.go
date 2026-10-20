package market

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/lab"
)

// asset is an asset profile request: a scope and an asset kind.
type asset struct {
	Scope Scope
	Kind  string
}

func testLab(t *testing.T) Lab {
	t.Helper()
	d, err := lab.Load()
	if err != nil {
		t.Fatal(err)
	}
	return Lab{Data: d, Stamp: "test build"}
}

func TestMarketListsEngineParity(t *testing.T) {
	t.Parallel()
	a := testLab(t)
	d := a.Data
	raw := Scope{From: 2017, To: 2026, Assets: []string{"LB"}}
	got := a.Lists(raw)
	engine := lab.LeagueMarket(d, LabScope(raw))
	want := listSections()
	for _, r := range engine.Rows {
		if lab.AssetOf(r.Kind) == "LB" {
			appendListRow(want, r)
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
	all := a.Lists(Scope{From: 2017, To: 2026})
	if got.Value.Summary != all.Value.Summary {
		t.Fatal("asset chip changed going rate")
	}
	age := a.Lists(Scope{From: 2017, To: 2026, Assets: []string{"LB"}, ByAge: true})
	found := false
	for _, section := range age.Value.Sections {
		for _, r := range section.Rows {
			found = found || strings.HasPrefix(r.Key, "LB:")
		}
	}
	if !found {
		t.Fatal("age split ignored")
	}
	empty := a.Lists(Scope{From: 2017, To: 2026, Phases: []string{}, Assets: []string{}})
	if !reflect.DeepEqual(empty.Value, all.Value) {
		t.Fatal("empty wire arrays narrowed scope")
	}
	winter := a.Lists(Scope{From: 2017, To: 2026, Phases: []string{"winter"}})
	if winter.Status != report.ReadingThin || winter.Count == 0 {
		t.Fatal("stretch scope ignored")
	}
}

func TestMarketClockEngineParity(t *testing.T) {
	t.Parallel()
	a := testLab(t)
	d := a.Data
	raw := Scope{From: 2017, To: 2026}
	clock := lab.PickClock(d, LabScope(raw))
	got := a.PickClock(raw)
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
	t.Parallel()
	a := testLab(t)
	d := a.Data
	raw := Scope{From: 2017, To: 2026, Assets: []string{"LB"}}
	scope := LabScope(raw)
	scope.To = d.LastFinished
	totals, err := lab.TradesByPhase(d, scope)
	if err != nil {
		t.Fatal(err)
	}
	got := a.Stretches(raw)
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
	t.Parallel()
	a := testLab(t)
	d := a.Data
	raw := asset{Scope: Scope{From: 2017, To: 2026}, Kind: "LB"}
	got := a.Profile(raw.Scope, raw.Kind)
	s := LabScope(raw.Scope)
	want := lab.AssetProfile(d, s, raw.Kind, lab.StretchPrices(d, s, Phases()))
	if got.Status != report.ReadingOK || !reflect.DeepEqual(got.Value.Sections, profileSections(want, profileDeliveryCount(d, s, raw.Kind))) {
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
	thin := a.Profile(Scope{From: 2100, To: 2101}, "LB")
	if thin.Status != report.ReadingThin || thin.AbsenceReason == nil || len(thin.Value.Sections) != 0 {
		t.Fatal("thin profile invented data")
	}
	for _, bad := range []asset{{Scope: raw.Scope, Kind: "unknown"}, {Scope: raw.Scope, Kind: "LB:prime"}, {Scope: raw.Scope, Kind: "pick4:next"}} {
		if ValidateAsset(bad.Scope, bad.Kind) == nil {
			t.Fatal("invalid asset accepted")
		}
	}
	raw.Scope.ByAge = true
	raw.Kind = "LB:prime"
	if ValidateAsset(raw.Scope, raw.Kind) != nil || a.Profile(raw.Scope, raw.Kind).Status != report.ReadingOK {
		t.Fatal("valid age profile rejected")
	}
}

func TestRangeOfWordsTheInterval(t *testing.T) {
	t.Parallel()
	r := RangeOf(&lab.Interval{Value: 0.4, Lo: 0.2, Hi: 0.8})
	if math.Abs(r.Value-0.4) > 1e-12 || r.Text != "40.0%; likely 20.0% to 80.0%" {
		t.Fatal("wrong displayed range")
	}
	if RangeOf(nil) != nil {
		t.Fatal("null range fabricated")
	}
}

func TestMarketAnchorClassification(t *testing.T) {
	t.Parallel()
	a := testLab(t)
	raw := Scope{From: 2017, To: 2026}
	lists := a.Lists(raw)
	for _, section := range lists.Value.Sections[:2] {
		for _, row := range section.Rows {
			if row.Key == lab.Anchor {
				t.Fatal("anchor classified as paid cheap/dear")
			}
		}
	}
	found := false
	for _, row := range lists.Value.Sections[3].Rows {
		found = found || row.Key == lab.Anchor
	}
	if !found {
		t.Fatal("Lab retains anchor in delivered sell list")
	}
	p := a.Profile(raw, lab.Anchor)
	paid := p.Value.Sections[0].Rows[0]
	if paid.Range != nil || !strings.Contains(paid.Detail, "Every other price is measured against this one") {
		t.Fatal("anchor profile must explain fixed price without bar")
	}
}
