package market

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func feedsFixture(t *testing.T) Feeds {
	t.Helper()
	id, err := playerid.New("16162")
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	return Feeds{
		marketfeeds.Projections: {Value: []marketfeeds.Feed{{Week: 6, Players: []marketfeeds.Player{{ID: id, Value: 12, Rank: 1}}}}, FetchedAt: at},
		marketfeeds.ADP:         {Value: []marketfeeds.Feed{{Samples: 40, Players: []marketfeeds.Player{{ID: id, Value: 3}}}}, FetchedAt: at, RefreshError: "stale"},
	}
}

func publishFeeds(h *Held, feeds Feeds) {
	for kind, feed := range feeds {
		h.PublishFeed(kind, feed)
	}
}

func mutateFeeds(feeds Feeds) {
	for kind, feed := range feeds {
		feed.Value[0].Players[0].Value = 99
		feed.Value[0].Week = 99
		feed.Value[0] = marketfeeds.Feed{}
		delete(feeds, kind)
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

func TestHeldPreservesNilAndEmpty(t *testing.T) {
	t.Parallel()
	if got := (&Held{}).SnapshotFeeds(); got != nil {
		t.Fatal("no publication reads as an empty map", got)
	}
	input := Feeds{
		"nil":          {},
		"empty":        {Value: []marketfeeds.Feed{}},
		"nil players":  {Value: []marketfeeds.Feed{{Week: 1}}},
		"none players": {Value: []marketfeeds.Feed{{Week: 1, Players: []marketfeeds.Player{}}}},
	}
	h := &Held{}
	publishFeeds(h, input)
	if got := h.SnapshotFeeds(); !reflect.DeepEqual(got, input) {
		t.Fatal("nil/empty changed", got, input)
	}
	if h.SnapshotModel() != nil {
		t.Fatal("model before publication")
	}
}

func TestHeldPublishReadRace(t *testing.T) {
	t.Parallel()
	h := &Held{}
	input := feedsFixture(t)
	publishFeeds(h, input)
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 300 {
				publishFeeds(h, input)
				h.PublishModel(fakeModel{})
				mutateFeeds(h.SnapshotFeeds())
				_ = h.SnapshotModel()
			}
		})
	}
	wg.Wait()
	if !reflect.DeepEqual(h.SnapshotFeeds(), input) {
		t.Fatal("concurrent reads mutated publication")
	}
}

func TestForWeekDropsOtherWeeksProjection(t *testing.T) {
	t.Parallel()
	feeds := feedsFixture(t)
	if got := ForWeek(feeds, 6); !reflect.DeepEqual(got, feeds) {
		t.Fatal("this week's projection dropped")
	}
	got := ForWeek(feeds, 7)
	if !reflect.DeepEqual(got[marketfeeds.Projections], season.Feed[[]marketfeeds.Feed]{}) || !reflect.DeepEqual(got[marketfeeds.ADP], feeds[marketfeeds.ADP]) {
		t.Fatal("week filter", got)
	}
	if ForWeek(nil, 1) == nil {
		t.Fatal("an unheld read is nil")
	}
}
