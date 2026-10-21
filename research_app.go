package main

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"time"

	researchapp "github.com/secureprospective/TheWarRoom/internal/app/research"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
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
	held := a.research.feeds[key]
	a.core.SeasonMu.Unlock()
	if !force && held.RefreshError == "" && time.Since(held.FetchedAt) < poolFreshFor {
		return
	}
	bodies := a.core.History.ArchivedBodies
	if week != "" {
		bodies = a.scopedBodies("W", week)
	}
	rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[researchfeeds.Feed]{
		export: kind, leagueless: researchfeeds.IsTrend(kind),
		fetch: func(ctx context.Context) ([]researchfeeds.Feed, error) {
			return researchfeeds.Fetch(ctx, a.core.MFLClient, a.core.Season, kind, week)
		},
		parse: func(body []byte) ([]researchfeeds.Feed, error) {
			feed, err := researchfeeds.Parse(body, kind, week)
			if err != nil {
				return nil, fmt.Errorf("research archive: %w", err)
			}
			return []researchfeeds.Feed{feed}, nil
		}, bodies: bodies,
	}, a.core.FallbackLog)
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	if a.research.feeds == nil {
		a.research.feeds = map[string]heldSeasonFeed[[]researchfeeds.Feed]{}
	}
	holdSeasonFeed(&held, rows, fresh, err)
	a.research.feeds[key] = held
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
	held := a.research.depth
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
	defer a.core.SeasonMu.Unlock()
	holdSeasonFeed(&held, rows, fresh, err)
	a.research.depth = held
	if lerr != nil {
		a.research.linkError = lerr.Error()
	} else {
		a.research.links = links
		a.research.linkError = ""
	}
}

