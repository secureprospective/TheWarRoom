package commissioner

import (
	"sort"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// CommandSafeguard is one commissioner command and the confirm Go asks for it: "confirm", "hold" or
// "reason". A rule change asks differently by timing, so it has a row for each timing and answer; Field
// names one rule that gets that answer.
type CommandSafeguard struct {
	Command   string `json:"command"`
	Effective string `json:"effective"`
	Field     string `json:"field"`
	Safeguard string `json:"safeguard"`
}

// CommandSafeguards lists every commissioner command the app can record with the confirm it needs, so a
// screen can be tested against Go's own list and a new command is covered the day it exists.
func CommandSafeguards() []CommandSafeguard {
	var out []CommandSafeguard
	commands := []string{Create, Revert}
	for _, d := range peopleDefinitions() {
		commands = append(commands, d.command)
	}
	succession := make([]string, 0, len(successionDefinitions()))
	for command := range successionDefinitions() {
		succession = append(succession, command)
	}
	sort.Strings(succession)
	for _, command := range append(commands, succession...) {
		d, err := DefinitionFor(command)
		if err != nil {
			continue
		}
		out = append(out, CommandSafeguard{Command: command, Effective: "now", Safeguard: safeguardFor(d.Gravity)})
	}
	return append(out, ruleSafeguards()...)
}

// ruleSafeguards gives, for each timing, one example rule for every confirm a rule change can ask.
func ruleSafeguards() []CommandSafeguard {
	var out []CommandSafeguard
	for _, effective := range []string{"now", "next_rollover"} {
		seen := map[string]bool{}
		for _, f := range leaguerules.Catalogue() {
			d, err := RawCommand{Command: RulesSet, Effective: effective,
				Args: RawArgs{Field: string(f.Path)}}.Definition()
			if err != nil || seen[d.Gravity] {
				continue
			}
			seen[d.Gravity] = true
			out = append(out, CommandSafeguard{Command: RulesSet, Effective: effective,
				Field: string(f.Path), Safeguard: safeguardFor(d.Gravity)})
		}
	}
	return out
}
