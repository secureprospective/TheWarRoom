// Package source adapts held nflverse statistics; it never fetches or writes a store.
package source

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

type RawWeek struct {
	Fields map[string]string
}

type PlayerWeek struct {
	GSIS        string
	Season      int
	Week        int
	GameID      string
	SeasonType  string
	RecoveryTDs leaguerules.Decimal
	Line        fantasyscore.StatLine
}

func mappings() map[leaguerules.StatID][]string {
	return map[leaguerules.StatID][]string{
		"passing_td": {"passing_tds"}, "passing_yards": {"passing_yards"},
		"interception_thrown": {"passing_interceptions"}, "passing_two_point": {"passing_2pt_conversions"},
		"rushing_td": {"rushing_tds"}, "rushing_yards": {"rushing_yards"},
		"rushing_attempt": {"carries"}, "rushing_two_point": {"rushing_2pt_conversions"},
		"receiving_td": {"receiving_tds"}, "receiving_yards": {"receiving_yards"},
		"reception": {"receptions"}, "receiving_two_point": {"receiving_2pt_conversions"},
		"punt_return_yards": {"punt_return_yards"}, "kickoff_return_yards": {"kickoff_return_yards"},
		"fumble_lost": {"fumbles_lost_total"}, "own_fumble_recovery": {"fumble_recovery_own"},
		"opponent_fumble_recovery":       {"fumble_recovery_opp"},
		"opponent_fumble_recovery_yards": {"fumble_recovery_yards_opp"},
		"forced_fumble":                  {"def_fumbles_forced"}, "interception_caught": {"def_interceptions"},
		"interception_return_yards": {"def_interception_yards"}, "pass_defensed": {"def_pass_defended"},
		"qb_hit": {"def_qb_hits"}, "blocked_fg": {"def_fg_blocks"}, "blocked_punt": {"def_punt_blocks"},
		"blocked_extra_point": {"def_pat_blocks"}, "tackle": {"def_tackles_solo", "def_tackles_with_assist"},
		"assist": {"def_tackle_assists"}, "sack": {"def_sacks"},
		"tackle_for_loss": {"def_tackles_for_loss"}, "safety": {"def_safeties"},
		"fg_made_0_39":  {"fg_made_0_19", "fg_made_20_29", "fg_made_30_39"},
		"fg_made_40_49": {"fg_made_40_49"}, "fg_made_50_59": {"fg_made_50_59"},
		"fg_missed_0_29":  {"fg_missed_0_19", "fg_missed_20_29"},
		"fg_missed_30_99": {"fg_missed_30_39", "fg_missed_40_49", "fg_missed_50_59", "fg_missed_60_"},
	}
}

func (r RawWeek) validateIdentity() error {
	id := r.Fields["player_id"]
	if !strings.HasPrefix(id, "00-") || len(id) != 10 {
		return fmt.Errorf("invalid GSIS player id")
	}
	for _, digit := range id[3:] {
		if digit < '0' || digit > '9' {
			return fmt.Errorf("invalid GSIS player id")
		}
	}
	for _, field := range []string{"season", "week"} {
		v, err := strconv.Atoi(r.Fields[field])
		if err != nil || v <= 0 {
			return fmt.Errorf("%s must be a positive integer", field)
		}
	}
	if r.Fields["game_id"] == "" {
		return fmt.Errorf("player week needs its NFL game")
	}
	if r.Fields["season_type"] != "REG" && r.Fields["season_type"] != "POST" {
		return fmt.Errorf("choose regular season or postseason")
	}
	return nil
}

func (r RawWeek) Validate() error {
	if err := r.validateIdentity(); err != nil {
		return err
	}
	for _, fields := range mappings() {
		for _, field := range fields {
			value, err := decimal(r.Fields[field])
			if err != nil {
				return fmt.Errorf("%s: %w", field, err)
			}
			if value < 0 && !strings.Contains(field, "yards") {
				return fmt.Errorf("%s cannot be negative", field)
			}
		}
	}
	for _, field := range []string{"pat_made", "pat_missed", "pat_blocked", "fg_blocked", "fumbles_total", "fumble_recovery_tds", "def_tds"} {
		if _, err := decimal(r.Fields[field]); err != nil {
			return fmt.Errorf("%s: %w", field, err)
		}
	}
	return r.validateDistances()
}

func (r RawWeek) validateDistances() error {
	made, err := distances(r.Fields["fg_made_list"])
	if err != nil {
		return fmt.Errorf("field goals made: %w", err)
	}
	for _, distance := range made {
		if distance < 0 || distance > 990000 {
			return fmt.Errorf("field goal distance outside the preset's 0–99 yard bands")
		}
	}
	count, err := decimal(r.Fields["fg_made_60_"])
	if err != nil {
		return fmt.Errorf("field goals of 60 yards or more: %w", err)
	}
	var n leaguerules.Decimal
	for _, distance := range made {
		if distance >= 600000 {
			n += 10000
		}
	}
	if n != count {
		return fmt.Errorf("field goal distances disagree with the 60-yard count")
	}
	return r.validateBlockedKicks()
}

