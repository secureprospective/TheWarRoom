package source

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// RawPlayerCredits contains only named player-credit columns, never description parsing.
type RawPlayerCredits map[string]string

type playerCredit struct {
	column string
	stat   leaguerules.StatID
	amount leaguerules.Decimal
}

func defensiveCredits() []playerCredit {
	return []playerCredit{
		{"solo_tackle_1_player_id", "tackle", 10000}, {"solo_tackle_2_player_id", "tackle", 10000},
		{"tackle_with_assist_1_player_id", "tackle", 10000}, {"tackle_with_assist_2_player_id", "tackle", 10000},
		{"assist_tackle_1_player_id", "assist", 10000}, {"assist_tackle_2_player_id", "assist", 10000},
		{"assist_tackle_3_player_id", "assist", 10000}, {"assist_tackle_4_player_id", "assist", 10000},
		{"sack_player_id", "sack", 10000}, {"half_sack_1_player_id", "sack", 5000}, {"half_sack_2_player_id", "sack", 5000},
		{"qb_hit_1_player_id", "qb_hit", 10000}, {"qb_hit_2_player_id", "qb_hit", 10000},
		{"tackle_for_loss_1_player_id", "tackle_for_loss", 10000}, {"tackle_for_loss_2_player_id", "tackle_for_loss", 10000},
		{"pass_defense_1_player_id", "pass_defensed", 10000}, {"pass_defense_2_player_id", "pass_defensed", 10000},
		{"interception_player_id", "interception_caught", 10000},
		{"forced_fumble_player_1_player_id", "forced_fumble", 10000}, {"forced_fumble_player_2_player_id", "forced_fumble", 10000},
	}
}

func creditColumns() []string {
	out := []string{"safety_player_id", "blocked_player_id", "punt_returner_player_id", "kickoff_returner_player_id", "return_yards", "fumbled_1_player_id", "fumbled_2_player_id",
		"fumble_recovery_1_player_id", "fumble_recovery_2_player_id", "fumble_recovery_1_team", "fumble_recovery_2_team",
		"fumble_recovery_1_yards", "fumble_recovery_2_yards", "fumbled_1_team", "fumbled_2_team", "defteam"}
	for _, c := range defensiveCredits() {
		out = append(out, c.column)
	}
	return out
}

func (r RawPlayerCredits) Validate() error {
	for column, value := range r {
		if value == "" || value == "NA" {
			continue
		}
		switch column {
		case "return_yards", "fumble_recovery_1_yards", "fumble_recovery_2_yards":
			if _, err := decimal(value); err != nil {
				return fmt.Errorf("player credit %s: %w", column, err)
			}
		case "defteam", "fumble_recovery_1_team", "fumble_recovery_2_team", "fumbled_1_team", "fumbled_2_team":
			if len(value) > 3 {
				return fmt.Errorf("invalid player credit team")
			}
		default:
			for _, id := range strings.Split(value, ";") {
				if err := (RawStatScope{GSIS: id, Season: 2026, Week: 1}).Validate(); err != nil {
					return fmt.Errorf("player credit %s: %w", column, err)
				}
			}
		}
	}
	return nil
}

func rawCredits(fields map[string]string) RawPlayerCredits {
	out := make(RawPlayerCredits)
	for column, value := range fields {
		if strings.HasSuffix(column, "player_id") || slices.Contains(creditColumns(), column) {
			out[column] = value
		}
	}
	return out
}

func (r RawPlayerCredits) complete() bool {
	for _, column := range creditColumns() {
		if _, ok := r[column]; !ok {
			return false
		}
	}
	return true
}

// MissingPlayWeeks adds only players absent from the weekly file. It never overwrites an official
// row or invents a zero from absence. A partial credit schema is not fallback evidence.
func MissingPlayWeeks(weeks []PlayerWeek, plays []RawPlay) ([]PlayerWeek, error) {
	if err := validateCoverage(weeks, plays); err != nil {
		return nil, err
	}
	known := make(map[string]bool)
	positions := make(map[string]string)
	for _, w := range weeks {
		known[playKey(w.GSIS, w.Season, w.Week)] = true
		positions[w.GSIS] = w.Position
	}
	found := make(map[string]*PlayerWeek)
	for _, p := range plays {
		if !hasPlayerCredits(p) {
			continue
		}
		season, _ := strconv.Atoi(p.Season)
		week, _ := strconv.Atoi(p.Week)
		add := func(id string, stat leaguerules.StatID, value leaguerules.Decimal) {
			if id == "" || id == "NA" || strings.Contains(id, ";") || value == 0 {
				return
			}
			key := playKey(id, season, week)
			if known[key] {
				return
			}
			w := found[key]
			if w == nil {
				w = &PlayerWeek{GSIS: id, Position: positions[id], Season: season, Week: week, SeasonType: p.SeasonType, GameID: p.GameID,
					Provenance: "nflverse play-by-play", Review: "nflverse missing row", Evidence: "player-id stat credits",
					Line: fantasyscore.StatLine{Quantities: make(map[leaguerules.StatID]leaguerules.Decimal), Events: make(map[leaguerules.StatID][]leaguerules.Decimal)}}
				found[key] = w
			}
			w.Line.Quantities[stat] += value
		}
		for _, c := range defensiveCredits() {
			add(p.Credits[c.column], c.stat, c.amount)
		}
		attachReturnCredits(p, add)
		attachFumbleCredits(p, add)
		attachOffensiveCredits(p, add)
		attachSpecialCredits(p, add)
	}
	out := slices.Clone(weeks)
	for _, key := range slices.Sorted(maps.Keys(found)) {
		found[key].RecoveryTDs = found[key].Line.Quantities["fumble_recovery_tds"]
		out = append(out, *found[key])
	}
	return out, nil
}

