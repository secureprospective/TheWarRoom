package lab

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"slices"
	"testing"
)

func TestMeasuredMarket(t *testing.T) {
	d := loadData(t)
	s := Scope{From: 2017, To: 2026}
	m := LeagueMarket(d, s)
	for _, k := range []string{"LB", Anchor} {
		var row *MarketRow
		for i := range m.Rows {
			if m.Rows[i].Kind == k {
				row = &m.Rows[i]
			}
		}
		if row == nil {
			t.Fatal(k)
		}
		side := "cheap"
		if k == Anchor {
			side = "dear"
		}
		if row.Side == nil || *row.Side != side || !row.Holds {
			t.Fatalf("%s must be %s in both halves", k, side)
		}
	}
	clock := PickClock(d, s)
	if len(clock.Halves) != 2 || clock.Halves[0].To >= clock.Halves[1].From {
		t.Fatal("halves overlap")
	}
	for _, h := range clock.Halves {
		if h.Clock == nil || !h.Clock.Clear || h.Clock.Value >= 0.5 {
			t.Fatal("later first must clear half next first")
		}
	}
	p := AssetProfile(d, s, "LB", StretchPrices(d, s, []string{"winter", "opening", "between", "rookieDraft", "summer", "early", "midseason", "deadline"}))
	if p.Paid.Hi >= 1 || p.Delivered.Ratio-Z*p.Delivered.SE <= 1 {
		t.Fatal("LB must be underpriced and over-deliver")
	}
	sum := 0.0
	found := false
	for _, a := range p.Against {
		if a.Kind == Anchor {
			found = true
			if !a.ByDelivered || a.Hi >= 1 || a.Side != "cheaper" {
				t.Fatal("LB vs first")
			}
		}
	}
	if !found {
		t.Fatal("missing first comparison")
	}
	for _, b := range p.Back {
		sum += b.Share
	}
	if math.Abs(sum-1) > 1e-9 {
		t.Fatal("return shares")
	}
	aged := LeagueMarket(d, Scope{From: 2017, To: 2026, ByAge: true})
	if *aged.Going != *m.Going {
		t.Fatal("age changed yardstick")
	}
	season := LeagueMarket(d, Scope{From: 2017, To: 2026, Phases: []string{"early", "midseason", "deadline"}})
	if season.Trades <= 400 || season.Trades >= m.Trades {
		t.Fatal("season scope")
	}
	winter := LeagueMarket(d, Scope{From: 2017, To: 2026, Phases: []string{"winter"}})
	if winter.Rows != nil || winter.Trades <= 0 {
		t.Fatal("thin winter")
	}
}
func TestBoundaries(t *testing.T) {
	for _, n := range []int{7, 8} {
		records := make([]Delivery, n)
		for i := range records {
			records[i] = Delivery{Expected: 1, Delivered: 1}
		}
		if (DeliveredRatio(records) != nil) != (n == 8) {
			t.Fatalf("threshold %d", n)
		}
	}
	if DeliveredRatio(make([]Delivery, 8)) != nil {
		t.Fatal("zero expectation")
	}
	if PerDelivered(&Price{Price: -1}, &Delivered{Ratio: 1}) != nil || PerDelivered(nil, nil) != nil {
		t.Fatal("invalid ratios")
	}
	if MeanVar(nil).Mean != nil || *MeanVar([]float64{2}).Var != 0 {
		t.Fatal("mean variance")
	}
	if evidence(&Range{Spread: 2.5}) != "solid" || evidence(&Range{Spread: 6}) != "rough" {
		t.Fatal("spread boundaries")
	}
	d := loadData(t)
	for _, n := range []int{39, 40, 59, 60, 79, 80} {
		small := *d
		small.Trades = append([]Trade{}, d.Trades[:n]...)
		small.Delivered = nil
		for i := range small.Trades {
			small.Trades[i].ID = i
			small.Trades[i].ANet = ptr(0.0)
			small.Trades[i].Season = ptr(2020)
			small.Trades[i].Window = ptr("summer")
			small.Trades[i].Phase = "summer"
			small.Trades[i].Legs = nil
		}
		s := Scope{From: 2020, To: 2020}
		if (PriceList(&small, s) != nil) != (n >= 40) {
			t.Fatalf("list %d", n)
		}
		if (LeagueMarket(&small, s).Rows != nil) != (n >= 80) {
			t.Fatalf("market %d", n)
		}
		if (StretchPrices(&small, s, []string{"summer"})[0].Fit != nil) != (n >= 60) {
			t.Fatalf("stretch %d", n)
		}
	}
}
func TestDecoder(t *testing.T) {
	d := loadData(t)
	if len(d.Trades) != 2634 || len(d.Legs) != 10316 || len(d.Players) != 3176 || d.LastFinished != 2025 {
		t.Fatal("inventory")
	}
	for _, l := range d.Legs {
		if l.Trade < 0 || l.Trade >= len(d.Trades) {
			t.Fatal("link")
		}
	}
	if _, err := DecodeGzip(frozen, "wrong"); err == nil {
		t.Fatal("hash mismatch accepted")
	}
	raw := d.Raw
	raw.Legs.Trade = append([]int{}, raw.Legs.Trade...)
	raw.Legs.Trade[0] = -1
	if err := raw.Validate(); err == nil {
		t.Fatal("bad link")
	}
	raw = d.Raw
	raw.Trades.Ts = nil
	if err := raw.Validate(); err == nil {
		t.Fatal("ragged columns")
	}
	raw = d.Raw
	raw.Schema = "future"
	b, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	var compressed bytes.Buffer
	w := gzip.NewWriter(&compressed)
	if _, err := w.Write(b); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	if _, err := DecodeGzip(compressed.Bytes(), hex.EncodeToString(sum[:])); err == nil {
		t.Fatal("unsupported schema")
	}
}

