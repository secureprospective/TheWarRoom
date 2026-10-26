package now

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

type LiveSource struct {
	Start   func(*livegame.Adapter) error
	Now     func() time.Time
	Windows func() []livegame.Window
	Week    func() int
	Seed    func(time.Time) livegame.Held
	Poll    func(context.Context, *LiveGame, time.Time, bool)
	Spines  func(context.Context, *LiveGameView, livegame.Adapter, string) error
}

// RefreshLiveGame mutates only the caller's detached poller; the caller serializes publication.
func RefreshLiveGame(ctx context.Context, live *LiveGame, open bool, follow, mine string, source LiveSource) (LiveGameView, error) {
	if err := source.Start(&live.Adapter); err != nil {
		return LiveGameView{}, err
	}
	now := source.Now()
	windows := source.Windows()
	week := source.Week()
	if live.Week != week {
		fetch, finals := live.Adapter.Fetch, live.Adapter.Finals
		*live = LiveGame{Week: week, Adapter: livegame.Adapter{Fetch: fetch, Finals: finals}}
	}
	adapter := &live.Adapter
	if len(adapter.Board.Games) == 0 {
		adapter.Board = source.Seed(now)
	}
	if follow != "" {
		if _, ok := adapter.Followed(follow); !ok {
			return LiveGameView{}, fmt.Errorf("game.follow id: not in the held week")
		}
	}
	adapter.Refresh(ctx, now, open, windows, follow)
	adapter.Final(ctx, now, follow)
	active := livegame.Live(now, adapter.EffectiveWindows(windows))
	source.Poll(ctx, live, now, open && active)
	view := LiveReading(LiveSnapshot{Board: adapter.Board, Details: maps.Clone(adapter.Details), Follow: follow, Active: active, Zone: time.Local})
	if err := source.Spines(ctx, &view, *adapter, mine); err != nil {
		view.DetailFailure += " " + report.FailureText(err)
	}
	return view, nil
}

func PollLiveMFL(ctx context.Context, live *LiveGame, now time.Time, active bool, refresh func(context.Context) bool) {
	if active && !now.Before(live.NextMFL) {
		failed := refresh(ctx)
		if failed {
			live.MFLFailures++
		} else {
			live.MFLFailures = 0
		}
		live.NextMFL = now.Add(30 * time.Second * time.Duration(1<<min(live.MFLFailures, 4)))
	}
}

func StartLiveGame(adapter *livegame.Adapter, create func() (livegame.Fetch, livegame.Finals, error)) error {
	if adapter.Fetch != nil {
		return nil
	}
	fetch, finals, err := create()
	if err != nil {
		return err
	}
	adapter.Fetch, adapter.Finals = fetch, finals
	return nil
}

type SpineSource struct {
	Available bool
	Known     func(context.Context) (map[string]bool, error)
	Inspect   func(context.Context, *LiveGameView) error
	Players   func(context.Context, string) (map[string]SpinePlayer, error)
}

func FillLiveSpines(ctx context.Context, v *LiveGameView, adapter livegame.Adapter, mine string, source SpineSource) error {
	held := HeldWeekGames(LiveSnapshot{Details: maps.Clone(adapter.Details)})
	v.WeekHeld = len(held)
	if v.Followed == nil && len(held) == 0 {
		return nil
	}
	if !source.Available {
		return fmt.Errorf("crosswalk directory unavailable")
	}
	known, err := source.Known(ctx)
	if err != nil {
		return fmt.Errorf("live game crosswalk: %w", err)
	}
	if v.Followed != nil {
		if err := source.Inspect(ctx, v); err != nil {
			return err
		}
	}
	players, err := source.Players(ctx, mine)
	if err != nil {
		return err
	}
	if v.Followed != nil {
		v.Spine, v.Unmatched = MatchSpine([]livegame.Game{*v.Followed}, known, players)
	}
	v.WeekSpine, _ = MatchSpine(held, known, players)
	return nil
}

