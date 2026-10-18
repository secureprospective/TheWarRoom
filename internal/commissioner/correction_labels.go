package commissioner

import (
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func correctionDescription(records []Record, r Record) string {
	c := r.Command.Args.Correction
	team := func(id string) string { return correctionAssetName(records, id, false) }
	player := func(id string) string { return correctionAssetName(records, id, true) }
	switch r.Command.Command {
	case CorrectRoster:
		if c.Action == "add" {
			return fmt.Sprintf("Added %s to %s to correct the roster", player(c.Player), team(c.Franchise))
		}
		return fmt.Sprintf("Removed %s from %s to correct the roster", player(c.Player), team(c.Franchise))
	case CorrectTransaction:
		for _, original := range records {
			if original.Receipt.Entry != c.Entry {
				continue
			}
			kind := "draft selection"
			if original.Command.Command == RosterCut {
				kind = "player release"
			}
			if original.Command.Command == WaiverProcess || original.Command.Command == WaiverOverride {
				kind = "waiver award"
			}
			return "Reversed a recorded " + kind
		}
	case CorrectProvenance:
		via := map[string]string{"held": "the starting roster", "draft": "the rookie draft", "waiver": "a waiver claim",
			"trade": "a trade", "free_agent": "free agency", "commissioner": "a commissioner assignment"}
		return fmt.Sprintf("Corrected %s's acquisition to %s", player(c.Player), via[c.Via])
	case CorrectPick:
		return fmt.Sprintf("Corrected the season %d round %d pick originally belonging to %s: now owned by %s",
			c.Pick.Season, c.Pick.Round, team(c.Pick.Original), team(c.Owner))
	case RuleException:
		scope := "for " + team(c.Scope.Franchise)
		if c.Scope.Player != "" {
			scope = "for " + player(c.Scope.Player)
		}
		if c.Scope.Transaction != "" {
			scope = "for one recorded transaction"
		}
		end := "with no expiry"
		if c.Until != "" {
			end = "until " + c.Until
		}
		return "Allowed an exception to " + correctionRuleLabel(c.Rule) + " " + scope + ", " + end
	}
	return correctionLabel(r)
}

func correctionAssetName(records []Record, id string, player bool) string {
	for _, r := range records {
		if r.Command.Args.Native == nil {
			continue
		}
		if player {
			for _, p := range r.Command.Args.Native.Players {
				if p.ID == id && p.Name != "" {
					return p.Name
				}
			}
		} else {
			for _, f := range r.Command.Args.Native.Franchises {
				if f.ID == id {
					return f.Name
				}
			}
		}
	}
	if player {
		return "Player " + id
	}
	return "Franchise " + strings.TrimPrefix(id, "franchise.")
}

func correctionRuleLabel(rule string) string {
	if strings.HasPrefix(rule, "calendar.") {
		return strings.ReplaceAll(strings.TrimPrefix(rule, "calendar."), "_", " ") + " window"
	}
	for _, f := range leaguerules.Catalogue() {
		if string(f.Path) == rule {
			return f.Label
		}
	}
	return "the recorded rule"
}
