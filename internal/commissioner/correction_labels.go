package commissioner

import (
	"fmt"
	"strings"
	"sync"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func correctionTransactionKind(command string) string {
	switch command {
	case RosterCut:
		return "player release"
	case RosterAdd:
		return "free-agent add"
	case WaiverProcess, WaiverOverride:
		return "waiver award"
	default:
		return "draft selection"
	}
}

func correctionDescription(names *assetNames, r Record) string {
	c := r.Command.Args.Correction
	team := func(id string) string { return names.name(id, false) }
	player := func(id string) string { return names.name(id, true) }
	switch r.Command.Command {
	case CorrectRoster:
		return rosterCorrectionDescription(c, player, team)
	case CorrectTransaction:
		for _, original := range names.records {
			if original.Receipt.Entry != c.Entry {
				continue
			}
			return "Reversed a recorded " + correctionTransactionKind(original.Command.Command)
		}
	case CorrectProvenance:
		via := map[string]string{
			"held": "the starting roster", "draft": "the rookie draft", "waiver": "a waiver claim",
			"trade": "a trade", "free_agent": "free agency", "commissioner": "a commissioner assignment",
		}
		return fmt.Sprintf("Corrected %s's acquisition to %s", player(c.Player), via[c.Via])
	case CorrectPick:
		return pickCorrectionDescription(c, player, team)
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
// names them. The index is built once per held revision and shared by every reading and owner on it
// (namesFor); a per-entry rebuild was ~65% of server CPU under load (CT 256 profile, 11-09).
type assetNames struct {
	records        []Record
	players, teams map[string]string
}

func newAssetNames(records []Record) *assetNames {
	n := &assetNames{records: records}
	n.players, n.teams = nameIndex(records)
	return n
}

// namesFor returns the shared index when records are exactly a held revision (memoFor checks every
// payload), and builds one otherwise. The maps are read-only once built.
func namesFor(records []Record) *assetNames {
	memo := memoFor(records)
	if memo == nil {
		return newAssetNames(records)
	}
	memo.names.once.Do(func() { memo.names.players, memo.names.teams = nameIndex(records) })
	return &assetNames{records: records, players: memo.names.players, teams: memo.names.teams}
}

type namesCache struct {
	once           sync.Once
	players, teams map[string]string
}

func nameIndex(records []Record) (players, teams map[string]string) {
	players, teams = map[string]string{}, map[string]string{}
	for _, r := range records {
		if r.Command.Args.Native == nil {
			continue
		}
		for _, p := range r.Command.Args.Native.Players {
			if _, seen := players[p.ID]; !seen && p.Name != "" {
				players[p.ID] = p.Name
			}
		}
		for _, f := range r.Command.Args.Native.Franchises {
			if _, seen := teams[f.ID]; !seen {
				teams[f.ID] = f.Name
			}
		}
	}
	return players, teams
}

func (n *assetNames) name(id string, player bool) string {
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

func rosterCorrectionDescription(c *RawCorrection, player, team func(string) string) string {
	switch c.Action {
	case "add":
		return fmt.Sprintf("Added %s to %s to correct the roster", player(c.Player), team(c.Franchise))
	case "status":
		return fmt.Sprintf("Set %s's roster status on %s to %s", player(c.Player), team(c.Franchise), c.Status)
	case "contract":
		return fmt.Sprintf("Updated %s's contract on %s", player(c.Player), team(c.Franchise))
	case "adjustment":
		return fmt.Sprintf("Recorded salary adjustment %s on %s", c.Charge.ID, team(c.Franchise))
	case "remove_adjustment":
		return fmt.Sprintf("Removed salary adjustment %s from %s", c.Charge.ID, team(c.Franchise))
	default:
		return fmt.Sprintf("Removed %s from %s to correct the roster", player(c.Player), team(c.Franchise))
	}
}

func pickCorrectionDescription(c *RawCorrection, player, team func(string) string) string {
	base := fmt.Sprintf("the season %d round %d pick originally belonging to %s",
		c.Pick.Season, c.Pick.Round, team(c.Pick.Original))
	if c.Pick.Original == "" {
		base = fmt.Sprintf("the season %d round %d slot %d pick", c.Pick.Season, c.Pick.Round, c.Pick.Slot)
	}
	switch {
	case c.Pick.Remove:
		return "Removed " + base
	case c.Pick.Player != "":
		return fmt.Sprintf("Corrected %s: owned by %s, selected %s", base, team(c.Owner), player(c.Pick.Player))
	case c.Pick.Skipped:
		return fmt.Sprintf("Corrected %s: owned by %s, skipped", base, team(c.Owner))
	case c.Pick.Clear:
		return fmt.Sprintf("Corrected %s: owned by %s, selection cleared", base, team(c.Owner))
	default:
		return fmt.Sprintf("Corrected %s: now owned by %s", base, team(c.Owner))
	}
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