func SpinePlayers(ids map[string]bool, espnIDs map[string]string, lookup normalize.Lookup, photosOff func() bool, photo func(string, string, bool) PhotoView) map[string]SpinePlayer {
	players := map[string]SpinePlayer{}
	for _, id := range slices.Sorted(maps.Keys(ids)) {
		espn := espnIDs[id]
		if espn == "" {
			continue
		}
		facts, ok := lookup.Facts(id)
		if !ok {
			continue
		}
		photo := photo(id, report.ProseName(facts.Name), photosOff())
		players[espn] = SpinePlayer{ID: id, Name: report.ProseName(facts.Name), Team: facts.Team, Mine: ids[id], Photo: photo}
	}
	return players
}

func InspectLiveBox(v *LiveGameView, ids map[string]string) {
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
	AttachReportEntities(&v.Box, slices.Collect(maps.Values(entities)))
}

func CheckScheduleWeek(raw nflschedule.RawWeek, week int) ([]nflschedule.RawWeek, error) {
	if raw.Week != strconv.Itoa(week) {
		return nil, fmt.Errorf("NFL schedule: requested week %d, received %s", week, raw.Week)
	}
	return []nflschedule.RawWeek{raw}, nil
}

func RefreshWeekAllowed(week int, weeks []leaguefeed.WeekResult, current int) error {
	listed := false
	for _, w := range weeks {
		listed = listed || w.Week == week
	}
	if !listed && week != current {
		return fmt.Errorf("now refresh: week %d is not on the league's schedule", week)
	}
	return nil
}

func RaceSnapshot(held season.Held, live Feed[leaguefeed.Lineups], unplayed bool) season.Snapshot {
	overlays := map[int]leaguefeed.WeekResult{}
	for _, w := range held.Weeks.Value {
		if w.Week == live.Value.Week {
			overlays[w.Week] = LiveWeek(live.Value, w)
		}
	}
	return season.Snapshot{Held: held, Live: season.Feed[leaguefeed.Lineups](live), LiveUnplayed: unplayed, LiveWeeks: overlays}
}

type WeekFeedSource struct {
	Pause    func(context.Context, time.Duration) error
	Since    func(time.Time) time.Duration
	Schedule func(context.Context)
	Byes     func(context.Context)
}

func FeedFresh(at time.Time, failure string, since func(time.Time) time.Duration) bool {
	return failure == "" && since(at) < 5*time.Minute
}

func RefreshWeekFeeds(ctx context.Context, force bool, scheduleAt time.Time, scheduleError string, byesAt time.Time, byesError string, source WeekFeedSource) {
	if force || !FeedFresh(scheduleAt, scheduleError, source.Since) {
		source.Schedule(ctx)
		if err := source.Pause(ctx, time.Second); err != nil {
			return
		}
	}
	if !force && byesError == "" && source.Since(byesAt) < 24*time.Hour {
		return
	}
	source.Byes(ctx)
}

func SeedClockSchedules(ctx context.Context, current func(context.Context), lastRegular func() int,
	read func(context.Context, int) ([]nflschedule.RawWeek, time.Time, bool, error),
	publish func(int, []nflschedule.RawWeek, time.Time), logFailure func(string)) {
	current(ctx)
	for _, rule := range leagueclock.Rules(lastRegular()) {
		if rule.Week == 0 {
			continue
		}
		rows, at, found, err := read(ctx, rule.Week)
		if err != nil {
			logFailure(err.Error())
			continue
		}
		if !found {
			continue
		}
		publish(rule.Week, rows, at)
	}
}

func RefreshNow(week int, force bool, race func(bool) error, refresh func(int, bool) error) error {
	if err := race(force); err != nil {
		return fmt.Errorf("now refresh: %w", err)
	}
	return refresh(week, force)
}

func SelectedRefreshWeek(week int, current func() int) int {
	if week == 0 {
		return current()
	}
	return week
}

func RefreshSelectedWeek(ctx context.Context, week int, weeks []leaguefeed.WeekResult, current int, force bool, refresh func(context.Context, int, bool)) error {
	if err := RefreshWeekAllowed(week, weeks, current); err != nil {
		return err
	}
	if week < 1 {
		return nil
	}
	refresh(ctx, week, force)
	return nil
}
