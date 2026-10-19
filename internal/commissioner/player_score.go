package commissioner

import (
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

type NativeScoreItem struct {
	Stat   leaguerules.StatID   `json:"stat"`
	Words  string               `json:"words"`
	Points *leaguerules.Decimal `json:"points,omitempty"`
}

// Points is present only for complete evidence and settled rules; incomplete totals are separate.
type NativePlayerScore struct {
	League        string               `json:"league"`
	Player        string               `json:"player"`
	Season        int                  `json:"season"`
	Week          int                  `json:"week"`
	Status        string               `json:"status"`
	Words         string               `json:"words"`
	Points        *leaguerules.Decimal `json:"points,omitempty"`
	PartialPoints *leaguerules.Decimal `json:"partialPoints,omitempty"`
	Position      leaguerules.Position `json:"position,omitempty"`
	PositionWhy   string               `json:"positionWhy,omitempty"`
	Missing       []string             `json:"missing"`
	Items         []NativeScoreItem    `json:"items"`
	Changes       []ScoreChange        `json:"changes"`
	SHA256        string               `json:"sha256,omitempty"`
	FetchedAt     *time.Time           `json:"fetchedAt,omitempty"`
}

func NativePlayerScoreReading(league string, scope source.RawNativeScope, records []Record,
	evidence source.NativeEvidence, now time.Time) (NativePlayerScore, error) {
	return NativePlayerScoreWith(league, scope, records, evidence, now, RulesInForce)
}

// NativePlayerScoreWith reads a player's score with the given resolver, so a reading that scores many players
// can resolve the season's rules once. The resolver is called only when the player's rules are needed.
func NativePlayerScoreWith(league string, scope source.RawNativeScope, records []Record,
	evidence source.NativeEvidence, now time.Time, resolve RulesFunc) (NativePlayerScore, error) {
	out := NativePlayerScore{League: league, Player: scope.Player, Season: scope.Season, Week: scope.Week,
		Status: "Unavailable", Missing: []string{}, Items: []NativeScoreItem{}, Changes: []ScoreChange{}}
	if err := scope.Validate(); err != nil {
		return out, fmt.Errorf("read player score: %w", err)
	}
	if len(records) == 0 {
		return out, fmt.Errorf("this app league has not been created")
	}
	if !seeded(records) {
		out.Words = "Record the starting snapshot first"
		return out, nil
	}
	rules, err := resolve(records, scope.Season, ReadingInstant(records, now))
	if err != nil {
		return out, err
	}
	changes, err := PlayerScoreChanges(records, scope.Player, scope.Season, scope.Week)
	if err != nil {
		return out, err
	}
	effects, err := foldStatEffects(records)
	if err != nil {
		return out, err
	}
	key := statKey{scope.Player, scope.Season, scope.Week}
	out.Changes = orderChanges(records, changes.Changes, effects.changes[key])
	out.Missing = append(out.Missing, evidence.Missing...)
	counted, err := countedVersion(evidence.Versions(), effects.baselines[key])
	if err != nil {
		return out, err
	}
	corrections := effects.corrections[key]
	var week source.PlayerWeek
	switch {
	case counted != nil:
		week = *counted.Held.Week
		out.SHA256, out.FetchedAt = counted.Held.SHA256, &counted.Held.FetchedAt
	case len(corrections) != 0:
		week = source.PlayerWeek{Season: scope.Season, Week: scope.Week}
		out.Missing = append(out.Missing, "Only the statistics the commissioner entered are counted; the NFL feed has none for this week")
	default:
		return out.withoutStatistics(evidence), nil
	}
	week = withCorrections(week, corrections)
	out.Position, out.PositionWhy = changes.PositionUsed(scoreFeedPosition(week.Position))
	if out.Position == "" {
		out.Status, out.Words = "Needs a ruling", "Choose this player's position for league scoring"
		return out, nil
	}
	return out.count(rules.Scoring, week.Line, changes.Delta)
}

// withoutStatistics says why no statistics are counted for the week; no zero line is made up.
func (out NativePlayerScore) withoutStatistics(evidence source.NativeEvidence) NativePlayerScore {
	switch {
	case evidence.GSIS == "":
		out.Words = "This player has no NFL statistics identity"
	case evidence.Held == nil:
		out.Words = "No NFL statistics are held for this season and week"
	default:
		out.SHA256, out.FetchedAt = evidence.Held.SHA256, &evidence.Held.FetchedAt
		out.Words = "This player is not reported this week in the newest held statistics"
	}
	return out
}

func (out NativePlayerScore) count(rules leaguerules.ScoringRules, line fantasyscore.StatLine,
	delta leaguerules.Decimal) (NativePlayerScore, error) {
	points, items, err := fantasyscore.Score(rules, out.Position, line)
	if err != nil {
		return out, fmt.Errorf("count held player statistics: %w", err)
	}
	for _, item := range items {
		out.Items = append(out.Items, nativeScoreItem(item))
	}
	if points.State != leaguerules.IsKnown {
		out.Status, out.Words = "Needs a ruling", "Needs a ruling: "+points.Citation
		return out, nil
	}
	total, err := addScoreDelta(points.Value, delta)
	if err != nil {
		return out, err
	}
	if len(out.Missing) != 0 {
		out.Status, out.PartialPoints = "Incomplete", &total
		out.Words = scorePointsWords(total) + " points counted so far; incomplete: " + strings.Join(out.Missing, "; ")
		return out, nil
	}
	out.Status, out.Points = "Complete", &total
	out.Words = scorePointsWords(total) + " points"
	return out, nil
}

func nativeScoreItem(item fantasyscore.LineItem) NativeScoreItem {
	out := NativeScoreItem{Stat: item.Stat,
		Words: item.Label + ": " + scorePointsWords(item.Quantity) + " recorded"}
	switch {
	case item.Unused:
		out.Words += "; not used for league scoring"
	case item.Points.State != leaguerules.IsKnown:
		out.Words += "; needs a ruling: " + item.Points.Citation
	default:
		points := item.Points.Value
		out.Points = &points
		out.Words += "; " + scorePointsWords(points) + " points"
	}
	return out
}

func scoreFeedPosition(position string) leaguerules.Position {
	switch position {
	case "K":
		return leaguerules.PK
	case "NT":
		return leaguerules.DT
	case "MLB", "ILB", "OLB":
		return leaguerules.LB
	case "FS", "SS":
		return leaguerules.S
	case "QB", "RB", "WR", "TE", "PK", "DT", "DE", "LB", "CB", "S":
		return leaguerules.Position(position)
	default:
		return ""
	}
}
