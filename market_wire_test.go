package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/lab"
)

// TestMarketWireMatchesGolden freezes the Market wire contract in testdata/wire/market: what the
// frontend receives from every Market binding, field for field. The bound methods run on fresh
// apps (the reports and the asset board hold nothing, so no clock or time zone reaches them; the
// Lab surfaces read the frozen Lab facts); the held states build in UTC.
// Regenerate with TWR_WRITE_WIRE=1 go test -run TestMarketWireMatchesGolden . and review the diff.
func TestMarketWireMatchesGolden(t *testing.T) {
	t.Parallel()
	fresh := targetTestApp(t)
	reports, err := fresh.TargetMarketReports()
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "market", "reports-unfetched", reports)
	held := map[string]heldSeasonFeed[[]marketfeeds.Feed]{}
	for kind, feed := range realMarketHeld(t) {
		feed.value[0].Players = feed.value[0].Players[:12] // a sample keeps the goldens readable
		held[kind] = feed
	}
	names := map[string]string{"16162": "Bijan Robinson", "16161": "Jahmyr Gibbs", "13589": "Josh Allen"}
	checkGolden(t, "market", "reports-held", buildMarketReports(held, names, 6, time.UTC))
	facts := marketFactsAt(held, time.UTC)
	checkGolden(t, "market", "facts", map[string][]MarketFact{"16162": facts["16162"], "13589": facts["13589"], "": facts[""]})
	empty, err := marketfeeds.Parse(marketfeeds.ADP, []byte(`{"adp":{"totalDrafts":"0","totalPicks":"0"}}`))
	if err != nil {
		t.Fatal(err)
	}
	adp := held[marketfeeds.ADP]
	adp.value, adp.refreshError = []marketfeeds.Feed{empty}, "offline"
	held[marketfeeds.ADP] = adp
	held[marketfeeds.Ranks] = heldSeasonFeed[[]marketfeeds.Feed]{refreshError: "offline"}
	checkGolden(t, "market", "reports-failed", buildMarketReports(held, nil, 0, time.UTC))
	board, err := fresh.TargetAssetBoard()
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "market", "assets-unheld", board)
	price, low, high, filler := 3.1, 2.4, 3.9, -0.03
	model := fakeAssets{
		"16162": {Valued: true, Discounted: 2.314, Seasons: 5, ClassLabel: "Running back, 24 or younger", Price: &price, Low: &low, High: &high},
		"13589": {Valued: false, Reason: "The player has no known position or birth date"},
		"99999": {Valued: true, Discounted: 0.05, Seasons: 5, ClassLabel: "Kicker, 28 or older", Price: &filler},
	}
	assets := buildAssetBoard(researchView{researchInput: realResearch(t)}, model, time.Now(), []string{"99999"})
	sample := slices.Clone(assets.Values[:6]) // the first scored players, then the modelled ones
	for _, v := range assets.Values[6:] {
		if _, ok := model[v.ID]; ok {
			sample = append(sample, v)
		}
	}
	assets.Values = sample
	checkGolden(t, "market", "assets-held", assets)

	a := NewApp()
	full := RawMarketScope{From: 2017, To: 2026}
	thin := RawMarketScope{From: 2100, To: 2101}
	invalid := RawMarketScope{From: 2026, To: 2017}
	filtered := RawMarketScope{From: 2017, To: 2026, Phases: []string{"winter", "deadline"}, Assets: []string{"LB", "pick1:next"}, ByAge: true}
	for name, scope := range map[string]RawMarketScope{"full": full, "thin": thin, "invalid": invalid, "filtered": filtered} {
		checkGolden(t, "market", "market-"+name, a.TargetMarket(scope))
		checkGolden(t, "market", "lists-"+name, a.TargetMarketLists(scope))
		checkGolden(t, "market", "clock-"+name, a.TargetPickClock(scope))
		checkGolden(t, "market", "stretches-"+name, a.TargetMarketStretches(scope))
	}
	for name, raw := range map[string]RawMarketAsset{
		"position": {Scope: full, Kind: "LB"},
		"anchor":   {Scope: full, Kind: lab.Anchor},
		"age":      {Scope: RawMarketScope{From: 2017, To: 2026, ByAge: true}, Kind: "WR:prime"},
		"thin":     {Scope: thin, Kind: "LB"},
		"invalid":  {Scope: full, Kind: "LB:prime"},
	} {
		checkGolden(t, "market", "profile-"+name, a.TargetAssetProfile(raw))
	}
	broken := &App{labData: func() (*lab.Data, error) { return nil, errors.New("fingerprint mismatch") }}
	checkGolden(t, "market", "lists-unloaded", broken.TargetMarketLists(full))
	checkGolden(t, "market", "profile-unloaded", broken.TargetAssetProfile(RawMarketAsset{Scope: full, Kind: "LB"}))
}

// TestMarketWireTwinsMatch proves wireSurface, wireReading, wireFact and wireAssetBoard lose
// nothing: a reading, a fact and an asset board with every field set, and empty ones, encode the
// same before and after conversion. A field added to one type set and not the other, a renamed
// JSON tag, or a field a converter skips fails here.
func TestMarketWireTwinsMatch(t *testing.T) {
	t.Parallel()
	var full report.Reading[marketapp.Surface]
	n := 0
	fillWire(reflect.ValueOf(&full).Elem(), &n)
	var fact marketapp.Fact
	fillWire(reflect.ValueOf(&fact).Elem(), &n)
	var assets marketapp.Board
	fillWire(reflect.ValueOf(&assets).Elem(), &n)
	for _, c := range []struct {
		name       string
		built, got any
	}{
		{"full reading", full, wireReading(full, wireSurface)},
		{"empty reading", report.Reading[marketapp.Surface]{}, wireReading(report.Reading[marketapp.Surface]{}, wireSurface)},
		{"empty section", marketapp.Surface{Sections: []marketapp.Section{{}}}, wireSurface(marketapp.Surface{Sections: []marketapp.Section{{}}})},
		{"fact", fact, wireFact(fact)},
		{"full asset board", assets, wireAssetBoard(assets)},
		{"empty asset board", marketapp.Board{}, wireAssetBoard(marketapp.Board{})},
	} {
		want, err := json.Marshal(c.built)
		if err != nil {
			t.Fatal(err)
		}
		got, err := json.Marshal(c.got)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(want, got) {
			t.Errorf("%s: wire twin encodes differently\n built %s\n wire  %s", c.name, want, got)
		}
	}
	if wireFact(fact).value != fact.Value {
		t.Error("fact: the pool's value was dropped")
	}
}
