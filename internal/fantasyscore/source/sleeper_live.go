package source

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// RawLiveSleeperStats retains numeric protocol fields. Unknown metrics (including fantasy
// totals) are never scoring inputs. Missing metrics remain absent, including missing bands.
type RawLiveSleeperStats map[string]json.Number

func sleeperFieldMap() map[string]leaguerules.StatID {
	fields := sleeperOffenseFields()
	maps.Copy(fields, sleeperDefenseFields())
	maps.Copy(fields, sleeperKickingFields())
	maps.Copy(fields, sleeperEdgesFields())
	return fields
}

func sleeperOffenseFields() map[string]leaguerules.StatID {
	//nolint:gosec // pass_* keys are public passing-stat names, not credentials.
	return map[string]leaguerules.StatID{
		"fum":           "fumbles_total",
		"fum_lost":      "fumble_lost",
		"fum_rec":       "own_fumble_recovery",
		"fum_rec_td":    "offensive_fumble_recovery_td",
		"pass_2pt":      "passing_two_point",
		"pass_att":      "pass_attempt",
		"pass_cmp":      "pass_completion",
		"pass_cmp_40p":  "long_pass_40",
		"pass_fd":       "passing_first_down",
		"pass_inc":      "pass_incompletion",
		"pass_int":      "interception_thrown",
		"pass_ypc":      "passing_average",
		"cmp_pct":       "pass_completion_percentage",
		"rush_ypa":      "rushing_average",
		"rec_ypr":       "receiving_average",
		"pass_rtg":      "passer_rating",
		"pass_rz_att":   "red_zone_pass_attempt",
		"pass_sack":     "times_sacked",
		"pass_sack_yds": "times_sacked_yards",
		"pass_td":       "passing_td",
		"pass_yd":       "passing_yards",
		"rec":           "reception",
		"rec_2pt":       "receiving_two_point",
		"rec_40p":       "long_reception_40",
		"rec_fd":        "receiving_first_down",
		"rec_rz_tgt":    "red_zone_target",
		"rec_td":        "receiving_td",
		"rec_tgt":       "target",
		"rec_yd":        "receiving_yards",
		"rush_2pt":      "rushing_two_point",
		"rush_40p":      "long_rush_40",
		"rush_att":      "rushing_attempt",
		"rush_fd":       "rushing_first_down",
		"rush_rz_att":   "red_zone_carry",
		"rush_td":       "rushing_td",
		"rush_tkl_loss": "rushing_tfl",
		"rush_yd":       "rushing_yards",
	}
}

func sleeperDefenseFields() map[string]leaguerules.StatID {
	return map[string]leaguerules.StatID{
		"idp_def_td":     "def_tds",
		"idp_ff":         "forced_fumble",
		"idp_fum_rec":    "opponent_fumble_recovery",
		"idp_fum_ret_yd": "opponent_fumble_recovery_yards",
		"idp_fum_td":     "fumble_recovery_tds",
		"idp_int":        "interception_caught",
		"idp_int_ret_yd": "interception_return_yards",
		"idp_int_td":     "interception_return_td",
		"idp_pass_def":   "pass_defensed",
		"idp_qb_hit":     "qb_hit",
		"idp_sack":       "sack",
		"idp_sack_yd":    "defensive_sack_yards",
		"idp_safe":       "safety",
		// idp_tkl is solo plus assisted; the tackle stat is the solo ones.
		"idp_tkl_solo": "tackle",
		"idp_tkl_ast":  "assist",
		"idp_tkl_loss": "tackle_for_loss",
	}
}

func sleeperKickingFields() map[string]leaguerules.StatID {
	return map[string]leaguerules.StatID{
		"fga":        "fg_attempt",
		"fgb":        "fg_blocked",
		"fgm":        "fg_made",
		"fgm_40_49":  "fg_made_40_49",
		"fgm_50_59":  "fg_made_50_59",
		"fgm_60p":    "fg_made_60_99",
		"fgmiss":     "fg_missed",
		"kr":         "kickoff_return",
		"kr_td":      "kickoff_return_td",
		"kr_yd":      "kickoff_return_yards",
		"pr":         "punt_return",
		"pr_td":      "punt_return_td",
		"pr_yd":      "punt_return_yards",
		"punt":       "punt",
		"punt_in_20": "punt_inside_20",
		"punt_yd":    "punt_yards",
		"xpb":        "pat_blocked",
		"xpm":        "pat_made",
		"xpmiss":     "pat_missed",
	}
}

