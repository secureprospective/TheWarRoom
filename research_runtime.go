package main

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"time"

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

func researchKey(kind, week string) string { return kind + ":" + week }
func (a *App) refreshResearchFeed(ctx context.Context, kind, week string, force bool) {
	key := researchKey(kind, week)
	a.core.SeasonMu.Lock()
	research := a.snapshotResearch()
	held := research.feeds[key]
	a.core.SeasonMu.Unlock()
	if !force && held.RefreshError == "" && time.Since(held.FetchedAt) < poolFreshFor {
		return
	}
	bodies := a.core.History.ArchivedBodies
	if week != "" {
		bodies = a.scopedBodies("W", week)
	}
	rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[researchfeeds.Feed]{
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
	a.core.SeasonMu.Lock()
	research = a.snapshotResearch()
	defer a.core.SeasonMu.Unlock()
	defer func() { a.publishResearch(research) }()
	if research.feeds == nil {
		research.feeds = map[string]heldSeasonFeed[[]researchfeeds.Feed]{}
	}
	holdSeasonFeed(&held, rows, fresh, err)
	research.feeds[key] = held
}
func (a *App) researchLinks(ctx context.Context, lookup normalize.Lookup) (map[string]string, error) {
	out := map[string]string{}
	for _, kind := range []string{"gsis", "espn"} {
		sources, err := a.core.History.SourceIDs(ctx, kind)
		if err != nil {
			return nil, fmt.Errorf("depth crosswalk: %w", err)
		}
		for _, id := range lookup.IDs() {
			if source := sources[id]; source != "" {
				out[kind+":"+source] = id
			}
		}
	}
	return out, nil
}
func (a *App) refreshResearchDepth(ctx context.Context, force bool) {
	a.core.SeasonMu.Lock()
	research := a.snapshotResearch()
	held := research.depth
	a.core.SeasonMu.Unlock()
	if !force && held.RefreshError == "" && time.Since(held.FetchedAt) < 24*time.Hour {
		return
	}
	rows, fresh, err := a.researchDepthOrArchive(ctx, a.fallbackParent())
	a.core.LookupMu.Lock()
	lookup := a.core.Lookup
	a.core.LookupMu.Unlock()
	links, lerr := a.researchLinks(ctx, lookup)
	a.core.SeasonMu.Lock()
	research = a.snapshotResearch()
	defer a.core.SeasonMu.Unlock()
	defer func() { a.publishResearch(research) }()
	holdSeasonFeed(&held, rows, fresh, err)
	research.depth = held
	if lerr != nil {
		research.linkError = lerr.Error()
	} else {
		research.links = links
		research.linkError = ""
	}
}

func (a *App) refreshResearch(force bool) error {
	if err := a.ready(); err != nil {
		return fmt.Errorf("research refresh: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.core.ResearchContext(), refreshTimeout)
	defer cancel()
	a.researchStatus("Waiting for the MFL refresh to finish")
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	defer a.researchStatus("")
	pause := a.core.SeasonPause
	if pause == nil {
		pause = seasonPause
	}
	week := a.marketWeek()
	for _, kind := range researchKinds() {
		a.researchStatus("Reading " + researchFeedLabel(kind))
		if err := pause(ctx, time.Second); err != nil {
			return err
		}
		window := ""
		if kind == "playerScores" {
			window = "YTD"
		}
		a.refreshResearchFeed(ctx, kind, window, force)
	}
	for w := 1; w <= week; w++ {
		a.researchStatus(fmt.Sprintf("Reading week %d of %d", w, week))
		if err := pause(ctx, time.Second); err != nil {
			return err
		}
		a.refreshResearchFeed(ctx, "playerScores", strconv.Itoa(w), force)
	}
	a.researchStatus("Reading NFL injuries")
	a.refreshInjuries(ctx)
	if week > 0 {
		a.researchStatus("Reading this week’s NFL schedule")
		a.refreshNowWeek(ctx, week, force)
	}
	a.researchStatus("Reading NFL depth charts")
	a.refreshResearchDepth(ctx, force)
	return nil
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
	if mine != "" {
		if err := (researchapp.RawResearchScope{Mine: mine}).Validate(); err != nil {
			return err //nolint:wrapcheck // Preserve the binding validation error text.
		}
	}
	if err := a.ready(); err != nil {
		return fmt.Errorf("news refresh: %w", err)
	}
	if mine != "" && a.core.Rulebook.FranchiseNames()[mine] == "" {
		return fmt.Errorf("news: unknown franchise")
	}
	ctx, cancel := context.WithTimeout(a.core.ResearchContext(), 30*time.Minute)
	defer cancel()
	if !a.core.NewsMu.TryLock() {
		return fmt.Errorf("news: already refreshing")
	}
	defer a.core.NewsMu.Unlock()
	mirror, _, _ := a.core.League.ContractsView()
	a.core.LookupMu.Lock()
	lookup := a.core.Lookup
	a.core.LookupMu.Unlock()
	ordered := researchapp.PlayerIDs(researchapp.RosterIDs(mirror, mine), lookup)
	pause := a.core.SeasonPause
	if pause == nil {
		pause = seasonPause
	}
	a.core.SeasonMu.Lock()
	research := a.snapshotResearch()
	held := maps.Clone(research.news)
	a.core.SeasonMu.Unlock()
	if held == nil {
		held = map[string]heldSeasonFeed[[]researchfeeds.News]{}
	}
	err := researchapp.ReadNewsBatches(ctx, held, a.core.Season, ordered, time.Now(), a.core.History.ArchivedBodies, pause,
		func(ctx context.Context, ids []string) (map[string][]researchfeeds.News, error) {
			return researchfeeds.FetchNews(ctx, a.core.MFLClient, a.core.Season, ids)
		}, a.core.FallbackLog)
	a.core.SeasonMu.Lock()
	research = a.snapshotResearch()
	research.news = held
	a.publishResearch(research)
	a.core.SeasonMu.Unlock()
	return err //nolint:wrapcheck // The service already supplies the original news error context.
}
func (a *App) researchStatus(text string) {
	a.core.SeasonMu.Lock()
	a.researchHeld.PublishStatus(text)
	a.core.SeasonMu.Unlock()
}

func (a *App) researchDepthOrArchive(ctx, fallback context.Context) ([]researchfeeds.RawDepth, Freshness, error) {
	rows, err := researchfeeds.FetchDepth(ctx, &http.Client{Transport: a.core.Fetches, Timeout: rasFetchTimeout}, a.core.Season)
	if err == nil {
		return rows, liveFreshness(time.Now()), nil
	}
	a.core.FallbackLog(fmt.Sprintf("the war room: depth charts: live fetch failed: %v", err))
	//nolint:wrapcheck // The archive service preserves the existing depth error text.
	return researchapp.ReadDepthArchive(fallback, a.core.Season, a.core.History.ArchivedBodies, err)
}
