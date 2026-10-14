package commissioner

import (
	"context"
	"fmt"
	"strings"
	"time"
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

// ResultFlipCheck compares one week's results before and after a proposed change. A game can only turn
// over if one of its sides changed or cannot be told; when either result is incomplete the answer is
// Not known, with the reason.
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
	before, err := weekResults(ctx, league, records, src, directory, published, season, week)
	if err != nil {
		return ResultFlip{}, err
	}
	after, err := weekResults(ctx, league, scoreTrial(raw, records), src, directory, published, season, week)
	if err != nil {
		return ResultFlip{}, err
	}
	if len(before.Games) == 0 {
		return ResultFlip{Status: FlipUnknown, Flips: []FlippedGame{},
			Words: fmt.Sprintf("We can't tell whether this changes a result: %s.", before.Words)}, nil
	}
	return compareWeeks(before, after), nil
}

// sideChange is how one franchise's week score moved.
type sideChange int

const (
	sideUnchanged sideChange = iota
	sideMoved
	sideUnknown
)

func scoreMoved(before, after FranchiseWeekScore) sideChange {
	switch {
	case before.Points == nil || after.Points == nil:
		return sideUnknown
	case *before.Points == *after.Points:
		return sideUnchanged
	}
	return sideMoved
}

func compareWeeks(before, after NativeResults) ResultFlip {
	out := ResultFlip{Flips: []FlippedGame{}}
	scores := func(r NativeResults) map[string]FranchiseWeekScore {
		m := map[string]FranchiseWeekScore{}
		for _, s := range r.Scores {
			m[s.Franchise] = s
		}
		return m
	}
	was, now := scores(before), scores(after)
	unknown := []string{}
	for i, g := range before.Games {
		h, a := g.Home, g.Away
		if scoreMoved(was[h], now[h]) == sideUnchanged && scoreMoved(was[a], now[a]) == sideUnchanged {
			continue
		}
		next := after.Games[i]
		if g.Status == GameNoResult || next.Status == GameNoResult {
			unknown = append(unknown, fmt.Sprintf("%s v %s: %s", g.HomeName, g.AwayName,
				strings.Join(append(append([]string{}, g.Why...), next.Why...), " ")))
		} else if g.Status != next.Status || g.Winner != next.Winner {
			out.Flips = append(out.Flips, FlippedGame{Week: g.Week, Home: g.Home, Away: g.Away,
				Before: g.Words, After: next.Words})
		}
	}
	switch {
	case len(out.Flips) != 0:
		out.Status = FlipYes
		out.Words = flipWords(out.Flips)
	case len(unknown) != 0:
		out.Status = FlipUnknown
		out.Words = "We can't tell whether this changes a result. " + strings.Join(unknown, " ")
	default:
		out.Status = FlipNo
		out.Words = fmt.Sprintf("This change does not turn over any result in week %d.", before.Week)
	}
	return out
}

func flipWords(flips []FlippedGame) string {
	lines := make([]string, len(flips))
	for i, f := range flips {
		lines[i] = fmt.Sprintf("%s becomes %s", f.Before, f.After)
	}
	return "This change turns over a result in week " + fmt.Sprint(flips[0].Week) + ": " + strings.Join(lines, "; ") + "."
}
