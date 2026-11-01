package leaguemigration

import (
	"encoding/json"
	"fmt"

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
