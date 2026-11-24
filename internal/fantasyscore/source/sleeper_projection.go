package source

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// SleeperProjection retains the feed's player key, resolved canonical id, true position and raw line.
type SleeperProjection struct {
	ExternalID  string
	CanonicalID string
	Position    leaguerules.Position
	Line        fantasyscore.StatLine
}

type rawSleeperProjection struct {
	PlayerID   string              `json:"player_id"`
	Team       string              `json:"team"`
	Week       json.Number         `json:"week"`
	Sport      string              `json:"sport"`
	SeasonType string              `json:"season_type"`
	Stats      RawLiveSleeperStats `json:"stats"`
	Player     struct {
		Position string `json:"position"`
	} `json:"player"`
}

// ParseSleeperProjections validates a Sleeper/RotoWire snapshot and maps known player ids. The
// identity map is external Sleeper id -> canonical league player id; team defense uses DEF:<team>.
func ParseSleeperProjections(body []byte, identities map[string]string) ([]SleeperProjection, map[leaguerules.Position]int, int, error) {
	var rows []rawSleeperProjection
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&rows); err != nil {
		return nil, nil, 0, fmt.Errorf("decode Sleeper projections: %w", err)
	}
	if dec.Decode(new(any)) == nil {
		return nil, nil, 0, fmt.Errorf("decode Sleeper projections: trailing JSON")
	}
	positions := map[leaguerules.Position]int{}
	out := make([]SleeperProjection, 0, len(rows))
	unmapped := 0
	for i, row := range rows {
		projection, supported, err := mapSleeperProjection(row, identities)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("sleeper projection row %d: %w", i, err)
		}
		if !supported {
			unmapped++
			continue
		}
		positions[projection.Position]++
		out = append(out, projection)
	}
	return out, positions, unmapped, nil
}

func mapSleeperProjection(row rawSleeperProjection, identities map[string]string) (SleeperProjection, bool, error) {
	if row.PlayerID == "" || row.Stats == nil || row.Sport != "nfl" || row.SeasonType != "regular" || row.Week == "" {
		return SleeperProjection{}, false, fmt.Errorf("has invalid identity, period, or stats")
	}
	position := projectionPosition(row.Player.Position)
	if row.PlayerID == row.Team && row.Team != "" {
		position = "DEF"
	}
	if position == "" {
		return SleeperProjection{}, false, nil
	}
	canonical := identities[row.PlayerID]
	if position == "DEF" {
		canonical = "DEF:" + row.Team
	}
	if canonical == "" {
		return SleeperProjection{}, false, nil
	}
	line, err := row.Stats.Convert()
	if err != nil {
		return SleeperProjection{}, false, fmt.Errorf("%s: %w", row.PlayerID, err)
	}
	if len(line.Quantities) == 0 {
		return SleeperProjection{}, false, nil
	}
	return SleeperProjection{ExternalID: row.PlayerID, CanonicalID: canonical, Position: position, Line: line}, true, nil
}

func projectionPosition(position string) leaguerules.Position {
	p := leaguerules.Position(strings.ToUpper(position))
	if p == "K" {
		return leaguerules.PK
	}
	switch p {
	case leaguerules.QB, leaguerules.RB, leaguerules.WR, leaguerules.TE, leaguerules.PK,
		leaguerules.DT, leaguerules.DE, leaguerules.LB, leaguerules.CB, leaguerules.S, leaguerules.OL:
		return p
	}
	return ""
}
