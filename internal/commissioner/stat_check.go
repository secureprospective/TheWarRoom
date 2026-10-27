package commissioner

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// StatFacts is what the held NFL files say about the one player's week a stat command is about: its
// versions, oldest first. Native checks read the ledger; the files are all they need besides.
type StatFacts struct{ Versions []source.LineRevision }

// CheckStatAppend refuses a stat decision the league cannot hold: one not waiting any more, one already
// decided, or a correction that changes nothing or names a statistic the league does not score.
func CheckStatAppend(raw RawCommand, records []Record, facts StatFacts, now time.Time) error {
	if !isStatCommand(raw.Command) {
		return nil
	}
	change, err := DecodeStatChange(raw.Args, raw.Command)
	if err != nil {
		return err
	}
	if !seeded(records) {
		return Refused(errors.New("record the starting snapshot first"))
	}
	effects, err := foldStatEffects(records)
	if err != nil {
		return err
	}
	if raw.Command == StatCorrect {
		return checkStatCorrection(change, records, effects, facts, now)
	}
	id, err := ParseStatItemID(change.Item)
	if err != nil {
		return err
	}
	return waitingItem(id, effects, facts)
}

// waitingItem is nil while an item is waiting for a decision, and otherwise a refusal that says why.
func waitingItem(id StatItemID, effects statEffects, facts StatFacts) error {
	versions := facts.Versions
	key := statKey{id.Player, id.Season, id.Week}
	if len(versions) == 0 || id.Seq >= len(versions) || !strings.HasPrefix(versions[id.Seq].Digest, id.Digest) {
		return Refused(errors.New("that change is not in the held NFL statistics"))
	}
	newest := versions[len(versions)-1]
	if id.Seq != newest.Seq {
		return Refused(errors.New("the NFL has changed this player's week again since you looked; read the queue again"))
	}
	counted, err := countedVersion(versions, effects.baselines[key])
	if err != nil {
		return err
	}
	switch {
	case effects.rejected[id]:
		return Refused(errors.New("this change was already turned down; undo that first to reconsider it"))
	case counted.Digest == newest.Digest:
		return Refused(errors.New("the league already counts these statistics"))
	}
	return nil
}

func checkStatCorrection(change RawStatChange, records []Record, effects statEffects, facts StatFacts,
	now time.Time) error {
	rules, err := RulesInForce(records, change.Season, ReadingInstant(records, now))
	if err != nil {
		return Refused(err)
	}
	if !scoresStat(rules.Scoring, change.Player, change.Stat) {
		return Refused(errors.New("the league's scoring has no such statistic"))
	}
	key := change.key()
	counted, err := countedVersion(facts.Versions, effects.baselines[key])
	if err != nil {
		return err
	}
	var week source.PlayerWeek
	if counted != nil {
		week = *counted.Held.Week
	}
	standing, recorded := withCorrections(week, effects.corrections[key]).Line.Quantities[change.Stat]
	if recorded && standing == *change.Value {
		return Refused(fmt.Errorf("the league already counts %s for this statistic", scorePointsWords(standing)))
	}
	return nil
}

// scoresStat says whether a correction names a statistic the league scores for this player: a catalogue
// row, or for an O-Line card one of the line unit's components.
func scoresStat(rules leaguerules.ScoringRules, player string, stat leaguerules.StatID) bool {
	if _, card := nativecheck.LineCardTeam(player); card {
		_, weighted := rules.LineUnit.Value.Weights[stat]
		return rules.LineUnit.State == leaguerules.IsKnown && rules.LineUnit.Value.Used && weighted
	}
	if rules.Catalogue.State != leaguerules.IsKnown {
		return true
	}
	_, scored := rules.Catalogue.Value[stat]
	return scored
}