func (a *App) TargetResearchRefresh(force bool) error {
	if err := a.ready(); err != nil {
		return fmt.Errorf("research refresh: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
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

func (a *App) TargetResearch(raw RawResearchScope) (ResearchBoard, error) {
	if err := raw.Validate(); err != nil {
		return ResearchBoard{}, err
	}
	if err := a.ready(); err != nil {
		return ResearchBoard{}, fmt.Errorf("research: %w", err)
	}
	if raw.Mine != "" && a.core.Rulebook.FranchiseNames()[raw.Mine] == "" {
		return ResearchBoard{}, fmt.Errorf("research: unknown franchise")
	}
	a.hydrateResearch()
	return buildResearch(a.heldResearchInput(), raw), nil
}

func (a *App) heldResearchInput() researchInput {
	a.core.SeasonMu.Lock()
	held := a.research
	held.feeds, held.news = maps.Clone(held.feeds), maps.Clone(held.news)
	held.links = maps.Clone(held.links)
	injuries := a.core.SeasonInjuries
	schedules := maps.Clone(a.now.schedules)
	a.core.SeasonMu.Unlock()
	a.core.LookupMu.Lock()
	lookup := a.core.Lookup
	a.core.LookupMu.Unlock()
	mirror, _, _ := a.core.League.ContractsView()
	return researchInput{held: held, lookup: lookup, mirror: mirror, injuries: injuries, schedules: schedules,
		week: a.marketWeek(), zone: time.Local, config: a.core.Rulebook.ActiveConfig()}
}

func (a *App) TargetResearchNewsRefresh(mine string) error {
	if mine != "" {
		if err := (RawResearchScope{Mine: mine}).Validate(); err != nil {
			return err
		}
	}
	if err := a.ready(); err != nil {
		return fmt.Errorf("news refresh: %w", err)
	}
	if mine != "" && a.core.Rulebook.FranchiseNames()[mine] == "" {
		return fmt.Errorf("news: unknown franchise")
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, 30*time.Minute)
	defer cancel()
	if !a.core.NewsMu.TryLock() {
		return fmt.Errorf("news: already refreshing")
	}
	defer a.core.NewsMu.Unlock()
	mirror, _, _ := a.core.League.ContractsView()
	a.core.LookupMu.Lock()
	lookup := a.core.Lookup
	a.core.LookupMu.Unlock()
	ordered := researchPlayerIDs(researchRosterIDs(mirror, mine), lookup)
	pause := a.core.SeasonPause
	if pause == nil {
		pause = seasonPause
	}
	a.core.SeasonMu.Lock()
	held := maps.Clone(a.research.news)
	a.core.SeasonMu.Unlock()
	if held == nil {
		held = map[string]heldSeasonFeed[[]researchfeeds.News]{}
	}
	err := readResearchNewsBatches(ctx, held, a.core.Season, ordered, time.Now(), a.core.History.ArchivedBodies, pause,
		func(ctx context.Context, ids []string) (map[string][]researchfeeds.News, error) {
			return researchfeeds.FetchNews(ctx, a.core.MFLClient, a.core.Season, ids)
		}, a.core.FallbackLog)
	a.core.SeasonMu.Lock()
	a.research.news = held
	a.core.SeasonMu.Unlock()
	return err
}
func (a *App) TargetOpenResearchArticle(id, article string) error {
	pid, err := playerid.New(id)
	if err != nil {
		return fmt.Errorf("news hand-off: %w", err)
	}
	if err := a.ready(); err != nil {
		return fmt.Errorf("news hand-off: %w", err)
	}
	a.core.SeasonMu.Lock()
	news := a.research.news[pid.String()]
	a.core.SeasonMu.Unlock()
	found := false
	for _, n := range news.Value {
		found = found || n.Article == article
	}
	if !found {
		return fmt.Errorf("news: article is not held for this player")
	}
	host := a.core.MFLClient.LeagueHost()
	if host == "" {
		return fmt.Errorf("news: MFL host not held; refresh from MFL first")
	}
	target := fmt.Sprintf("https://%s/%d/view_news_article?ID=%s", host, a.core.Season, url.QueryEscape(article))
	a.core.OpenURL(a.core.Ctx, target)
	return nil
}

func (a *App) TargetResearchPlayer(id string) (ResearchPlayerReading, error) {
	pid, err := playerid.New(id)
	if err != nil {
		return ResearchPlayerReading{}, fmt.Errorf("research player: %w", err)
	}
	if err := a.ready(); err != nil {
		return ResearchPlayerReading{}, fmt.Errorf("research player: %w", err)
	}
	a.hydrateResearch(pid.String())
	if a.core.Fetches != nil {
		ctx, cancel := context.WithTimeout(a.fallbackParent(), rasFetchTimeout)
		defer cancel()
		a.refreshResearchDepth(ctx, false)
	}
	in := a.heldResearchInput()
	if _, _, ok := in.lookup.DisplayFacts(pid.String()); !ok {
		return ResearchPlayerReading{}, fmt.Errorf("research: player not in the held directory")
	}
	return wireResearchPlayer(researchapp.ReadPlayer(in.snapshot(), pid.String(), a.core.Rulebook.FranchiseNames(), "How this is counted")), nil
}

func (a *App) researchStatus(text string) {
	a.core.SeasonMu.Lock()
	a.research.status = text
	a.core.SeasonMu.Unlock()
}
func (a *App) TargetResearchStatus() string {
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	return a.research.status
}

func researchRosterIDs(mirror state.MirrorSnapshot, mine string) []string {
	ids := map[string]bool{}
	for _, p := range mirror.Players {
		if mine == "" || p.FranchiseID == mine {
			ids[p.MFLID] = true
		}
	}
	ordered := make([]string, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	return ordered
}

func researchPlayerIDs(ids []string, lookup normalize.Lookup) []string {
	return slices.DeleteFunc(slices.Clone(ids), func(id string) bool {
		_, position, _ := lookup.DisplayFacts(id)
		return researchfeeds.TeamUnit(position)
	})
}

func (a *App) researchDepthOrArchive(ctx, fallback context.Context) ([]researchfeeds.RawDepth, Freshness, error) {
	rows, err := researchfeeds.FetchDepth(ctx, &http.Client{Transport: a.core.Fetches, Timeout: rasFetchTimeout}, a.core.Season)
	if err == nil {
		return rows, liveFreshness(time.Now()), nil
	}
	a.core.FallbackLog(fmt.Sprintf("the war room: depth charts: live fetch failed: %v", err))
	return readResearchDepthArchive(fallback, a.core.Season, a.core.History.ArchivedBodies, err)
}
func readResearchDepthArchive(ctx context.Context, season int, bodies archivedBodies, fetchErr error) ([]researchfeeds.RawDepth, Freshness, error) {
	rctx, cancel := context.WithTimeout(ctx, fallbackTimeout)
	defer cancel()
	var fresh Freshness
	var rows []researchfeeds.RawDepth
	found, aerr := bodies(rctx, researchfeeds.DepthURL(season), func(src string, body []byte, at time.Time) bool {
		if src != researchfeeds.DepthURL(season) {
			return false
		}
		parsed, perr := researchfeeds.ParseDepth(body)
		if perr != nil {
			return false
		}
		rows = parsed
		fresh = staleFreshness(at, fetchErr)
		return true
	})
	if aerr != nil {
		return nil, Freshness{}, fmt.Errorf("depth chart archive: %w", aerr)
	}
	if !found {
		return nil, Freshness{}, fmt.Errorf("depth charts: no archived copy: %w", fetchErr)
	}
	return rows, fresh, nil
}
