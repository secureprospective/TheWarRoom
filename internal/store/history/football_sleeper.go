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