func TestLoadImmutableAcrossScopes(t *testing.T) {
	d := loadData(t)
	fresh, err := DecodeGzip(frozen, DataSHA)
	if err != nil {
		t.Fatal(err)
	}
	computed(d, Scope{From: 2024, To: 2026, ByAge: true}, "age-last3")
	second := Scope{From: 2017, To: 2026, Phases: []string{"early", "midseason", "deadline"}}
	if !reflect.DeepEqual(computed(d, second, "season"), computed(fresh, second, "season")) {
		t.Fatal("first scope changed second scope results")
	}
	if !reflect.DeepEqual(d, fresh) {
		t.Fatal("engine mutated frozen facts")
	}
	for i := range 8 {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			shared := d // concurrent readers share one decode, as the App does
			if !reflect.DeepEqual(computed(shared, second, "season"), computed(fresh, second, "season")) {
				t.Fatal("concurrent scope results changed")
			}
		})
	}
}

func TestStableReturnTies(t *testing.T) {
	d := loadData(t)
	zero := *d
	zero.Trades = slices.Clone(d.Trades)
	for i := range zero.Trades {
		zero.Trades[i].Legs = slices.Clone(d.Trades[i].Legs)
		for j := range zero.Trades[i].Legs {
			zero.Trades[i].Legs[j].XValue = ptr(0.0)
		}
	}
	for _, tc := range []struct {
		name, kind, first, second, inputFirst, inputSecond string
		scope                                              Scope
	}{
		{"age-last3", "LB:young", "pick3:next", "pick3:later", "pick3:next", "pick3:later", Scope{From: 2024, To: 2026, ByAge: true}},
		{"season", "QB", "LB", "pick1:later", "pick1:later", "LB", Scope{From: 2017, To: 2026, Phases: []string{"early", "midseason", "deadline"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			trades := ValuedTrades(d, InScope(tc.scope))
			back, _, _ := profileReturns(d, trades, tc.kind, classifier(tc.scope.ByAge))
			i := slices.IndexFunc(back, func(r Return) bool { return r.Kind == tc.first })
			if i < 0 || i+1 >= len(back) || back[i+1].Kind != tc.second || math.Abs(back[i].Wins-back[i+1].Wins) >= 1e-12 {
				t.Fatalf("reference near-tie order changed: %+v", back)
			}
			tied, _, _ := profileReturns(&zero, trades, tc.kind, classifier(tc.scope.ByAge))
			first := slices.IndexFunc(tied, func(r Return) bool { return r.Kind == tc.inputFirst })
			second := slices.IndexFunc(tied, func(r Return) bool { return r.Kind == tc.inputSecond })
			if first < 0 || second <= first {
				t.Fatalf("exact tie lost JS input order: %+v", tied)
			}
		})
	}
}
