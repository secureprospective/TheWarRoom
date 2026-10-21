package control

import (
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/sitenews"
)

func heldAt(at time.Time) domain.Freshness { return report.LiveFreshness(at) }

func TestHeldDetachesIngressAndEgress(t *testing.T) {
	t.Parallel()
	h := &Held{}
	at := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	names := map[string]string{"#P": "Passing TD"}
	got := h.PublishRuleNames(names, heldAt(at), nil)
	names["#P"] = "changed"
	got.Value["#P"] = "changed too"
	if v := h.SnapshotRuleNames().Value["#P"]; v != "Passing TD" {
		t.Fatalf("rule names aliased: %q", v)
	}
	snap := h.SnapshotRuleNames()
	snap.Value["#P"] = "snapshot write"
	if v := h.SnapshotRuleNames().Value["#P"]; v != "Passing TD" {
		t.Fatalf("rule names snapshot aliased: %q", v)
	}

	items := siteNewsItems(t)
	want := siteNewsItems(t)
	news := h.PublishSiteNews(items, heldAt(at), nil)
	items[0], items[1] = items[1], items[0]
	news.Value[0] = news.Value[1]
	if !reflect.DeepEqual(h.SnapshotSiteNews().Value, want) {
		t.Fatal("site news aliased its input or its publication result")
	}
	out := h.SnapshotSiteNews()
	out.Value[0] = out.Value[1]
	if held := h.SnapshotSiteNews(); !reflect.DeepEqual(held.Value, want) || !held.FetchedAt.Equal(at) {
		t.Fatal("site news snapshot aliased the holder")
	}
}

func siteNewsItems(t *testing.T) []sitenews.RawItem {
	t.Helper()
	items, err := sitenews.Parse(readTestdata(t, "internal/ingestion/sitenews/testdata/siteNews.json"))
	if err != nil || len(items) < 2 || reflect.DeepEqual(items[0], items[1]) {
		t.Fatal("site news fixture", err)
	}
	return items
}

func TestHeldKeepsNilEmptyAndLastGood(t *testing.T) {
	t.Parallel()
	h := &Held{}
	if h.SnapshotRuleNames().Value != nil || h.SnapshotSiteNews().Value != nil {
		t.Fatal("an empty holder must hold nil")
	}
	at := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	if v := h.PublishRuleNames(map[string]string{}, heldAt(at), nil).Value; v == nil || len(v) != 0 {
		t.Fatal("an empty map must stay empty, not nil")
	}
	if v := h.PublishSiteNews([]sitenews.RawItem{}, heldAt(at), nil).Value; v == nil || len(v) != 0 {
		t.Fatal("an empty slice must stay empty, not nil")
	}
	if v := h.PublishSiteNews(nil, heldAt(at), nil).Value; v != nil {
		t.Fatal("nil site news must stay nil")
	}
	h.PublishRuleNames(map[string]string{"#P": "Passing TD"}, heldAt(at), nil)
	failed := h.PublishRuleNames(nil, domain.Freshness{}, errors.New("mfl down"))
	if failed.Value["#P"] != "Passing TD" || failed.RefreshError != "mfl down" || !failed.FetchedAt.Equal(at) {
		t.Fatalf("a failed read must keep the last good names: %+v", failed)
	}
}

func TestHeldPublishAndReadRace(t *testing.T) {
	t.Parallel()
	h := &Held{}
	at := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	var wg sync.WaitGroup
	for i := range 4 {
		wg.Go(func() {
			for j := range 200 {
				h.PublishRuleNames(map[string]string{"#P": "names"}, heldAt(at.Add(time.Duration(i*j))), nil)
				h.PublishSiteNews([]sitenews.RawItem{{}}, heldAt(at), nil)
			}
		})
		wg.Go(func() {
			for range 200 {
				if names := h.SnapshotRuleNames(); names.Value != nil {
					names.Value["#X"] = "reader write"
				}
				news := h.SnapshotSiteNews()
				if len(news.Value) > 0 {
					news.Value[0] = sitenews.RawItem{}
				}
			}
		})
	}
	wg.Wait()
	if _, ok := h.SnapshotRuleNames().Value["#X"]; ok {
		t.Fatal("a reader's write reached the holder")
	}
}
