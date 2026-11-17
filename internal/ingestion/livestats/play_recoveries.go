package livestats

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// The box's fumbles REC column counts every recovery a player made, his own team's ball (MFL OFC) and the
// opponent's (FC) alike. The play text says which team recovered and the play's start says which team
// had the ball. A player's split is kept only when it adds up to his REC, and kicks and punts are left
// out because possession turns over on them.
func attachFumbleRecoveries(game livegame.Game, lines map[string]*Line) {
	held := boxRecoveries(game, lines)
	if len(held) == 0 {
		return
	}
	own, opponent := collectRecoveries(game, lines)
	for line, total := range held {
		if own[line]+opponent[line] != total {
			continue
		}
		line.Stats.Quantities["own_fumble_recovery"] = leaguerules.Decimal(own[line]) * 10000
		line.Stats.Quantities["opponent_fumble_recovery"] = leaguerules.Decimal(opponent[line]) * 10000
	}
}

// boxRecoveries reads each player's REC from the box's fumbles table.
func boxRecoveries(game livegame.Game, lines map[string]*Line) map[*Line]int {
	held := map[*Line]int{}
	for _, box := range game.Box {
		column := slices.Index(box.Labels, "REC")
		if box.Category != "fumbles" || column < 0 {
			continue
		}
		for _, row := range box.Rows {
			line, ok := lines[row.ID]
			if !ok || column >= len(row.Stats) {
				continue
			}
			if n, err := strconv.Atoi(row.Stats[column]); err == nil && n >= 0 {
				held[line] = n
			}
		}
	}
	return held
}

var recoveredPlay = regexp.MustCompile(`(?i)(?:FUMBLES(?: \([^)]*\))?)[^.]*?,? recovered by ([A-Z]{2,3})-(.+?) at `)

func collectRecoveries(game livegame.Game, lines map[string]*Line) (own, opponent map[*Line]int) {
	own, opponent = map[*Line]int{}, map[*Line]int{}
	for _, play := range game.Plays {
		text := strings.TrimSpace(play.Text)
		possession, ok := scrimmageFumble(game, play, text)
		if !ok {
			continue
		}
		if m := muffRecovery.FindStringSubmatch(text); m != nil {
			if line := uniquePlayName(lines, m[1], ""); line != nil {
				own[line]++
			}
			continue
		}
		if m := recoveredPlay.FindStringSubmatch(text); m != nil {
			team := playTeam(m[1])
			if line := uniquePlayName(lines, m[2], team); line != nil && team == possession {
				own[line]++
			} else if line != nil {
				opponent[line]++
			}
			continue
		}
		if strings.Contains(text, "), and recovers at ") {
			m := receptionPlay.FindStringSubmatch(text)
			if m == nil {
				m = rushPlay.FindStringSubmatch(text)
			}
			if m != nil {
				if line := uniquePlayName(lines, m[1], possession); line != nil {
					own[line]++
				}
			}
			continue
		}
		m := ownRecovery.FindStringSubmatch(text)
		if m == nil {
			continue
		}
		if line := uniquePlayName(lines, m[1], possession); line != nil {
			own[line]++
		}
	}
	return own, opponent
}

var (
	muffRecovery = regexp.MustCompile(`(?i)\b([A-Z]\.\S+) MUFFS catch, and recovers at `)
	ownRecovery  = regexp.MustCompile(`(?i)(?:^|[. ]+)([A-Z]\.\S+(?: \S+)?)(?: FUMBLES(?: \([^)]*\))?(?: at [^.]+)?|\) and recovers at)`)
)

// scrimmageFumble names the team that had the ball on a counted scrimmage play with a fumble. A punt
// muff is the exception: its catcher recovers his own muff even though the kicking team had possession.
func scrimmageFumble(game livegame.Game, play livegame.Play, text string) (string, bool) {
	lower := strings.ToLower(text)
	if voided(lower) {
		return "", false
	}
	if strings.Contains(lower, "muffs catch") {
		return "", true
	}
	if !strings.Contains(lower, "fumbles") || strings.Contains(lower, "punts") || strings.Contains(lower, "kicks") {
		return "", false
	}
	return offense(game, play)
}

// playTeam turns the NFL's play-text team codes into ESPN's abbreviations; the rest already agree.
func playTeam(code string) string {
	if espn, ok := map[string]string{"ARZ": "ARI", "BLT": "BAL", "CLV": "CLE", "HST": "HOU", "LA": "LAR", "WAS": "WSH"}[code]; ok {
		return espn
	}
	return code
}
