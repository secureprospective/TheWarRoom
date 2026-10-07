package envelope

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/lineup"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func validateExpected(s Spec) error {
	if s.Intent == "trade.accept" {
		return validateTradeExpected(s)
	}
	if s.Expected.Trade != nil {
		return fmt.Errorf("envelope: trade change requires trade.accept")
	}
	if s.Intent == "lineup.set" {
		return validateLineupExpected(s)
	}

	if s.Expected.Lineup != nil || s.Expected.Player.String() == "" || s.Expected.RosterStatus == "" ||
		!slices.Contains(s.Subject.Players, s.Expected.Player) {
		return fmt.Errorf("envelope: expected change must name a subject player and status")
	}
	return nil
}

func validateLineupExpected(s Spec) error {
	l := s.Expected.Lineup
	if l == nil || l.Week < 1 || len(l.Starters) == 0 ||
		s.Expected.Player.String() != "" || s.Expected.RosterStatus != "" {
		return fmt.Errorf("envelope: lineup requires week and starters, without roster change")
	}
	seen := make(map[playerid.PlayerID]bool)
	for _, id := range l.Starters {
		if id.String() == "" || seen[id] {
			return fmt.Errorf("envelope: empty or duplicate starter")
		}
		seen[id] = true
	}
	if len(s.Subject.Players) != len(l.Starters) || !samePlayers(s.Subject.Players, l.Starters) {
		return fmt.Errorf("envelope: lineup subjects must equal starters")
	}
	return nil
}

func samePlayers(a, b []playerid.PlayerID) bool {
	left, right := make(map[playerid.PlayerID]bool), make(map[playerid.PlayerID]bool)
	for _, id := range a {
		left[id] = true
	}
	for _, id := range b {
		right[id] = true
	}
	if len(left) != len(right) {
		return false
	}
	for id := range left {
		if !right[id] {
			return false
		}
	}
	return true
}

// LineupCheck is lineup.set's pre-flight. Blocks are reasons found outside the snapshot (week,
// lock, MFL host, rules); BaselineKnown is false when MFL's saved lineup could not be read, so a
// later match could not be told apart from no change. A known empty baseline is fine.
type LineupCheck struct {
	Rules         lineup.Rules
	BaselineKnown bool
	Blocks        []string
}

// Evaluate blocks on every outside reason, an unknown baseline, a starter not on the active
// roster, an illegal lineup, or a lineup equal to the saved one; partial but legal passes.
func (c LineupCheck) Evaluate(s Spec, snap snapshot.Snapshot) CheckResult {
	if s.Intent != "lineup.set" || s.Expected.Lineup == nil {
		return CheckResult{Blocked: true, Note: "lineup check does not cover this intent"}
	}
	blocks := slices.Clone(c.Blocks)
	if !c.BaselineKnown {
		blocks = append(blocks, "MFL's saved lineup is unknown, so a change could not be confirmed")
	}
	if len(blocks) > 0 {
		return CheckResult{Blocked: true, Note: strings.Join(blocks, "; ")}
	}
	if snap.Rosters.Provenance.Freshness.State == domain.FreshFail {
		return CheckResult{Blocked: true, Note: "roster unavailable"}
	}
	active := make(map[playerid.PlayerID]bool)
	for _, roster := range snap.Rosters.Value {
		if roster.FranchiseID != s.FranchiseID {
			continue
		}
		for _, p := range roster.Players {
			active[p.ID] = p.RosterStatus == domain.RosterActive
		}
	}
	for _, id := range s.Expected.Lineup.Starters {
		if !active[id] {
			return CheckResult{Blocked: true, Note: fmt.Sprintf(
				"starter %s is not on this franchise's active roster (ROSTER-status)", id)}
		}
	}
	result := lineup.Check(c.Rules, LineupPositions(s.Expected.Lineup.Starters, snap))
	notes := []string{}
	for _, p := range result.Problems {
		notes = append(notes, p.Message)
	}
	if !result.Legal {
		return CheckResult{Blocked: true, Note: strings.Join(notes, "; ")}
	}
	if samePlayers(s.Expected.Lineup.Starters, s.Expected.Lineup.Baseline) {
		return CheckResult{Blocked: true,
			Note: fmt.Sprintf("already your saved week %d lineup", s.Expected.Lineup.Week)}
	}
	note := "legal"
	if !result.Full {
		note = "legal, partial; " + strings.Join(notes, "; ")
	}
	return CheckResult{Note: note}
}

