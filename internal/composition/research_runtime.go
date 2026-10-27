package composition

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	researchapp "github.com/secureprospective/TheWarRoom/internal/app/research"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

type researchHeld struct {
	feeds       map[string]heldSeasonFeed[[]researchfeeds.Feed]
	depth       heldSeasonFeed[[]researchfeeds.RawDepth]
	links       map[string]string
	linkError   string
	linkVersion string
	finishedAt  map[int]time.Time
	status      string
	news        map[string]heldSeasonFeed[[]researchfeeds.News]
	// archiveRead names the held copies already looked up in the archive this season, so a
	// copy the archive does not hold is looked up once, not on every read while the refresh runs.
	archiveRead map[string]bool
}

func researchKey(kind, week string) string { return researchapp.Key(kind, week) }
func (a *App) refreshResearchFeed(ctx context.Context, kind, week string, force bool) {
	a.core.SeasonMu.Lock()
	held := a.snapshotResearch().feeds[researchKey(kind, week)]
	a.core.SeasonMu.Unlock()
	researchapp.RefreshFeed(ctx, held, force, func(ctx context.Context) ([]researchfeeds.Feed, Freshness, error) {
		bodies := (researchapp.ArchiveFeeds{Bodies: a.core.History.ArchivedBodies, Scoped: a.scopedBodies}).ForWeek(week)
		return liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[researchfeeds.Feed]{
			Export: kind, Leagueless: researchfeeds.IsTrend(kind),
			Fetch: func(ctx context.Context) ([]researchfeeds.Feed, error) {
				return researchfeeds.Fetch(ctx, a.core.MFLClient, a.core.Season, kind, week)
			},
			Parse: func(body []byte) ([]researchfeeds.Feed, error) {
				feed, err := researchfeeds.Parse(body, kind, week)
				if err != nil {
					return nil, fmt.Errorf("research archive: %w", err)
				}
				return []researchfeeds.Feed{feed}, nil
			}, Bodies: bodies,
		}, a.core.FallbackLog)
	}, func(held researchapp.Feed[[]researchfeeds.Feed]) {
		a.core.SeasonMu.Lock()
		defer a.core.SeasonMu.Unlock()
		values := a.researchHeld.Snapshot()
		researchapp.PublishFeed(&values, kind, week, held)
		a.researchHeld.Publish(values)
	})
}
func (a *App) researchLinks(ctx context.Context, lookup normalize.Lookup) (map[string]string, error) {
	return researchapp.DepthLinks(ctx, lookup, a.core.History.SourceIDs) //nolint:wrapcheck // Preserve depth crosswalk errors.
}
func (a *App) refreshResearchDepth(ctx context.Context, force bool) {
	a.core.SeasonMu.Lock()
	held := a.snapshotResearch().depth
	a.core.SeasonMu.Unlock()
	researchapp.RefreshDepth(ctx, held, force, func(ctx context.Context) ([]researchfeeds.RawDepth, Freshness, error) {
		return a.researchDepthOrArchive(ctx, a.fallbackParent())
	}, func(ctx context.Context) (map[string]string, error) {
		a.core.LookupMu.Lock()
		lookup := a.core.Lookup
		a.core.LookupMu.Unlock()
		return a.researchLinks(ctx, lookup)
	}, func(held researchapp.Feed[[]researchfeeds.RawDepth], links map[string]string, err error) {
		a.core.SeasonMu.Lock()
		defer a.core.SeasonMu.Unlock()
		values := a.researchHeld.Snapshot()
		researchapp.PublishDepth(&values, held, links, err)
		a.researchHeld.Publish(values)
	})
}