func (r RawWeek) validateBlockedKicks() error {
	blocked, err := distances(r.Fields["fg_blocked_list"])
	if err != nil {
		return fmt.Errorf("blocked field goals: %w", err)
	}
	for _, distance := range blocked {
		if distance < 0 || distance > 990000 {
			return fmt.Errorf("blocked field goal distance is outside 0–99 yards")
		}
	}
	count, _ := decimal(r.Fields["fg_blocked"])
	if leaguerules.Decimal(len(blocked))*10000 != count {
		return fmt.Errorf("blocked field goal distances disagree with their count")
	}
	return nil
}

func (r RawWeek) Convert() (PlayerWeek, error) {
	if err := r.Validate(); err != nil {
		return PlayerWeek{}, fmt.Errorf("nflverse player week: %w", err)
	}
	season, _ := strconv.Atoi(r.Fields["season"])
	week, _ := strconv.Atoi(r.Fields["week"])
	line := fantasyscore.StatLine{Quantities: make(map[leaguerules.StatID]leaguerules.Decimal),
		Events: make(map[leaguerules.StatID][]leaguerules.Decimal), Labels: map[leaguerules.StatID]string{
			"pat_made": "Extra points made", "pat_missed": "Extra points missed",
			"pat_blocked": "Extra points blocked", "fg_blocked": "Field goals blocked",
			"fumbles_total": "Fumbles", "fumble_recovery_tds": "Fumble recovery touchdowns",
			"def_tds": "Defensive touchdowns"}}
	for id, fields := range mappings() {
		var sum leaguerules.Decimal
		for _, field := range fields {
			v, _ := decimal(r.Fields[field])
			var err error
			sum, err = safeSum(sum, v)
			if err != nil {
				return PlayerWeek{}, fmt.Errorf("%s: %w", id, err)
			}
		}
		line.Quantities[id] = sum
	}
	for _, field := range []string{"pat_made", "pat_missed", "pat_blocked", "fg_blocked", "fumbles_total", "fumble_recovery_tds", "def_tds"} {
		line.Quantities[leaguerules.StatID(field)], _ = decimal(r.Fields[field])
	}
	line.Quantities["fg_made_60_99"], _ = decimal(r.Fields["fg_made_60_"])
	blocked, _ := distances(r.Fields["fg_blocked_list"])
	for _, distance := range blocked {
		id := leaguerules.StatID("fg_missed_30_99")
		if distance < 300000 {
			id = "fg_missed_0_29"
		}
		line.Quantities[id] += 10000
	}
	return PlayerWeek{GSIS: r.Fields["player_id"], Season: season, Week: week,
		GameID: r.Fields["game_id"], SeasonType: r.Fields["season_type"], RecoveryTDs: line.Quantities["fumble_recovery_tds"], Line: line}, nil
}

func ReadWeeks(input io.Reader) ([]PlayerWeek, error) {
	var result []PlayerWeek
	seen := make(map[string]bool)
	err := readRecords(input, func(fields map[string]string) error {
		row, err := (RawWeek{Fields: fields}).Convert()
		if err != nil {
			return err
		}
		key := playKey(row.GSIS, row.Season, row.Week)
		if seen[key] {
			return fmt.Errorf("duplicate nflverse player week %s", key)
		}
		seen[key] = true
		result = append(result, row)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("nflverse weekly statistics are empty")
	}
	return result, nil
}

func readRecords(input io.Reader, consume func(map[string]string) error) error {
	reader := csv.NewReader(input)
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("nflverse header: %w", err)
	}
	seen := make(map[string]bool, len(header))
	for _, name := range header {
		if name == "" || seen[name] {
			return fmt.Errorf("nflverse header is empty or duplicated")
		}
		seen[name] = true
	}
	for {
		cells, readErr := reader.Read()
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return fmt.Errorf("nflverse row: %w", readErr)
		}
		fields := make(map[string]string, len(header))
		for i, name := range header {
			fields[name] = cells[i]
		}
		if err := consume(fields); err != nil {
			return err
		}
	}
}

func decimal(raw string) (leaguerules.Decimal, error) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return 0, fmt.Errorf("missing or malformed stat quantity")
	}
	digits := strings.TrimPrefix(raw, "-")
	for _, digit := range digits {
		if (digit < '0' || digit > '9') && digit != '.' {
			return 0, fmt.Errorf("malformed decimal quantity")
		}
	}
	parts := strings.Split(raw, ".")
	if len(parts) > 2 || parts[0] == "" || parts[0] == "-" || (len(parts) == 2 && parts[1] == "") {
		return 0, fmt.Errorf("malformed decimal quantity")
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if len(fraction) > 4 {
		return 0, fmt.Errorf("stat needs more than four decimal places")
	}
	encoded := parts[0] + fraction + strings.Repeat("0", 4-len(fraction))
	v, err := strconv.ParseInt(encoded, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("stat quantity: %w", err)
	}
	return leaguerules.Decimal(v), nil
}

func distances(raw string) ([]leaguerules.Decimal, error) {
	if raw == "" {
		return nil, nil
	}
	var result []leaguerules.Decimal
	for _, value := range strings.Split(raw, ";") {
		distance, err := decimal(value)
		if err != nil {
			return nil, err
		}
		result = append(result, distance)
	}
	return result, nil
}

func safeSum(a, b leaguerules.Decimal) (leaguerules.Decimal, error) {
	if (b > 0 && a > math.MaxInt64-b) || (b < 0 && a < math.MinInt64-b) {
		return 0, fmt.Errorf("stat quantity exceeds the supported range")
	}
	return a + b, nil
}