// LineupPositions maps starters to the snapshot's positions; a player it lacks maps to "", which
// lineup.Check reports as an unknown position.
func LineupPositions(ids []playerid.PlayerID, snap snapshot.Snapshot) []domain.Position {
	positions := make(map[playerid.PlayerID]domain.Position, len(snap.Players.Value))
	for _, p := range snap.Players.Value {
		positions[p.ID] = p.Position
	}
	out := make([]domain.Position, 0, len(ids))
	for _, id := range ids {
		out = append(out, positions[id])
	}
	return out
}

// LineupPredicate lands a lineup.set when MFL's saved starters for the week equal the drafted
// set (order ignored); the baseline again is no change; any other set is Not verified.
type LineupPredicate struct{}

func (LineupPredicate) Sources() []Source { return []Source{Lineups} }

func (LineupPredicate) Evaluate(s Spec, obs Observation) Verdict {
	l := s.Expected.Lineup
	reason := ""
	switch {
	case l == nil || s.Intent != "lineup.set":
		reason = "lineup intent missing"
	case obs.Lineups.Provenance.Freshness.State == domain.FreshFail:
		reason = "lineup feed failed"
	case obs.LeagueID != s.LeagueID:
		reason = "wrong league"
	case obs.Lineups.Value.Week != l.Week:
		reason = "held lineup week differs from drafted week"
	}
	if reason != "" {
		return Verdict{Event: Partial, Note: "Not verified: " + reason}
	}
	for _, saved := range obs.Lineups.Value.Franchises {
		if saved.Franchise != s.FranchiseID {
			continue
		}
		if samePlayers(saved.Starters, l.Starters) {
			return Verdict{Event: Match, Note: "MFL saved the drafted starters"}
		}
		if samePlayers(saved.Starters, l.Baseline) {
			return Verdict{Event: NoChange, Note: "No matching MFL change observed"}
		}
		k := 0
		for _, id := range l.Starters {
			if slices.Contains(saved.Starters, id) {
				k++
			}
		}
		return Verdict{Event: Partial, Note: fmt.Sprintf(
			"Not verified: MFL shows a different lineup (%d of %d drafted starters)", k, len(l.Starters))}
	}
	return Verdict{Event: Partial, Note: "Not verified: franchise absent from saved lineups"}
}

// LineupRequest is what TargetDraftLineup gathers for DraftLineup.
type LineupRequest struct {
	LeagueID    string
	FranchiseID string
	Week        int
	Starters    []playerid.PlayerID
	Baseline    []playerid.PlayerID
	Target      Target
	Deadline    *time.Time
}

// DraftLineup builds and checks a lineup.set envelope (G1, reversible); an unknown baseline is
// stored empty and its draft is Blocked, so it is never observed.
func DraftLineup(
	at time.Time, req LineupRequest, snap snapshot.Snapshot, check LineupCheck, generate func() string,
) (Envelope, error) {
	if !check.BaselineKnown {
		req.Baseline = nil
	}
	spec := Spec{
		Intent: "lineup.set", LeagueID: req.LeagueID, FranchiseID: req.FranchiseID,
		Subject: Subject{Players: req.Starters},
		Expected: ExpectedChange{Lineup: &ExpectedLineup{
			Week: req.Week, Starters: req.Starters, Baseline: req.Baseline,
		}},
		Gravity: G1, Undo: Reversible, Target: req.Target, Deadline: req.Deadline,
	}
	e, err := New(at, spec, generate)
	if err != nil {
		return Envelope{}, fmt.Errorf("draft lineup: %w", err)
	}
	checked, err := e.Check(at, snap, check)
	if err != nil {
		return Envelope{}, fmt.Errorf("draft lineup: check: %w", err)
	}
	return checked, nil
}
