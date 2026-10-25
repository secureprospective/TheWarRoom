package composition

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"
	platformapp "github.com/secureprospective/TheWarRoom/internal/app/platform"
	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
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
	next, err := nowapp.ReadFirsts(ctx, prev, a.core.Season, a.core.History.ArchivedBodies)
	if err != nil {
		a.core.FallbackLog("the war room: " + err.Error())
		return
	}
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	a.nowHeld.PublishFirsts(next)
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
	if force || !feedFresh(held.FetchedAt, held.RefreshError) {
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
		if err := pause(ctx, time.Second); err != nil {
			return
		}
	}
	if !force && byes.RefreshError == "" && time.Since(byes.FetchedAt) < 24*time.Hour {
		return
	}
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
	a.seedCurrentWeek(ctx)
	for _, rule := range leagueclock.Rules(a.lastRegularWeek()) {
		if rule.Week == 0 {
			continue
		}
		rows, at, found, err := readArchivedFeed(ctx, a.core.Season, a.scheduleFeed(rule.Week))
		if err != nil {
			a.core.FallbackLog(err.Error())
			continue
		}
		if !found {
			continue
		}
		a.core.SeasonMu.Lock()
		if a.now.schedules == nil {
			a.now.schedules = map[int]heldSeasonFeed[[]nflschedule.RawWeek]{}
		}
		a.now.schedules[rule.Week] = heldSeasonFeed[[]nflschedule.RawWeek]{Value: rows, FetchedAt: at}
		a.core.SeasonMu.Unlock()
	}
}

func checkScheduleWeek(raw nflschedule.RawWeek, week int) ([]nflschedule.RawWeek, error) {
	if raw.Week != strconv.Itoa(week) {
		return nil, fmt.Errorf("NFL schedule: requested week %d, received %s", week, raw.Week)
	}
	return []nflschedule.RawWeek{raw}, nil
}

func feedFresh(at time.Time, refreshError string) bool {
	return refreshError == "" && time.Since(at) < poolFreshFor
}

