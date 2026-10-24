package commissioner

import (
	"fmt"
	"sort"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// LineupPlayer is one player on a franchise's active roster as the week's lineup sees him: where he plays,
// whether his game has locked him in place, and when, in the owner's own time.
type LineupPlayer struct {
	Player   string     `json:"player"`
	Name     string     `json:"name"`
	Position string     `json:"position"`
	Team     string     `json:"team,omitempty"`
	Locked   bool       `json:"locked"`
	LocksAt  *time.Time `json:"locksAt,omitempty"`
	Words    string     `json:"words"`
}

// NativeLineup is one franchise's lineup for one week: who starts, who sits, who is locked and when, and
// the league's verdict on the lineup in plain words. Set is false when the franchise has none; that is a
// missed lineup, flagged and never penalised.
type NativeLineup struct {
	League        string                    `json:"league"`
	Name          string                    `json:"name"`
	Revision      int64                     `json:"revision"`
	Season        int                       `json:"season"`
	Week          int                       `json:"week"`
	Franchise     string                    `json:"franchise"`
	FranchiseName string                    `json:"franchiseName"`
	Zone          string                    `json:"zone"`
	Set           bool                      `json:"set"`
	Corrected     bool                      `json:"corrected"`
	Words         string                    `json:"words"`
	Reason        string                    `json:"reason,omitempty"`
	Starters      []LineupPlayer            `json:"starters"`
	Bench         []LineupPlayer            `json:"bench"`
	Verdict       nativecheck.LineupVerdict `json:"verdict"`
	Rules         []RuleLine                `json:"rules"`
}

// NativeLineupReading reads the lineup standing for a franchise's week at an instant. The lineup comes
// from the ledger alone, never from MFL's held starters.
func NativeLineupReading(league, franchise, principal string, week int, records []Record, at time.Time,
	env LineupFacts) (NativeLineup, error) {
	if week < 1 || week > 22 {
		return NativeLineup{}, fmt.Errorf("choose a week")
	}
	base, err := readNativeBase(league, records, at)
	if err != nil {
		return NativeLineup{}, err
	}
	if !base.assets.Seeded {
		return NativeLineup{}, fmt.Errorf("record the league's starting snapshot before reading a lineup")
	}
	seat, ok := base.directory.seat(franchise)
	if !ok {
		return NativeLineup{}, fmt.Errorf("choose one of this league's franchises")
	}
	instant := ReadingInstant(records, at)
	season, err := SeasonAt(records, instant)
	if err != nil {
		return NativeLineup{}, err
	}
	rules, err := RulesInForce(records, season, instant)
	if err != nil {
		return NativeLineup{}, err
	}
	held, set, err := RecordedLineupFor(records, franchise, season, week)
	if err != nil {
		return NativeLineup{}, err
	}
	if err := authorizeLineupReading(league, principal, seat.Owner, held.Corrected, records); err != nil {
		return NativeLineup{}, err
	}
	out := NativeLineup{League: league, Name: Replay(records).Name, Revision: int64(len(records)), Season: season,
		Week: week, Franchise: franchise, FranchiseName: seat.Name, Zone: env.zone().String(), Set: set,
		Corrected: held.Corrected, Reason: held.Reason, Starters: []LineupPlayer{}, Bench: []LineupPlayer{},
		Rules: ruleLines(base.rules, "lineup.positions", "lineup.groups", "lineup.invalidPenalty", "lineup.missedFlag")}
	starting := map[string]bool{}
	for _, id := range held.Starters {
		starting[id] = true
	}
	for _, id := range held.Starters {
		out.Starters = append(out.Starters, lineupPlayer(id, env, instant))
	}
	for _, p := range base.playersOf()[franchise] {
		if p.Status == domain.RosterActive && !starting[p.ID] {
			out.Bench = append(out.Bench, lineupPlayer(p.ID, env, instant))
		}
	}
	sortLineupPlayers(out.Bench)
	out.Verdict, _ = judgeStarters(rules.Lineup, set, held.Starters, env.Facts)
	out.Words = lineupWords(out, seat.Name)
	return out, nil
}

func lineupPlayer(id string, env LineupFacts, at time.Time) LineupPlayer {
	f := env.Facts[id]
	p := LineupPlayer{Player: id, Name: env.nameOf(id), Position: string(f.Position), Team: f.Team}
	kickoff, kind := env.lockOf(id)
	switch kind {
	case lockNoGame:
		p.Words = "No NFL game this week, so this player never locks"
	case lockUnknown:
		p.Words = "This week's NFL kickoff times are not held, so we can't tell when this player locks"
	case lockAt:
		local := kickoff.In(env.zone())
		p.LocksAt, p.Locked = &local, !at.Before(kickoff)
		p.Words = "Locks at kickoff, " + env.localTime(kickoff)
		if p.Locked {
			p.Words = "Locked at kickoff, " + env.localTime(kickoff)
		}
	}
	return p
}

func sortLineupPlayers(players []LineupPlayer) {
	order := []string{"QB", "RB", "WR", "TE", "PK", "DT", "DE", "LB", "CB", "S"}
	rank := func(position string) int {
		for i, o := range order {
			if o == position {
				return i
			}
		}
		return len(order)
	}
	sort.SliceStable(players, func(i, j int) bool {
		a, b := players[i], players[j]
		if rank(a.Position) != rank(b.Position) {
			return rank(a.Position) < rank(b.Position)
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.Player < b.Player
	})
}

func lineupWords(l NativeLineup, franchise string) string {
	switch {
	case !l.Set:
		return fmt.Sprintf("%s, week %d: %s.", franchise, l.Week, l.Verdict.Words)
	case l.Corrected:
		return fmt.Sprintf("A commissioner set %s's week %d lineup (%s). %s.", franchise, l.Week, l.Reason, l.Verdict.Words)
	}
	return fmt.Sprintf("%s's week %d lineup is set. %s.", franchise, l.Week, l.Verdict.Words)
}

func authorizeLineupReading(league, principal, owner string, corrected bool, records []Record) error {
	if corrected {
		return nil
	}
	state, err := GovernanceState(records)
	if err != nil {
		return err
	}
	if principal != "" && (SamePrincipal(records, owner, principal) || privilegedLedgerReader(league, principal, GovernanceAuthorizer{State: state})) {
		return nil
	}
	return fmt.Errorf("%s", "This lineup is private to its team and the commissioner.")
}
