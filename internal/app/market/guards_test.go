package market

import (
	"reflect"
	"testing"
	"time"
)

type fakeAssets map[string]Dynasty

func (f fakeAssets) Player(pid string, _ time.Time) Dynasty { return f[pid] }

func (fakeAssets) Pick(int, int, string, time.Time) Dynasty { return Dynasty{} }

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

func TestHeldDetachesIngressAndEgress(t *testing.T) {
	t.Parallel()
	h := &Held{}
	input, want := feedsFixture(t), feedsFixture(t)
	publishFeeds(h, input)
	mutateFeeds(input)
	if got := h.SnapshotFeeds(); !reflect.DeepEqual(got, want) {
		t.Fatal("publication retained caller storage", got)
	}
	mutateFeeds(h.SnapshotFeeds())
	if got := h.SnapshotFeeds(); !reflect.DeepEqual(got, want) {
		t.Fatal("snapshot exposed held storage", got)
	}
}