// KnownFranchise reports whether id is one of the rulebook's franchises.
func (h nowHost) KnownFranchise(id string) bool {
	_, ok := h.Rulebook.FranchiseNames()[id]
	return ok
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
	if err := raceapp.New(a.core, a.raceHeld).TargetRaceRefresh(force); err != nil {
		return fmt.Errorf("now refresh: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
	defer cancel()
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	a.refreshScoreFirsts(ctx)
	if week == 0 {
		week = a.marketWeek()
	}
	a.core.SeasonMu.Lock()
	listed := false
	for _, w := range a.raceHeld.Snapshot().Weeks.Value {
		listed = listed || w.Week == week
	}
	a.core.SeasonMu.Unlock()
	// The current NFL week is always fetched: its schedule does not depend on the league's weeks being
	// held yet (just after launch they may not be). Any other week must be on the league's schedule.
	current := week == a.marketWeek()
	if !listed && !current {
		return fmt.Errorf("now refresh: week %d is not on the league's schedule", week)
	}
	if week < 1 {
		return nil
	}
	a.refreshNowWeek(ctx, week, force)
	return nil
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
	if err := a.startLiveGame(&live.Adapter); err != nil {
		return nowapp.LiveGameView{}, err
	}
	now := time.Now()
	windows := h.liveGameWindows()
	week := a.marketWeek()
	if live.Week != week {
		fetch, finals := live.Adapter.Fetch, live.Adapter.Finals
		live = nowapp.LiveGame{Week: week, Adapter: livegame.Adapter{Fetch: fetch, Finals: finals}}
	}
	adapter := &live.Adapter
	if len(adapter.Board.Games) == 0 {
		adapter.Board = a.seedLiveGames(now)
	}
	if follow != "" {
		if _, ok := adapter.Followed(follow); !ok {
			return nowapp.LiveGameView{}, fmt.Errorf("game.follow id: not in the held week")
		}
	}
	adapter.Refresh(ctx, now, open, windows, follow)
	adapter.Final(ctx, now, follow)
	active := livegame.Live(now, adapter.EffectiveWindows(windows))
	a.pollLiveMFL(ctx, &live, now, open && active)
	view := liveGameReading(*adapter, follow, active)
	if err := a.liveGameSpines(ctx, &view, *adapter, mine); err != nil {
		view.DetailFailure += " " + report.FailureText(err)
	}
	return view, nil
}

func (a *App) startLiveGame(adapter *livegame.Adapter) error {
	if adapter.Fetch != nil {
		return nil
	}
	endpoints, err := nowapp.LiveEndpoints()
	if err != nil {
		return err //nolint:wrapcheck // LiveEndpoints keeps the original "live game sources" text.
	}
	adapter.Fetch = livegame.HTTPFetch(&http.Client{Timeout: 10 * time.Second, Transport: a.core.Network}, endpoints)
	if a.core.History != nil {
		adapter.Finals = nowapp.HeldFinals{Store: a.core.History, Endpoints: endpoints}
	}
	return nil
}

func (a *App) pollLiveMFL(ctx context.Context, live *nowapp.LiveGame, now time.Time, active bool) {
	if active && !now.Before(live.NextMFL) {
		a.core.RefreshMu.Lock()
		a.refreshSeasonLineups(ctx)
		a.core.RefreshMu.Unlock()
		a.core.SeasonMu.Lock()
		failed := a.core.SeasonLineups.RefreshError != ""
		a.core.SeasonMu.Unlock()
		if failed {
			live.MFLFailures++
		} else {
			live.MFLFailures = 0
		}
		live.NextMFL = now.Add(30 * time.Second * time.Duration(1<<min(live.MFLFailures, 4)))
	}
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
	held := nowapp.HeldWeekGames(nowapp.LiveSnapshot{Details: maps.Clone(adapter.Details)})
	v.WeekHeld = len(held)
	if v.Followed == nil && len(held) == 0 {
		return nil
	}
	if a.core.History == nil {
		return fmt.Errorf("crosswalk directory unavailable")
	}
	known, err := a.core.History.KnownIDs(ctx, "espn")
	if err != nil {
		return fmt.Errorf("live game crosswalk: %w", err)
	}
	if v.Followed != nil {
		if err := a.inspectLiveBox(ctx, v); err != nil {
			return err
		}
	}
	players, err := a.spinePlayers(ctx, mine)
	if err != nil {
		return err
	}
	if v.Followed != nil {
		v.Spine, v.Unmatched = nowapp.MatchSpine([]livegame.Game{*v.Followed}, known, players)
	}
	v.WeekSpine, _ = nowapp.MatchSpine(held, known, players)
	return nil
}

func (a *App) spinePlayers(ctx context.Context, mine string) (map[string]nowapp.SpinePlayer, error) {
	ids := a.heldMatchupStarters(mine)
	players := map[string]nowapp.SpinePlayer{}
	espnIDs, err := a.core.History.SourceIDs(ctx, "espn")
	if err != nil {
		return nil, fmt.Errorf("live game player crosswalk: %w", err)
	}
	a.core.LookupMu.Lock()
	lookup := a.core.Lookup
	a.core.LookupMu.Unlock()
	for _, id := range slices.Sorted(maps.Keys(ids)) {
		espn := espnIDs[id]
		if espn == "" {
			continue
		}
		facts, ok := lookup.Facts(id)
		if !ok {
			continue
		}
		photo := platformapp.Photo(id, proseName(facts.Name), a.core.PhotosOff.Load())
		players[espn] = nowapp.SpinePlayer{ID: id, Name: proseName(facts.Name), Team: facts.Team, Mine: ids[id], Photo: nowapp.PhotoView(photo)}
	}
	return players, nil
}

func (a *App) inspectLiveBox(ctx context.Context, v *nowapp.LiveGameView) error {
	ids, err := a.core.History.SourceIDs(ctx, "espn")
	if err != nil {
		return fmt.Errorf("box score crosswalk: %w", err)
	}
	bySource := map[string]string{}
	for id, source := range ids {
		bySource[source] = id
	}
	entities := map[string]report.Entity{}
	for _, box := range v.Followed.Box {
		for _, p := range box.Rows {
			if id := bySource[p.ID]; id != "" {
				entities[id] = report.Entity{Text: p.Name, Kind: "player", ID: id}
			}
		}
	}
	nowapp.AttachReportEntities(&v.Box, slices.Collect(maps.Values(entities)))
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
func nflGames(w []nflschedule.RawWeek, at time.Time) map[string]nowapp.NFLGame {
	return nowapp.NFLGames(w, at)
}
func weekKickedOff(g map[string]nowapp.NFLGame, at time.Time) bool {
	return nowapp.WeekKickedOff(g, at)
}
func liveWeekUnplayed(l heldSeasonFeed[leaguefeed.Lineups], s map[int]heldSeasonFeed[[]nflschedule.RawWeek], at time.Time) bool {
	schedules := make(map[int]nowapp.Feed[[]nflschedule.RawWeek], len(s))
	for k, v := range s {
		schedules[k] = nowFeed(v)
	}
	return nowapp.LiveWeekUnplayed(nowFeed(l), schedules, at)
}
func possessive(s string) string { return nowapp.Possessive(s) }
