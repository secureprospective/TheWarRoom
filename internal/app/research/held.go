package research

import (
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
)

// Held owns private caches; composition retains the outer season lock for joint captures.
// It is never embedded or bound to a transport.
type Held struct {
	mu    sync.Mutex
	value Values
}

func (h *Held) Publish(value Values) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.value = cloneValues(value)
}

func (h *Held) Snapshot() Values {
	h.mu.Lock()
	defer h.mu.Unlock()
	return cloneValues(h.value)
}

func (h *Held) PublishStatus(text string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.value.Status = text
}

func (h *Held) SnapshotStatus() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.value.Status
}

func cloneValues(value Values) Values {
	value.Feeds = maps.Clone(value.Feeds)
	for key, feed := range value.Feeds {
		feed.Value = slices.Clone(feed.Value)
		for i := range feed.Value {
			feed.Value[i].Players = slices.Clone(feed.Value[i].Players)
			feed.Value[i].Allowed = slices.Clone(feed.Value[i].Allowed)
		}
		value.Feeds[key] = feed
	}
	value.Depth.Value = slices.Clone(value.Depth.Value)
	for i := range value.Depth.Value {
		row := &value.Depth.Value[i]
		row.GSIS, row.ESPN = cloneDepthID(row.GSIS), cloneDepthID(row.ESPN)
	}
	value.News = maps.Clone(value.News)
	for key, feed := range value.News {
		feed.Value = slices.Clone(feed.Value)
		value.News[key] = feed
	}
	value.Links = maps.Clone(value.Links)
	value.FinishedAt = maps.Clone(value.FinishedAt)
	value.ArchiveRead = maps.Clone(value.ArchiveRead)
	return value
}

func cloneDepthID(id *string) *string {
	if id == nil {
		return nil
	}
	value := *id
	return &value
}

// Values is a detached publication, including archive bookkeeping and the current status.
// Snapshot's reading construction uses only its feed, depth, link and news fields.
type Values struct {
	Feeds       map[string]Feed[[]researchfeeds.Feed]
	Depth       Feed[[]researchfeeds.RawDepth]
	Links       map[string]string
	LinkError   string
	LinkVersion string
	FinishedAt  map[int]time.Time
	Status      string
	News        map[string]Feed[[]researchfeeds.News]
	ArchiveRead map[string]bool
}
