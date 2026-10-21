package research

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
)

func heldFixture() Values {
	id := "fixture"
	return Values{
		Feeds: map[string]Feed[[]researchfeeds.Feed]{"populated": {Value: []researchfeeds.Feed{{Players: []researchfeeds.Entry{{Value: 1}}, Allowed: []researchfeeds.Allowed{{Team: "BUF"}}}}}, "empty": {Value: []researchfeeds.Feed{}}, "nil": {}, "nested": {Value: []researchfeeds.Feed{{Players: []researchfeeds.Entry{}, Allowed: []researchfeeds.Allowed{}}, {}}}},
		Depth: Feed[[]researchfeeds.RawDepth]{Value: []researchfeeds.RawDepth{{GSIS: &id, ESPN: &id}}},
		News:  map[string]Feed[[]researchfeeds.News]{"populated": {Value: []researchfeeds.News{{Headline: "fixture"}}}, "empty": {Value: []researchfeeds.News{}}, "nil": {}},
		Links: map[string]string{"fixture": "0042"}, FinishedAt: map[int]time.Time{1: time.Unix(1, 0)}, ArchiveRead: map[string]bool{"fixture": true}, LinkError: "fixture", LinkVersion: "fixture", Status: "fixture",
	}
}

func mutateHeld(value Values) {
	value.Feeds["populated"].Value[0].Players[0].Value = 2
	value.Feeds["populated"].Value[0].Allowed[0].Team = "NYJ"
	*value.Depth.Value[0].GSIS = "changed"
	*value.Depth.Value[0].ESPN = "changed"
	value.News["populated"].Value[0].Headline = "changed"
	value.Links["fixture"] = "changed"
	value.FinishedAt[1] = time.Time{}
	value.ArchiveRead["fixture"] = false
	delete(value.Feeds, "empty")
}

func heldJSON(t *testing.T, value Values) string {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestHeldDetachesIngressEgressAndNestedEmptyValues(t *testing.T) {
	h := &Held{}
	value := heldFixture()
	want := heldJSON(t, value)
	h.Publish(value)
	mutateHeld(value)
	if got := heldJSON(t, h.Snapshot()); got != want {
		t.Fatal("publication retained input", got)
	}
	out := h.Snapshot()
	mutateHeld(out)
	if got := heldJSON(t, h.Snapshot()); got != want {
		t.Fatal("snapshot retained output", got)
	}
	out = h.Snapshot()
	if out.Feeds["nil"].Value != nil || out.Feeds["empty"].Value == nil || out.News["nil"].Value != nil || out.News["empty"].Value == nil || out.Feeds["nested"].Value[0].Players == nil || out.Feeds["nested"].Value[0].Allowed == nil || out.Feeds["nested"].Value[1].Players != nil || out.Feeds["nested"].Value[1].Allowed != nil {
		t.Fatal("nil/empty changed", out)
	}
	for _, value := range []Values{{}, {Feeds: map[string]Feed[[]researchfeeds.Feed]{}, News: map[string]Feed[[]researchfeeds.News]{}, Links: map[string]string{}, FinishedAt: map[int]time.Time{}, ArchiveRead: map[string]bool{}, Depth: Feed[[]researchfeeds.RawDepth]{Value: []researchfeeds.RawDepth{}}}} {
		want := heldJSON(t, value)
		h.Publish(value)
		if got := heldJSON(t, h.Snapshot()); got != want {
			t.Fatal("nil/empty publication", got, want)
		}
	}
}

func TestHeldPublicationRacesDetachedReaders(t *testing.T) {
	h := &Held{}
	h.Publish(heldFixture())
	var workers sync.WaitGroup
	for worker := range 5 {
		workers.Go(func() {
			for range 200 {
				if worker == 0 {
					h.Publish(heldFixture())
					continue
				}
				out := h.Snapshot()
				if out.Feeds["populated"].Value[0].Players[0].Value != 1 || out.News["populated"].Value[0].Headline != "fixture" {
					t.Error("torn publication")
				}
				mutateHeld(out)
			}
		})
	}
	workers.Wait()
}
