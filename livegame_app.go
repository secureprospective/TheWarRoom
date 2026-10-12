package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"time"

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
	a.weekMu.Lock()
	defer a.weekMu.Unlock()
	out := make([]livegame.Window, 0, len(a.week.Games))
	for _, g := range a.week.Games {
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
	if a.history != nil {
		a.liveGame.adapter.Finals = heldFinals{store: a.history, endpoints: endpoints}
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
		if _, ok := a.rulebook.FranchiseNames()[mine]; !ok {
			return LiveGameView{}, fmt.Errorf("live game: unknown franchise")
		}
	}
	ctx, cancel := context.WithTimeout(a.ctx, 35*time.Second)
	defer cancel()
	a.liveGameMu.Lock()
	defer a.liveGameMu.Unlock()
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
		a.refreshMu.Lock()
		a.refreshSeasonLineups(ctx)
		a.refreshMu.Unlock()
		a.seasonMu.Lock()
		failed := a.seasonLineups.refreshError != ""
		a.seasonMu.Unlock()
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
	a.seasonMu.Lock()
	held := a.now.schedules[week]
	a.seasonMu.Unlock()
	out := []livegame.Game{}
	seen := map[string]bool{}
	for _, g := range nflGames(held.value, at) {
		id := g.away + "-" + g.home
		if seen[id] {
			continue
		}
		seen[id] = true
		status := "pre"
		if g.started {
			status = "in"
		}
		if g.final {
			status = "post"
		}
		out = append(out, livegame.Game{ID: id, Kickoff: g.kickoff, Home: livegame.Team{Abbreviation: g.home, Score: g.scores[1]}, Away: livegame.Team{Abbreviation: g.away, Score: g.scores[0]}, Status: status, Box: []livegame.Box{}, Plays: []livegame.Play{}})
	}
	slices.SortFunc(out, func(a, b livegame.Game) int { return a.Kickoff.Compare(b.Kickoff) })
	return livegame.Held{Games: out, Source: "Held MFL NFL schedule", At: held.fetchedAt, Failure: held.refreshError}
}
func liveGameReading(a livegame.Adapter, follow string, active bool) LiveGameView {
	source := a.Board.Source
	if source == "" {
		source = "Held NFL schedule; scores unavailable"
	}
	if a.Board.Stale {
		source = "Held " + source
	}
	v := LiveGameView{Games: a.Board.Games, Source: source + " · play feeds and headshots: personal use only", AsOf: reportTime(a.Board.At, time.Local), Failure: a.Board.Failure, Scoring: []livegame.Play{}, Spine: []SpineEntry{}, WeekSpine: []SpineEntry{}, WeekGames: len(a.Board.Games), Active: active}
	if g, ok := a.Followed(follow); ok {
		d := a.Details[g.ID]
		v.DetailFailure = d.Failure
		if !d.At.IsZero() {
			v.DetailAsOf = reportTime(d.At, time.Local)
			g.Box = d.Game.Box
			g.Plays = d.Game.Plays
			if a.Board.Source == "ESPN" && d.At.After(a.Board.At) {
				g = d.Game
			} else {
				g.State, g.Last = d.Game.State, d.Game.Last
			}
			v.Scoring = livegame.ScoringPlays(g.Plays)
		}
		v.Followed = &g
	}
	v.Box = liveBoxReport(v)
	return v
}
func liveBoxReport(v LiveGameView) Report {
	asOf := ""
	if v.DetailAsOf != "" {
		asOf = "As of: " + v.DetailAsOf
	}
	r := Report{ID: "X-01", Title: "NFL box score", Status: "empty", Source: "Source: ESPN game details · play feeds and headshots: personal use only", AsOf: asOf, Note: v.DetailFailure, How: "Passing, rushing, receiving and defense by team, as ESPN supplies them. Yahoo supplies scores and last play only. Held details keep their own time.", Columns: []string{"Team", "Category", "Player", "Statistics"}, Rows: []ReportRow{}, Summary: []ReportRow{}}
	if v.Followed == nil {
		r.Note = "Follow a game from the week strip."
		return r
	}
	if v.Followed.Status == "pre" {
		r.Note = "Not started; no game box score yet."
		return r
	}
	for _, b := range v.Followed.Box {
		for _, p := range b.Rows {
			text := ""
			for i, s := range p.Stats {
				if i > 0 {
					text += " · "
				}
				text += b.Labels[i] + " " + s
			}
			r.Rows = append(r.Rows, ReportRow{Cells: []string{b.Team, b.Category, p.Name, text}})
		}
	}
	if len(r.Rows) > 0 {
		r.Status = "ok"
		if v.DetailFailure != "" {
			r.Status = "stale"
		}
	} else {
		r.Note += " Box score not held."
	}
	return r
}
