package livegame

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
)

// Canonical returns a team code in ESPN's form, whichever source supplied it. The week strip is
// seeded from MFL's schedule and the live board comes from ESPN, so one game must be the same
// game under either source's codes.
func Canonical(team string) string {
	switch team {
	case "GBP":
		return "GB"
	case "KCC":
		return "KC"
	case "LVR":
		return "LV"
	case "NEP":
		return "NE"
	case "NOS":
		return "NO"
	case "SFO":
		return "SF"
	case "TBB":
		return "TB"
	case "JAC":
		return "JAX"
	case "WAS":
		return "WSH"
	}
	return team
}

// sameGame reports whether two games are the same fixture: the same two teams (in either source's
// codes) kicking off within twelve hours of each other.
func sameGame(a, b Game) bool {
	if Canonical(a.Home.Abbreviation) != Canonical(b.Home.Abbreviation) ||
		Canonical(a.Away.Abbreviation) != Canonical(b.Away.Abbreviation) {
		return false
	}
	gap := a.Kickoff.Sub(b.Kickoff)
	return a.Kickoff.IsZero() || b.Kickoff.IsZero() || (gap < 12*time.Hour && gap > -12*time.Hour)
}

// Finals is where a final game's ESPN bodies are kept, so a final is fetched once and read
// without the network after a restart. Bodies calls each with every kept body of kind
// ("summary" or "plays") for event id (any id when id is empty), newest first, until each
// returns false.
type Finals interface {
	Keep(ctx context.Context, kind, id string, body []byte, at time.Time) error
	Bodies(ctx context.Context, kind, id string, each func(id string, body []byte, at time.Time) bool) error
}

// ErrNotListed means ESPN's scoreboard does not list the followed game, so its event id is unknown.
var ErrNotListed = errors.New("ESPN's scoreboard does not list this game")

// Final reads a followed game the source marks final, once per follow: from the kept bodies
// when they are held, otherwise by one fetch of its summary and plays, which are then kept.
// A final never changes, so it is never fetched again; a failure is shown and is retried only
// when the game is followed again, never on a timer. Polling rules for live games are untouched.
func (a *Adapter) Final(ctx context.Context, at time.Time, follow string) {
	if follow != a.finalTried {
		a.finalTried = ""
	}
	g, ok := a.Followed(follow)
	if !ok || g.Status != "post" || a.Details[g.ID].Final || a.finalTried == follow {
		return
	}
	a.finalTried = follow
	if a.Details == nil {
		a.Details = map[string]Detail{}
	}
	if d, ok := a.keptFinal(ctx, g); ok {
		a.holdFinal(g, d)
		return
	}
	d, err := a.fetchFinal(ctx, at, g)
	if err != nil {
		previous := a.Details[g.ID]
		previous.Failure = "Final game details: " + err.Error() + ". Follow the game again to retry."
		a.Details[g.ID] = previous
		return
	}
	a.holdFinal(g, d)
}

// holdFinal keeps a final's details under the strip's id and ESPN's event id, so the game reads
// the same whether the strip is seeded from MFL's schedule or comes from ESPN's live board.
func (a *Adapter) holdFinal(g Game, d Detail) {
	a.Details[g.ID] = d
	a.Details[d.Game.ID] = d
}

func (a *Adapter) keptFinal(ctx context.Context, g Game) (Detail, bool) {
	if a.Finals == nil {
		return Detail{}, false
	}
	var found Detail
	_ = a.Finals.Bodies(ctx, "summary", eventID(g), func(id string, body []byte, at time.Time) bool {
		s, err := ParseSummary(body)
		if err != nil || s.ID != id || s.Status != "post" || !sameGame(s, g) {
			return true
		}
		found = Detail{Game: s, At: at, Final: true}
		return false
	})
	if !found.Final {
		return Detail{}, false
	}
	held := false
	_ = a.Finals.Bodies(ctx, "plays", found.Game.ID, func(_ string, body []byte, _ time.Time) bool {
		plays, err := ParseCore(body)
		if err != nil {
			return true
		}
		withPlays(&found.Game, plays)
		held = true
		return false
	})
	return found, held
}

func eventID(g Game) string {
	if digits(g.ID) {
		return g.ID
	}
	return ""
}

func (a *Adapter) fetchFinal(ctx context.Context, at time.Time, g Game) (Detail, error) {
	if a.Fetch == nil {
		return Detail{}, fmt.Errorf("transport unavailable")
	}
	id := eventID(g)
	if id == "" {
		games, err := a.fetchBoard(ctx, "espn", ParseScoreboard)
		if err != nil {
			return Detail{}, err
		}
		i := slices.IndexFunc(games, func(e Game) bool { return sameGame(e, g) })
		if i < 0 {
			return Detail{}, ErrNotListed
		}
		id = games[i].ID
	}
	summary, err := a.Fetch(ctx, "summary", id)
	if err != nil {
		return Detail{}, fmt.Errorf("ESPN summary: %w", err)
	}
	s, err := ParseSummary(summary)
	if err != nil {
		return Detail{}, fmt.Errorf("ESPN summary: %w", err)
	}
	if s.ID != id {
		return Detail{}, fmt.Errorf("ESPN summary.header.id: requested %s received %s", id, s.ID)
	}
	if s.Status != "post" {
		return Detail{}, fmt.Errorf("ESPN does not mark this game final yet")
	}
	body, err := a.Fetch(ctx, "plays", id)
	if err != nil {
		return Detail{}, fmt.Errorf("ESPN plays: %w", err)
	}
	plays, err := ParseCore(body)
	if err != nil {
		return Detail{}, fmt.Errorf("ESPN plays: %w", err)
	}
	withPlays(&s, plays)
	d := Detail{Game: s, At: at, Final: true}
	if a.Finals != nil {
		if err := a.Finals.Keep(ctx, "summary", id, summary, at); err != nil {
			d.Failure = "Held for this session only: " + err.Error()
		} else if err := a.Finals.Keep(ctx, "plays", id, body, at); err != nil {
			d.Failure = "Held for this session only: " + err.Error()
		}
	}
	return d, nil
}

// withPlays replaces a game's plays with the core feed's, which carry participants.
func withPlays(g *Game, plays []Play) {
	g.Plays = plays
	if len(plays) > 0 {
		p := plays[len(plays)-1]
		g.Last = &p
		g.State = p.End
		g.Clock = p.Clock
		g.Period = p.Period
	}
}

// ScoringPlays returns the plays after which either score changed, in game order.
func ScoringPlays(plays []Play) []Play {
	out := []Play{}
	home, away := 0, 0
	for _, p := range plays {
		if p.HomeScore != home || p.AwayScore != away {
			out = append(out, p)
		}
		home, away = p.HomeScore, p.AwayScore
	}
	return out
}
