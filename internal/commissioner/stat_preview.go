package commissioner

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// StatChangePreview says what a stat command would do before it is sent: the change, what it is worth in
// points and whether it turns over a result. A flip is a warning, not a block. It records nothing.
func StatChangePreview(ctx context.Context, league string, records []Record, src ResultSources, feed StatFeed,
	week StatWeek, raw RawCommand, now time.Time) (ScorePreview, error) {
	change, err := DecodeStatChange(raw.Args, raw.Command)
	if err != nil {
		return ScorePreview{}, err
	}
	if err := CheckStatAppend(raw, records, StatFacts{Versions: week.Evidence.Versions(), OLine: week.Evidence.OLine}, now); err != nil {
		return ScorePreview{}, err
	}
	rules, err := RulesInForce(records, change.key().season, ReadingInstant(records, now))
	if err != nil {
		return ScorePreview{}, err
	}
	effects, err := foldStatEffects(records)
	if err != nil {
		return ScorePreview{}, err
	}
	b := queueBuilder{league: league, season: change.key().season, records: records, feed: feed, src: src,
		rules: rules, effects: effects, now: now}
	var out ScorePreview
	switch raw.Command {
	case StatApprove:
		out, err = b.approvalPreview(ctx, week)
	case StatReject:
		out, err = b.rejectionPreview(week)
	default:
		out, err = b.correctionPreview(ctx, change, week, raw)
	}
	if err != nil {
		return ScorePreview{}, err
	}
	out.Command = raw.Command
	if out.Flip.Status != FlipNo {
		out.Warnings = append(out.Warnings, out.Flip.Words)
	}
	return out, nil
}

func (b queueBuilder) approvalPreview(ctx context.Context, week StatWeek) (ScorePreview, error) {
	item, waiting, err := b.candidate(week)
	if err != nil || !waiting {
		return ScorePreview{}, fmt.Errorf("nothing is waiting for this player's week")
	}
	if err := b.price(ctx, &item, week); err != nil {
		return ScorePreview{}, err
	}
	return ScorePreview{Words: "Count the NFL's update: " + item.Words, Warnings: []string{}, Flip: item.flip}, nil
}

// rejectionPreview moves no score, so it cannot turn over a result.
func (b queueBuilder) rejectionPreview(week StatWeek) (ScorePreview, error) {
	item, waiting, err := b.candidate(week)
	if err != nil || !waiting {
		return ScorePreview{}, fmt.Errorf("nothing is waiting for this player's week")
	}
	flip := ResultFlip{Status: FlipNo, Flips: []FlippedGame{},
		Words: "Turning this change down moves no score, so no result changes."}
	return ScorePreview{Warnings: []string{}, Flip: flip, Words: fmt.Sprintf(
		"Keep the league's count and turn down the NFL's update: %s, week %d: %s; the score stays as it is",
		item.Name, item.Week, changesWords(item.Changes))}, nil
}

func (b queueBuilder) correctionPreview(ctx context.Context, change RawStatChange, week StatWeek,
	raw RawCommand) (ScorePreview, error) {
	key := change.key()
	scope := source.RawNativeScope{Player: key.player, Season: key.season, Week: key.week}
	before, err := NativePlayerScoreReading(b.league, scope, b.records, week.Evidence, b.now)
	if err != nil {
		return ScorePreview{}, err
	}
	trial := scoreTrial(raw, b.records)
	after, err := NativePlayerScoreReading(b.league, scope, trial, week.Evidence, b.now)
	if err != nil {
		return ScorePreview{}, err
	}
	flip, err := ResultFlipCheck(ctx, b.league, b.records, b.src, raw, key.season, key.week, b.now)
	if err != nil {
		return ScorePreview{}, err
	}
	was := b.enteredQuantity(week, key, change.Stat)
	changes := []StatChange{{Stat: string(change.Stat), Label: b.statLabel(change.Stat),
		Before: was, After: scorePointsWords(*change.Value)}}
	return ScorePreview{Warnings: []string{}, Flip: flip, Words: fmt.Sprintf(
		"Enter this value: %s, week %d: %s, %s, %s (source: %s)", b.name(key.player), key.week, changesWords(changes),
		signedPointsWords(pointsChange(before, after)), flipSummary(flip.Status), change.Source)}, nil
}

// enteredQuantity is the value the league counts for a stat now, in words.
func (b queueBuilder) enteredQuantity(week StatWeek, key statKey, stat leaguerules.StatID) string {
	facts := StatFacts{Versions: week.Evidence.Versions(), OLine: week.Evidence.OLine}
	quantity, recorded, err := standingQuantity(key, stat, facts, b.effects)
	if err != nil {
		return quantityWords(0, false)
	}
	return quantityWords(quantity, recorded)
}
