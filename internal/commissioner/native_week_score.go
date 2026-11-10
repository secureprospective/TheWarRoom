package commissioner

import (
	"context"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// Starter is one player who counts for a franchise in a week.
type Starter struct {
	Player string `json:"player"`
	Name   string `json:"name"`
}

// WeekLineup is the lineup that counts for one franchise in one week. Set is false until a lineup exists.
type WeekLineup struct {
	Set      bool
	Starters []Starter
}

// ResultSources supplies what a week's scores are made of. Lineup and PlayerScore read the given ledger, so the
// same week can be scored as it would be after a proposed change.
type ResultSources interface {
	Lineup(ctx context.Context, records []Record, franchise string, week int) (WeekLineup, error)
	PlayerScore(ctx context.Context, records []Record, player string, week int) (NativePlayerScore, error)
}

// StarterScore is one starter's contribution to a franchise's week.
type StarterScore struct {
	Source       string                        `json:"source,omitempty"`
	FieldSources map[leaguerules.StatID]string `json:"fieldSources,omitempty"`
	Player       string                        `json:"player"`
	Name         string                        `json:"name"`
	Position     leaguerules.Position          `json:"position,omitempty"`
	Words        string                        `json:"words"`
	// Line is what the starter recorded this week, by statistic.
	Line   map[leaguerules.StatID]leaguerules.Decimal `json:"line,omitempty"`
	Points *leaguerules.Decimal                       `json:"points,omitempty"`
	// Revision is how the final statistics moved the starter's points off the last live line held.
	Revision *StatRevision `json:"revision,omitempty"`
	// Provisional is what the starter has scored so far from evidence that is not final: the game
	// is still being played, or the final feed has not landed. It is never a result.
	Provisional *leaguerules.Decimal `json:"provisional,omitempty"`
}

// FranchiseWeekScore is a franchise's score for one week: its starters' points plus any franchise
// adjustment. Points is present only when every starter's score is complete.
type FranchiseWeekScore struct {
	RecordedResult string               `json:"recordedResult,omitempty"`
	Franchise      string               `json:"franchise"`
	Name           string               `json:"name"`
	Status         string               `json:"status"`
	Points         *leaguerules.Decimal `json:"points,omitempty"`
	Words          string               `json:"words"`
	Starters       []StarterScore       `json:"starters"`
	// Provisional is the week's running score from evidence that is not final: every starter's
	// points so far, plus the franchise adjustment. Present only while Points is not.
	Provisional *leaguerules.Decimal `json:"provisional,omitempty"`
	Adjustment  leaguerules.Decimal  `json:"adjustment"`
	Changes     []ScoreChange        `json:"changes"`
	// Lineup is the league's judgement of the lineup: a missed one is flagged and never penalised, an invalid
	// one costs what the league's penalty rule says. Zeroed names the starter whose points that took away.
	Lineup nativecheck.LineupVerdict `json:"lineup"`
	Zeroed string                    `json:"zeroed,omitempty"`
}

// WeekScoreMemo is a ResultSources that can share a franchise's week score between readings of the same
// ledger, evidence and rules. score computes it; the memo decides whether a held copy may stand for it.
// Whatever it returns must be the caller's own: no reading may change what another one reads.
type WeekScoreMemo interface {
	MemoWeekScore(ctx context.Context, records []Record, rules leaguerules.LineupRules, seat NativeDirectorySeat,
		season, week int, score func() (FranchiseWeekScore, error)) (FranchiseWeekScore, error)
}

func franchiseWeekScore(ctx context.Context, records []Record, src ResultSources, rules leaguerules.LineupRules,
	seat NativeDirectorySeat, season, week int) (FranchiseWeekScore, error) {
	if memo, ok := src.(WeekScoreMemo); ok {
		out, err := memo.MemoWeekScore(ctx, records, rules, seat, season, week, func() (FranchiseWeekScore, error) {
			return scoreFranchiseWeek(ctx, records, src, rules, seat, season, week)
		})
		if err != nil {
			return out, fmt.Errorf("read memoized week score: %w", err)
		}
		return out, nil
	}
	return scoreFranchiseWeek(ctx, records, src, rules, seat, season, week)
}

func scoreFranchiseWeek(ctx context.Context, records []Record, src ResultSources, rules leaguerules.LineupRules,
	seat NativeDirectorySeat, season, week int) (FranchiseWeekScore, error) {
	if recorded, ok := recordedWeekScore(records, seat, season, week); ok {
		return recorded, nil
	}
	out := FranchiseWeekScore{Franchise: seat.Franchise, Name: seat.Name, Status: "Incomplete",
		Starters: []StarterScore{}, Changes: []ScoreChange{}}
	lineup, err := src.Lineup(ctx, records, seat.Franchise, week)
	if err != nil {
		return out, fmt.Errorf("read %s lineup: %w", seat.Name, err)
	}
	if !lineup.Set || len(lineup.Starters) == 0 {
		out.noLineup(records, rules, seat, season, week)
		return out, nil
	}
	changes, err := FranchiseScoreChanges(records, seat.Franchise, season, week)
	if err != nil {
		return out, err
	}
	out.Adjustment, out.Changes = changes.Delta, changes.Changes
	total, running, err := out.addStarters(ctx, records, src, lineup, week)
	if err != nil {
		return out, err
	}
	if len(running.missing) != 0 {
		return out, out.incomplete(seat, running, changes.Delta)
	}
	if total, err = out.judge(rules, total); err != nil {
		return out, err
	}
	if total, err = addScoreDelta(total, changes.Delta); err != nil {
		return out, err
	}
	out.Status, out.Points = "Complete", &total
	out.Words = fmt.Sprintf("%s scored %s points", seat.Name, scorePointsWords(total))
	if out.Zeroed != "" {
		out.Words += fmt.Sprintf("; the lineup is invalid, so %s counts for nothing", out.zeroedName())
	}
	return out, nil
}

// noLineup is the week of a franchise that set no lineup. Nothing is taken from it: a missed
// lineup is flagged only when the league flags it, and not while the league's lineup sync has not
// read the week yet, because the league has not asked for that lineup yet.
func (out *FranchiseWeekScore) noLineup(records []Record, rules leaguerules.LineupRules,
	seat NativeDirectorySeat, season, week int) {
	out.Lineup = nativecheck.JudgeLineup(rules, false, nil)
	out.Words = fmt.Sprintf("%s has no lineup set for week %d", seat.Name, week)
	if out.Lineup.Flagged && LineupSyncPending(records, season, week) {
		out.Lineup.Flagged = false
		out.Lineup.Words = "No lineup was set. The league reads lineups from a sync source that has not " +
			"reached this week yet, so nothing is flagged to the commissioner"
		out.Words += "; the league has not read this week's lineups yet, so nothing is flagged."
		return
	}
	if out.Lineup.Flagged {
		out.Words += "; it is flagged to the commissioner as a missed lineup, and nothing is zeroed"
	}
	out.Words += "."
}

// incomplete is a week that has no result yet. It keeps a running score when any starter has
// points so far, so a game being played can be followed without ever being called a result.
func (out *FranchiseWeekScore) incomplete(seat NativeDirectorySeat, running runningScore,
	delta leaguerules.Decimal) error {
	out.Lineup = nativecheck.LineupVerdict{Status: nativecheck.LineupUnjudged, Problems: []nativecheck.LineupProblem{},
		Words: "The lineup is judged once every starter's score is complete"}
	out.Words = fmt.Sprintf("%s's score is incomplete: %s", seat.Name, strings.Join(running.missing, "; "))
	if !running.counted {
		return nil
	}
	provisional, err := addScoreDelta(running.points, delta)
	if err != nil {
		return err
	}
	out.Provisional = &provisional
	out.Words = fmt.Sprintf("%s has %s points so far, from evidence that is not final: %s",
		seat.Name, scorePointsWords(provisional), strings.Join(running.missing, "; "))
	return nil
}

// provisional is the side's running score while the week has no result: the points it has from
// final evidence plus the points it has so far from evidence that is not.
func (out FranchiseWeekScore) provisional() *leaguerules.Decimal {
	if out.Points != nil {
		return out.Points
	}
	return out.Provisional
}

func (out FranchiseWeekScore) zeroedName() string {
	for _, s := range out.Starters {
		if s.Player == out.Zeroed {
			return s.Name
		}
	}
	return "a starter"
}

// judge holds complete starters to the league's lineup limits. An invalid lineup loses what the penalty
// rule takes, through the same function the native lineup check uses; any other lineup counts in full.
func (out *FranchiseWeekScore) judge(rules leaguerules.LineupRules, total leaguerules.Decimal) (leaguerules.Decimal, error) {
	starters := make([]nativecheck.Starter, len(out.Starters))
	for i, s := range out.Starters {
		if s.Points == nil || s.Position == "" {
			out.Lineup = nativecheck.LineupVerdict{Status: nativecheck.LineupUnjudged, Problems: []nativecheck.LineupProblem{},
				Words: "Can't tell: where a starter plays is not held"}
			return total, nil
		}
		starters[i] = nativecheck.Starter{ID: s.Player, Position: s.Position, Points: *s.Points}
	}
	out.Lineup = nativecheck.JudgeLineup(rules, true, starters)
	if out.Lineup.Status != nativecheck.LineupInvalid {
		return total, nil
	}
	penalised := nativecheck.ApplyPenalty(rules.InvalidPenalty, starters)
	if penalised.Zeroed == "" {
		return total, nil
	}
	out.Zeroed = penalised.Zeroed
	var kept leaguerules.Decimal
	for i, s := range penalised.Starters {
		points := s.Points
		out.Starters[i].Points = &points
		if s.ID == penalised.Zeroed {
			out.Starters[i].Words += "; zeroed because the lineup is invalid"
		}
		var err error
		if kept, err = addScoreDelta(kept, points); err != nil {
			return 0, err
		}
	}
	return kept, nil
}

// runningScore is what a week has so far while it is not final: the points counted from every
// starter, whether that starter's evidence is final or still live, and why the week is incomplete.
type runningScore struct {
	points  leaguerules.Decimal
	counted bool
	missing []string
}

// addStarters scores each starter once. It sums the starters that are complete into the week's
// final total, and every starter's points so far — final or provisional — into the running score.
func (out *FranchiseWeekScore) addStarters(ctx context.Context, records []Record, src ResultSources, lineup WeekLineup,
	week int) (leaguerules.Decimal, runningScore, error) {
	var total leaguerules.Decimal
	running := runningScore{missing: []string{}}
	seen := map[string]bool{}
	for _, starter := range lineup.Starters {
		if seen[starter.Player] {
			return 0, running, fmt.Errorf("a lineup lists player %s twice", starter.Player)
		}
		seen[starter.Player] = true
		name := starter.Name
		if name == "" {
			name = "player " + starter.Player
		}
		score, err := src.PlayerScore(ctx, records, starter.Player, week)
		if err != nil {
			return 0, running, fmt.Errorf("score %s: %w", name, err)
		}
		item := StarterScore{Player: starter.Player, Name: name, Words: score.Words, Position: score.Position, Source: score.Source, FieldSources: score.FieldSources,
			Line: score.Line, Revision: score.Revision}
		switch {
		case score.Status == "Complete" && score.Points != nil:
			item.Points = score.Points
			if total, err = addScoreDelta(total, *score.Points); err != nil {
				return 0, running, err
			}
			if running.points, err = addScoreDelta(running.points, *score.Points); err != nil {
				return 0, running, err
			}
			running.counted = true
		default:
			running.missing = append(running.missing, name+": "+score.Words)
			if score.PartialPoints == nil {
				break
			}
			item.Provisional = score.PartialPoints
			if running.points, err = addScoreDelta(running.points, *score.PartialPoints); err != nil {
				return 0, running, err
			}
			running.counted = true
		}
		out.Starters = append(out.Starters, item)
	}
	return total, running, nil
}