func sleeperEdgesFields() map[string]leaguerules.StatID {
	return map[string]leaguerules.StatID{
		"def_snp":     "defensive_snap",
		"off_snp":     "offensive_snap",
		"pass_air_yd": "passing_air_yards",
		"penalty":     "penalty",
		"penalty_yd":  "penalty_yards",
		"rec_air_yd":  "receiving_air_yards",
		"rec_drop":    "receiving_drop",
		"rec_yar":     "receiving_yards_after_catch",
		"rush_btkl":   "rushing_broken_tackle",
		"rush_yac":    "rushing_yards_after_contact",
		"st_snp":      "special_teams_snap",
		"st_tkl_solo": "special_teams_solo_tackle",
	}
}
func (r RawLiveSleeperStats) Validate() error {
	for key, id := range sleeperFieldMap() {
		number, ok := r[key]
		if !ok {
			continue
		}
		value, err := decimal(number.String())
		if err != nil {
			return fmt.Errorf("sleeper live %s: %w", id, err)
		}
		if value < 0 && !strings.Contains(string(id), "yards") && !strings.HasSuffix(string(id), "_average") {
			return fmt.Errorf("sleeper live %s cannot be negative", id)
		}
	}
	for _, keys := range sleeperBands() {
		for _, key := range keys {
			number, ok := r[key]
			if !ok {
				continue
			}
			value, err := decimal(number.String())
			if err != nil {
				return fmt.Errorf("sleeper %s: %w", key, err)
			}
			if value < 0 {
				return fmt.Errorf("sleeper %s cannot be negative", key)
			}
		}
	}

	return nil
}

func (r RawLiveSleeperStats) Convert() (fantasyscore.StatLine, error) {
	if err := r.Validate(); err != nil {
		return fantasyscore.StatLine{}, err
	}
	line := fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{}}
	for key, id := range sleeperFieldMap() {
		if number, ok := r[key]; ok {
			line.Quantities[id], _ = decimal(number.String())
		}
	}
	fillSleeperBands(r, &line)
	leaguerules.DropTotals(line.Quantities)

	return line, nil
}

// LiveSleeperLines checks the whole response before returning any player evidence.
func LiveSleeperLines(raw RawSleeperWeek) (map[string]fantasyscore.StatLine, error) {
	if err := raw.Validate(); err != nil {
		return nil, err
	}
	var rows map[string]*RawLiveSleeperStats
	if err := json.Unmarshal([]byte(raw.Body), &rows); err != nil {
		return nil, fmt.Errorf("decode Sleeper live: %w", err)
	}
	out := map[string]fantasyscore.StatLine{}
	for id, row := range rows {
		if row == nil {
			return nil, fmt.Errorf("sleeper live %s needs statistics", id)
		}
		line, err := row.Convert()
		if err != nil {
			return nil, fmt.Errorf("sleeper live %s: %w", id, err)
		}
		if len(line.Quantities) != 0 {
			out[id] = line
		}
	}
	return out, nil
}

func sleeperBands() map[leaguerules.StatID][]string {
	return map[leaguerules.StatID][]string{
		"fg_made_0_39":    {"fgm_0_19", "fgm_20_29", "fgm_30_39"},
		"fg_missed_0_29":  {"fgmiss_0_19", "fgmiss_20_29"},
		"fg_missed_30_99": {"fgmiss_30_39", "fgmiss_40_49", "fgmiss_50_59", "fgmiss_60p"},
	}
}

func fillSleeperBands(r RawLiveSleeperStats, line *fantasyscore.StatLine) {
	for id, keys := range sleeperBands() {
		var sum leaguerules.Decimal
		complete := true
		for _, key := range keys {
			number, ok := r[key]
			if !ok {
				complete = false
				continue
			}
			value, _ := decimal(number.String())
			sum += value
		}
		// Sleeper may omit zero bins. Only a reconciled total proves those missing bins zero.
		total := "fgmiss"
		if id == "fg_made_0_39" {
			total = "fgm"
		}
		if complete || sleeperBinsReconcile(r, total) {
			line.Quantities[id] = sum
		}
	}
}

func sleeperBinsReconcile(r RawLiveSleeperStats, total string) bool {
	number, ok := r[total]
	if !ok {
		return false
	}
	want, _ := decimal(number.String())
	var sum leaguerules.Decimal
	prefix := total + "_"
	for key, number := range r {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if key != prefix+"0_19" && key != prefix+"20_29" && key != prefix+"30_39" && key != prefix+"40_49" && key != prefix+"50_59" && key != prefix+"60p" {
			continue
		}
		value, err := decimal(number.String())
		if err != nil || value < 0 {
			return false
		}
		sum += value
	}
	return sum == want
}
