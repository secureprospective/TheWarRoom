package livestats

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// attachPlayDetails adds what the box leaves out but the play-by-play states: kick distances, PAT misses,
// fumble-recovery touchdowns, long gains and the own/opponent split of fumble recoveries.
func attachPlayDetails(game livegame.Game, lines map[string]*Line) {
	for _, line := range lines {
		attachPATMisses(line)
	}
	attachKickDistances(game, lines)
	attachRecoveryTDs(game, lines)
	attachLongGains(game, lines)
	attachFumbleRecoveries(game, lines)
}

// The play text's field goals, recovery touchdowns, catches and carries. Compiled once: the collector
// reads every live game on each poll.
var (
	kickPlay       = regexp.MustCompile(`^(\S+) (\d+) yard field goal is (GOOD|No Good)[,.]`)
	recoveryTDPlay = regexp.MustCompile(`(?i)recovered by ([A-Z]+)-(.+?) at [A-Z]+ -?\d+[,.] TOUCHDOWN`)
)

// attachKickDistances records each field goal's distance and its band. ESPN's box only totals kicks, so a
// distance is kept only when the uniquely named kicker's parsed attempts reconcile with his box line.
func attachKickDistances(game livegame.Game, lines map[string]*Line) {
	for line, events := range collectKicks(game, lines) {
		attachKicks(line, events)
	}
}

func attachKicks(line *Line, events map[leaguerules.StatID][]leaguerules.Decimal) {
	attempts, hasAttempts := line.Stats.Quantities["fg_attempt"]
	made, hasMade := line.Stats.Quantities["fg_made"]
	if !hasAttempts || !hasMade ||
		leaguerules.Decimal(len(events["fg_made_distance"]))*10000 != made ||
		leaguerules.Decimal(len(events["fg_made_distance"])+len(events["fg_missed_distance"]))*10000 != attempts {
		return
	}
	if line.Stats.Events == nil {
		line.Stats.Events = map[leaguerules.StatID][]leaguerules.Decimal{}
	}
	for _, id := range []leaguerules.StatID{"fg_made_0_39", "fg_made_40_49", "fg_made_50_59", "fg_made_60_99", "fg_missed_0_29", "fg_missed_30_99"} {
		line.Stats.Quantities[id] = 0
	}
	for id, distances := range events {
		line.Stats.Events[id] = distances
		for _, distance := range distances {
			line.Stats.Quantities[kickBand(id, distance)] += 10000
		}
	}
}

func kickBand(id leaguerules.StatID, distance leaguerules.Decimal) leaguerules.StatID {
	if id == "fg_missed_distance" {
		if distance < 300000 {
			return "fg_missed_0_29"
		}
		return "fg_missed_30_99"
	}
	switch {
	case distance < 400000:
		return "fg_made_0_39"
	case distance < 500000:
		return "fg_made_40_49"
	case distance < 600000:
		return "fg_made_50_59"
	default:
		return "fg_made_60_99"
	}
}

// uniquePlayName is the one line whose player the play text names ("C.Kiner", "A.St. Brown"), or nil when
// none or several match. A team narrows the search when the text gives one.
func uniquePlayName(lines map[string]*Line, name, team string) *Line {
	var found *Line
	for _, line := range lines {
		if (team != "" && line.Team != team) || playName(line.Name) != name {
			continue
		}
		if found != nil {
			return nil
		}
		found = line
	}
	return found
}

// playName is how play text writes a player: first initial, a dot, then the rest of the name without a
// generational suffix ("Travis Etienne Jr." is "T.Etienne").
func playName(full string) string {
	parts := slices.DeleteFunc(strings.Fields(full), func(part string) bool {
		switch part {
		case "Jr.", "Sr.", "II", "III", "IV", "V":
			return true
		}
		return false
	})
	if len(parts) < 2 {
		return ""
	}
	return parts[0][:1] + "." + strings.Join(parts[1:], " ")
}

// attachRecoveryTDs credits a fumble returned for a touchdown: to the offense's own recoverer (MFL's own
// fumble recovery TD) or to the defender who took it away. Kicks and punts are left out, as for recoveries.
func attachRecoveryTDs(game livegame.Game, lines map[string]*Line) {
	for _, play := range game.Plays {
		text := strings.ToLower(play.Text)
		if !strings.Contains(text, "touchdown") || voided(text) || strings.Contains(text, "punts") || strings.Contains(text, "kicks") {
			continue
		}
		m := recoveryTDPlay.FindStringSubmatch(play.Text)
		if m == nil {
			continue
		}
		team := playTeam(m[1])
		line := uniquePlayName(lines, m[2], team)
		possession, ok := offense(game, play)
		if line == nil || !ok {
			continue
		}
		id := leaguerules.StatID("fumble_recovery_tds")
		if possession == team {
			id = "offensive_fumble_recovery_td"
		}
		line.Stats.Quantities[id] += 10000
	}
}

// voided says a play did not stand: it was wiped out (no play), its score was nullified by a penalty, or
// offsetting penalties replayed the down.
func voided(lower string) bool {
	return strings.Contains(lower, "no play") || strings.Contains(lower, "nullified") || strings.Contains(lower, "offsetting")
}

// offense is the abbreviation of the team that had the ball when the play started.
func offense(game livegame.Game, play livegame.Play) (string, bool) {
	if play.Start == nil {
		return "", false
	}
	switch play.Start.Possession {
	case game.Home.ID:
		return game.Home.Abbreviation, true
	case game.Away.ID:
		return game.Away.Abbreviation, true
	}
	return "", false
}

// collectKicks reads each standing field goal's distance by its uniquely named kicker.
func collectKicks(game livegame.Game, lines map[string]*Line) map[*Line]map[leaguerules.StatID][]leaguerules.Decimal {
	kicks := map[*Line]map[leaguerules.StatID][]leaguerules.Decimal{}
	for _, play := range game.Plays {
		text := strings.TrimSpace(play.Text)
		m := kickPlay.FindStringSubmatch(text)
		if m == nil || voided(strings.ToLower(text)) {
			continue
		}
		line := uniquePlayName(lines, m[1], "")
		distance, err := strconv.Atoi(m[2])
		if line == nil || err != nil || distance < 1 || distance > 99 {
			continue
		}
		id := leaguerules.StatID("fg_missed_distance")
		if m[3] == "GOOD" {
			id = "fg_made_distance"
		}
		if kicks[line] == nil {
			kicks[line] = map[leaguerules.StatID][]leaguerules.Decimal{"fg_made_distance": {}, "fg_missed_distance": {}}
		}
		kicks[line][id] = append(kicks[line][id], leaguerules.Decimal(distance)*10000)
	}
	return kicks
}

func attachPATMisses(line *Line) {
	made, hasMade := line.Stats.Quantities["pat_made"]
	attempts, hasAttempts := line.Stats.Quantities["pat_attempt"]
	if hasMade && hasAttempts && attempts >= made {
		// ESPN does not split blocked from missed: count all as missed so the scored total is whole;
		// the final nflverse revision carries the split.
		line.Stats.Quantities["pat_missed"] = attempts - made
		if _, held := line.Stats.Quantities["pat_blocked"]; !held {
			line.Stats.Quantities["pat_blocked"] = 0
		}
	}
}
