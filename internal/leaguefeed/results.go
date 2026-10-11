package leaguefeed

import "github.com/secureprospective/TheWarRoom/internal/playerid"

// Points are hundredths of a fantasy point, so totals and tiebreaks compare exactly.
type Points int64

// WeekResult is one week as MFL scored it: its head-to-head games, and the franchises that
// scored without a game (a playoff bye, or a team out of the playoffs).
type WeekResult struct {
	Week   int
	Games  []Game
	Others []Side
}

// Game is one head-to-head game; Regular is false for a playoff game.
type Game struct {
	Regular    bool
	Home, Away Side
}

// Result is MFL's verdict for a side: "W", "L" or "T".
type Result string

// Side is one franchise's week. Until the week is scored, Scored is false and Points, Best and
// Result are empty; a set lineup is already listed in Players.
type Side struct {
	Franchise string
	Scored    bool
	Points    Points
	Best      Points // the best lineup MFL found from the same roster
	Result    Result
	Players   []WeekPlayer
}

// WeekPlayer is one rostered player's week: started or benched, his points, and MFL's verdict
// that he belonged in the best lineup.
type WeekPlayer struct {
	ID          playerid.PlayerID
	Starter     bool
	ShouldStart bool
	Points      Points
}

// Bracket is one playoff bracket the league defines and its games so far.
type Bracket struct {
	ID        int
	Name      string
	Title     string // what the winner is called, e.g. "AFC Conference Champion"
	StartWeek int
	Teams     int
	Games     []BracketGame
}

// BracketGame is one bracket game; a side is known by its seed, by the game it won, or both.
type BracketGame struct {
	ID         int
	Week       int
	Home, Away BracketSide
}

// BracketSide is a franchise in a bracket game. Franchise is empty until MFL places one; Seed is
// 0 for a side that advanced from game WinnerOf, of bracket FromBracket when that is another one.
type BracketSide struct {
	Franchise   string
	Seed        int
	WinnerOf    int
	FromBracket int
	Scored      bool
	Points      Points
}
