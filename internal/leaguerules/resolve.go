package leaguerules

import (
	"fmt"
	"time"
)

type EffectiveKind uint8

const (
	NextRollover EffectiveKind = iota
	Now
	Instant
)

// At is the ledger-supplied rollover point or chosen instant, never an inferred date.
type Effective struct {
	Kind EffectiveKind
	At   time.Time
}
type Change struct {
	LeagueID   string
	Field      FieldPath
	New        Setting
	Effective  Effective
	RecordedAt time.Time
	Season     int
	// Franchises, set on a Teams or Divisions change, is the league's franchises in order once the
	// change applies. Resolving the change keeps the division alignment to them (see alignTo). It is
	// omitted when empty, so a stored change without it encodes, and digests, as it did.
	Franchises []string `json:",omitempty"`
}

func (c Change) Validate() error {
	return c.validate(ruleFields())
}

func (c Change) validate(fields []ruleField) error {
	if c.LeagueID == "" {
		return fmt.Errorf("choose a league")
	}
	if c.Season <= 0 {
		return fmt.Errorf("choose a season")
	}
	if c.RecordedAt.IsZero() {
		return fmt.Errorf("record when this rule was changed")
	}
	if c.New == nil {
		return fmt.Errorf("choose a rule value or mark it Needs a ruling")
	}
	if _, err := c.starts(); err != nil {
		return err
	}
	for _, f := range fields {
		if f.pathName() != c.Field {
			continue
		}
		var r RuleSet
		if err := f.apply(&r, c.New); err != nil {
			return err
		}
		return validateRules(r)
	}
	return fmt.Errorf("choose a recognised league rule")
}
func (c Change) starts() (time.Time, error) {
	switch c.Effective.Kind {
	case Now:
		if !c.Effective.At.IsZero() && !c.Effective.At.Equal(c.RecordedAt) {
			return time.Time{}, fmt.Errorf("apply now must start when the change was recorded")
		}
		return c.RecordedAt, nil
	case Instant, NextRollover:
		if c.Effective.At.IsZero() {
			return time.Time{}, fmt.Errorf("needs a ruling: choose the date this rule starts")
		}
		if c.Effective.Kind == NextRollover && !c.Effective.At.After(c.RecordedAt) {
			return time.Time{}, fmt.Errorf("the next rollover must be after this change was recorded")
		}
		return c.Effective.At, nil
	default:
		return time.Time{}, fmt.Errorf("choose when this rule starts")
	}
}

// Resolve projects one league's ordered ledger. An unseeded league has no competitive defaults.
func Resolve(changes []Change, season int, at time.Time) (RuleSet, error) {
	result := RuleSet{Season: season}
	if season <= 0 || at.IsZero() {
		return RuleSet{}, fmt.Errorf("choose a season and a reading time")
	}
	fields := ruleFields()
	for _, c := range changes {
		if result.LeagueID == "" {
			result.LeagueID = c.LeagueID
		}
		if result.LeagueID != c.LeagueID {
			return RuleSet{}, fmt.Errorf("read rules for one league at a time")
		}
		if c.Season != season {
			continue
		}
		if err := c.validate(fields); err != nil {
			return RuleSet{}, fmt.Errorf("rule change: %w", err)
		}
		start, err := c.starts()
		if err != nil {
			return RuleSet{}, fmt.Errorf("rule start: %w", err)
		}
		if at.Before(start) || at.Before(c.RecordedAt) {
			continue
		}
		for _, f := range fields {
			if f.pathName() == c.Field {
				if err := f.apply(&result, c.New); err != nil {
					return RuleSet{}, fmt.Errorf("rule value: %w", err)
				}
				alignTo(&result, c)
				break
			}
		}
	}
	if err := validateRules(result); err != nil {
		return RuleSet{}, fmt.Errorf("league rules: %w", err)
	}
	return result, nil
}

// SeedChanges records the selected preset explicitly, so other leagues never inherit Legacy policy.
func SeedChanges(r RuleSet, leagueID string, season int, at time.Time) []Change {
	fields := ruleFields()
	result := make([]Change, len(fields))
	for i, f := range fields {
		result[i] = Change{LeagueID: leagueID, Field: f.pathName(), New: f.read(cloneValue(r)),
			Effective: Effective{Kind: Now}, RecordedAt: at, Season: season}
	}
	return result
}
