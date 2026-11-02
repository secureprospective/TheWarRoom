package source

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// RawLiveSleeperStats adds core fields for live gaps without changing pinned import ownership.
// Only supplied fields are emitted: an absent field is not evidence of zero.
type RawLiveSleeperStats struct {
	RawSleeperStats
	PassYards      json.Number `json:"pass_yd"`
	PassTD         json.Number `json:"pass_td"`
	PassAttempt    json.Number `json:"pass_att"`
	PassCompletion json.Number `json:"pass_cmp"`
	PassINT        json.Number `json:"pass_int"`
	PassTwo        json.Number `json:"pass_2pt"`
	RushYards      json.Number `json:"rush_yd"`
	RushTD         json.Number `json:"rush_td"`
	RushAttempt    json.Number `json:"rush_att"`
	RushTwo        json.Number `json:"rush_2pt"`
	RecYards       json.Number `json:"rec_yd"`
	RecTD          json.Number `json:"rec_td"`
	Reception      json.Number `json:"rec"`
	Target         json.Number `json:"rec_tgt"`
	RecTwo         json.Number `json:"rec_2pt"`
	Fumble         json.Number `json:"fum"`
	FumbleLost     json.Number `json:"fum_lost"`
	Tackle         json.Number `json:"idp_tkl"`
	Assist         json.Number `json:"idp_tkl_ast"`
	Sack           json.Number `json:"idp_sack"`
	TFL            json.Number `json:"idp_tkl_loss"`
	QBHit          json.Number `json:"idp_qb_hit"`
	PD             json.Number `json:"idp_pass_def"`
	INT            json.Number `json:"idp_int"`
	ForcedFumble   json.Number `json:"idp_ff"`
	Recovery       json.Number `json:"idp_fum_rec"`
	DefTD          json.Number `json:"idp_def_td"`
	Safety         json.Number `json:"idp_safe"`
	PuntYards      json.Number `json:"pr_yd"`
	KickYards      json.Number `json:"kr_yd"`
	PAT            json.Number `json:"xpm"`
}

func (r RawLiveSleeperStats) fields() map[leaguerules.StatID]json.Number {
	fields := r.RawSleeperStats.fields()
	fields["passing_yards"] = r.PassYards
	fields["passing_td"] = r.PassTD
	fields["pass_attempt"] = r.PassAttempt
	fields["pass_completion"] = r.PassCompletion
	fields["interception_thrown"] = r.PassINT
	fields["passing_two_point"] = r.PassTwo
	fields["rushing_yards"] = r.RushYards
	fields["rushing_td"] = r.RushTD
	fields["rushing_attempt"] = r.RushAttempt
	fields["rushing_two_point"] = r.RushTwo
	fields["receiving_yards"] = r.RecYards
	fields["receiving_td"] = r.RecTD
	fields["reception"] = r.Reception
	fields["target"] = r.Target
	fields["receiving_two_point"] = r.RecTwo
	fields["fumbles_total"] = r.Fumble
	fields["fumble_lost"] = r.FumbleLost
	fields["tackle"] = r.Tackle
	fields["assist"] = r.Assist
	fields["sack"] = r.Sack
	fields["tackle_for_loss"] = r.TFL
	fields["qb_hit"] = r.QBHit
	fields["pass_defensed"] = r.PD
	fields["interception_caught"] = r.INT
	fields["forced_fumble"] = r.ForcedFumble
	fields["opponent_fumble_recovery"] = r.Recovery
	fields["def_tds"] = r.DefTD
	fields["safety"] = r.Safety
	fields["punt_return_yards"] = r.PuntYards
	fields["kickoff_return_yards"] = r.KickYards
	fields["pat_made"] = r.PAT
	return fields
}

func (r RawLiveSleeperStats) Validate() error {
	for id, number := range r.fields() {
		if number == "" {
			continue
		}
		value, err := decimal(number.String())
		if err != nil {
			return fmt.Errorf("sleeper live %s: %w", id, err)
		}
		if value < 0 && !strings.Contains(string(id), "yards") && id != "receiving_yards_after_catch" {
			return fmt.Errorf("sleeper live %s cannot be negative", id)
		}
	}
	return nil
}

func (r RawLiveSleeperStats) Convert() (fantasyscore.StatLine, error) {
	if err := r.Validate(); err != nil {
		return fantasyscore.StatLine{}, err
	}
	line := fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{}}
	for id, number := range r.fields() {
		if number != "" {
			line.Quantities[id], _ = decimal(number.String())
		}
	}
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
