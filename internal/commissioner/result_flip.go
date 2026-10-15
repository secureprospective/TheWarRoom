package commissioner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// Flip verdicts. Not known is the honest answer while a result is incomplete, never a quiet "no".
const (
	FlipNo      = "No result changes"
	FlipYes     = "A result changes"
	FlipUnknown = "Not known"
)

// FlippedGame names a game whose result a proposed change would turn over.
type FlippedGame struct {
	Week   int    `json:"week"`
	Home   string `json:"home"`
	Away   string `json:"away"`
	Before string `json:"before"`
	After  string `json:"after"`
	// Decided is true when the game had no result and the change gives it one.
	Decided bool `json:"decided,omitempty"`
}

// ResultFlip says whether a proposed score change turns over any game's result in its week.
type ResultFlip struct {
	Status string        `json:"status"`
	Words  string        `json:"words"`
	Flips  []FlippedGame `json:"flips"`
}

// ScorePreview is what a score change would do, ahead of recording it. A flip is a warning, not a block.
type ScorePreview struct {
	Command  string     `json:"command"`
	Words    string     `json:"words"`
	Warnings []string   `json:"warnings"`
	Flip     ResultFlip `json:"flip"`
	// Lineup is the league's judgement of a corrected lineup; absent for a score change.
	Lineup *nativecheck.LineupVerdict `json:"lineup,omitempty"`
}

// ScoreChangePreview scores the change's week as it stands and as it would stand after the change.
func ScoreChangePreview(ctx context.Context, league string, records []Record, src ResultSources, raw RawCommand,
	now time.Time) (ScorePreview, error) {
	if raw.Command != ScorePlayer && raw.Command != ScoreFranchise {
		return ScorePreview{}, fmt.Errorf("only a score adjustment can be previewed for a changed result")
	}
	change, err := DecodeScoreChange(raw.Args, raw.Command)
	if err != nil {
		return ScorePreview{}, err
	}
	if err := checkScoreAppend(raw, records); err != nil {
		return ScorePreview{}, err
	}
	flip, err := ResultFlipCheck(ctx, league, records, src, raw, change.Season, change.Week, now)
	if err != nil {
		return ScorePreview{}, err
	}
	out := ScorePreview{Command: raw.Command, Words: scoreChangeLabel(raw.Command, change),
		Warnings: []string{}, Flip: flip}
	if flip.Status != FlipNo {
		out.Warnings = append(out.Warnings, flip.Words)
	}
	return out, nil
}

// ResultFlipCheck compares one week's results before and after a proposed change. Only the games of the
// franchises the change touches can turn over; when one of those results is incomplete the answer is Not
// known, with the reason.
func ResultFlipCheck(ctx context.Context, league string, records []Record, src ResultSources, raw RawCommand, season, week int,
	now time.Time) (ResultFlip, error) {
	directory, err := NativeDirectoryReading(league, records, now)
	if err != nil {
		return ResultFlip{}, err
	}
	published, err := resultSchedule(records, season, now)
	if err != nil {
		return ResultFlip{}, err
	}
	rules, err := RulesInForce(records, season, ReadingInstant(records, now))
	if err != nil {
		return ResultFlip{}, err
	}
	before, err := weekResults(ctx, league, records, src, rules.Lineup, directory, published, season, week)
	if err != nil {
		return ResultFlip{}, err
	}
	after, err := weekResults(ctx, league, scoreTrial(raw, records), src, rules.Lineup, directory, published, season, week)
	if err != nil {
		return ResultFlip{}, err
	}
	if len(before.Games) == 0 {
		return ResultFlip{Status: FlipUnknown, Flips: []FlippedGame{},
			Words: fmt.Sprintf("We can't tell whether this changes a result: %s.", before.Words)}, nil
	}
	return compareWeeks(before, after, franchisesTouched(raw, before, after)), nil
}

// franchisesTouched names the franchises a proposed change can move: the one it names, or for a player's
// score the one that starts him in either reading of the week.
func franchisesTouched(raw RawCommand, readings ...NativeResults) map[string]bool {
	touched := map[string]bool{}
	switch {
	case raw.Command == LineupCorrect && raw.Args.Lineup != nil:
		touched[raw.Args.Lineup.Franchise] = true
	case raw.Command == ScoreFranchise && raw.Args.Score != nil:
		touched[raw.Args.Score.Franchise] = true
	case raw.Command == ScorePlayer && raw.Args.Score != nil:
		for _, reading := range readings {
			for _, score := range reading.Scores {
				for _, starter := range score.Starters {
					if starter.Player == raw.Args.Score.Player {
						touched[score.Franchise] = true
					}
				}
			}
		}
	}
	return touched
}

