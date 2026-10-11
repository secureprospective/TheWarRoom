package lab

import (
	"encoding/json"
	"math"
	"testing"
)

type pickGolden struct {
	ID       int      `json:"id"`
	Segment  string   `json:"segment"`
	NFL      string   `json:"nfl"`
	Judged   bool     `json:"judged"`
	Got      *float64 `json:"got"`
	Expected *float64 `json:"expected"`
}

func draftPickValues(picks []JudgedPick) []pickGolden {
	out := []pickGolden{}
	for _, p := range picks {
		out = append(out, pickGolden{p.ID, p.Segment, p.NFL, p.Judged, p.Got, p.Expected})
	}
	return out
}
func draftValues(d *Data, s DraftScopeFilter) map[string]any {
	picks := DraftScope(d, s)
	curve := SlotCurve(picks)
	positions := make(map[string]any)
	buys := make(map[string]DraftBuyReading)
	market := LeagueMarket(d, Scope{From: max(2017, s.From), To: s.To, ByAge: true})
	groups := []string{"QB", "RB", "WR", "TE", "PK", "DL", "LB", "DB"}
	for _, g := range groups {
		p := PositionDraft(picks, g)
		buys[g] = DraftBuy(market, p.All, g)
		positions[g] = map[string]any{"group": p.Group, "n": p.N, "all": draftPickValues(p.All), "overall": p.Overall, "hits": p.Hits, "kept": p.Kept, "where": p.Where, "when": p.When, "nfl": p.NFL, "age": p.Age, "curve": p.Curve}
	}
	likes := []*int{PlaysLike(curve, nil), PlaysLike(curve, ptr(0.0)), PlaysLike(curve, ptr(1.0)), PlaysLike(curve, ptr(10.0))}
	return map[string]any{"draftScope": draftPickValues(picks), "vsSlot": VsSlot(picks), "bySeason": BySeason(picks), "slotCurve": curve, "playsLike": likes, "hitRate": HitRate(picks, picks), "positionDraft": positions, "draftMap": DraftMap(picks, groups), "draftBuy": buys}
}

type draftGolden struct {
	Header struct {
		DataSHA   string `json:"dataSHA"`
		EngineSHA string `json:"engineSHA"`
		DraftSHA  string `json:"draftSHA"`
		Node      string `json:"node"`
	} `json:"header"`
	Fixtures []struct {
		Name   string                     `json:"name"`
		Scope  DraftScopeFilter           `json:"scope"`
		Values map[string]json.RawMessage `json:"values"`
	} `json:"fixtures"`
	Boundary struct {
		Picks   []pickGolden `json:"picks"`
		HitRate *DraftHits   `json:"hitRate"`
	} `json:"boundary"`
}

