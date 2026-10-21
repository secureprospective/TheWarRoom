package now

import (
	"slices"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

// Held is Now's private held state: the season's bye weeks, each week's first held scores and
// the live-game poller. It is never bound. mu guards only these values and is never held across
// a Host call, a callback, I/O or another lock; every value is cloned in and cloned out.
type Held struct {
	mu     sync.Mutex
	byes   Feed[[]nflschedule.RawBye]
	firsts FirstScores
	live   LiveGame
}

// LiveGame is the live-game poller's held state for one NFL week: its ESPN adapter and the MFL
// live-scoring backoff.
type LiveGame struct {
	Week        int
	Adapter     livegame.Adapter
	NextMFL     time.Time
	MFLFailures int
}

func (h *Held) PublishByes(f Feed[[]nflschedule.RawBye]) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.byes = cloneByes(f)
}

func (h *Held) SnapshotByes() Feed[[]nflschedule.RawBye] {
	h.mu.Lock()
	defer h.mu.Unlock()
	return cloneByes(h.byes)
}

func (h *Held) PublishFirsts(f FirstScores) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.firsts = cloneFirsts(f)
}

func (h *Held) SnapshotFirsts() FirstScores {
	h.mu.Lock()
	defer h.mu.Unlock()
	return cloneFirsts(h.firsts)
}

func (h *Held) PublishLive(l LiveGame) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.live = cloneLive(l)
}

func (h *Held) SnapshotLive() LiveGame {
	h.mu.Lock()
	defer h.mu.Unlock()
	return cloneLive(h.live)
}

func cloneByes(f Feed[[]nflschedule.RawBye]) Feed[[]nflschedule.RawBye] {
	f.Value = slices.Clone(f.Value)
	return f
}

func cloneFirsts(f FirstScores) FirstScores {
	if f.Weeks != nil {
		weeks := make(map[int]FirstScore, len(f.Weeks))
		for k, v := range f.Weeks {
			v.Week = cloneWeekResult(v.Week)
			weeks[k] = v
		}
		f.Weeks = weeks
	}
	return f
}

func cloneWeekResult(w leaguefeed.WeekResult) leaguefeed.WeekResult {
	w.Games = slices.Clone(w.Games)
	for i := range w.Games {
		w.Games[i].Home = cloneSide(w.Games[i].Home)
		w.Games[i].Away = cloneSide(w.Games[i].Away)
	}
	w.Others = slices.Clone(w.Others)
	for i := range w.Others {
		w.Others[i] = cloneSide(w.Others[i])
	}
	return w
}

func cloneSide(s leaguefeed.Side) leaguefeed.Side {
	s.Players = slices.Clone(s.Players)
	return s
}

// cloneLive copies the adapter by value (its unexported backoff fields too) and deep-copies its
// held games. Fetch and Finals are shared transports, not held data.
func cloneLive(l LiveGame) LiveGame {
	l.Adapter.Board.Games = cloneGames(l.Adapter.Board.Games)
	if l.Adapter.Details != nil {
		details := make(map[string]livegame.Detail, len(l.Adapter.Details))
		for k, d := range l.Adapter.Details {
			d.Game = cloneGame(d.Game)
			details[k] = d
		}
		l.Adapter.Details = details
	}
	return l
}

func cloneGames(games []livegame.Game) []livegame.Game {
	games = slices.Clone(games)
	for i := range games {
		games[i] = cloneGame(games[i])
	}
	return games
}

func cloneGame(g livegame.Game) livegame.Game {
	g.State = clonePtr(g.State)
	if g.Last != nil {
		last := clonePlay(*g.Last)
		g.Last = &last
	}
	g.Box = slices.Clone(g.Box)
	for i := range g.Box {
		g.Box[i].Labels = slices.Clone(g.Box[i].Labels)
		g.Box[i].Rows = slices.Clone(g.Box[i].Rows)
		for j := range g.Box[i].Rows {
			g.Box[i].Rows[j].Stats = slices.Clone(g.Box[i].Rows[j].Stats)
		}
	}
	g.Plays = slices.Clone(g.Plays)
	for i := range g.Plays {
		g.Plays[i] = clonePlay(g.Plays[i])
	}
	return g
}

func clonePlay(p livegame.Play) livegame.Play {
	p.Start, p.End = clonePtr(p.Start), clonePtr(p.End)
	p.Participants = slices.Clone(p.Participants)
	return p
}

func clonePtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