func compareWeeks(before, after NativeResults, touched map[string]bool) ResultFlip {
	out := ResultFlip{Flips: []FlippedGame{}}
	scores := map[string]FranchiseWeekScore{}
	for _, s := range after.Scores {
		scores[s.Franchise] = s
	}
	unknown := []string{}
	for i, g := range before.Games {
		if !touched[g.Home] && !touched[g.Away] {
			continue
		}
		next := after.Games[i]
		switch {
		case g.Status != GameNoResult && next.Status != GameNoResult:
			if g.Status != next.Status || g.Winner != next.Winner {
				out.Flips = append(out.Flips, FlippedGame{Week: g.Week, Home: g.Home, Away: g.Away, Before: g.Words, After: next.Words})
			}
		case g.Status == GameNoResult && next.Status != GameNoResult:
			out.Flips = append(out.Flips, FlippedGame{Week: g.Week, Home: g.Home, Away: g.Away, Before: g.Words,
				After: next.Words, Decided: true})
		default:
			unknown = append(unknown, fmt.Sprintf("%s v %s: %s", next.HomeName, next.AwayName, incompleteReason(next, scores)))
		}
	}
	switch {
	case len(out.Flips) != 0:
		out.Status = FlipYes
		out.Words = flipWords(out.Flips)
	case len(unknown) != 0:
		out.Status = FlipUnknown
		out.Words = "We can't tell yet whether this changes " + strings.Join(unknown, "; ") + "."
	default:
		out.Status = FlipNo
		out.Words = fmt.Sprintf("This change does not turn over any result in week %d.", before.Week)
	}
	return out
}

// incompleteReason says in one clause why a game cannot be called, naming the side or sides that hold it up.
func incompleteReason(g NativeGame, scores map[string]FranchiseWeekScore) string {
	home, away := scores[g.Home], scores[g.Away]
	switch {
	case home.Points == nil && away.Points == nil:
		homeNone, awayNone := len(home.Starters) == 0, len(away.Starters) == 0
		switch {
		case homeNone && awayNone:
			return "neither team has a lineup set"
		case !homeNone && !awayNone:
			return "neither score is complete"
		}
		return waitingOn(home) + " and " + waitingOn(away)
	case home.Points == nil:
		return waitingOn(home)
	}
	return waitingOn(away)
}

func waitingOn(s FranchiseWeekScore) string {
	if len(s.Starters) == 0 {
		return s.Name + " has no lineup set"
	}
	return s.Name + "'s score is not complete"
}

func flipWords(flips []FlippedGame) string {
	lines := make([]string, len(flips))
	for i, f := range flips {
		lines[i] = fmt.Sprintf("%s becomes %s", f.Before, f.After)
		if f.Decided {
			lines[i] = "a game with no result yet becomes: " + f.After
		}
	}
	return "This change turns over a result in week " + fmt.Sprint(flips[0].Week) + ": " + strings.Join(lines, "; ") + "."
}

// LineupCorrectionPreview reads a CM-029 correction in the form it would be sent: what it would count, the
// league's judgement of that lineup, and a warning when it would turn over a game's result. It records nothing.
func LineupCorrectionPreview(ctx context.Context, league string, records []Record, src ResultSources, raw RawCommand,
	env LineupFacts, now time.Time) (ScorePreview, error) {
	if raw.Command != LineupCorrect {
		return ScorePreview{}, fmt.Errorf("only a lineup correction can be previewed here")
	}
	change, err := DecodeLineupChange(raw.Args, raw.Command)
	if err != nil {
		return ScorePreview{}, err
	}
	if err := CheckLineupAppend(raw, records, now, env); err != nil {
		return ScorePreview{}, err
	}
	flip, err := ResultFlipCheck(ctx, league, records, src, raw, change.Season, change.Week, now)
	if err != nil {
		return ScorePreview{}, err
	}
	rules, err := RulesInForce(records, change.Season, ReadingInstant(records, now))
	if err != nil {
		return ScorePreview{}, err
	}
	verdict, _ := judgeStarters(rules.Lineup, true, change.Starters, env.Facts)
	name := change.Franchise
	if directory, err := NativeDirectoryReading(league, records, now); err == nil {
		name = directory.seats().of(change.Franchise).Name
	}
	out := ScorePreview{Command: raw.Command, Warnings: []string{}, Flip: flip, Lineup: &verdict,
		Words: fmt.Sprintf("Count these %d starters as %s's week %d lineup, lock or no lock", len(change.Starters), name, change.Week)}
	if flip.Status != FlipNo {
		out.Warnings = append(out.Warnings, flip.Words)
	}
	switch verdict.Status {
	case nativecheck.LineupInvalid:
		out.Warnings = append(out.Warnings, verdict.Words+". "+verdict.Penalty+" when the week is scored.")
	case nativecheck.LineupUnjudged:
		out.Warnings = append(out.Warnings, verdict.Words+".")
	case nativecheck.LineupValid, nativecheck.LineupMissed:
	}
	return out, nil
}