func readDraftGolden(t *testing.T) draftGolden {
	t.Helper()
	var g draftGolden
	if err := json.Unmarshal(readGoldenBytes(t, "testdata/goldens/draft.json.gz"), &g); err != nil {
		t.Fatal(err)
	}
	if g.Header.DataSHA != DataSHA || g.Header.EngineSHA != "d7a914bcf5f12c22b004a513d0c1a443356fad2bffc3ef6080f788283d296cfd" || g.Header.DraftSHA != "e042a4ca88588413b730c8986d6599875d171f0b96397b28e30fc2598e4cc62b" || g.Header.Node != "v24.21.0" {
		t.Fatal("invalid draft golden provenance")
	}
	return g
}
func TestDraftParity(t *testing.T) {
	d := loadData(t)
	g := readDraftGolden(t)
	maxima := make(map[string]float64)
	for _, fixture := range g.Fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			got := draftValues(d, fixture.Scope)
			for _, name := range keysRaw(fixture.Values) {
				var want any
				if err := json.Unmarshal(fixture.Values[name], &want); err != nil {
					t.Fatal(err)
				}
				maximum := 0.0
				if err := parity(want, jsonValue(t, got[name]), name, &maximum); err != nil {
					t.Error(err)
				}
				maxima[name] = math.Max(maxima[name], maximum)
			}
		})
	}
	for _, name := range keysFloat(maxima) {
		t.Logf("%-18s max absolute error %.17g", name, maxima[name])
	}
}
func TestDraftMeasured(t *testing.T) {
	picks := DraftScope(loadData(t), DraftScopeFilter{From: 2017, To: 2026, Seasons: 3})
	if judgedCount(picks) <= 1000 {
		t.Fatal("missing judged picks")
	}
	p := PositionDraft(picks, "DL")
	if p.When[0].Tone != "down" || p.When[3].Ratio == nil || *p.When[3].Ratio <= 0.9 {
		t.Fatal("measured DL late pay no longer holds")
	}
	c := SlotCurve(picks)
	for i := 2; i < len(c); i++ {
		if c[i] == nil || c[i-1] == nil || *c[i] > *c[i-1] {
			t.Fatal("curve must never rise")
		}
	}
	if *PlaysLike(c, ptr(*c[1]*2)) != 1 {
		t.Fatal("above top value not first pick")
	}
	db := PositionDraft(picks, "DB")
	if len(db.Where) != 7 || db.N != judgedCount(db.All) {
		t.Fatal("DB accounting")
	}
	for _, p := range DraftScope(loadData(t), DraftScopeFilter{From: 2017, To: 2026, Seasons: 3, Segments: []string{"r3"}, NFL: []string{"n1"}}) {
		if p.Round != 3 || p.NflRound == nil || *p.NflRound != 1 {
			t.Fatal("filter ignored")
		}
	}
}
func TestMutantsDraft(t *testing.T) {
	d := loadData(t)
	g := readDraftGolden(t)
	picks := DraftScope(d, DraftScopeFilter{From: 2017, To: 2017, Seasons: 3})
	mine := []JudgedPick{}
	for _, p := range picks {
		if p.Judged && len(mine) < 7 {
			mine = append(mine, p)
		}
	}
	if HitRate(mine, mine) != nil || g.Boundary.HitRate != nil {
		t.Fatal("normal hit boundary not null")
	}
	maximum := 0.0
	if err := parity(jsonValue(t, g.Boundary.HitRate), jsonValue(t, hitRate(mine, mine, 7)), "hitRate.threshold7", &maximum); err == nil {
		t.Fatal("draft hit threshold mutant survived")
	} else {
		t.Logf("draft mutant killed: %v", err)
	}
	if err := parity(jsonValue(t, g.Boundary.Picks), jsonValue(t, draftPickValues(mine)), "boundary.picks", &maximum); err != nil {
		t.Fatal(err)
	}
}

func TestDraftStableTiesAndClassifiers(t *testing.T) {
	group := "DB"
	picks := []JudgedPick{
		{DraftPick: DraftPick{Round: 2, Group: &group}, Judged: true},
		{DraftPick: DraftPick{Round: 1, Group: &group}, Judged: true},
	}
	if got := draftRoundOrder(picks); len(got) != 2 || got[0] != "pick2:next" {
		t.Fatal("tied modal rounds lost first-seen order")
	}
	m := Market{Rows: []MarketRow{
		{Kind: "DB:young", Cost: &Interval{Value: 1}},
		{Kind: "DB:prime", Cost: &Interval{Value: 1}},
		{Kind: "pick1:next", Cost: &Interval{Value: 1}},
		{Kind: "pick2:next", Cost: &Interval{Value: 1}},
	}}
	b := DraftBuy(m, picks, group)
	if b.Cheapest == nil || *b.Cheapest != "DB:young" || b.Via == nil || *b.Via != "pick2:next" {
		t.Fatal("stable cheapest/modal route tie")
	}
	c := markExtremes([]DraftCell{{Ratio: ptr(1.0), Lo: ptr(1.0), Hi: ptr(1.0)}, {Ratio: ptr(1.0), Lo: ptr(1.0), Hi: ptr(1.0)}})
	for _, x := range c {
		if x.Best || x.Worst {
			t.Fatal("tied cells labelled extreme")
		}
	}
	for _, x := range []struct {
		round, slot int
		want        string
	}{{1, 10, "r1a"}, {1, 11, "r1b"}, {2, 16, "r2a"}, {2, 17, "r2b"}, {3, 1, "r3"}, {4, 1, "r4"}, {5, 1, "r5"}, {6, 1, "r5"}} {
		if SegmentOf(DraftPick{Round: x.round, Slot: x.slot}) != x.want {
			t.Fatal("draft stretch boundary")
		}
	}
	if NFLOf(DraftPick{}) != "nx" || NFLOf(DraftPick{NflRound: ptr(4)}) != "n4" || ageBand(JudgedPick{DraftPick: DraftPick{Age: ptr(22.0)}}) != "a22" || ageBand(JudgedPick{DraftPick: DraftPick{Age: ptr(23.0)}}) != "a23" {
		t.Fatal("NFL/age boundary")
	}
}
