package livegame

import (
	"context"

	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/offline"
)

type Fetch func(context.Context, string, string) ([]byte, error)
type Window struct {
	Teams   [2]string
	Kickoff time.Time
	Final   bool
}
type Held struct {
	Stale   bool      `json:"stale"`
	Games   []Game    `json:"games"`
	Source  string    `json:"source"`
	At      time.Time `json:"at"`
	Failure string    `json:"failure"`
}
type Detail struct {
	Game    Game      `json:"game"`
	At      time.Time `json:"at"`
	Failure string    `json:"failure"`
	Final   bool      `json:"final"` // read once from a final game; never fetched again
}
type Adapter struct {
	Fetch          Fetch
	Finals         Finals
	Board          Held
	Details        map[string]Detail
	NextBoard      time.Time
	NextDetail     time.Time
	boardFailures  int
	detailFailures int
	followed       string
	finalTried     string // the follow a final's one-shot read was last tried on
}

// MaxGame bounds a window when no source has said the game is final: no NFL game runs six hours past kickoff,
// so a stale schedule and two failed live sources can never keep polling a game that ended yesterday.
const MaxGame = 6 * time.Hour

func (w Window) open(at time.Time) bool {
	return !w.Final && !w.Kickoff.IsZero() && !at.Before(w.Kickoff) && at.Before(w.Kickoff.Add(MaxGame))
}
func Live(at time.Time, windows []Window) bool {
	return slices.ContainsFunc(windows, func(w Window) bool { return w.open(at) })
}
func GameLive(at time.Time, windows []Window, g Game) bool {
	return slices.ContainsFunc(windows, func(w Window) bool {
		return w.open(at) && w.plays(g)
	}) && g.Status != "post"
}

// plays reports whether g is this window's game, whichever source's team codes each uses.
func (w Window) plays(g Game) bool {
	teams := []string{Canonical(w.Teams[0]), Canonical(w.Teams[1])}
	return slices.Contains(teams, Canonical(g.Home.Abbreviation)) && slices.Contains(teams, Canonical(g.Away.Abbreviation))
}
func delay(base time.Duration, failures int) time.Duration {
	return base * time.Duration(1<<min(failures, 4))
}
func (a *Adapter) EffectiveWindows(windows []Window) []Window {
	out := slices.Clone(windows)
	if a.Board.Source != "ESPN" && a.Board.Source != "Yahoo failover" {
		return out
	}
	for i, w := range out {
		for _, g := range a.Board.Games {
			if !w.Kickoff.Equal(g.Kickoff) || !w.plays(g) {
				continue
			}
			d := a.Details[g.ID]
			if g.Status == "post" || (!d.At.IsZero() && d.Game.Status == "post") {
				out[i].Final = true
			}
		}
	}
	return out
}
func (a *Adapter) Refresh(ctx context.Context, at time.Time, open bool, windows []Window, follow string) {
	windows = a.EffectiveWindows(windows)
	if !open || !Live(at, windows) {
		return
	}
	if !at.Before(a.NextBoard) {
		a.refreshBoard(ctx, at)
		a.NextBoard = at.Add(delay(30*time.Second, a.boardFailures))
	}
	windows = a.EffectiveWindows(windows)
	g, ok := a.Followed(follow)
	if !ok || !GameLive(at, windows, g) {
		return
	}
	if follow != a.followed {
		a.NextDetail = time.Time{}
		a.detailFailures = 0
		a.followed = follow
	}
	if at.Before(a.NextDetail) {
		return
	}
	a.refreshDetail(ctx, at, g)
	a.NextDetail = at.Add(delay(15*time.Second, a.detailFailures))
}
func (a *Adapter) refreshBoard(ctx context.Context, at time.Time) {
	primary, err := a.fetchBoard(ctx, "espn", ParseScoreboard)
	if err == nil {
		a.Board = Held{Games: primary, Source: "ESPN", At: at}
		a.boardFailures = 0
		return
	}
	backup, second := a.fetchBoard(ctx, "yahoo", ParseYahoo)
	a.boardFailures++
	if second == nil {
		// Preserve the ESPN selection key only on an exact team pair and kickoff match.
		for i := range backup {
			for _, old := range a.Board.Games {
				if old.Home.Abbreviation == backup[i].Home.Abbreviation && old.Away.Abbreviation == backup[i].Away.Abbreviation && old.Kickoff.Equal(backup[i].Kickoff) {
					backup[i].ID = old.ID
				}
			}
		}
		a.Board = Held{Games: backup, Source: "Yahoo failover", At: at, Failure: offline.Text(err)}
		return
	}
	a.Board.Failure = offline.Text(err) + "; " + offline.Text(second)
	a.Board.Stale = true
}
func (a *Adapter) fetchBoard(ctx context.Context, source string, parse func([]byte) ([]Game, error)) ([]Game, error) {
	if a.Fetch == nil {
		return nil, fmt.Errorf("%s: transport unavailable", source)
	}
	body, err := a.Fetch(ctx, source, "")
	if err != nil {
		return nil, fmt.Errorf("%s scoreboard: %w", source, err)
	}
	games, err := parse(body)
	if err != nil {
		return nil, fmt.Errorf("%s scoreboard: %w", source, err)
	}
	return games, nil
}
func (a *Adapter) Followed(id string) (Game, bool) {
	for _, g := range a.Board.Games {
		if g.ID == id || Canonical(g.Away.Abbreviation)+"-"+Canonical(g.Home.Abbreviation) == canonicalPair(id) {
			return g, true
		}
	}
	return Game{}, false
}

// canonicalPair puts an "AWAY-HOME" follow id into ESPN's team codes.
func canonicalPair(id string) string {
	away, home, ok := strings.Cut(id, "-")
	if !ok {
		return id
	}
	return Canonical(away) + "-" + Canonical(home)
}
func (a *Adapter) refreshDetail(ctx context.Context, at time.Time, g Game) {
	if a.Details == nil {
		a.Details = map[string]Detail{}
	}
	previous := a.Details[g.ID]
	if !digits(g.ID) || a.Board.Source != "ESPN" {
		previous.Failure = "Yahoo failover supplies scores and last play only; box score and participant plays remain held."
		a.Details[g.ID] = previous
		return
	}
	game, err := a.fetchDetail(ctx, g.ID)
	if err != nil {
		previous.Failure = offline.Text(err)
		a.Details[g.ID] = previous
		a.detailFailures++
		return
	}
	a.Details[g.ID] = Detail{Game: game, At: at}
	a.detailFailures = 0
}
func (a *Adapter) fetchDetail(ctx context.Context, id string) (Game, error) {
	body, err := a.Fetch(ctx, "summary", id)
	if err != nil {
		return Game{}, fmt.Errorf("ESPN summary: %w", err)
	}
	g, err := ParseSummary(body)
	if err != nil {
		return Game{}, fmt.Errorf("ESPN summary: %w", err)
	}
	if g.ID != id {
		return Game{}, fmt.Errorf("ESPN summary.header.id: requested %s received %s", id, g.ID)
	}
	if g.Status == "pre" {
		return g, nil
	}
	body, err = a.Fetch(ctx, "plays", id)
	if err != nil {
		return Game{}, fmt.Errorf("ESPN plays: %w", err)
	}
	plays, err := ParseCore(body)
	if err != nil {
		return Game{}, fmt.Errorf("ESPN plays: %w", err)
	}
	withPlays(&g, plays)
	return g, nil
}
