package main

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/lineup"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type PulseReading struct {
	Week       int                 `json:"week"`
	Matchups   []PulseMatchup      `json:"matchups"`
	Provenance snapshot.Provenance `json:"provenance"`
}

type PulseMatchup struct {
	Home PulseSide `json:"home"`
	Away PulseSide `json:"away"`
}

type PulseSide struct {
	FranchiseID      string        `json:"franchiseId"`
	Name             string        `json:"name"`
	Score            float64       `json:"score"`
	SecondsRemaining int           `json:"secondsRemaining"`
	Playing          int           `json:"playing"`
	YetToPlay        int           `json:"yetToPlay"`
	Starters         []PulsePlayer `json:"starters"`
}

type PulsePlayer struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Position         string  `json:"position"`
	Team             string  `json:"team"`
	Score            float64 `json:"score"`
	SecondsRemaining int     `json:"secondsRemaining"`
}

// TargetPulseNow reads held feeds and directory only; it never waits for refreshMu.
func (a *App) TargetPulseNow() (PulseReading, error) {
	season, err := a.TargetSeason()
	if err != nil {
		return PulseReading{}, fmt.Errorf("target pulse now: %w", err)
	}
	r := PulseReading{
		Week: season.Lineups.Value.Week, Matchups: []PulseMatchup{}, Provenance: season.Lineups.Provenance,
	}
	if envelope.TradeFeedReason(r.Provenance) != "" {
		return r, nil
	}
	a.targetSnapshotMu.Lock()
	snap, loaded := a.targetSnapshot, a.hasTargetSnapshot
	a.targetSnapshotMu.Unlock()
	if !loaded {
		return PulseReading{}, fmt.Errorf("target pulse now: no snapshot loaded")
	}
	rules, err := lineup.ParseRules(a.rulebook.ActiveConfig().Starters)
	if err != nil {
		return PulseReading{}, fmt.Errorf("target pulse now: starter order: %w", err)
	}
	sides := make(map[string]PulseSide, len(season.Lineups.Value.Franchises))
	for _, saved := range season.Lineups.Value.Franchises {
		sides[saved.Franchise] = pulseSide(saved, snap, season, rules)
	}
	for _, m := range season.Lineups.Value.Matchups {
		r.Matchups = append(r.Matchups, PulseMatchup{Home: heldSide(sides, m.Home), Away: heldSide(sides, m.Away)})
	}
	return r, nil
}

// heldSide keeps a matchup franchise the feed listed without a lineup row as an empty side,
// never a null starters array.
func heldSide(sides map[string]PulseSide, id string) PulseSide {
	if side, ok := sides[id]; ok {
		return side
	}
	return PulseSide{FranchiseID: id, Starters: []PulsePlayer{}}
}

func pulseSide(
	saved leaguefeed.Lineup, snap snapshot.Snapshot, season SeasonReading, rules lineup.Rules,
) PulseSide {
	r := PulseSide{
		FranchiseID: saved.Franchise, Name: tradeFranchiseName(snap, saved.Franchise),
		Score: saved.Score, SecondsRemaining: saved.SecondsRemaining,
		Playing: saved.Playing, YetToPlay: saved.YetToPlay, Starters: []PulsePlayer{},
	}
	ordered := LineupReading{Franchise: saved.Franchise, Starters: []LineupPlayer{}, Bench: []LineupPlayer{}}
	ordered.populate(snap, season, rules)
	players := make(map[string]snapshot.Player, len(snap.Players.Value))
	for _, p := range snap.Players.Value {
		players[p.ID.String()] = p
	}
	scores := make(map[string]leaguefeed.PlayerScore, len(saved.Players))
	for _, p := range saved.Players {
		scores[p.ID.String()] = p
	}
	for _, starter := range ordered.Starters {
		p, score := players[starter.ID], scores[starter.ID]
		name := p.Name
		if name == "" {
			name = starter.ID
		}
		r.Starters = append(r.Starters, PulsePlayer{
			ID: starter.ID, Name: name, Position: string(p.Position), Team: p.Team,
			Score: score.Score, SecondsRemaining: score.SecondsRemaining,
		})
	}
	return r
}
