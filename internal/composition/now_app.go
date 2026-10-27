package composition

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"
	platformapp "github.com/secureprospective/TheWarRoom/internal/app/platform"
	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

// nowSchedules is the NFL schedule of each week opened. It stays on App under SeasonMu because
// Race, Week, Research and the finished-scores check read it too; Now's private byes, first scores
// and live-game poller are in app.nowHeld.
type nowSchedules struct {
	schedules map[int]heldSeasonFeed[[]nflschedule.RawWeek]
}

// nowHost is the Now API's Host. It is never bound: only now.API exposes methods to Wails. The
// refresh and the live-game poll stay here because they span the refresh, season and live-game
// locks and call the race, week, market and photo features' root code.
type nowHost struct {
	*core.State
	app *App
}

func (a *App) nowAPI() *nowapp.API { return nowapp.New(nowHost{State: a.core, app: a}, a.nowHeld) }

func (a *App) refreshScoreFirsts(ctx context.Context) {
	a.core.SeasonMu.Lock()
	prev := a.nowHeld.SnapshotFirsts()
	a.core.SeasonMu.Unlock()
	nowapp.RefreshFirsts(ctx, prev, a.core.Season, a.core.History.ArchivedBodies, func(next nowapp.FirstScores) {
		a.core.SeasonMu.Lock()
		defer a.core.SeasonMu.Unlock()
		a.nowHeld.PublishFirsts(next)
	}, a.core.FallbackLog)
}

// refreshNowWeek reads the chosen week's NFL schedule and the season's byes, each live or from
// the archive, unless held and fresh; force always reads them.
func (a *App) refreshNowWeek(ctx context.Context, week int, force bool) {
	pause := a.core.SeasonPause
	if pause == nil {
		pause = seasonPause
	}
	year := strconv.Itoa(a.core.Season)
	a.core.SeasonMu.Lock()
	held, byes := a.now.schedules[week], a.nowHeld.SnapshotByes()
	a.core.SeasonMu.Unlock()
	nowapp.RefreshWeekFeeds(ctx, force, held.FetchedAt, held.RefreshError, byes.FetchedAt, byes.RefreshError,
		nowapp.WeekFeedSource{Pause: pause, Since: time.Since, Schedule: func(ctx context.Context) {
			rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.core.Season, a.scheduleFeed(week), a.core.FallbackLog)
			a.core.SeasonMu.Lock()
			schedules := maps.Clone(a.now.schedules)
			if schedules == nil {
				schedules = map[int]heldSeasonFeed[[]nflschedule.RawWeek]{}
			}
			h := schedules[week]
			holdSeasonFeed(&h, rows, fresh, err)
			schedules[week] = h
			a.now.schedules = schedules
			a.core.SeasonMu.Unlock()
		}, Byes: func(ctx context.Context) {
			rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[nflschedule.RawBye]{
				Export: nflschedule.ByeExport, Leagueless: true,
				Fetch: func(ctx context.Context) ([]nflschedule.RawBye, error) {
					return nflschedule.FetchByes(ctx, a.core.MFLClient, year)
				},
				Parse: nflschedule.ParseByes, Bodies: a.core.History.ArchivedBodies,
			}, a.core.FallbackLog)
			a.core.SeasonMu.Lock()
			defer a.core.SeasonMu.Unlock()
			next := heldFeed(a.nowHeld.SnapshotByes())
			holdSeasonFeed(&next, rows, fresh, err)
			a.nowHeld.PublishByes(nowFeed(next))
		}})
}

func (a *App) scheduleFeed(week int) archivedFeed[nflschedule.RawWeek] {
	year := strconv.Itoa(a.core.Season)
	return archivedFeed[nflschedule.RawWeek]{
		Export: nflschedule.Export, Leagueless: true,
		Fetch: func(ctx context.Context) ([]nflschedule.RawWeek, error) {
			raw, ferr := nflschedule.Fetch(ctx, a.core.MFLClient, year, week)
			if ferr != nil {
				return nil, fmt.Errorf("NFL schedule: %w", ferr)
			}
			return checkScheduleWeek(raw, week)
		},
		Parse: func(body []byte) ([]nflschedule.RawWeek, error) {
			raw, perr := nflschedule.Parse(body)
			if perr != nil {
				return nil, fmt.Errorf("NFL schedule: %w", perr)
			}
			return checkScheduleWeek(raw, week)
		},
		Bodies: a.scopedBodies("W", strconv.Itoa(week)),
	}
}

func (a *App) seedClockSchedules(ctx context.Context) {
	nowapp.SeedClockSchedules(ctx, a.seedCurrentWeek, a.lastRegularWeek,
		func(ctx context.Context, week int) ([]nflschedule.RawWeek, time.Time, bool, error) {
			return readArchivedFeed(ctx, a.core.Season, a.scheduleFeed(week))
		}, func(week int, rows []nflschedule.RawWeek, at time.Time) {
			a.core.SeasonMu.Lock()
			if a.now.schedules == nil {
				a.now.schedules = map[int]heldSeasonFeed[[]nflschedule.RawWeek]{}
			}
			a.now.schedules[week] = heldSeasonFeed[[]nflschedule.RawWeek]{Value: rows, FetchedAt: at}
			a.core.SeasonMu.Unlock()
		}, a.core.FallbackLog)
}

