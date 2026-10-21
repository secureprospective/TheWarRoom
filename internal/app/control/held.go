package control

import (
	"maps"
	"slices"
	"sync"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/sitenews"
)

// Held owns the Control Room's private held MFL feeds: MFL's names for its scoring codes and its
// site news. Only this feature writes or reads them, so their lock guards nothing else and is never
// held across a Host call, a fetch or an event. It is never bound to a transport.
type Held struct {
	mu        sync.Mutex
	ruleNames report.HeldFeed[map[string]string]
	siteNews  report.HeldFeed[[]sitenews.RawItem]
}

// SnapshotRuleNames returns a detached copy of the held rule names; nil means none are held.
func (h *Held) SnapshotRuleNames() report.HeldFeed[map[string]string] {
	h.mu.Lock()
	defer h.mu.Unlock()
	return cloneRuleNames(h.ruleNames)
}

// PublishRuleNames holds one read of the rule names, keeping the last good copy when the read
// failed, and returns a detached copy of what is now held.
func (h *Held) PublishRuleNames(value map[string]string, fresh domain.Freshness,
	err error) report.HeldFeed[map[string]string] {
	value = maps.Clone(value)
	h.mu.Lock()
	defer h.mu.Unlock()
	report.HoldFeed(&h.ruleNames, value, fresh, err)
	return cloneRuleNames(h.ruleNames)
}

// SnapshotSiteNews returns a detached copy of the held site news.
func (h *Held) SnapshotSiteNews() report.HeldFeed[[]sitenews.RawItem] {
	h.mu.Lock()
	defer h.mu.Unlock()
	return cloneSiteNews(h.siteNews)
}

// PublishSiteNews holds one read of the site news, keeping the last good copy when the read failed,
// and returns a detached copy of what is now held. Each raw item holds only value strings.
func (h *Held) PublishSiteNews(value []sitenews.RawItem, fresh domain.Freshness,
	err error) report.HeldFeed[[]sitenews.RawItem] {
	value = slices.Clone(value)
	h.mu.Lock()
	defer h.mu.Unlock()
	report.HoldFeed(&h.siteNews, value, fresh, err)
	return cloneSiteNews(h.siteNews)
}

func cloneRuleNames(feed report.HeldFeed[map[string]string]) report.HeldFeed[map[string]string] {
	feed.Value = maps.Clone(feed.Value)
	return feed
}

func cloneSiteNews(feed report.HeldFeed[[]sitenews.RawItem]) report.HeldFeed[[]sitenews.RawItem] {
	feed.Value = slices.Clone(feed.Value)
	return feed
}