func hasPlayerCredits(p RawPlay) bool {
	return p.Credits.complete() && p.Deleted != "1" && p.TwoPoint != "1" && p.PlayType != "no_play"
}

func attachReturnCredits(p RawPlay, add func(string, leaguerules.StatID, leaguerules.Decimal)) {
	yards, _ := decimal(p.Credits["return_yards"])
	for _, c := range []struct {
		column, flag string
		stat         leaguerules.StatID
	}{
		{"punt_returner_player_id", p.Punt, "punt_return_yards"}, {"kickoff_returner_player_id", p.Kickoff, "kickoff_return_yards"},
		{"interception_player_id", p.Interception, "interception_return_yards"},
	} {
		if c.flag == "1" {
			add(p.Credits[c.column], c.stat, yards)
		}
	}
}

func attachFumbleCredits(p RawPlay, add func(string, leaguerules.StatID, leaguerules.Decimal)) {
	for _, slot := range []string{"1", "2"} {
		id := p.Credits["fumble_recovery_"+slot+"_player_id"]
		team := p.Credits["fumble_recovery_"+slot+"_team"]
		if team == "" || team == "NA" {
			continue
		}
		stat := leaguerules.StatID("own_fumble_recovery")
		lostBy := p.Credits["fumbled_"+slot+"_team"]
		if lostBy == "" || lostBy == "NA" {
			continue
		}
		if team != lostBy {
			stat = "opponent_fumble_recovery"
		}
		add(id, stat, 10000)
		if p.Touchdown == "1" && p.Scorer == id && p.Punt != "1" && p.Kickoff != "1" {
			add(id, "fumble_recovery_tds", 10000)
		}
		if stat == "opponent_fumble_recovery" {
			yards, _ := decimal(p.Credits["fumble_recovery_"+slot+"_yards"])
			add(id, "opponent_fumble_recovery_yards", yards)
		}
	}
}

func attachOffensiveCredits(p RawPlay, add func(string, leaguerules.StatID, leaguerules.Decimal)) {
	if p.RushAttempt == "1" {
		yards, _ := decimal(p.Rushing)
		add(p.Rusher, "rushing_attempt", 10000)
		add(p.Rusher, "rushing_yards", yards)
	}
	if p.CompletePass == "1" {
		yards, _ := decimal(p.Passing)
		add(p.Passer, "passing_yards", yards)
		yards, _ = decimal(p.Receiving)
		add(p.Receiver, "receiving_yards", yards)
		add(p.Receiver, "reception", 10000)
	}
	if p.Interception == "1" {
		add(p.Passer, "interception_thrown", 10000)
	}
	if p.Touchdown == "1" {
		if p.Scorer == p.Rusher {
			add(p.Rusher, "rushing_td", 10000)
		}
		if p.Scorer == p.Receiver {
			add(p.Receiver, "receiving_td", 10000)
			add(p.Passer, "passing_td", 10000)
		}
	}
}

func attachSpecialCredits(p RawPlay, add func(string, leaguerules.StatID, leaguerules.Decimal)) {
	if p.Safety == "1" {
		add(p.Credits["safety_player_id"], "safety", 10000)
	}
	if p.PuntBlocked == "1" {
		add(p.Credits["blocked_player_id"], "blocked_punt", 10000)
	}
	if p.FieldGoal == "1" && p.FieldResult == "blocked" {
		add(p.Credits["blocked_player_id"], "blocked_fg", 10000)
	}
	for _, slot := range []string{"1", "2"} {
		add(p.Credits["fumbled_"+slot+"_player_id"], "fumbles_total", 10000)
	}
	if p.FumbleLost == "1" {
		add(p.Credits["fumbled_1_player_id"], "fumble_lost", 10000)
	}
}
