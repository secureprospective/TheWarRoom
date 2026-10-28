package market

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/playerrecord"
)

type fakeAssets map[string]Dynasty

func (f fakeAssets) Player(pid string, _ time.Time) Dynasty { return f[pid] }
func (fakeAssets) Pick(int, int, string, time.Time) Dynasty { return Dynasty{} }

func TestAssetBoardDynastyCellsInWords(t *testing.T) {
	t.Parallel()
	model := fakeAssets{
		"1": {Valued: true, Discounted: 9.314, Seasons: 3, ClassLabel: "Native value"},
		"2": {Valued: false, Reason: "Not in this season's model run"},
	}
	cells := map[string]Value{}
	for _, value := range BuildBoard(OnField{Cells: map[string]Cell{}}, model, time.Now(), []string{"1", "2"}).Values {
		cells[value.ID] = value
	}
	if d, m := cells["1"].Dynasty, cells["1"].Market; d.Text != "9.31 PPG" || d.Sub != "dynasty · next 3 seasons" ||
		!strings.HasPrefix(d.Detail, "Native value · ") || m.Text != "No price" || m.Held {
		t.Fatal(d, m)
	}
	if cells["1"].DynastyValue != 9.314 || cells["2"].DynastyValue != 0 {
		t.Fatal("dynasty sort keys", cells["1"].DynastyValue, cells["2"].DynastyValue)
	}
	if d := cells["2"].Dynasty; d.Held || d.Text != "Not valued" || d.Sub != "Not in this season's model run" {
		t.Fatal(d)
	}
}

func TestSignedPricesNeverReadMinusZero(t *testing.T) {
	t.Parallel()
	for v, want := range map[float64]string{-0.03: "−0.03", -0.001: "0.00", 0: "0.00", 0.114: "0.11", -1.5: "−1.50"} {
		if got := Signed(v); got != want {
			t.Errorf("Signed(%v) = %q, want %q", v, got, want)
		}
	}
}

func TestTopPercentRoundsUp(t *testing.T) {
	t.Parallel()
	for _, c := range [][3]int{{1, 50, 2}, {14, 54, 26}, {54, 54, 100}, {1, 1, 100}, {3, 200, 2}} {
		if got := TopPercent(c[0], c[1]); got != c[2] {
			t.Errorf("TopPercent(%d, %d) = %d, want %d", c[0], c[1], got, c[2])
		}
	}
}

// The native model values a player by his run value and a pick by Pick Value v1 in the same
// units; a past draft and a player the run did not value say why.
func TestNativeValuesPlayersAndPicksInOneUnit(t *testing.T) {
	t.Parallel()
	var records []playerrecord.Record
	values := map[string]float64{}
	for i := range 4 * 260 {
		mfl := fmt.Sprintf("%05d", 10000+i)
		overall := i%260 + 1
		records = append(records, playerrecord.Record{MFL: mfl, Draft: playerrecord.Draft{Status: playerrecord.Drafted,
			Year: 2023 + i/260, Overall: overall}})
		values[mfl] = 16*math.Exp(-float64(overall)/70) + 0.5
	}
	n := NewNative(NativeInput{Values: values, Records: records, Season: 2026, Seasons: 3, Discount: 0.75, Teams: 32})
	if d := n.Player("10000", time.Now()); !d.Valued || d.Discounted != values["10000"] || d.Seasons != 3 {
		t.Fatalf("player = %+v", d)
	}
	if d := n.Player("0001", time.Now()); d.Valued || d.Reason == "" {
		t.Fatalf("unvalued player = %+v", d)
	}
	first, third := n.Pick(2026, 1, "0001", time.Now()), n.Pick(2026, 3, "0001", time.Now())
	if !first.Valued || !third.Valued || !(first.Discounted > third.Discounted) || first.Seasons != 3 ||
		!strings.HasPrefix(first.ClassLabel, "Pick Value v1, round 1") {
		t.Fatalf("picks: %+v %+v", first, third)
	}
	if later := n.Pick(2027, 1, "0001", time.Now()); math.Abs(later.Discounted-0.75*first.Discounted) > 1e-9 ||
		!strings.HasSuffix(later.ClassLabel, "1 year out") {
		t.Fatalf("a pick a year out = %+v, want %.4f", later, 0.75*first.Discounted)
	}
	if past := n.Pick(2025, 1, "0001", time.Now()); past.Valued {
		t.Fatalf("a past draft = %+v", past)
	}
	if none := NewNative(NativeInput{Values: values, Season: 2026, Seasons: 3, Discount: 0.75, Teams: 32}).Pick(2026, 1, "", time.Now()); none.Valued ||
		!strings.HasPrefix(none.Reason, "Pick Value v1 not available") {
		t.Fatalf("no records = %+v", none)
	}
}

func TestDynastyDisplayNeverShowsMinusZero(t *testing.T) {
	t.Parallel()
	for _, v := range []float64{-0.0001, -0.004, 0} {
		cell, _, raw := DynastyCells(fakeAssets{"1": {Valued: true, Discounted: v, Seasons: 5}}, "1", time.Now())
		if cell.Text != "0.00 PPG" || raw != v {
			t.Fatalf("display rounded zero or raw value changed: %+v, %v", cell, raw)
		}
	}
	cell, _, _ := DynastyCells(fakeAssets{"1": {Valued: true, Discounted: -0.01}}, "1", time.Now())
	if cell.Text != "−0.01 PPG" {
		t.Fatal(cell.Text)
	}
}

func TestPoolValuesPreserveIDsAndMissingPickModel(t *testing.T) {
	t.Parallel()
	id, err := playerid.New("0099")
	if err != nil {
		t.Fatal(err)
	}
	values, source := PoolAssetValues(OnField{Cells: map[string]Cell{}, Source: "held source"}, nil, []playerid.PlayerID{id, id}, time.Now())
	if len(values) != 1 || values["0099"].ID != "0099" || values["0099"].Now.Text != "No score" || source != "held source" {
		t.Fatalf("pool values: %v %q", values, source)
	}
	if cell := PickValue(nil, 2027, 1, "0001", time.Now()); cell.Text != "Not valued" || cell.Sub != "no model run read yet" || cell.Held {
		t.Fatal(cell)
	}
}
