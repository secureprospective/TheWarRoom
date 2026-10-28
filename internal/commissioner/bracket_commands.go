package commissioner

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	BracketBuild      = "bracket.build"
	BracketEdit       = "bracket.edit"
	BracketSeed       = "bracket.seed"
	BracketPresetSave = "bracket.preset.save"
	BracketAdvance    = "bracket.advance"

	// presetLeagueRules is the bracket shape set in the league's own playoff rules; the league presets
	// that carry a playoff graph name the others (builtInBrackets). A saved shape may take none of them.
	presetLeagueRules = "league_rules"

	maxBracketGames = 64
	maxBracketWeek  = 22
)

// BracketNodeSpec is one game of a bracket shape in plain values: each team comes from a fixed seed or
// from a rule such as "winner of semi_1" or "lowest remaining seed".
type BracketNodeSpec struct {
	ID       string `json:"id"`
	Week     int    `json:"week"`
	HomeSeed int    `json:"homeSeed,omitempty"`
	AwaySeed int    `json:"awaySeed,omitempty"`
	HomeFrom string `json:"homeFrom,omitempty"`
	AwayFrom string `json:"awayFrom,omitempty"`
}

// RawBracketChange is a bracket command's argument (Args.Bracket). Started says the commissioner knows the
// first playoff game has kicked off; Go checks it against the NFL calendar, as it does for a running draft.
type RawBracketChange struct {
	Season  int               `json:"season"`
	Started bool              `json:"started,omitempty"`
	Preset  string            `json:"preset,omitempty"`
	Node    string            `json:"node,omitempty"`
	Slot    int               `json:"slot,omitempty"`
	Game    string            `json:"game,omitempty"`
	Team    string            `json:"team,omitempty"`
	Name    string            `json:"name,omitempty"`
	Shape   []BracketNodeSpec `json:"shape,omitempty"`
}

func isBracketCommand(command string) bool {
	switch command {
	case BracketBuild, BracketEdit, BracketSeed, BracketPresetSave, BracketAdvance:
		return true
	}
	return false
}

// changesAfterKickoff are the bracket commands that need a recorded reason once the playoffs have begun.
func changesAfterKickoff(command string) bool {
	return isBracketCommand(command) && command != BracketAdvance
}

// DecodeBracketChange reads a bracket command's argument and refuses a detail that belongs to another command.
func DecodeBracketChange(a RawArgs, command string) (RawBracketChange, error) {
	other := a
	other.Bracket = nil
	if a.Bracket == nil || !reflect.DeepEqual(other, RawArgs{}) {
		return RawBracketChange{}, fmt.Errorf("send only the details of this bracket change")
	}
	raw := *a.Bracket
	if raw.Season < 1920 || raw.Season > 9999 {
		return raw, fmt.Errorf("choose the season this bracket is for")
	}
	if raw.Started && !changesAfterKickoff(command) {
		return raw, fmt.Errorf("an advancement always needs a reason, so it does not say whether the playoffs have started")
	}
	want := raw.shared(command)
	if !reflect.DeepEqual(raw, want) {
		return raw, fmt.Errorf("send only the details of this bracket change")
	}
	return raw, raw.validate(command)
}

// shared keeps only the fields the command takes, so any other field fails the comparison.
func (r RawBracketChange) shared(command string) RawBracketChange {
	keep := RawBracketChange{Season: r.Season, Started: r.Started}
	switch command {
	case BracketBuild:
		keep.Preset = r.Preset
	case BracketEdit:
		keep.Node, keep.Team = r.Node, r.Team
	case BracketSeed:
		keep.Slot, keep.Team = r.Slot, r.Team
	case BracketPresetSave:
		keep.Name, keep.Shape = r.Name, r.Shape
	default:
		keep.Game, keep.Team = r.Game, r.Team
	}
	return keep
}

