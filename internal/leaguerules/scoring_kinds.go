package leaguerules

import "fmt"

// The rule kinds below extend the stat catalogue. Each is its own commissioner setting so a league
// that never uses one records it as empty; Legacy records every one as empty.

// PositionExtra adds a rate on top of a stat's base rate for the listed positions. Overlapping rows
// add up and each row's award is rounded on its own, as MFL does.
type PositionExtra struct {
	Positions []Position
	Points    Decimal
}

// StatRange is MFL's inclusive scoring range: a week whose value falls outside it scores nothing
// for that stat. Bounds are in the stat's own ten-thousandths.
type StatRange struct {
	Min Decimal
	Max Decimal
}

// DistanceScale prices each recorded play by its distance: plays shorter than From score nothing;
// the rest score Base + Above × max(0, distance − Pivot) + Below × max(0, Pivot − distance),
// capped at Cap when Capped. The week's sum is the stat's value, scored at its catalogue rate.
type DistanceScale struct {
	From   Decimal
	Base   Decimal
	Pivot  Decimal
	Above  Decimal
	Below  Decimal
	Capped bool
	Cap    Decimal
}

// LineUnit scores a team-game for an NFL team's offensive line. A team that did not play scores 0.
type LineUnit struct {
	Used    bool
	Weights map[StatID]Decimal
	Labels  map[StatID]string
	// NoSack is awarded when the team played and the recorded sacks are zero.
	NoSack  Decimal
	Sacks   StatID
	Clamped bool
	Clamp   Decimal
}

func emptyScoringKinds(r *ScoringRules, citation string) {
	r.Extras = cited(map[StatID][]PositionExtra{}, citation)
	r.Ranges = cited(map[StatID]StatRange{}, citation)
	r.Scales = cited(map[StatID]DistanceScale{}, citation)
	r.Totals = cited(map[StatID][]StatID{}, citation)
	r.LineUnit = cited(LineUnit{Weights: map[StatID]Decimal{}, Labels: map[StatID]string{}}, citation)
}

func scoringKindFields() []ruleField {
	return []ruleField{
		field[map[StatID][]PositionExtra]{path: "scoring.extras", label: "Extra points by position", access: func(r *RuleSet) *Value[map[StatID][]PositionExtra] { return &r.Scoring.Extras }},
		field[map[StatID]StatRange]{path: "scoring.ranges", label: "Scoring ranges", access: func(r *RuleSet) *Value[map[StatID]StatRange] { return &r.Scoring.Ranges }},
		field[map[StatID]DistanceScale]{path: "scoring.scales", label: "Points by play distance", access: func(r *RuleSet) *Value[map[StatID]DistanceScale] { return &r.Scoring.Scales }},
		field[map[StatID][]StatID]{path: "scoring.totals", label: "Stats counted as a total of others", access: func(r *RuleSet) *Value[map[StatID][]StatID] { return &r.Scoring.Totals }},
		field[LineUnit]{path: "scoring.lineUnit", label: "Team offensive line scoring", access: func(r *RuleSet) *Value[LineUnit] { return &r.Scoring.LineUnit }},
	}
}

func validateScoringKinds(r ScoringRules) error {
	if r.Ranges.State == IsKnown {
		for id, bounds := range r.Ranges.Value {
			if bounds.Min > bounds.Max {
				return fmt.Errorf("%s: a scoring range cannot start above its end", id)
			}
		}
	}
	if r.Scales.State == IsKnown {
		for id, scale := range r.Scales.Value {
			if !scale.Capped && scale.Cap != 0 {
				return fmt.Errorf("%s: an uncapped distance scale carries no cap", id)
			}
		}
	}
	if r.LineUnit.State == IsKnown {
		unit := r.LineUnit.Value
		if unit.Clamp < 0 || (!unit.Clamped && unit.Clamp != 0) {
			return fmt.Errorf("the offensive line clamp must be zero or more and only set when clamped")
		}
		if _, weighted := unit.Weights[unit.Sacks]; unit.NoSack != 0 && !weighted {
			return fmt.Errorf("the no-sack bonus needs a weighted sack stat")
		}
	}
	return nil
}
