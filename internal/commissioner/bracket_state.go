package commissioner

import (
	"cmp"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

type handSeed struct {
	Slot int
	Team string
}

// bracketState is what the commissioner's bracket commands stand for in one season, after every
// compensation is folded: whether the bracket is built and from which shape, the seeds and places set by
// hand, and the games whose winner was chosen.
type bracketState struct {
	Built    bool
	Preset   string
	Seeds    []handSeed
	Edits    map[string]string
	Advances map[string]string
	Changes  []ScoreChange
	Presets  map[string][]BracketNodeSpec
}

// foldBracket reads the active bracket commands of a season in the order they were recorded. A later
// command for the same seed, place or game replaces an earlier one. Saved shapes belong to the league, not
// a season. name turns a franchise id into the name shown in each change.
func foldBracket(records []Record, season int, name func(string) string) bracketState {
	s := bracketState{Edits: map[string]string{}, Advances: map[string]string{},
		Changes: []ScoreChange{}, Presets: map[string][]BracketNodeSpec{}}
	active := Replay(records).Active
	for _, r := range records {
		if !isBracketCommand(r.Command.Command) || r.Command.Args.Bracket == nil {
			continue
		}
		change := *r.Command.Args.Bracket
		if r.Command.Command == BracketPresetSave && active[r.Receipt.Entry] {
			s.Presets[change.Name] = change.Shape
		}
		if change.Season != season {
			continue
		}
		status := "Reversed"
		if active[r.Receipt.Entry] {
			status = "Applied"
			s.apply(r.Command.Command, change)
		}
		s.Changes = append(s.Changes, ScoreChange{Entry: r.Receipt.Entry,
			What: bracketLabel(r.Command.Command, change, name), Reason: r.Command.Reason, Status: status})
	}
	return s
}

func (s *bracketState) apply(command string, c RawBracketChange) {
	switch command {
	case BracketBuild:
		s.Built, s.Preset = true, c.Preset
	case BracketSeed:
		s.Seeds = slices.DeleteFunc(s.Seeds, func(h handSeed) bool { return h.Team == c.Team })
		s.Seeds = append(s.Seeds, handSeed{Slot: c.Slot, Team: c.Team})
	case BracketEdit:
		s.Edits[c.Node] = c.Team
	case BracketAdvance:
		s.Advances[c.Game] = c.Team
	}
}

// graph is the bracket shape a season is built from: a built-in one (the league's own, or a preset's) or a
// saved one.
func (s bracketState) graph(rules leaguerules.RuleSet) ([]leaguerules.BracketNode, string) {
	if b, ok := builtInBracketByID(cmp.Or(s.Preset, presetLeagueRules)); ok {
		return b.graph(rules), ""
	}
	shape, ok := s.Presets[s.Preset]
	if !ok {
		return nil, "The saved bracket shape " + s.Preset + " is no longer saved"
	}
	return nodesOf(shape), ""
}

// nodesOf turns a saved shape back into the rules' own bracket nodes.
func nodesOf(shape []BracketNodeSpec) []leaguerules.BracketNode {
	slot := func(seed int, from string) (leaguerules.Value[int], leaguerules.Value[string]) {
		if seed > 0 {
			return leaguerules.Known(seed), leaguerules.Known("")
		}
		return leaguerules.Needs[int]("A saved bracket shape"), leaguerules.Known(from)
	}
	out := make([]leaguerules.BracketNode, len(shape))
	for i, n := range shape {
		out[i] = leaguerules.BracketNode{ID: leaguerules.Known(n.ID), Week: leaguerules.Known(n.Week)}
		out[i].HomeSeed, out[i].HomeWinner = slot(n.HomeSeed, n.HomeFrom)
		out[i].AwaySeed, out[i].AwayWinner = slot(n.AwaySeed, n.AwayFrom)
	}
	return out
}

// handFields lists each conference's seeds set by hand, with a later seed replacing an earlier one for the
// same slot. layout says which conference each team plays in.
func (s bracketState) handFields(layout conferenceLayout) map[string]map[int]string {
	hand := map[string]map[int]string{}
	for _, h := range s.Seeds {
		conference := layout.Of[h.Team]
		if hand[conference] == nil {
			hand[conference] = map[int]string{}
		}
		hand[conference][h.Slot] = h.Team
	}
	return hand
}
