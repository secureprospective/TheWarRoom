package labcompile

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/facts"
)

func TestConservativeValueBoundaries(t *testing.T) {
	// Empty design, constant targets, and constant gains are undefined in Python.
	for _, cost := range []bool{false, true} {
		if prices, meta := fitPrices(nil, []string{anchor}, cost); prices != nil || meta != nil {
			t.Fatal("empty fit fabricated prices")
		}
	}
	pairs := make([][2]float64, 30)
	for i := range pairs {
		pairs[i] = [2]float64{1, 2}
	}
	if forecastCheck(pairs) != nil || forecastCheck(pairs[:29]) != nil {
		t.Fatal("undefined forecast accuracy")
	}
	if summarizeBacktest(nil, 2025) != nil || summarizeBacktest([]backtestRow{{1, 2, 2020}, {1, 3, 2021}}, 2025) != nil {
		t.Fatal("undefined backtest slope")
	}
	if deliveredRatio(make([]deliveryObservation, 8)) != nil || deliveredRatio([]deliveryObservation{{1, 1, 2020}}) != nil {
		t.Fatal("unsupported delivery ratio")
	}
	if deliveredInterval(1, 0, 0, 0) != nil || deliveredInterval(1, 1, 1, 1e300) != nil {
		t.Fatal("undefined delivery interval")
	}
	empty := market(nil, []string{anchor}, nil)
	if len(empty.Classes) != 0 || empty.PerCapMillion != nil || empty.PerCapMillionSE != nil || empty.CapTrades != 0 {
		t.Fatal("empty market fabricated evidence")
	}
	l, err := newLab(facts.Facts{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	w := newWins(l)
	f := newForecast(w, nil)
	p := newPicks(w, nil, f.last)
	if f.last != 0 || len(f.check()) != 0 || w.currencyCheck() != nil || p.rates(1, 0) != nil {
		t.Fatal("empty archive fabricated evidence")
	}
	if len(w.spread) != 0 {
		t.Fatal("unexpected spread")
	}
	v := newValuer(w, f, p, nil, nil, 0)
	if v.player("absent", 2026, 0) != nil || v.realizedPick(Asset{Kind: "pick", Year: ptr(int64(2026)), Round: ptr(int64(1))}, 2026) != nil {
		t.Fatal("absent asset fabricated value")
	}
}

func TestAssetValueUnavailable(t *testing.T) {
	l, err := newLab(facts.Facts{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	w := newWins(l)
	f := newForecast(w, nil)
	m := &Model{valuer: newValuer(w, f, newPicks(w, nil, 0), nil, nil, 0)}
	at := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	var absent *Model
	for _, got := range []AssetValue{absent.Player("absent", at), m.Player("absent", at), m.Player("absent", time.Time{}), m.Pick(2026, 0, 0, "0001", at), m.Pick(2026, 1, 0, "0001", at), m.Player("absent", at.AddDate(1, 0, 0))} {
		if got.Valued || got.Reason == "" || got.Forecast != nil || got.Price != nil || got.Interval != nil {
			t.Fatalf("fabricated asset value: %+v", got)
		}
	}
	m.valuer.wins.lab.Birth["known"] = at.AddDate(-25, 0, 0).Unix()
	m.valuer.wins.lab.Positions["known"] = map[int64]string{2026: "QB"}
	got := m.Player("known", at)
	if got.Valued || got.Reason != "No production or rookie forecast is available" {
		t.Fatalf("no forecast: %+v", got)
	}
}