func checkScheduleWeek(raw nflschedule.RawWeek, week int) ([]nflschedule.RawWeek, error) {
	return nowapp.CheckScheduleWeek(raw, week) //nolint:wrapcheck // Preserve requested-week refusal text.
}

// KnownFranchise reports whether id is one of the rulebook's franchises.
func (h nowHost) KnownFranchise(id string) bool {
	return nowapp.KnownFranchise(h.Rulebook.FranchiseNames(), id)
}

// liveGameWindows copies the held week's NFL game windows under the week lock.
func (h nowHost) liveGameWindows() []livegame.Window {
	h.WeekMu.Lock()
	defer h.WeekMu.Unlock()
	out := make([]livegame.Window, 0, len(h.Week.Games))
	for _, g := range h.Week.Games {
		out = append(out, livegame.Window{Teams: g.Teams, Kickoff: g.Kickoff, Final: g.Final})
	}
	return out
}

// NowRefresh keeps the binding's original order: the race refresh, then the refresh lock over the
// first scores, the week check and the week's schedule.
func (h nowHost) NowRefresh(week int, force bool) error {
	a := h.app
	//nolint:wrapcheck // Preserve the feature refusal text.
	return nowapp.RefreshNow(week, force, raceapp.New(raceHost{a.core}, a.raceHeld).TargetRaceRefresh,
		func(week int, force bool) error {
			ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
			defer cancel()
			a.core.RefreshMu.Lock()
			defer a.core.RefreshMu.Unlock()
			a.refreshScoreFirsts(ctx)
			week = nowapp.SelectedRefreshWeek(week, a.marketWeek)
			a.core.SeasonMu.Lock()
			weeks := a.raceHeld.Snapshot().Weeks.Value
			a.core.SeasonMu.Unlock()
			return nowapp.RefreshSelectedWeek(ctx, week, weeks, a.marketWeek(), force, a.refreshNowWeek)
		})
}

// NowInput is the joint capture: under SeasonMu it snapshots Now's holder, then copies the shared
// race, schedule and lineup feeds, so both come from one section.
func (h nowHost) NowInput(week, reportWeek int, mine string) nowapp.Snapshot {
	a := h.app
	current := a.marketWeek()
	projections := a.heldMarketForWeek(current)[marketfeeds.Projections]
	a.core.SeasonMu.Lock()
	byes, firsts := a.nowHeld.SnapshotByes(), a.nowHeld.SnapshotFirsts()
	schedules := make(map[int]nowapp.Feed[[]nflschedule.RawWeek], len(a.now.schedules))
	for k, v := range a.now.schedules {
		schedules[k] = nowFeed(v)
	}
	rh := a.raceHeld.Snapshot()
	race := season.Held{Weeks: rh.Weeks, Brackets: rh.Brackets, Standings: rh.Standings}
	in := nowapp.Snapshot{Race: season.Snapshot{Held: race}, Held: nowapp.Feeds{Schedules: schedules, Byes: byes, Firsts: firsts},
		Live: nowFeed(a.core.SeasonLineups), Projections: nowFeed(projections), At: a.timeNow(), Current: current, Week: week,
		ReportWeek: reportWeek}
	a.core.SeasonMu.Unlock()
	a.core.LookupMu.Lock()
	in.Race.Lookup, in.Race.HasLookup = a.core.Lookup, a.core.HasLookup
	a.core.LookupMu.Unlock()
	in.Race.Config, in.Race.Mine, in.Race.Zone = a.core.Rulebook.ActiveConfig(), mine, time.Local
	return in
}

// LiveGame runs after the API's readiness, follow and franchise checks, in their original order.
// The poller's held state is taken from the holder under the live-game lock and published back
// before that lock is released, so the lock keeps its original span.
func (h nowHost) LiveGame(open bool, follow, mine string) (nowapp.LiveGameView, error) {
	a := h.app
	ctx, cancel := context.WithTimeout(a.core.Ctx, 35*time.Second)
	defer cancel()
	a.core.LiveGameMu.Lock()
	defer a.core.LiveGameMu.Unlock()
	live := a.nowHeld.SnapshotLive()
	defer func() { a.nowHeld.PublishLive(live) }()
	//nolint:wrapcheck // Preserve the feature refusal text.
	return nowapp.RefreshLiveGame(ctx, &live, open, follow, mine, nowapp.LiveSource{
		Start: a.startLiveGame, Now: time.Now, Windows: h.liveGameWindows, Week: a.marketWeek,
		Seed: a.seedLiveGames, Poll: a.pollLiveMFL, Spines: a.liveGameSpines,
	})
}

