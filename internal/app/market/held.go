package market

import (
	"maps"
	"slices"
	"sync"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
)

// Feeds are the held MFL market reports by kind: each kind's last good copy (one parsed feed, or
// none before the first read) and its read state.
type Feeds = map[string]season.Feed[[]marketfeeds.Feed]

// Held owns Market's private published data: the held market reports and the native value model.
// Root refresh writers and readers retain SeasonMu around each feed publication and capture, as
// they did when the reports lived in shared state; the model needs no outer lock.
type Held struct {
	mu    sync.Mutex
	feeds Feeds
	model Model
}

// PublishFeed replaces one kind's held report with a detached copy of feed.
func (h *Held) PublishFeed(kind string, feed season.Feed[[]marketfeeds.Feed]) {
	feed.Value = cloneFeeds(feed.Value)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.feeds == nil {
		h.feeds = Feeds{}
	}
	h.feeds[kind] = feed
}

// SnapshotFeeds returns a detached copy of every held report; nil when none was ever published.
func (h *Held) SnapshotFeeds() Feeds {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := maps.Clone(h.feeds)
	for kind, feed := range out {
		feed.Value = cloneFeeds(feed.Value)
		out[kind] = feed
	}
	return out
}

// PublishModel holds the native value model. The fitted model is immutable and its public
// API exposes no mutation, so the same value is shared rather than copied.
func (h *Held) PublishModel(model Model) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.model = model
}

// SnapshotModel returns the held value model; nil until a model run is read.
func (h *Held) SnapshotModel() Model {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.model
}

func cloneFeeds(in []marketfeeds.Feed) []marketfeeds.Feed {
	out := slices.Clone(in)
	for i := range out {
		out[i].Players = slices.Clone(out[i].Players)
	}
	return out
}

// ForWeek is the held reports as this week reads them: a projection made for another week is
// not this week's, so it reads as never fetched. The result is never nil.
func ForWeek(feeds Feeds, week int) Feeds {
	out := Feeds{}
	for kind, feed := range feeds {
		if kind == marketfeeds.Projections && len(feed.Value) == 1 && feed.Value[0].Week != week {
			feed = season.Feed[[]marketfeeds.Feed]{}
		}
		out[kind] = feed
	}
	return out
}

// HeldFeeds is the held reports in the shape the report and fact builders read.
func HeldFeeds(feeds Feeds) map[string]HeldFeed {
	out := make(map[string]HeldFeed, len(feeds))
	for kind, feed := range feeds {
		out[kind] = HeldFeed{Value: feed.Value, Feed: report.Feed{FetchedAt: feed.FetchedAt, RefreshError: feed.RefreshError, ReadPending: feed.ReadPending}}
	}
	return out
}
