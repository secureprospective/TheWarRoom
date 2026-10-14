package commissioner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

const (
	PlayerPosition      = "commish.player.position"
	PlayerPositionClear = "commish.player.position.clear"
	ScorePlayer         = "commish.score.player"
	ScoreFranchise      = "commish.score.franchise"
)

// RawScoreChange is carried in Args.Value; Delta is exact ten-thousandths, never a replacement total.
type RawScoreChange struct {
	Player    string               `json:"player,omitempty"`
	Position  leaguerules.Position `json:"position,omitempty"`
	Franchise string               `json:"franchise,omitempty"`
	Season    int                  `json:"season,omitempty"`
	Week      int                  `json:"week,omitempty"`
	Delta     *leaguerules.Decimal `json:"delta,omitempty"`
}

func DecodeScoreChange(a RawArgs, command string) (RawScoreChange, error) {
	var raw RawScoreChange
	if a.Name != "" || a.Entry != "" || a.People != nil || a.Field != "" || a.Preset != "" ||
		a.FranchiseCount != 0 || len(a.Seed) != 0 || len(a.Franchises) != 0 {
		return raw, fmt.Errorf("send only the details of this score change")
	}
	decoder := json.NewDecoder(bytes.NewReader(a.Value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return raw, fmt.Errorf("read score change: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return raw, fmt.Errorf("send exactly one score change")
	}
	if err := raw.Validate(command); err != nil {
		return raw, err
	}
	return raw, nil
}

func (r RawScoreChange) Validate(command string) error {
	if command != ScoreFranchise {
		id, err := playerid.New(r.Player)
		if err != nil {
			return fmt.Errorf("choose a player: %w", err)
		}
		if id.String() != r.Player {
			return fmt.Errorf("use the player's full league id")
		}
	} else if !strings.HasPrefix(r.Franchise, "franchise.") || strings.TrimSpace(r.Franchise) != r.Franchise {
		return fmt.Errorf("choose an app franchise")
	}
	switch command {
	case PlayerPosition, PlayerPositionClear:
		return r.validatePosition(command)
	case ScorePlayer, ScoreFranchise:
		return r.validateDelta(command)
	default:
		return fmt.Errorf("this score change is not available")
	}
}

func (r RawScoreChange) validatePosition(command string) error {
	if r.Franchise != "" || r.Season != 0 || r.Week != 0 || r.Delta != nil {
		return fmt.Errorf("send only the player and position")
	}
	if command == PlayerPositionClear && r.Position == "" {
		return nil
	}
	if command == PlayerPosition && slices.Contains([]leaguerules.Position{leaguerules.QB,
		leaguerules.RB, leaguerules.WR, leaguerules.TE, leaguerules.PK, leaguerules.DT,
		leaguerules.DE, leaguerules.LB, leaguerules.CB, leaguerules.S}, r.Position) {
		return nil
	}
	return fmt.Errorf("choose a recognised position, or clear the override")
}

func (r RawScoreChange) validateDelta(command string) error {
	if r.Position != "" || (command == ScorePlayer && r.Franchise != "") ||
		(command == ScoreFranchise && r.Player != "") {
		return fmt.Errorf("adjust one player or one franchise, not both")
	}
	if r.Season < 1920 || r.Season > 9999 || r.Week < 1 || r.Week > 22 || r.Delta == nil || *r.Delta == 0 {
		return fmt.Errorf("choose a season, week and a nonzero points adjustment")
	}
	return nil
}

func isScoreCommand(command string) bool {
	switch command {
	case PlayerPosition, PlayerPositionClear, ScorePlayer, ScoreFranchise:
		return true
	default:
		return false
	}
}

func (r RawCommand) validateScorePlacement() error {
	if r.Effective != "now" || r.Visibility != "league" {
		return fmt.Errorf("score and position changes take effect now and are shown to the league")
	}
	if strings.TrimSpace(r.Reason) == "" {
		return fmt.Errorf("give a reason for this change")
	}
	return nil
}

// ResultFlipCheck is unknown until native results can prove whether a player adjustment flips a game.
func ResultFlipCheck(_ RawScoreChange) leaguerules.Value[bool] {
	return leaguerules.Needs[bool]("A change of game result is not known yet")
}

func scoreChangeLabel(command string, change RawScoreChange) string {
	switch command {
	case PlayerPosition:
		return fmt.Sprintf("Set player %s to %s for league scoring", change.Player, change.Position)
	case PlayerPositionClear:
		return fmt.Sprintf("Restored the feed's position for player %s", change.Player)
	default:
		target := "player " + change.Player
		if command == ScoreFranchise {
			target = "franchise " + change.Franchise
		}
		return fmt.Sprintf("Adjusted %s by %s points in season %d, week %d", target,
			scorePointsWords(*change.Delta), change.Season, change.Week)
	}
}

func (a RawArgs) validateExclusiveArgs(command string) error {
	if isScoreCommand(command) {
		return nil
	}
	if (command != RulesSet && (a.Field != "" || len(a.Value) != 0)) || (command != Create &&
		(a.Preset != "" || a.FranchiseCount != 0 || len(a.Seed) != 0 || len(a.Franchises) != 0)) {
		return fmt.Errorf("send only the details of this league action")
	}
	return nil
}
