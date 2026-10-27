package history

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

func (r RawFootballPlayer) validateSourceIDs() error {
	if (strings.TrimSpace(r.Identity) == "" && (r.Identity != "" || r.Sleeper == "")) || len(r.Identity) > 200 {
		return fmt.Errorf("invalid player source identity")
	}
	if r.Sleeper != "" && !sleeperPlayerID(r.Sleeper) {
		return fmt.Errorf("invalid Sleeper player id")
	}
	return nil
}

func sleeperPlayerID(id string) bool {
	if id == "" || len(id) > 20 {
		return false
	}
	for _, digit := range id {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}

func (w RawFootballWeek) validateSleeper() error {
	seen := make(map[int]bool, len(w.Sleeper))
	for _, raw := range w.Sleeper {
		if err := raw.Validate(); err != nil {
			return fmt.Errorf("validate pinned Sleeper statistics: %w", err)
		}
		if raw.Season != w.Season || (w.Week != 0 && raw.Week != w.Week) || seen[raw.Week] {
			return fmt.Errorf("sleeper bodies must match the statistic season/week, one per week")
		}
		seen[raw.Week] = true
	}
	return nil
}

// sleeperRevision extends the existing revision fingerprint only when new bodies are present.
// The legacy branch is deliberately byte-identical for previously pinned evidence.
func (r RawFootballEvidence) sleeperRevision(bodies []RawFootballWeek, legacy string) ([]RawFootballWeek, string, error) {
	var sleeper []source.RawSleeperWeek
	for _, body := range bodies {
		if err := body.validateSleeper(); err != nil {
			return nil, "", err
		}
		sleeper = append(sleeper, body.Sleeper...)
	}
	if len(sleeper) == 0 {
		return bodies, legacy, nil
	}
	bridge := make(map[string]string)
	for mfl, player := range r.Players {
		if player.Sleeper != "" {
			bridge[player.Sleeper] = mfl
		}
	}
	encoded, err := json.Marshal(struct {
		Legacy  string                  `json:"legacy"`
		Sleeper []source.RawSleeperWeek `json:"sleeper"`
		Bridge  map[string]string       `json:"bridge"`
	}{legacy, sleeper, bridge})
	if err != nil {
		return nil, "", fmt.Errorf("fingerprint Sleeper statistics: %w", err)
	}
	return bodies, footballDigest(encoded), nil
}

// canonicalWeek identifies a regular-season week by the pinned league identity.
type canonicalWeek struct {
	player string
	week   int
}

func (r RawFootballEvidence) attachSleeper(weeks []source.PlayerWeek, bodies []RawFootballWeek) (map[canonicalWeek]source.PlayerWeek, map[int]int, error) {
	canonical := make(map[canonicalWeek]source.PlayerWeek)
	gsis := make(map[string]string)
	bridge := make(map[string]string)
	for mfl, player := range r.Players {
		if player.Identity != "" {
			gsis[player.Identity] = mfl
		}
		if player.Sleeper != "" {
			bridge[player.Sleeper] = mfl
		}
	}
	for _, week := range weeks {
		if mfl, ok := gsis[week.GSIS]; ok && week.SeasonType == "REG" {
			key := canonicalWeek{mfl, week.Week}
			if _, exists := canonical[key]; !exists {
				canonical[key] = clonePlayerWeek(week)
			}
		}
	}
	unknown := make(map[int]int)
	for _, body := range bodies {
		for _, raw := range body.Sleeper {
			count, err := r.mergeSleeperWeek(raw, bridge, canonical)
			if err != nil {
				return nil, nil, err
			}
			unknown[raw.Season] += count
		}
	}
	return canonical, unknown, nil
}

func (r RawFootballEvidence) mergeSleeperWeek(raw source.RawSleeperWeek, bridge map[string]string, canonical map[canonicalWeek]source.PlayerWeek) (int, error) {
	var rows map[string]source.RawSleeperStats
	if err := json.Unmarshal([]byte(raw.Body), &rows); err != nil {
		return 0, fmt.Errorf("decode pinned Sleeper statistics: %w", err)
	}
	unknown := 0
	for id, stats := range rows {
		if id == "" || strings.Trim(id, "0123456789") != "" {
			continue
		}
		line, err := stats.Convert()
		if err != nil {
			return 0, fmt.Errorf("convert Sleeper player %s: %w", id, err)
		}
		nonzero := false
		for _, value := range line.Quantities {
			if value != 0 {
				nonzero = true
				break
			}
		}
		if !nonzero {
			continue
		}
		mfl, ok := bridge[id]
		if !ok {
			unknown++
			continue
		}
		key := canonicalWeek{mfl, raw.Week}
		week, ok := canonical[key]
		if !ok {
			week = source.PlayerWeek{Position: string(r.Players[mfl].Facts.Position), Season: raw.Season, Week: raw.Week, SeasonType: "REG", Line: line}
		} else {
			for stat, value := range line.Quantities {
				week.Line.Quantities[stat] = value
			}
		}
		canonical[key] = week
	}
	return unknown, nil
}
