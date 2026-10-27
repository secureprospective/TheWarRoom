package source

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// RawSleeperWeek retains an exact regular-season response and the scope absent from its body.
type RawSleeperWeek struct {
	Season int    `json:"season"`
	Week   int    `json:"week"`
	Body   string `json:"body"`
}

func (r RawSleeperWeek) Validate() error {
	if r.Season < 1920 || r.Season > 9999 || r.Week < 1 || r.Week > 22 {
		return fmt.Errorf("sleeper statistics need a season and week")
	}
	var rows map[string]*RawSleeperStats
	if err := json.Unmarshal([]byte(r.Body), &rows); err != nil {
		return fmt.Errorf("decode Sleeper statistics: %w", err)
	}
	if rows == nil {
		return fmt.Errorf("sleeper statistics need an object")
	}
	for id, row := range rows {
		if row == nil {
			return fmt.Errorf("sleeper row %s needs statistics", id)
		}
		if err := row.Validate(); err != nil {
			return fmt.Errorf("sleeper row %s: %w", id, err)
		}
	}
	return nil
}

// RawSleeperStats is the owned edge-stat subset; other response fields are ignored.
// Missing numbers mean zero. Air yards and receiving YAC may be negative; sack yards may be fractional.
// Conversions, red-zone counts and rushing TFL belong to play-by-play, not this subset.
type RawSleeperStats struct {
	RushYAC             json.Number `json:"rush_yac"`
	ReceivingYAC        json.Number `json:"rec_yar"`
	ReceivingAirYards   json.Number `json:"rec_air_yd"`
	PassingAirYards     json.Number `json:"pass_air_yd"`
	BrokenTackles       json.Number `json:"rush_btkl"`
	Drops               json.Number `json:"rec_drop"`
	Penalties           json.Number `json:"penalty"`
	PenaltyYards        json.Number `json:"penalty_yd"`
	SackYards           json.Number `json:"idp_sack_yd"`
	SpecialTeamsTackles json.Number `json:"st_tkl_solo"`
	OffensiveSnaps      json.Number `json:"off_snp"`
	DefensiveSnaps      json.Number `json:"def_snp"`
	SpecialTeamsSnaps   json.Number `json:"st_snp"`
}

func (r RawSleeperStats) fields() map[leaguerules.StatID]json.Number {
	fields := map[leaguerules.StatID]json.Number{
		"rushing_yards_after_contact": r.RushYAC, "receiving_yards_after_catch": r.ReceivingYAC,
		"receiving_air_yards": r.ReceivingAirYards, "passing_air_yards": r.PassingAirYards,
		"rushing_broken_tackle": r.BrokenTackles, "receiving_drop": r.Drops,
		"penalty": r.Penalties, "penalty_yards": r.PenaltyYards,
		"defensive_sack_yards": r.SackYards, "special_teams_solo_tackle": r.SpecialTeamsTackles,
		"offensive_snap": r.OffensiveSnaps, "defensive_snap": r.DefensiveSnaps,
		"special_teams_snap": r.SpecialTeamsSnaps,
	}
	for id, value := range fields {
		if value == "" {
			fields[id] = "0"
		}
	}
	return fields
}

func (r RawSleeperStats) Validate() error {
	for id, number := range r.fields() {
		value, err := decimal(number.String())
		if err != nil {
			return fmt.Errorf("sleeper %s: %w", id, err)
		}
		signed := id == "receiving_yards_after_catch" || strings.Contains(string(id), "air_yards")
		if value < 0 && !signed {
			return fmt.Errorf("sleeper %s cannot be negative", id)
		}
	}
	return nil
}

// Convert emits only the Sleeper-owned quantities, never replacement nflverse statistics.
func (r RawSleeperStats) Convert() (fantasyscore.StatLine, error) {
	if err := r.Validate(); err != nil {
		return fantasyscore.StatLine{}, err
	}
	quantities := make(map[leaguerules.StatID]leaguerules.Decimal, 13)
	for id, number := range r.fields() {
		quantities[id], _ = decimal(number.String())
	}
	return fantasyscore.StatLine{Quantities: quantities}, nil
}
