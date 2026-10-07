package main

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/lineup"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// LineupReading is one franchise's saved MFL lineup for the held week, the rest of its active
// roster as the bench, and whether the league's own starter rules accept it. StarterCount is a
// full lineup's size under those rules (0 when they are unreadable).
type LineupReading struct {
	Franchise    string              `json:"franchise"`
	Week         int                 `json:"week"`
	StarterCount int                 `json:"starterCount"`
	Starters     []LineupPlayer      `json:"starters"`
	Bench        []LineupPlayer      `json:"bench"`
	Check        lineup.Result       `json:"check"`
	Provenance   snapshot.Provenance `json:"provenance"`
	RulesSource  snapshot.Provenance `json:"rulesSource"`
}

// LineupPlayer is a player in the lineup; Position is empty when the directory lacks him.
type LineupPlayer struct {
	ID       string          `json:"id"`
	Position domain.Position `json:"position,omitempty" ts_type:"string"`
}

// TargetLineup reads held state only (no network, never refreshMu). A missing lineup or
// unreadable rules is not an error: the reading's provenance and Check say why.
func (a *App) TargetLineup(franchiseID string) (LineupReading, error) {
	season, err := a.TargetSeason()
	if err != nil {
		return LineupReading{}, fmt.Errorf("target lineup: %w", err)
	}
	a.targetSnapshotMu.Lock()
	snap, loaded := a.targetSnapshot, a.hasTargetSnapshot
	a.targetSnapshotMu.Unlock()
	if !loaded {
		return LineupReading{}, fmt.Errorf("target lineup: no snapshot loaded")
	}
	known := slices.ContainsFunc(snap.Franchises.Value, func(f snapshot.Franchise) bool {
		return f.ID == franchiseID
	})
	if !known {
		return LineupReading{}, fmt.Errorf("target lineup: unknown franchise %q", franchiseID)
	}
	cfg := a.rulebook.ActiveConfig()
	rules, rulesErr := lineup.ParseRules(cfg.Starters)
	r := LineupReading{
		Franchise: franchiseID, Week: season.Lineups.Value.Week, StarterCount: rules.StarterCount,
		Starters: []LineupPlayer{}, Bench: []LineupPlayer{}, Provenance: season.Lineups.Provenance,
		RulesSource: snapshot.Provenance{
			Kind: "live", Source: cfg.Source,
			Freshness: Freshness{State: FreshStale, Note: "active rulebook; fetch time unknown"},
		},
	}
	r.populate(snap, season, rules)
	r.judge(rules, rulesErr)
	return r, nil
}

// judge fills Check; unreadable rules are reported in the reading, not as the binding's error.
func (r *LineupReading) judge(rules lineup.Rules, rulesErr error) {
	if rulesErr != nil {
		note := "The league's lineup rules could not be read: " + rulesErr.Error()
		r.Check = lineup.Result{Problems: []lineup.Problem{
			{Subject: "Rules", Kind: lineup.KindUnknown, Message: note},
		}}
		r.RulesSource.Freshness = Freshness{State: FreshFail, Note: note}
		return
	}
	positions := make([]domain.Position, 0, len(r.Starters))
	for _, p := range r.Starters {
		positions = append(positions, p.Position)
	}
	r.Check = lineup.Check(rules, positions)
}

func (r *LineupReading) populate(snap snapshot.Snapshot, season SeasonReading, rules lineup.Rules) {
	players := make(map[string]snapshot.Player, len(snap.Players.Value))
	for _, p := range snap.Players.Value {
		players[p.ID.String()] = p
	}
	starting := make(map[string]bool)
	found := false
	for _, saved := range season.Lineups.Value.Franchises {
		if saved.Franchise != r.Franchise {
			continue
		}
		found = true
		for _, id := range saved.Starters {
			starting[id.String()] = true
			r.Starters = append(r.Starters, LineupPlayer{ID: id.String(), Position: players[id.String()].Position})
		}
		break
	}
	if !found && r.Provenance.Freshness.State != FreshFail {
		fresh := &r.Provenance.Freshness
		fresh.State = FreshFail
		fresh.Note = joinNote(fresh.Note, "no saved lineup for franchise "+r.Franchise)
	}
	for _, roster := range snap.Rosters.Value {
		if roster.FranchiseID != r.Franchise {
			continue
		}
		for _, p := range roster.Players {
			id := p.ID.String()
			if p.RosterStatus == domain.RosterActive && !starting[id] {
				r.Bench = append(r.Bench, LineupPlayer{ID: id, Position: players[id].Position})
			}
		}
	}
	// Stable: MFL's own order within a position.
	slices.SortStableFunc(r.Starters, func(a, b LineupPlayer) int {
		return cmp.Compare(rules.Order(a.Position), rules.Order(b.Position))
	})
	slices.SortFunc(r.Bench, func(a, b LineupPlayer) int {
		if order := cmp.Compare(rules.Order(a.Position), rules.Order(b.Position)); order != 0 {
			return order
		}
		if name := cmp.Compare(players[a.ID].Name, players[b.ID].Name); name != 0 {
			return name
		}
		return cmp.Compare(a.ID, b.ID)
	})
}

func joinNote(notes ...string) string {
	kept := slices.DeleteFunc(notes, func(n string) bool { return n == "" })
	return strings.Join(kept, "; ")
}
