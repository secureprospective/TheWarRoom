package leaguemigration

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// WaitingSetting is an unanswered rule, using the same catalogue as the rules screen.
type WaitingSetting struct {
	Field  string                  `json:"field"`
	Label  string                  `json:"label"`
	Schema leaguerules.ValueSchema `json:"schema"`
}

// DefaultPreset suggests a starting bundle; it never limits the league's settings.
func DefaultPreset(teams int) string {
	switch teams {
	case 16:
		return "mirror16"
	case 32:
		return "legacy_nfl"
	default:
		return "default32"
	}
}

// Base resolves only creation-time suggestions. MFL's own values take precedence.
func Base(preset string) (leaguerules.RuleSet, error) {
	switch preset {
	case "":
		return leaguerules.RuleSet{}, nil
	case "legacy_nfl":
		return leaguerules.LegacyNFL(), nil
	case "default32":
		return leaguerules.Default32(), nil
	case "mirror16":
		return leaguerules.Mirror16(), nil
	default:
		return leaguerules.RuleSet{}, fmt.Errorf("choose a starting settings bundle")
	}
}

// ImportOptions carries the creation-time bundle and answers to rules still waiting.
type ImportOptions struct {
	Preset  string
	Answers []commissioner.SeedValue
}

func answerCommands(plan Plan, league, reason string, answers []commissioner.SeedValue) ([]commissioner.RawCommand, error) {
	waiting := make(map[string]bool, len(plan.Waiting))
	for _, field := range plan.Waiting {
		waiting[field.Field] = true
	}
	out := make([]commissioner.RawCommand, 0, len(answers))
	for _, answer := range answers {
		if !waiting[answer.Field] {
			return nil, fmt.Errorf("%s is not a setting waiting for an answer", answer.Field)
		}
		if string(answer.Value) == "null" || !json.Valid(answer.Value) {
			return nil, fmt.Errorf("answer %s before importing", answer.Field)
		}
		if _, err := leaguerules.DecodeSetting(leaguerules.FieldPath(answer.Field), answer.Value); err != nil {
			return nil, fmt.Errorf("check %s: %w", answer.Field, err)
		}
		delete(waiting, answer.Field)
		out = append(out, commissioner.RawCommand{League: league, Command: commissioner.RulesSet,
			Args: commissioner.RawArgs{Field: answer.Field, Value: answer.Value}, Request: "mfl-import.answer." + answer.Field,
			Reason: reason, Effective: "now", Visibility: "league"})
	}
	return out, nil
}

func followCommand(league, sourceLeague string, season int, reason string) (commissioner.RawCommand, error) {
	follow := leaguerules.ScheduleFollow{}
	if sourceLeague != "" {
		follow = leaguerules.ScheduleFollow{Source: "mfl", League: strings.TrimSpace(sourceLeague), Season: season}
	}
	value, err := json.Marshal(follow)
	if err != nil {
		return commissioner.RawCommand{}, fmt.Errorf("encode reference schedule: %w", err)
	}
	return commissioner.RawCommand{League: league, Command: commissioner.RulesSet,
		Args: commissioner.RawArgs{Field: "schedule.follow", Value: value}, Request: "schedule.follow." + strconv.Itoa(season),
		Reason: reason, Effective: "now", Visibility: "league"}, nil
}

// BackfillFollowing records the reference setting once for imports predating it.
// Any recorded setting, including a clear, belongs to the commissioner and is never replaced.
func BackfillFollowing(ctx context.Context, ledger Ledger, league string) error {
	records, err := ledger.Records(ctx, league)
	if err != nil {
		return fmt.Errorf("read reference setting: %w", err)
	}
	if len(records) == 0 || commissioner.CutOver(records) {
		return nil
	}
	for _, r := range records {
		if r.Command.Command == commissioner.RulesSet && r.Command.Args.Field == "schedule.follow" {
			return nil
		}
	}
	for _, seed := range records[0].Command.Args.Seed {
		if seed.Field == "schedule.follow" {
			return nil
		}
	}
	assets, err := commissioner.StartingAssetsAt(records, time.Time{})
	if err != nil {
		return fmt.Errorf("read starting assets for reference setting: %w", err)
	}
	if assets != leaguerules.HeldAssets {
		return nil
	}
	season := records[0].Command.Args.Season
	prefix, suffix := "native.mfl-", "-"+strconv.Itoa(season)
	id := strings.TrimSuffix(strings.TrimPrefix(league, prefix), suffix)
	if id == "" || AppLeagueID(id, season) != league {
		// Held assets do not imply that a league has an MFL reference.
		return nil
	}
	raw, err := followCommand(league, id, season, "Record reference schedule setting from the original import")
	if err != nil {
		return err
	}
	raw.Request = "schedule.follow.backfill"
	raw.Expected = int64(len(records))
	if _, err := ledger.Execute(ctx, raw, records[0].Principal); err != nil {
		return fmt.Errorf("record reference setting: %w", err)
	}
	return nil
}
