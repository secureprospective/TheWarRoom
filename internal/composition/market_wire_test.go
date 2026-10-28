package composition

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
)

// TestMarketWireMatchesGolden keeps the composition goldens in testdata/wire/market: the reports
// and asset board through the bound API over a fresh app's real host, the held asset board over
// Research's held weeks, and the pool's market facts. Every other Market golden is frozen beside
// the API in internal/app/market. The bound methods run on fresh apps (the reports and the asset
// board hold nothing, so no clock or time zone reaches them); the held states build in UTC.
// Regenerate with TWR_WRITE_WIRE=1 go test -run TestMarketWireMatchesGolden . and review the diff.
func TestMarketWireMatchesGolden(t *testing.T) {
	t.Parallel()
	fresh := newMarketAPI(targetTestApp(t))
	reports, err := fresh.TargetMarketReports()
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "market", "reports-unfetched", reports)
	held := map[string]heldSeasonFeed[[]marketfeeds.Feed]{}
	for kind, feed := range realMarketHeld(t) {
		feed.Value[0].Players = feed.Value[0].Players[:12] // a sample keeps the goldens readable
		held[kind] = feed
	}
	facts := marketFactsAt(held, time.UTC)
	checkGolden(t, "market", "facts", map[string][]MarketFact{"16162": facts["16162"], "13589": facts["13589"], "": facts[""]})
	board, err := fresh.TargetAssetBoard()
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "market", "assets-unheld", board)
	model := fakeAssets{
		"16162": {Valued: true, Discounted: 12.314, Seasons: 3, ClassLabel: "Native value"},
		"13589": {Valued: false, Reason: "Not in this season's model run"},
		"99999": {Valued: true, Discounted: 0.05, Seasons: 3, ClassLabel: "Native value"},
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
}

// TestMarketFactWireTwinMatches proves wireFact loses nothing: a fact with every field set
// encodes the same before and after conversion, and keeps the pool's value.
func TestMarketFactWireTwinMatches(t *testing.T) {
	t.Parallel()
	var fact marketapp.Fact
	n := 0
	fillWire(reflect.ValueOf(&fact).Elem(), &n)
	want, err := json.Marshal(fact)
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(wireFact(fact))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, got) {
		t.Errorf("fact: wire twin encodes differently\n built %s\n wire  %s", want, got)
	}
	if wireFact(fact).value != fact.Value {
		t.Error("fact: the pool's value was dropped")
	}
}

type fakeAssets map[string]dynastyValue

func (f fakeAssets) Player(pid string, _ time.Time) dynastyValue { return f[pid] }
func (fakeAssets) Pick(int, int, string, time.Time) dynastyValue { return dynastyValue{} }

func TestAssetBoardOnFieldNowFromHeldWeeks(t *testing.T) {
	t.Parallel()
	in := realResearch(t) // week 5: weeks 1-4 are finished
	v := researchView{researchInput: in}
	b := buildAssetBoard(v, nil, time.Now(), []string{"99999"})
	if !strings.Contains(b.NowSource, "4 of 4 finished weeks held") {
		t.Fatal(b.NowSource)
	}
	// Recount one player by hand from the weekly feeds: points per game over the weeks he scored.
	const id = "16162"
	sum, games := 0.0, 0
	for w := 1; w <= 4; w++ {
		for _, e := range researchRows(v.feed("playerScores", strconv.Itoa(w))) {
			if e.ID.String() == id {
				sum, games = sum+e.Value, games+1
			}
		}
	}
	if games == 0 {
		t.Fatal("fixture player has no weekly scores")
	}
	cells := map[string]AssetValue{}
	for _, value := range b.Values {
		cells[value.ID] = value
	}
	got := cells[id].Now
	position := v.player(id).Position
	ppg := strconv.FormatFloat(sum/float64(games), 'f', 2, 64) + " PPG"
	if !got.Held || got.Text != ppg || !strings.HasPrefix(got.Sub, position+" ") || !strings.Contains(got.Sub, "· top ") ||
		cells[id].NowValue != sum/float64(games) ||
		!strings.Contains(got.Detail, strconv.Itoa(games)+" scored week") {
		t.Fatalf("%s: %+v, want %q", id, got, ppg)
	}
	// Ranks run 1..n at each position, the best points per game first.
	firsts := 0
	for _, value := range cells {
		if f := strings.Fields(value.Now.Sub); value.Now.Held && len(f) == 5 && strings.HasPrefix(f[1], "1/") {
			firsts++
		}
	}
	if firsts < 8 {
		t.Fatal("expected a leader at every position", firsts)
	}
	if c := cells["99999"].Now; c.Held || c.Text != "No score" || c.Sub != "this season" {
		t.Fatal("unscored rostered player", c)
	}
	if c := cells[id].Dynasty; c.Held || c.Text != "Not valued yet" || c.Sub != "no model run read yet" || b.DynastySource == "" {
		t.Fatal("no model", c, b.DynastySource)
	}
}

func TestAppLabDecodesOnce(t *testing.T) {
	t.Parallel()
	a := NewApp()
	first, err := a.lab()
	if err != nil {
		t.Fatal(err)
	}
	second, err := a.lab()
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("the App decoded the frozen Lab facts twice")
	}
}
