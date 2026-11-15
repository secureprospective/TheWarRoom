package source

import (
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

const (
	ESPNLive = "ESPN live"
	MFLLive  = "MFL live"
	Sleeper  = "Sleeper"
)

// MergeStatSources takes the first recorded value for each field, including explicit zero.
// Events and quantities of the same stat are one field; lower sources cannot supply both.
// MFL totals are display-only and never raw evidence. Commissioner corrections remain a
// separate ledger overlay, applied after feed selection by the existing scoring engine.
func MergeStatSources(lines map[string]fantasyscore.StatLine) (fantasyscore.StatLine, map[leaguerules.StatID]string) {
	out := fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{}, Events: map[leaguerules.StatID][]leaguerules.Decimal{}, Labels: map[leaguerules.StatID]string{}}
	owners := map[leaguerules.StatID]string{}
	for _, authority := range []string{NFLVerse, ESPNLive, Sleeper} {
		line := lines[authority]
		for id, value := range line.Quantities {
			if _, exists := owners[id]; exists || countedFromPlays(id, owners, authority) {
				continue
			}
			out.Quantities[id], owners[id], out.Labels[id] = value, authority, line.Labels[id]
		}
		for id, events := range line.Events {
			if owner, exists := owners[id]; exists && owner != authority {
				continue
			}
			out.Events[id], owners[id], out.Labels[id] = slices.Clone(events), authority, line.Labels[id]
		}
	}
	return out, owners
}

// countedFromPlays says an earlier source already holds the play distances this count is worked out from
// (R20 from each run's length), so a later source's stated count, which can trail a play behind, must not
// replace the exact one.
func countedFromPlays(id leaguerules.StatID, owners map[leaguerules.StatID]string, authority string) bool {
	def, ok := leaguerules.StatByID(id)
	if !ok {
		return false
	}
	for _, play := range def.Plays {
		if owner, held := owners[play]; held && owner != authority {
			return true
		}
	}
	return false
}

// LiveAuthority names only contributors, not merely feeds that were fetched.
func LiveAuthority(owners map[leaguerules.StatID]string) string {
	used := map[string]bool{}
	for _, owner := range owners {
		used[owner] = true
	}
	names := []string{}
	for _, name := range []string{ESPNLive, Sleeper} {
		if used[name] {
			names = append(names, name)
		}
	}
	return strings.Join(names, " + ")
}

// FillStatGaps shares the source chain with pinned imports without mutating archived lines.
func FillStatGaps(primary, gaps fantasyscore.StatLine) fantasyscore.StatLine {
	line, _ := MergeStatSources(map[string]fantasyscore.StatLine{NFLVerse: primary, Sleeper: gaps})
	return line
}
