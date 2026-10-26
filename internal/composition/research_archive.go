package composition

import (
	"context"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/store/state"

	researchapp "github.com/secureprospective/TheWarRoom/internal/app/research"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/injuries"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func (a *App) archivedResearchFeed(ctx context.Context, kind, week string) heldSeasonFeed[[]researchfeeds.Feed] {
	return a.researchArchiveFeeds().Read(ctx, a.core.Season, kind, week)
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
	research := a.snapshotResearch()
	input := researchapp.ArchiveInput{Feeds: research.feeds, Depth: research.depth, Injuries: a.core.SeasonInjuries, News: research.news}
	a.core.SeasonMu.Unlock()
	input = researchapp.HydrateArchive(ctx, input, extraIDs, researchapp.ArchiveSource{
		Feeds: a.researchArchiveFeeds(), Season: a.core.Season, Week: a.marketWeek, FirstRead: a.firstArchiveRead,
		Roster: func() state.MirrorSnapshot { mirror, _, _ := a.core.League.ContractsView(); return mirror },
		Injuries: func() researchapp.Feed[[]leaguefeed.Injury] {
			return archivedOnly(ctx, a, "injuries", true, func(body []byte) ([]leaguefeed.Injury, error) { return injuries.ParseToInjuries(ctx, body) })
		},
		Log: a.core.FallbackLog,
	})
	a.mergeResearchArchive(ctx, input.Feeds, input.Depth, input.Injuries, input.News)
}

// firstArchiveRead reports whether key has not been looked up in the archive yet, and marks it.
func (a *App) firstArchiveRead(key string) bool {
	a.core.SeasonMu.Lock()
	research := a.snapshotResearch()
	defer a.core.SeasonMu.Unlock()
	defer func() { a.publishResearch(research) }()
	return researchapp.FirstArchiveRead(&research.archiveRead, key)
}

func (a *App) mergeResearchArchive(ctx context.Context, feeds map[string]heldSeasonFeed[[]researchfeeds.Feed],
	depth heldSeasonFeed[[]researchfeeds.RawDepth], injury heldSeasonFeed[[]leaguefeed.Injury],
	news map[string]heldSeasonFeed[[]researchfeeds.News]) {
	a.core.LookupMu.Lock()
	lookup := a.core.Lookup
	a.core.LookupMu.Unlock()
	a.core.SeasonMu.Lock()
	research := a.snapshotResearch()
	links, version := research.links, research.linkVersion
	a.core.SeasonMu.Unlock()
	loaded := a.researchCrosswalkVersion()
	links, err := researchapp.ArchiveLinks(links, version, loaded, func() (map[string]string, error) { return a.researchLinks(ctx, lookup) })
	a.core.SeasonMu.Lock()
	research = a.snapshotResearch()
	defer a.core.SeasonMu.Unlock()
	defer func() { a.publishResearch(research) }()
	value, heldInjuries := researchapp.MergeArchive(research.values(), a.core.SeasonInjuries, researchapp.ArchiveInput{Feeds: feeds, Depth: depth, Injuries: injury, News: news}, links, loaded, err, time.Now)
	research = researchFromValues(value)
	a.core.SeasonInjuries = heldInjuries
}

func (a *App) researchArchiveFeeds() researchapp.ArchiveFeeds {
	return researchapp.ArchiveFeeds{Bodies: a.core.History.ArchivedBodies, Scoped: a.scopedBodies}
}
