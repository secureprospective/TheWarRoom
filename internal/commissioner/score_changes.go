package commissioner

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// ScoreChange records the provenance of a position override or a points adjustment.
type ScoreChange struct {
	Entry  string `json:"entry"`
	What   string `json:"what"`
	Reason string `json:"reason"`
	Status string `json:"status"`
}

type ScoreChanges struct {
	Position leaguerules.Position `json:"position,omitempty"`
	Delta    leaguerules.Decimal  `json:"delta"`
	Changes  []ScoreChange        `json:"changes"`
}

// PlayerScoreChanges folds compensation before selecting the last position and adding each delta once.
func PlayerScoreChanges(records []Record, player string, season, week int) (ScoreChanges, error) {
	return playerScoreChanges(records, Replay(records).Active, player, season, week)
}

func playerScoreChanges(records []Record, active map[string]bool, player string,
	season, week int) (ScoreChanges, error) {
	id, err := playerid.New(player)
	if err != nil || id.String() != player || season < 1920 || season > 9999 || week < 1 || week > 22 {
		return ScoreChanges{}, fmt.Errorf("choose a player, season and week")
	}
	return scoreChangesFrom(records, active, player, "", season, week)
}

func FranchiseScoreChanges(records []Record, franchise string, season, week int) (ScoreChanges, error) {
	if franchise == "" || season < 1920 || season > 9999 || week < 1 || week > 22 {
		return ScoreChanges{}, fmt.Errorf("choose a franchise, season and week")
	}
	return scoreChanges(records, "", franchise, season, week)
}

func scoreChanges(records []Record, player, franchise string, season, week int) (ScoreChanges, error) {
	return scoreChangesFrom(records, Replay(records).Active, player, franchise, season, week)
}

// scoreChangesFrom reads the changes with the ledger's active entries already replayed.
func scoreChangesFrom(records []Record, active map[string]bool, player, franchise string,
	season, week int) (ScoreChanges, error) {
	out := ScoreChanges{Changes: []ScoreChange{}}
	for _, r := range records {
		command := r.Command.Command
		switch command {
		case PlayerPosition, PlayerPositionClear, ScorePlayer, ScoreFranchise:
		default:
			continue
		}
		change, err := DecodeScoreChange(r.Command.Args, command)
		if err != nil {
			return ScoreChanges{}, err
		}
		if change.Player != player || change.Franchise != franchise {
			continue
		}
		position := command == PlayerPosition || command == PlayerPositionClear
		if !position && (change.Season != season || change.Week != week) {
			continue
		}
		status := "Reversed"
		if active[r.Receipt.Entry] {
			status = "Applied"
			if position {
				out.Position = change.Position
			} else {
				out.Delta, err = addScoreDelta(out.Delta, *change.Delta)
				if err != nil {
					return ScoreChanges{}, err
				}
			}
		}
		out.Changes = append(out.Changes, ScoreChange{Entry: r.Receipt.Entry,
			What: scoreChangeLabel(command, change), Reason: r.Command.Reason, Status: status})
	}
	return out, nil
}

func checkScoreAppend(raw RawCommand, records []Record) error {
	change, err := DecodeScoreChange(raw.Args, raw.Command)
	if err != nil {
		return err
	}
	trial := scoreTrial(raw, records)
	if raw.Command == ScorePlayer || raw.Command == ScoreFranchise {
		if _, err := scoreChanges(trial, change.Player, change.Franchise, change.Season, change.Week); err != nil {
			return Refused(err)
		}
	}
	if raw.Command != ScoreFranchise {
		return nil
	}
	for _, r := range records {
		if r.Command.Command == Create {
			for _, id := range r.Command.Args.Franchises {
				if id == change.Franchise {
					return nil
				}
			}
		}
	}
	return Refused(fmt.Errorf("that franchise is not in this app league"))
}

func scoreTrial(raw RawCommand, records []Record) []Record {
	trial := make([]Record, len(records), len(records)+1)
	copy(trial, records)
	return append(trial, Record{Command: raw, Receipt: Receipt{Entry: "proposed." + raw.Request}})
}

func checkScoreCompensation(raw RawCommand, records []Record) error {
	trial := scoreTrial(raw, records)
	for _, r := range records {
		if r.Command.Command != ScorePlayer && r.Command.Command != ScoreFranchise {
			continue
		}
		change, err := DecodeScoreChange(r.Command.Args, r.Command.Command)
		if err != nil {
			return err
		}
		if _, err := scoreChanges(trial, change.Player, change.Franchise, change.Season, change.Week); err != nil {
			return Refused(err)
		}
	}
	return nil
}

func (s ScoreChanges) PositionUsed(feed leaguerules.Position) (leaguerules.Position, string) {
	if s.Position != "" {
		return s.Position, "The commissioner chose this position for league scoring"
	}
	return feed, "The position reported with the NFL statistics"
}
