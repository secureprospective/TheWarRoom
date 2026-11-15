package livestats

import (
	"regexp"
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
		column := -1
		for i, label := range box.Labels {
			if box.Category == "fumbles" && label == "REC" {
				column = i
			}
		}
		if column < 0 {
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

func collectRecoveries(game livegame.Game, lines map[string]*Line) (own, opponent map[*Line]int) {
	recovered := regexp.MustCompile(`(?i)FUMBLES \([^)]*\)[^.]*?,? recovered by ([A-Z]{2,3})-(.+?) at `)
	reception, rush := gainPatterns()
	own, opponent = map[*Line]int{}, map[*Line]int{}
	for _, play := range game.Plays {
		text := strings.TrimSpace(play.Text)
		possession, ok := scrimmageFumble(game, play, text)
		if !ok {
			continue
		}
		if m := recovered.FindStringSubmatch(text); m != nil {
			team := playTeam(m[1])
			if line := uniquePlayName(lines, m[2], team); line != nil && team == possession {
				own[line]++
			} else if line != nil {
				opponent[line]++
			}
			continue
		}
		if !strings.Contains(text, "), and recovers at ") {
			continue
		}
		// The ball carrier fell on his own fumble.
		m := reception.FindStringSubmatch(text)
		if m == nil {
			m = rush.FindStringSubmatch(text)
		}
		if m == nil {
			continue
		}
		if line := uniquePlayName(lines, m[1], possession); line != nil {
			own[line]++
		}
	}
	return own, opponent
}

// scrimmageFumble names the team that had the ball on a counted scrimmage play with a fumble. Kicks and
// punts are left out: the start's possession is the kicking team, not the one that lost the ball.
func scrimmageFumble(game livegame.Game, play livegame.Play, text string) (string, bool) {
	lower := strings.ToLower(text)
	if !strings.Contains(lower, "fumbles") || strings.Contains(lower, "no play") || strings.Contains(lower, "nullified") ||
		strings.Contains(lower, "punts") || strings.Contains(lower, "kicks") || play.Start == nil {
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

// playTeam turns the NFL's play-text team codes into ESPN's abbreviations; the rest already agree.
func playTeam(code string) string {
	if espn, ok := map[string]string{"ARZ": "ARI", "BLT": "BAL", "CLV": "CLE", "HST": "HOU", "LA": "LAR", "WAS": "WSH"}[code]; ok {
		return espn
	}
	return code
}
