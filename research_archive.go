package main

import (
	"context"
	"fmt"
	"maps"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/injuries"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func researchKinds() []string {
	return []string{"playerScores", "topAdds", "topDrops", "topStarters", "topOwns", "topTrades", "pointsAllowed"}
}
func researchFeedLabel(kind string) string {
	return map[string]string{"playerScores": "fantasy points", "topAdds": "most added players",
		"topDrops": "most dropped players", "topStarters": "most started players", "topOwns": "most owned players",
		"topTrades": "most traded players", "pointsAllowed": "points allowed"}[kind]
}

func (a *App) archivedResearchFeed(ctx context.Context, kind, week string) heldSeasonFeed[[]researchfeeds.Feed] {
	bodies := a.history.ArchivedBodies
	if week != "" {
		bodies = a.scopedBodies("W", week)
	}
	h := heldSeasonFeed[[]researchfeeds.Feed]{}
	found, err := bodies(ctx, "TYPE="+kind, func(src string, body []byte, at time.Time) bool {
		if !sameExport(src, kind, a.season, researchfeeds.IsTrend(kind)) {
			return false
		}
		feed, err := researchfeeds.Parse(body, kind, week)
		if err != nil {
			return false
		}
		h.value, h.fetchedAt, h.refreshError = []researchfeeds.Feed{feed}, at, "Archived copy; refresh pending"
		return true
	})
	if err != nil {
		h.refreshError = "Archive unreadable: " + err.Error()
	}
	if !found && err == nil {
		h.refreshError = "No archived copy held"
	}
	return h
}

// Opening Research reads disk, never the network or the launch refresh lock.
func (a *App) hydrateResearch(extraIDs ...string) {
	a.researchArchiveMu.Lock()
	defer a.researchArchiveMu.Unlock()
	if a.history == nil {
		return
	}
	ctx, cancel := context.WithTimeout(a.fallbackParent(), fallbackTimeout)
	defer cancel()
	a.seasonMu.Lock()
	feeds := maps.Clone(a.research.feeds)
	depth, injury := a.research.depth, a.seasonInjuries
	news := maps.Clone(a.research.news)
	a.seasonMu.Unlock()
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
	if depth.fetchedAt.IsZero() && a.firstArchiveRead("depth") {
		rows, fresh, err := readResearchDepthArchive(ctx, a.season, a.history.ArchivedBodies, fmt.Errorf("archived copy; refresh pending"))
		holdSeasonFeed(&depth, rows, fresh, err)
	}
	if injury.fetchedAt.IsZero() && a.firstArchiveRead("injuries") {
		injury = archivedOnly(ctx, a, "injuries", true, func(body []byte) ([]leaguefeed.Injury, error) {
			return injuries.ParseToInjuries(ctx, body)
		})
		if !injury.fetchedAt.IsZero() {
			injury.refreshError = "Archived copy; refresh pending"
		}
	}
	mirror, _, _ := a.league.ContractsView()
	missing := []string{}
	for _, id := range append(researchRosterIDs(mirror, ""), extraIDs...) {
		if news[id].fetchedAt.IsZero() && a.firstArchiveRead("news:"+id) {
			missing = append(missing, id)
		}
	}
	cached, err := archivedNews(ctx, a.history.ArchivedBodies, a.season, missing, func(time.Time) bool { return true })
	if err != nil {
		a.fallbackLog(err.Error())
	}
	a.mergeResearchArchive(ctx, feeds, depth, injury, cached)
}

// firstArchiveRead reports whether key has not been looked up in the archive yet, and marks it.
func (a *App) firstArchiveRead(key string) bool {
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
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
	if feeds[key].fetchedAt.IsZero() && a.firstArchiveRead("feed:"+key) {
		feeds[key] = a.archivedResearchFeed(ctx, kind, week)
	}
}
func (a *App) mergeResearchArchive(ctx context.Context, feeds map[string]heldSeasonFeed[[]researchfeeds.Feed],
	depth heldSeasonFeed[[]researchfeeds.RawDepth], injury heldSeasonFeed[[]leaguefeed.Injury],
	news map[string]heldSeasonFeed[[]researchfeeds.News]) {
	a.lookupMu.Lock()
	lookup := a.lookup
	a.lookupMu.Unlock()
	a.seasonMu.Lock()
	links, version := a.research.links, a.research.linkVersion
	a.seasonMu.Unlock()
	a.crosswalkMu.Lock()
	loaded := a.crosswalk.LoadedAt
	a.crosswalkMu.Unlock()
	var err error
	if links == nil || version != loaded {
		links, err = a.researchLinks(ctx, lookup)
	}
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	if a.research.feeds == nil {
		a.research.feeds = map[string]heldSeasonFeed[[]researchfeeds.Feed]{}
	}
	for key, h := range feeds {
		if a.research.feeds[key].fetchedAt.IsZero() {
			a.research.feeds[key] = h
		}
	}
	if a.research.depth.fetchedAt.IsZero() {
		a.research.depth = depth
	}
	if a.seasonInjuries.fetchedAt.IsZero() {
		a.seasonInjuries = injury
	}
	if a.research.news == nil {
		a.research.news = map[string]heldSeasonFeed[[]researchfeeds.News]{}
	}
	for id, h := range news {
		if a.research.news[id].fetchedAt.IsZero() {
			h.refreshError = "Archived copy; refresh pending"
			a.research.news[id] = h
		}
	}
	if err != nil {
		a.research.linkError = err.Error()
	} else {
		a.research.links, a.research.linkError, a.research.linkVersion = links, "", loaded
	}
}
