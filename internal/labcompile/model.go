package labcompile

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/facts"
)

// Interval is the market's confidence interval for a class price per expected win.
type Interval struct{ Low, High float64 }

// AssetValue uses the same five-season frame and market as the Lab trade legs.
// Unvalued assets have no forecast, price or interval; Reason explains the missing evidence.
// Interval is the class price per expected win; Range is this asset's price at its ends.
type AssetValue struct {
	Forecast   []float64
	Discounted float64
	Class      string
	Label      string
	Price      *float64
	Interval   *Interval
	Range      *Interval
	Valued     bool
	Reason     string
}

// Model holds a fitted, immutable Lab valuer. Player and Pick do not retrain it.
type Model struct {
	valuer   *valuer
	prices   map[string]MarketClass
	perAsset float64
}

// Build returns the complete Lab document and the model that valued its trade legs.
func Build(ctx context.Context, f facts.Facts, in Evidence) (*Model, []byte, error) {
	var model *Model
	body, err := compile(ctx, f, in, func(ctx context.Context, l *Lab, trades []Trade, draft []Pick, known, estimated map[teamYear]int64) (Analysis, error) {
		a, m, err := analyse(ctx, l, trades, draft, known, estimated)
		model = m
		return a, err
	})
	if err != nil {
		return nil, nil, err
	}
	return model, body, nil
}

// Player values a player at an instant, including only unplayed weeks of that season.
func (m *Model) Player(pid string, at time.Time) AssetValue {
	return m.asset(Asset{Kind: "player", ID: &pid}, at)
}

// Pick values a rookie pick, using its exact slot when known and its round otherwise.
// Slot zero means unknown. Orig is the original franchise ID, not the current holder.
func (m *Model) Pick(year, round, slot int, orig string, at time.Time) AssetValue {
	return m.asset(Asset{Kind: "pick", Year: ptr(int64(year)), Round: ptr(int64(round)), Slot: ptr(int64(slot)), Orig: &orig}, at)
}
func (m *Model) asset(a Asset, at time.Time) AssetValue {
	if m == nil || m.valuer == nil {
		return AssetValue{Reason: "No fitted value model is available"}
	}
	if at.IsZero() {
		return AssetValue{Reason: "A valuation moment is required"}
	}
	year := int64(dayOf(at.Unix()).Year())
	if year < firstYear || year > lastYear {
		return AssetValue{Reason: "The valuation moment is outside the archive calendar"}
	}
	if a.Kind == "pick" && (value(a.Year) <= 0 || value(a.Round) <= 0 || value(a.Slot) < 0) {
		return AssetValue{Reason: "Pick year, round and slot are invalid"}
	}
	season, done := m.valuer.frame(year, at.Unix())
	class := assetClass(a, m.valuer, season, at.Unix())
	if class == "player:unknown" {
		return AssetValue{Class: class, Label: classLabel(class), Reason: "The player has no known position or birth date"}
	}
	var forecast []float64
	if a.Kind == "player" {
		forecast = m.valuer.player(value(a.ID), season, done)
	} else {
		forecast = m.valuer.pick(a, season)
	}
	if forecast == nil {
		return AssetValue{Class: class, Label: classLabel(class), Reason: "No production or rookie forecast is available"}
	}
	x := discounted(forecast, discount)
	out := AssetValue{Forecast: slices.Clone(forecast), Discounted: pythonRound(x, 3), Class: class, Label: classLabel(class), Valued: true}
	if price, ok := m.prices[class]; ok {
		out.Price = ptr(pythonRound(x*price.Price+m.perAsset, 3))
		out.Interval = &Interval{price.Lo, price.Hi}
		out.Range = &Interval{pythonRound(x*price.Lo+m.perAsset, 3), pythonRound(x*price.Hi+m.perAsset, 3)}
	}
	return out
}
func analyse(ctx context.Context, l *Lab, trades []Trade, draft []Pick, known, estimated map[teamYear]int64) (Analysis, *Model, error) {
	if err := ctx.Err(); err != nil {
		return Analysis{}, nil, fmt.Errorf("value analysis: %w", err)
	}
	w := newWins(l)
	f := newForecast(w, nil)
	p := newPicks(w, draft, f.last)
	v := newValuer(w, f, p, known, estimated, 0)
	rows := valueTrades(l, trades, v, rookieSalaries(l, draft))
	classes := []string{}
	for _, row := range rows {
		for _, side := range row.Sides {
			for _, it := range side {
				if !slices.Contains(classes, it.Class) {
					classes = append(classes, it.Class)
				}
			}
		}
	}
	slices.Sort(classes)
	bt, err := backtest(ctx, l, trades, draft, known, estimated, w)
	if err != nil {
		return Analysis{}, nil, err
	}
	prices := market(rows, classes, bt.Delivered)
	m := &Model{valuer: v, prices: map[string]MarketClass{}, perAsset: prices.PerAsset}
	for _, c := range prices.Classes {
		m.prices[c.ID] = c
	}
	a := Analysis{Market: prices, Checks2: AnalysisChecks{Forecast: f.check(), ValuedTrades: int64(len(rows)), Currency: w.currencyCheck(), Replacement: map[string]float64{}, Backtest: bt.Check}, Delivered: bt.Records, AllPlay: w.allPlay(), Weeks: map[int64]int64{}}
	for key, r := range w.replacement {
		a.Checks2.Replacement[fmt.Sprintf("%s:%d", key.Position, key.Year)] = pythonRound(r, 2)
	}
	for _, year := range w.years {
		a.Weeks[year] = int64(len(w.played[year]))
	}
	a.PickSlots, a.PickSeasons = p.expectations()
	fillValueLegs(&a, rows, m.prices, prices.PerAsset)
	return a, m, nil
}