func (r RawBracketChange) validate(command string) error {
	switch command {
	case BracketBuild:
		if !validPresetID(r.Preset, true) {
			return fmt.Errorf("choose the bracket shape to build from")
		}
	case BracketEdit:
		slotNamed := strings.HasSuffix(r.Node, "/home") || strings.HasSuffix(r.Node, "/away")
		if !slotNamed || !strings.HasPrefix(r.Team, "franchise.") {
			return fmt.Errorf("choose a bracket place, such as a game's home or away team, and an app franchise")
		}
	case BracketSeed:
		if r.Slot < 1 || r.Slot > maxFranchises || !strings.HasPrefix(r.Team, "franchise.") {
			return fmt.Errorf("choose a seed and an app franchise")
		}
	case BracketPresetSave:
		return r.validatePreset()
	default:
		if strings.TrimSpace(r.Game) == "" || !strings.HasPrefix(r.Team, "franchise.") {
			return fmt.Errorf("choose a bracket game and an app franchise")
		}
	}
	return nil
}

// validPresetID accepts a short lower-case name; the two built-in names are allowed only when asked for.
func validPresetID(id string, builtIn bool) bool {
	if id == "" || len(id) > 40 || strings.Trim(id, "abcdefghijklmnopqrstuvwxyz0123456789_-") != "" {
		return false
	}
	_, taken := builtInBracketByID(id)
	return builtIn || !taken
}

func (r RawBracketChange) validatePreset() error {
	if !validPresetID(r.Name, false) {
		return fmt.Errorf("name the bracket shape in lower-case letters, numbers, - or _; the built-in names are taken")
	}
	if len(r.Shape) == 0 || len(r.Shape) > maxBracketGames {
		return fmt.Errorf("a bracket shape needs between 1 and %d games", maxBracketGames)
	}
	for _, n := range r.Shape {
		if !n.valid() {
			return fmt.Errorf("every bracket game needs a name without a slash, a week from 1 to %d, and for each team "+
				"either a seed or a rule", maxBracketWeek)
		}
	}
	return nil
}

// valid says whether a game of a saved shape is named, has a week the NFL season holds, and gives each team
// either a seed or a rule, never both.
func (n BracketNodeSpec) valid() bool {
	named := strings.TrimSpace(n.ID) == n.ID && n.ID != "" && !strings.Contains(n.ID, "/")
	weeks := n.Week >= 1 && n.Week <= maxBracketWeek
	home := n.HomeSeed >= 0 && len(n.HomeFrom) <= 60 && (n.HomeSeed > 0) != (n.HomeFrom != "")
	away := n.AwaySeed >= 0 && len(n.AwayFrom) <= 60 && (n.AwaySeed > 0) != (n.AwayFrom != "")
	return named && weeks && home && away
}

// bracketLabel says what a bracket command did; name turns a franchise id into the name an owner knows.
func bracketLabel(command string, c RawBracketChange, name func(string) string) string {
	switch command {
	case BracketBuild:
		return fmt.Sprintf("Built the season %d playoff bracket from %s", c.Season, presetWords(c.Preset))
	case BracketEdit:
		return fmt.Sprintf("Put %s in as %s for the season %d playoffs", name(c.Team), placeWords(c.Node), c.Season)
	case BracketSeed:
		return fmt.Sprintf("Set %s as playoff seed %d for season %d", name(c.Team), c.Slot, c.Season)
	case BracketPresetSave:
		return fmt.Sprintf("Saved the bracket shape %q with %d games", c.Name, len(c.Shape))
	}
	return fmt.Sprintf("Advanced %s from the %s game in the season %d playoffs", name(c.Team), gameWords(c.Game), c.Season)
}

// gameWords reads a game's id the way an owner would say it: AFC/wildcard_2 is "AFC wildcard 2".
func gameWords(id string) string { return strings.NewReplacer("/", " ", "_", " ").Replace(id) }

// placeWords reads a place in a game, such as AFC/wildcard_2/away, as "the away team of the AFC wildcard 2 game".
func placeWords(place string) string {
	i := strings.LastIndex(place, "/")
	if i < 0 {
		return gameWords(place)
	}
	team := map[string]string{"home": "the home team", "away": "the away team"}[place[i+1:]]
	return fmt.Sprintf("%s of the %s game", team, gameWords(place[:i]))
}

func presetWords(id string) string {
	if b, ok := builtInBracketByID(id); ok {
		return b.words
	}
	return "the saved shape " + id
}
