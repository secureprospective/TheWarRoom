package history

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// HeldLines is every team-game's offensive line components from one pinned statistic revision.
// A game whose advanced rushing is not yet published lacks yards before contact, so the unit
// scores as not recorded rather than zero.
type HeldLines struct {
	SHA   string
	At    time.Time
	Games map[source.LineGame]fantasyscore.UnitLine
}

// lineFile builds a revision's line games when it carries advanced rushing. The play-by-play body
// must then also carry the line columns; a revision without advanced rushing holds no line facts,
// exactly as before line facts existed.
func lineFile(raw RawFootballStats, isRB func(pfrID string) bool) (string, map[source.LineGame]fantasyscore.UnitLine, error) {
	if raw.Rushing == "" {
		return "", nil, nil
	}
	plays, err := source.ReadLinePlays(strings.NewReader(raw.Plays))
	if err != nil {
		return "", nil, fmt.Errorf("read pinned line plays: %w", err)
	}
	rushing, err := source.ReadLineRushing(strings.NewReader(raw.Rushing))
	if err != nil {
		return "", nil, fmt.Errorf("read pinned advanced rushing: %w", err)
	}
	games, err := source.LineGames(plays, rushing, isRB)
	if err != nil {
		return "", nil, fmt.Errorf("build pinned line games: %w", err)
	}
	sha := footballDigest([]byte(fmt.Sprintf("%d:%s%s", len(raw.Plays), raw.Plays, raw.Rushing)))
	return sha, games, nil
}

// leagueRB answers whether the league lists the player with this PFR id at RB: the RB designation
// yards before contact counts (DEFAULT-32 spec §4.6). A PFR id no player carries is not an RB.
func (r RawFootballEvidence) leagueRB() func(pfrID string) bool {
	rbs := make(map[string]bool)
	for _, player := range r.Players {
		if player.PFR != "" && player.Facts.Position == leaguerules.RB {
			rbs[player.PFR] = true
		}
	}
	return func(pfrID string) bool { return rbs[pfrID] }
}

func (r RawFootballEvidence) validatePFR() error {
	seen := make(map[string]bool)
	for _, player := range r.Players {
		if player.PFR == "" {
			continue
		}
		if len(player.PFR) > 200 || strings.TrimSpace(player.PFR) != player.PFR || seen[player.PFR] {
			return fmt.Errorf("a PFR identity must be trimmed, at most 200 characters and name one player")
		}
		seen[player.PFR] = true
	}
	return nil
}

// LineGames serves the newest pinned revision that holds line facts. ok is false when none does.
func (p *PinnedFootball) LineGames(ctx context.Context, season int) (HeldLines, bool, error) {
	if err := ctx.Err(); err != nil {
		return HeldLines{}, false, fmt.Errorf("read pinned line games: %w", err)
	}
	if season != p.raw.Season {
		return HeldLines{}, false, fmt.Errorf("pinned statistics do not hold season %d", season)
	}
	for _, f := range p.stats {
		if f.lines == nil {
			continue
		}
		games := make(map[source.LineGame]fantasyscore.UnitLine, len(f.lines))
		for game, line := range f.lines {
			games[game] = fantasyscore.UnitLine{Played: line.Played, Quantities: maps.Clone(line.Quantities)}
		}
		return HeldLines{SHA: f.lineSHA, At: f.at, Games: games}, true, nil
	}
	return HeldLines{}, false, nil
}