func (a *App) startLiveGame(adapter *livegame.Adapter) error {
	//nolint:wrapcheck // Preserve the feature refusal text.
	return nowapp.StartLiveGame(adapter, func() (livegame.Fetch, livegame.Finals, error) {
		endpoints, err := nowapp.LiveEndpoints()
		if err != nil {
			return nil, nil, err //nolint:wrapcheck // LiveEndpoints supplies its existing refusal.
		}
		fetch := livegame.HTTPFetch(&http.Client{Timeout: 10 * time.Second, Transport: a.core.Network}, endpoints)
		finals := adapter.Finals
		if a.core.History != nil {
			finals = nowapp.HeldFinals{Store: a.core.History, Endpoints: endpoints}
		}
		return fetch, finals, nil
	})
}

func (a *App) pollLiveMFL(ctx context.Context, live *nowapp.LiveGame, now time.Time, active bool) {
	nowapp.PollLiveMFL(ctx, live, now, active, func(ctx context.Context) bool {
		a.core.RefreshMu.Lock()
		a.refreshSeasonLineups(ctx)
		a.core.RefreshMu.Unlock()
		a.core.SeasonMu.Lock()
		failed := a.core.SeasonLineups.RefreshError != ""
		a.core.SeasonMu.Unlock()
		return failed
	})
}

func (a *App) seedLiveGames(at time.Time) livegame.Held {
	week := a.marketWeek()
	a.core.SeasonMu.Lock()
	held := a.now.schedules[week]
	a.core.SeasonMu.Unlock()
	return nowapp.SeedLiveGames(nowFeed(held), at)
}

func liveGameReading(a livegame.Adapter, follow string, active bool) nowapp.LiveGameView {
	return nowapp.LiveReading(nowapp.LiveSnapshot{Board: a.Board, Details: maps.Clone(a.Details), Follow: follow, Active: active, Zone: time.Local})
}

func (a *App) heldMatchupStarters(mine string) map[string]bool {
	week := a.marketWeek()
	a.core.SeasonMu.Lock()
	live, weeks := a.core.SeasonLineups.Value, a.raceHeld.Snapshot().Weeks.Value
	a.core.SeasonMu.Unlock()
	return nowapp.HeldMatchupStarters(live, weeks, week, mine)
}

// liveGameSpines fills the followed game's spine and the week's: every held play (followed
// games and finals already read) with a starter from my matchup. Nothing is fetched here.
func (a *App) liveGameSpines(ctx context.Context, v *nowapp.LiveGameView, adapter livegame.Adapter, mine string) error {
	//nolint:wrapcheck // Preserve the feature refusal text.
	return nowapp.FillLiveSpines(ctx, v, adapter, mine, nowapp.SpineSource{Available: a.core.History != nil,
		Known:   func(ctx context.Context) (map[string]bool, error) { return a.core.History.KnownIDs(ctx, "espn") },
		Inspect: a.inspectLiveBox, Players: a.spinePlayers})
}

func (a *App) spinePlayers(ctx context.Context, mine string) (map[string]nowapp.SpinePlayer, error) {
	ids := a.heldMatchupStarters(mine)
	espnIDs, err := a.core.History.SourceIDs(ctx, "espn")
	if err != nil {
		return nil, fmt.Errorf("live game player crosswalk: %w", err)
	}
	a.core.LookupMu.Lock()
	lookup := a.core.Lookup
	a.core.LookupMu.Unlock()
	return nowapp.SpinePlayers(ids, espnIDs, lookup, a.core.PhotosOff.Load,
		func(id, name string, off bool) nowapp.PhotoView {
			return nowapp.PhotoView(platformapp.Photo(id, name, off))
		}), nil
}

func (a *App) inspectLiveBox(ctx context.Context, v *nowapp.LiveGameView) error {
	ids, err := a.core.History.SourceIDs(ctx, "espn")
	if err != nil {
		return fmt.Errorf("box score crosswalk: %w", err)
	}
	nowapp.InspectLiveBox(v, ids)
	return nil
}

func nowFeed[T any](f heldSeasonFeed[T]) nowapp.Feed[T] {
	return nowapp.Feed[T]{Value: f.Value, FetchedAt: f.FetchedAt, RefreshError: f.RefreshError, ReadPending: f.ReadPending}
}
func heldFeed[T any](f nowapp.Feed[T]) heldSeasonFeed[T] {
	return heldSeasonFeed[T]{Value: f.Value, FetchedAt: f.FetchedAt, RefreshError: f.RefreshError, ReadPending: f.ReadPending}
}
func liveWeek(l leaguefeed.Lineups, base leaguefeed.WeekResult) leaguefeed.WeekResult {
	return nowapp.LiveWeek(l, base)
}
func liveWeekUnplayed(l heldSeasonFeed[leaguefeed.Lineups], s map[int]heldSeasonFeed[[]nflschedule.RawWeek], at time.Time) bool {
	schedules := make(map[int]nowapp.Feed[[]nflschedule.RawWeek], len(s))
	for k, v := range s {
		schedules[k] = nowFeed(v)
	}
	return nowapp.LiveWeekUnplayed(nowFeed(l), schedules, at)
}
func possessive(s string) string { return nowapp.Possessive(s) }
