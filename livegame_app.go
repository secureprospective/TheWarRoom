package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"time"

	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
)

//go:embed livegame_sources.json
var liveGameSources []byte

type liveGameHeld struct {
	week        int
	adapter     livegame.Adapter
	nextMFL     time.Time
	mflFailures int
}
type LiveGameView struct {
	Games         []livegame.Game `json:"games"`
	Followed      *livegame.Game  `json:"followed"`
	Source        string          `json:"source"`
	AsOf          string          `json:"asOf"`
	Failure       string          `json:"failure"`
	DetailAsOf    string          `json:"detailAsOf"`
	DetailFailure string          `json:"detailFailure"`
	Scoring       []livegame.Play `json:"scoring"`
	Spine         []SpineEntry    `json:"spine"`
	Unmatched     int             `json:"unmatched"`
	WeekSpine     []SpineEntry    `json:"weekSpine"`
	WeekHeld      int             `json:"weekHeld"`
	WeekGames     int             `json:"weekGames"`
	Box           Report          `json:"box"`
	Active        bool            `json:"active"`
}

func (a *App) liveGameWindows() []livegame.Window {
	a.core.WeekMu.Lock()
	defer a.core.WeekMu.Unlock()
	out := make([]livegame.Window, 0, len(a.core.Week.Games))
	for _, g := range a.core.Week.Games {
		out = append(out, livegame.Window{Teams: g.Teams, Kickoff: g.Kickoff, Final: g.Final})
	}
	return out
}
func (a *App) startLiveGame() error {
	if a.liveGame.adapter.Fetch != nil {
		return nil
	}
	var endpoints livegame.Endpoints
	if err := json.Unmarshal(liveGameSources, &endpoints); err != nil {
		return fmt.Errorf("live game sources: %w", err)
	}
	if err := endpoints.Validate(); err != nil {
		return fmt.Errorf("live game sources: %w", err)
	}
	a.liveGame.adapter.Fetch = livegame.HTTPFetch(&http.Client{Timeout: 10 * time.Second}, endpoints)
	if a.core.History != nil {
		a.liveGame.adapter.Finals = heldFinals{store: a.core.History, endpoints: endpoints}
	}
	return nil
}

// TargetLiveGame is a demand-driven poll: only the mounted Now view asks to refresh.
func (a *App) TargetLiveGame(open bool, follow, mine string) (LiveGameView, error) {
	if err := a.ready(); err != nil {
		return LiveGameView{}, fmt.Errorf("live game: startup: %w", err)
	}
	if !regexp.MustCompile(`^(?:[0-9]{1,20}|[A-Z]{2,3}-[A-Z]{2,3}|nfl\.g\.[0-9]+)?$`).MatchString(follow) {
		return LiveGameView{}, fmt.Errorf("game.follow id: invalid")
	}
	if mine != "" {
		if _, ok := a.core.Rulebook.FranchiseNames()[mine]; !ok {
			return LiveGameView{}, fmt.Errorf("live game: unknown franchise")
		}
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, 35*time.Second)
	defer cancel()
	a.core.LiveGameMu.Lock()
	defer a.core.LiveGameMu.Unlock()
	if err := a.startLiveGame(); err != nil {
		return LiveGameView{}, err
	}
	now := time.Now()
	windows := a.liveGameWindows()
	adapter := &a.liveGame.adapter
	week := a.marketWeek()
	if a.liveGame.week != week {
		fetch, finals := adapter.Fetch, adapter.Finals
		a.liveGame = liveGameHeld{week: week, adapter: livegame.Adapter{Fetch: fetch, Finals: finals}}
		adapter = &a.liveGame.adapter
	}
	if len(adapter.Board.Games) == 0 {
		adapter.Board = a.seedLiveGames(now)
	}
	if follow != "" {
		if _, ok := adapter.Followed(follow); !ok {
			return LiveGameView{}, fmt.Errorf("game.follow id: not in the held week")
		}
	}
	adapter.Refresh(ctx, now, open, windows, follow)
	adapter.Final(ctx, now, follow)
	active := livegame.Live(now, adapter.EffectiveWindows(windows))
	a.pollLiveMFL(ctx, now, open && active)
	view := liveGameReading(*adapter, follow, active)
	if err := a.liveGameSpines(ctx, &view, *adapter, mine); err != nil {
		view.DetailFailure += " " + err.Error()
	}
	return view, nil
}
func (a *App) pollLiveMFL(ctx context.Context, now time.Time, active bool) {
	if active && !now.Before(a.liveGame.nextMFL) {
		a.core.RefreshMu.Lock()
		a.refreshSeasonLineups(ctx)
		a.core.RefreshMu.Unlock()
		a.core.SeasonMu.Lock()
		failed := a.core.SeasonLineups.RefreshError != ""
		a.core.SeasonMu.Unlock()
		if failed {
			a.liveGame.mflFailures++
		} else {
			a.liveGame.mflFailures = 0
		}
		a.liveGame.nextMFL = now.Add(30 * time.Second * time.Duration(1<<min(a.liveGame.mflFailures, 4)))
	}
}

func (a *App) seedLiveGames(at time.Time) livegame.Held {
	week := a.marketWeek()
	a.core.SeasonMu.Lock()
	held := a.now.schedules[week]
	a.core.SeasonMu.Unlock()
	return nowapp.SeedLiveGames(nowFeed(held), at)
}
func liveGameReading(a livegame.Adapter, follow string, active bool) LiveGameView {
	return wireLive(nowapp.LiveReading(nowapp.LiveSnapshot{Board: a.Board, Details: maps.Clone(a.Details), Follow: follow, Active: active, Zone: time.Local}))
}
