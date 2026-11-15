package livestats

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// The box states a player's catches and carries but not how long each was, and the league scores the
// long ones (C20, C40, R20, R40). The play text names each gain; a player's gains are kept only when
// their count and yards reconcile with that player's box line, so a missed or misread play never
// becomes a made-up distance. Distances are events, from which the dictionary counts each threshold.

type gainKind struct {
	event, count, yards leaguerules.StatID
}

func attachLongGains(game livegame.Game, lines map[string]*Line) {
	gains := collectGains(game, lines)
	for _, line := range lines {
		for _, kind := range []gainKind{{"long_reception", "reception", "receiving_yards"}, {"long_rush", "rushing_attempt", "rushing_yards"}} {
			held := gains[line][kind.event]
			if !reconciles(line, kind, held) {
				continue
			}
			if line.Stats.Events == nil {
				line.Stats.Events = map[leaguerules.StatID][]leaguerules.Decimal{}
			}
			line.Stats.Events[kind.event] = append([]leaguerules.Decimal{}, held...)
		}
	}
}

// reconciles says whether the parsed gains are exactly the box line's count and yards.
func reconciles(line *Line, kind gainKind, held []leaguerules.Decimal) bool {
	count, hasCount := line.Stats.Quantities[kind.count]
	yards, hasYards := line.Stats.Quantities[kind.yards]
	if !hasCount || !hasYards {
		return false
	}
	var total leaguerules.Decimal
	for _, gain := range held {
		total += gain
	}
	return leaguerules.Decimal(len(held))*10000 == count && total == yards
}

// collectGains reads each counted catch and carry from the play text, by the uniquely named player.
func collectGains(game livegame.Game, lines map[string]*Line) map[*Line]map[leaguerules.StatID][]leaguerules.Decimal {
	receptionPlay, rushPlay := gainPatterns()
	gains := map[*Line]map[leaguerules.StatID][]leaguerules.Decimal{}
	for _, play := range game.Plays {
		text := strings.TrimSpace(play.Text)
		lower := strings.ToLower(text)
		if strings.Contains(lower, "no play") || strings.Contains(lower, "nullified") ||
			strings.Contains(lower, "incomplete") || strings.Contains(lower, "intercepted") {
			continue
		}
		event, m := leaguerules.StatID("long_reception"), receptionPlay.FindStringSubmatch(text)
		if m == nil {
			event, m = "long_rush", rushPlay.FindStringSubmatch(text)
		}
		if m == nil {
			continue
		}
		line, yards, ok := uniquePlayName(lines, m[1], ""), 0, true
		if m[2] != "no gain" {
			yards, ok = gainYards(m[2])
		}
		if line == nil || !ok {
			continue
		}
		if gains[line] == nil {
			gains[line] = map[leaguerules.StatID][]leaguerules.Decimal{}
		}
		gains[line][event] = append(gains[line][event], leaguerules.Decimal(yards)*10000)
	}
	return gains
}

// gainPatterns read a catch (pass ... to NAME ... for N yards) and a carry (NAME up the middle ... for N yards).
func gainPatterns() (reception, rush *regexp.Regexp) {
	return regexp.MustCompile(`pass (?:(?:short|deep) )?(?:(?:left|right|middle) )?to (.+?)` +
			`(?: (?:to|ran ob at|pushed ob at) (?:[A-Z]{2,3} )?\d+)?\.? for (-?\d+ yards?|no gain)`),
		regexp.MustCompile(`^(?:\([^)]*\) )?(\S+(?: \S+)??) (?:scrambles )?` +
			`(?:up the middle|left end|right end|left tackle|right tackle|left guard|right guard|kneels)` +
			`(?: (?:to|ran ob at|pushed ob at) (?:[A-Z]{2,3} )?\d+)?\.? for (-?\d+ yards?|no gain)`)
}

func gainYards(words string) (int, bool) {
	n, err := strconv.Atoi(strings.Fields(words)[0])
	return n, err == nil && n >= -99 && n <= 99
}
