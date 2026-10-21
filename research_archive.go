package main

import (
	"context"
	"fmt"
	"maps"
	"strconv"
	"time"

	researchapp "github.com/secureprospective/TheWarRoom/internal/app/research"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/injuries"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func researchKinds() []string {
	return researchapp.Kinds()
}
func researchFeedLabel(kind string) string {
	return researchapp.FeedLabel(kind)
}

func (a *App) archivedResearchFeed(ctx context.Context, kind, week string) heldSeasonFeed[[]researchfeeds.Feed] {
	bodies := a.core.History.ArchivedBodies
	if week != "" {
		bodies = a.scopedBodies("W", week)
	}
	h := heldSeasonFeed[[]researchfeeds.Feed]{}
	found, err := bodies(ctx, "TYPE="+kind, func(src string, body []byte, at time.Time) bool {
		if !sameExport(src, kind, a.core.Season, researchfeeds.IsTrend(kind)) {
			return false
		}
		feed, err := researchfeeds.Parse(body, kind, week)
		if err != nil {
			return false
		}
		h.Value, h.FetchedAt, h.RefreshError = []researchfeeds.Feed{feed}, at, "Archived copy; refresh pending"
		return true
	})
	if err != nil {
		h.RefreshError = "Archive unreadable: " + err.Error()
	}
	if !found && err == nil {
		h.RefreshError = "No archived copy held"
	}
	return h
}

// Opening Research reads disk, never the network or the launch refresh lock.
func (a *App) hydrateResearch(extraIDs ...string) {
	ctx, cancel := context.WithTimeout(a.fallbackParent(), fallbackTimeout)
	defer cancel()
	a.hydrateResearchContext(ctx, extraIDs...)
}

// hydrateResearchContext publishes archived research; one hydration at a time, so a reader
// never sees lookups marked done before their results are held.
func (a *App) hydrateResearchContext(ctx context.Context, extraIDs ...string) {
	a.core.ResearchArchiveMu.Lock()
	defer a.core.ResearchArchiveMu.Unlock()
	if a.core.History == nil {
		return
	}
	a.core.SeasonMu.Lock()
	feeds := maps.Clone(a.research.feeds)
	depth, injury := a.research.depth, a.core.SeasonInjuries
	news := maps.Clone(a.research.news)
	a.core.SeasonMu.Unlock()
	if feeds == nil {
		feeds = map[string]heldSeasonFeed[[]researchfeeds.Feed]{}
	}
	for _, kind := range researchKinds() {
		week := ""
		if kind == "playerScores" {
			week = "YTD"
		}
		a.hydrateResearchFeed(ctx, feeds, kind, week)
	}
	for w := 1; w <= a.marketWeek(); w++ {
		a.hydrateResearchFeed(ctx, feeds, "playerScores", strconv.Itoa(w))
	}
	if depth.FetchedAt.IsZero() && a.firstArchiveRead("depth") {
		rows, fresh, err := readResearchDepthArchive(ctx, a.core.Season, a.core.History.ArchivedBodies, fmt.Errorf("archived copy; refresh pending"))
		holdSeasonFeed(&depth, rows, fresh, err)
	}
	if injury.FetchedAt.IsZero() && a.firstArchiveRead("injuries") {
		injury = archivedOnly(ctx, a, "injuries", true, func(body []byte) ([]leaguefeed.Injury, error) {
			return injuries.ParseToInjuries(ctx, body)
		})
		if !injury.FetchedAt.IsZero() {
			injury.RefreshError = "Archived copy; refresh pending"
		}
	}
	mirror, _, _ := a.core.League.ContractsView()
	missing := []string{}
	for _, id := range append(researchRosterIDs(mirror, ""), extraIDs...) {
		if news[id].FetchedAt.IsZero() && a.firstArchiveRead("news:"+id) {
			missing = append(missing, id)
		}
	}
	cached, err := archivedNews(ctx, a.core.History.ArchivedBodies, a.core.Season, missing, func(time.Time) bool { return true })
	if err != nil {
		a.core.FallbackLog(err.Error())
	}
	a.mergeResearchArchive(ctx, feeds, depth, injury, cached)
}

// firstArchiveRead reports whether key has not been looked up in the archive yet, and marks it.
func (a *App) firstArchiveRead(key string) bool {
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	if a.research.archiveRead[key] {
		return false
	}
	if a.research.archiveRead == nil {
		a.research.archiveRead = map[string]bool{}
	}
	a.research.archiveRead[key] = true
	return true
}
func (a *App) hydrateResearchFeed(ctx context.Context, feeds map[string]heldSeasonFeed[[]researchfeeds.Feed], kind, week string) {
	key := researchKey(kind, week)
	if feeds[key].FetchedAt.IsZero() && a.firstArchiveRead("feed:"+key) {
		feeds[key] = a.archivedResearchFeed(ctx, kind, week)
	}
}
func (a *App) mergeResearchArchive(ctx context.Context, feeds map[string]heldSeasonFeed[[]researchfeeds.Feed],
	depth heldSeasonFeed[[]researchfeeds.RawDepth], injury heldSeasonFeed[[]leaguefeed.Injury],
	news map[string]heldSeasonFeed[[]researchfeeds.News]) {
	a.core.LookupMu.Lock()
	lookup := a.core.Lookup
	a.core.LookupMu.Unlock()
	a.core.SeasonMu.Lock()
	links, version := a.research.links, a.research.linkVersion
	a.core.SeasonMu.Unlock()
	a.core.CrosswalkMu.Lock()
	loaded := a.crosswalk.LoadedAt
	a.core.CrosswalkMu.Unlock()
	var err error
	if links == nil || version != loaded {
		links, err = a.researchLinks(ctx, lookup)
	}
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	if a.research.feeds == nil {
		a.research.feeds = map[string]heldSeasonFeed[[]researchfeeds.Feed]{}
	}
	for key, h := range feeds {
		if a.research.feeds[key].FetchedAt.IsZero() {
			a.research.feeds[key] = h
		}
	}
	if a.research.depth.FetchedAt.IsZero() {
		a.research.depth = depth
	}
	if a.core.SeasonInjuries.FetchedAt.IsZero() {
		a.core.SeasonInjuries = injury
	}
	if a.research.news == nil {
		a.research.news = map[string]heldSeasonFeed[[]researchfeeds.News]{}
	}
	for id, h := range news {
		if a.research.news[id].FetchedAt.IsZero() {
			if h.FetchedAt.In(time.Local).Format("2006-01-02") != time.Now().In(time.Local).Format("2006-01-02") {
				h.RefreshError = "Archived copy; refresh pending"
			}
			a.research.news[id] = h
		}
	}
	if err != nil {
		a.research.linkError = err.Error()
	} else {
		a.research.links, a.research.linkError, a.research.linkVersion = links, "", loaded
	}
}