func (a *App) refreshResearch(force bool) error {
	//nolint:wrapcheck // Preserve research startup and refresh cancellation errors.
	return researchapp.RefreshReading(force, a.ready, a.core.ResearchContext, a.researchStatus,
		func(run func() error) error {
			a.core.RefreshMu.Lock()
			defer a.core.RefreshMu.Unlock()
			return run()
		}, func() researchapp.RefreshSteps {
			return researchapp.RefreshSteps{Week: a.marketWeek, Status: a.researchStatus, Pause: report.RefreshPause(a.core.SeasonPause),
				Feed: a.refreshResearchFeed, Injuries: a.refreshInjuries, Schedule: a.refreshNowWeek, Depth: a.refreshResearchDepth}
		})
}

func (a *App) heldResearchInput() researchInput {
	a.core.SeasonMu.Lock()
	research := a.snapshotResearch()
	held := research
	injuries := a.core.SeasonInjuries
	injuries.Value = slices.Clone(injuries.Value)
	schedules := maps.Clone(a.now.schedules)
	for key, feed := range schedules {
		feed.Value = slices.Clone(feed.Value)
		for i := range feed.Value {
			feed.Value[i].Matchups = slices.Clone(feed.Value[i].Matchups)
		}
		schedules[key] = feed
	}
	a.core.SeasonMu.Unlock()
	lookup := a.core.ResearchDirectory()
	mirror := a.core.ResearchRoster()
	return researchInput{held: held, lookup: lookup, mirror: mirror, injuries: injuries, schedules: schedules,
		week: a.marketWeek(), zone: a.core.ResearchZone(), config: a.core.ResearchConfig()}
}

func (a *App) refreshResearchNews(mine string) error {
	var pause func(context.Context, time.Duration) error
	return (researchapp.NewsRefresh{ //nolint:wrapcheck // Preserve original news refresh errors.
		Ready: a.ready, Names: func() map[string]string { return a.core.Rulebook.FranchiseNames() },
		Parent: a.core.ResearchContext, TryLock: a.core.NewsMu.TryLock, Unlock: a.core.NewsMu.Unlock,
		Players: func(mine string) []string {
			mirror, _, _ := a.core.League.ContractsView()
			a.core.LookupMu.Lock()
			lookup := a.core.Lookup
			a.core.LookupMu.Unlock()
			ordered := researchapp.PlayerIDs(researchapp.RosterIDs(mirror, mine), lookup)
			pause = a.seasonPauseFunc()
			return ordered
		},
		Held: func() map[string]researchapp.Feed[[]researchfeeds.News] {
			a.core.SeasonMu.Lock()
			defer a.core.SeasonMu.Unlock()
			return maps.Clone(a.snapshotResearch().news)
		},
		Read: func(ctx context.Context, held map[string]researchapp.Feed[[]researchfeeds.News], ordered []string) error {
			return researchapp.ReadNewsBatches(ctx, held, a.core.Season, ordered, time.Now(), a.core.History.ArchivedBodies, pause,
				func(ctx context.Context, ids []string) (map[string][]researchfeeds.News, error) {
					return researchfeeds.FetchNews(ctx, a.core.MFLClient, a.core.Season, ids)
				}, a.core.FallbackLog)
		},
		Publish: func(held map[string]researchapp.Feed[[]researchfeeds.News]) {
			a.core.SeasonMu.Lock()
			defer a.core.SeasonMu.Unlock()
			research := a.snapshotResearch()
			research.news = held
			a.publishResearch(research)
		},
	}).Run(mine)
}

func (a *App) researchStatus(text string) {
	a.core.SeasonMu.Lock()
	a.researchHeld.PublishStatus(text)
	a.core.SeasonMu.Unlock()
}

func (a *App) researchDepthOrArchive(ctx, fallback context.Context) ([]researchfeeds.RawDepth, Freshness, error) {
	return researchapp.ReadDepth(ctx, fallback, a.core.Season, a.core.History.ArchivedBodies, //nolint:wrapcheck // Preserve depth fallback errors.
		func(ctx context.Context) ([]researchfeeds.RawDepth, error) {
			return researchfeeds.FetchDepth(ctx, &http.Client{Transport: a.core.Fetches, Timeout: rasFetchTimeout}, a.core.Season)
		}, a.core.FallbackLog)
}
