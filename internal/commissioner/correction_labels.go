package commissioner

import (
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func correctionDescription(names *assetNames, r Record) string {
	c := r.Command.Args.Correction
	team := func(id string) string { return names.name(id, false) }
	player := func(id string) string { return names.name(id, true) }
	switch r.Command.Command {
	case CorrectRoster:
		if c.Action == "add" {
			return fmt.Sprintf("Added %s to %s to correct the roster", player(c.Player), team(c.Franchise))
		}
		return fmt.Sprintf("Removed %s from %s to correct the roster", player(c.Player), team(c.Franchise))
	case CorrectTransaction:
		for _, original := range names.records {
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

// assetNames resolves player and franchise names from the first native snapshot in record order that
// names them. The index is built once on first use: a per-lookup scan of every snapshot made corrections
// quadratic in the ledger (lab profile, 11-08).
type assetNames struct {
	records        []Record
	players, teams map[string]string
}

func newAssetNames(records []Record) *assetNames { return &assetNames{records: records} }

func (n *assetNames) name(id string, player bool) string {
	if n.players == nil {
		n.players, n.teams = map[string]string{}, map[string]string{}
		for _, r := range n.records {
			if r.Command.Args.Native == nil {
				continue
			}
			for _, p := range r.Command.Args.Native.Players {
				if _, seen := n.players[p.ID]; !seen && p.Name != "" {
					n.players[p.ID] = p.Name
				}
			}
			for _, f := range r.Command.Args.Native.Franchises {
				if _, seen := n.teams[f.ID]; !seen {
					n.teams[f.ID] = f.Name
				}
			}
		}
	}
	if player {
		if name, ok := n.players[id]; ok {
			return name
		}
		return "Player " + id
	}
	if name, ok := n.teams[id]; ok {
		return name
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
