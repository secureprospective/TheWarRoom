package leagueservice

import (
	"fmt"
	"slices"
	"sync/atomic"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
)

// liveGamesStale matches the score overlay's staleness: a board older than this is labelled held.
const liveGamesStale = 10 * time.Minute

// LiveGames holds the server's latest validated NFL scoreboard. It is public NFL data shared by every
// league on the server; it never carries a league's own facts.
type LiveGames struct {
	held    atomic.Pointer[livegame.Held]
	details atomic.Pointer[map[string]NFLGame]
	now     func() time.Time
}

// NewLiveGames starts an empty holder; readers see no games until the collector sets a board.
func NewLiveGames() *LiveGames { return &LiveGames{now: time.Now} }

// Set replaces the held board with a detached copy, so later edits by the caller never leak in.
func (g *LiveGames) Set(games []livegame.Game, at time.Time, source string) {
	if g == nil {
		return
	}
	board := livegame.Held{Games: slices.Clone(games), At: at.UTC(), Source: source}
	g.held.Store(&board)
}

// Board answers the held board, marked stale once it is older than liveGamesStale.
func (g *LiveGames) Board() livegame.Held {
	if g == nil {
		return livegame.Held{Games: []livegame.Game{}}
	}
	held := g.held.Load()
	if held == nil {
		return livegame.Held{Games: []livegame.Game{}}
	}
	board := *held
	board.Games = slices.Clone(board.Games)
	board.Stale = g.now().Sub(board.At) > liveGamesStale
	return board
}

// NFLGames is the hosted NFL scoreboard reading: the current week's games, provisional and public.
type NFLGames struct {
	League string        `json:"league"`
	Board  livegame.Held `json:"board"`
}

// WithLiveGames gives a hosted service the server's shared scoreboard; nil keeps the reading empty.
func (a *Service) WithLiveGames(games *LiveGames) *Service {
	a.games = games
	return a
}

// TargetNFLGames reads the held NFL scoreboard. Any league member may read it: it is admitted exactly
// like the league directory, and it carries only public NFL data.
func (a *Service) TargetNFLGames(league string) (NFLGames, error) {
	if league == "" {
		return NFLGames{}, fmt.Errorf("nfl games: league is required")
	}
	if err := a.admitRead(league, "native.directory.read", ""); err != nil {
		return NFLGames{}, err
	}
	board := a.games.Board()
	if board.Games == nil {
		board.Games = []livegame.Game{}
	}
	return NFLGames{League: league, Board: board}, nil
}
