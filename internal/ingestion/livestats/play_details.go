package livestats

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// ESPN's box only totals kicks. Accept its explicit play distance only when the
// uniquely named kicker's parsed attempts reconcile with that same box score.
func attachPlayDetails(game livegame.Game, lines map[string]*Line) {
	kicks := regexp.MustCompile(`^(\S+) (\d+) yard field goal is (GOOD|No Good)[,.]`)
	for _, line := range lines {
		attachPATMisses(line)
		attempts, hasAttempts := line.Stats.Quantities["fg_attempt"]
		made, hasMade := line.Stats.Quantities["fg_made"]
		if !hasAttempts || !hasMade {
			continue
		}
		events := map[leaguerules.StatID][]leaguerules.Decimal{"fg_made_distance": {}, "fg_missed_distance": {}}
		collectKicks(game, lines, line, kicks, events)
		if leaguerules.Decimal(len(events["fg_made_distance"]))*10000 != made ||
			leaguerules.Decimal(len(events["fg_made_distance"])+len(events["fg_missed_distance"]))*10000 != attempts {
			continue
		}
		line.Stats.Events = events
		for _, id := range []leaguerules.StatID{"fg_made_0_39", "fg_made_40_49", "fg_made_50_59", "fg_made_60_99", "fg_missed_0_29", "fg_missed_30_99"} {
			line.Stats.Quantities[id] = 0
		}
		for id, distances := range events {
			for _, distance := range distances {
				line.Stats.Quantities[kickBand(id, distance)] += 10000
			}
		}
	}
	attachRecoveryTDs(game, lines)
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

func uniquePlayName(lines map[string]*Line, name, team string) *Line {
	var found *Line
	for _, line := range lines {
		parts := strings.Fields(line.Name)
		if len(parts) < 2 || (team != "" && line.Team != team) {
			continue
		}
		short := parts[0][:1] + "." + strings.Join(parts[1:], " ")
		if short != name {
			continue
		}
		if found != nil {
			return nil
		}
		found = line
	}
	return found
}

func attachRecoveryTDs(game livegame.Game, lines map[string]*Line) {
	recovery := regexp.MustCompile(`(?i)recovered by ([A-Z]+)-(.+?) at [A-Z]+ 0[,.] TOUCHDOWN`)
	for _, play := range game.Plays {
		text := strings.ToLower(play.Text)
		if !strings.Contains(text, "touchdown") || strings.Contains(text, "no play") ||
			strings.Contains(text, "nullified") || strings.Contains(text, "punts") || strings.Contains(text, "kicks") || play.Start == nil {
			continue
		}
		m := recovery.FindStringSubmatch(play.Text)
		if len(m) == 0 {
			continue
		}
		line := uniquePlayName(lines, m[2], m[1])
		if line == nil {
			continue
		}
		possession := ""
		if play.Start.Possession == game.Home.ID {
			possession = game.Home.Abbreviation
		}
		if play.Start.Possession == game.Away.ID {
			possession = game.Away.Abbreviation
		}
		if possession == "" {
			continue
		}
		id := leaguerules.StatID("fumble_recovery_tds")
		if possession == m[1] {
			id = "offensive_fumble_recovery_td"
		}
		line.Stats.Quantities[id] += 10000
	}
}

func collectKicks(game livegame.Game, lines map[string]*Line, line *Line, kicks *regexp.Regexp,
	events map[leaguerules.StatID][]leaguerules.Decimal) {
	for _, play := range game.Plays {
		m := kicks.FindStringSubmatch(strings.TrimSpace(play.Text))
		if len(m) == 0 || uniquePlayName(lines, m[1], "") != line {
			continue
		}
		distance, err := strconv.Atoi(m[2])
		if err != nil || distance < 1 || distance > 99 {
			continue
		}
		id := leaguerules.StatID("fg_missed_distance")
		if m[3] == "GOOD" {
			id = "fg_made_distance"
		}
		events[id] = append(events[id], leaguerules.Decimal(distance)*10000)
	}
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
