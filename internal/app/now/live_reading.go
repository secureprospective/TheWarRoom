package now

import (
	"maps"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

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
	Box           report.Report   `json:"box"`
	Active        bool            `json:"active"`
}

func LiveReading(in LiveSnapshot) LiveGameView {
	a := livegame.Adapter{Board: in.Board, Details: in.Details}
	follow, active := in.Follow, in.Active
	source := a.Board.Source
	if source == "" {
		source = "Held NFL schedule; scores unavailable"
	}
	if a.Board.Stale {
		source = "Held " + source
	}
	v := LiveGameView{Games: a.Board.Games, Source: source + " · play feeds and headshots: personal use only", AsOf: reportTime(a.Board.At, in.Zone), Failure: a.Board.Failure, Scoring: []livegame.Play{}, Spine: []SpineEntry{}, WeekSpine: []SpineEntry{}, WeekGames: len(a.Board.Games), Active: active}
	if g, ok := a.Followed(follow); ok {
		d := a.Details[g.ID]
		v.DetailFailure = d.Failure
		if !d.At.IsZero() {
			v.DetailAsOf = reportTime(d.At, in.Zone)
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
func liveBoxReport(v LiveGameView) report.Report {
	asOf := ""
	if v.DetailAsOf != "" {
		asOf = "As of: " + v.DetailAsOf
	}
	r := report.Report{ID: "X-01", Title: "NFL box score", Status: "empty", Source: "Source: ESPN game details · play feeds and headshots: personal use only", AsOf: asOf, Note: v.DetailFailure, How: "Passing, rushing, receiving and defense by team, as ESPN supplies them. Yahoo supplies scores and last play only. Held details keep their own time.", Columns: []string{"Team", "Category", "Player", "Statistics"}, Rows: []report.Row{}, Summary: []report.Row{}}
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
			r.Rows = append(r.Rows, report.Row{Cells: []string{b.Team, b.Category, p.Name, text}})
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

type SpinePlayer struct {
	ID    string    `json:"id"`
	Name  string    `json:"name"`
	Team  string    `json:"team"`
	Mine  bool      `json:"mine"`
	Photo PhotoView `json:"photo"`
}
type SpineEntry struct {
	Player SpinePlayer   `json:"player"`
	Play   livegame.Play `json:"play"`
	Game   string        `json:"game"` // "AWAY at HOME", so a week-wide entry names its game
}

func MatchupStarterIDs(l leaguefeed.Lineups, mine string) map[string]bool {
	opponent := ""
	for _, m := range l.Matchups {
		if m.Home == mine {
			opponent = m.Away
		}
		if m.Away == mine {
			opponent = m.Home
		}
	}
	out := map[string]bool{}
	if opponent == "" || mine == "" {
		return out
	}
	for _, f := range l.Franchises {
		if f.Franchise != mine && f.Franchise != opponent {
			continue
		}
		for _, id := range f.Starters {
			out[id.String()] = f.Franchise == mine
		}
	}
	return out
}

func HeldWeekGames(adapter LiveSnapshot) []livegame.Game {
	out := []livegame.Game{}
	seen := map[string]bool{}
	for _, key := range slices.Sorted(maps.Keys(adapter.Details)) {
		d := adapter.Details[key]
		if d.At.IsZero() || len(d.Game.Plays) == 0 || seen[d.Game.ID] {
			continue
		}
		seen[d.Game.ID] = true
		out = append(out, d.Game)
	}
	return out
}

// MatchSpine joins plays to players by ESPN id only, newest first: by the play's wall clock when
// both plays carry one, otherwise by the game's own play order.
func MatchSpine(games []livegame.Game, known map[string]bool, players map[string]SpinePlayer) ([]SpineEntry, int) {
	out := []SpineEntry{}
	unmatched := 0
	for _, g := range games {
		label := g.Away.Abbreviation + " at " + g.Home.Abbreviation
		for _, p := range g.Plays {
			for _, id := range p.Participants {
				if !known[id] {
					unmatched++
					continue
				}
				if player, ok := players[id]; ok {
					out = append(out, SpineEntry{Player: player, Play: p, Game: label})
				}
			}
		}
	}
	slices.SortStableFunc(out, func(a, b SpineEntry) int {
		if !a.Play.At.IsZero() && !b.Play.At.IsZero() && !a.Play.At.Equal(b.Play.At) {
			return b.Play.At.Compare(a.Play.At)
		}
		if a.Game != b.Game {
			return 0
		}
		return b.Play.Sequence - a.Play.Sequence
	})
	return out, unmatched
}

type PhotoView struct {
	Src string `json:"src"`
	Alt string `json:"alt"`
}

func SeedLiveGames(held Feed[[]nflschedule.RawWeek], at time.Time) livegame.Held {
	out := []livegame.Game{}
	seen := map[string]bool{}
	for _, g := range NFLGames(held.Value, at) {
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
	return livegame.Held{Games: out, Source: "Held MFL NFL schedule", At: held.FetchedAt, Failure: held.RefreshError}
}

func HeldMatchupStarters(live leaguefeed.Lineups, weeks []leaguefeed.WeekResult, week int, mine string) map[string]bool {
	if live.Week == week {
		if ids := MatchupStarterIDs(live, mine); len(ids) > 0 {
			return ids
		}
	}
	out := map[string]bool{}
	for _, w := range weeks {
		if w.Week != week {
			continue
		}
		if g, ok := findGame(w, mine); ok {
			for _, side := range []leaguefeed.Side{g.Home, g.Away} {
				for _, p := range side.Players {
					if p.Starter {
						out[p.ID.String()] = side.Franchise == mine
					}
				}
			}
		}
	}
	return out
}
