package commissioner

import (
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// StatBaseline is an explicit commissioner choice; automatic approvals remain feed-following.
type StatBaseline struct {
	Seq int
	// Digest is the start of the approved version's digest; it is empty until a version is approved.
	Digest string
	Entry  string
}

// statEffects walks the stat commands in the order they were recorded, folding compensation, and says
// what each did. It is the one place the three commands are read back.
type statEffects struct {
	baselines   map[statKey]StatBaseline
	rejected    map[StatItemID]bool
	corrections map[statKey]map[leaguerules.StatID]StatCorrection
	changes     map[statKey][]ScoreChange
}

// StatCorrection is a value the commissioner entered where the feed was silent or wrong.
type StatCorrection struct {
	Value  leaguerules.Decimal
	Source string
	Entry  string
}

func foldStatEffects(records []Record) (statEffects, error) {
	return foldStatEffectsFrom(records, Replay(records).Active)
}

func foldStatEffectsFrom(records []Record, active map[string]bool) (statEffects, error) {
	out := statEffects{baselines: map[statKey]StatBaseline{}, rejected: map[StatItemID]bool{},
		corrections: map[statKey]map[leaguerules.StatID]StatCorrection{}, changes: map[statKey][]ScoreChange{}}
	for _, r := range records {
		command := r.Command.Command
		if !isStatCommand(command) {
			continue
		}
		change, err := DecodeStatChange(r.Command.Args, command)
		if err != nil {
			return statEffects{}, err
		}
		for _, line := range change.Lines() {
			if err := out.record(command, line, r, active[r.Receipt.Entry]); err != nil {
				return statEffects{}, err
			}
		}
	}
	return out, nil
}

func (e *statEffects) record(command string, change RawStatChange, r Record, active bool) error {
	status := "Reversed"
	if active {
		status = "Applied"
		if err := e.apply(command, change, r); err != nil {
			return err
		}
	}
	if command != StatReject {
		key := change.key()
		e.changes[key] = append(e.changes[key], ScoreChange{Entry: r.Receipt.Entry,
			What: statChangeLabel(command, change), Reason: r.Command.Reason, Status: status})
	}
	return nil
}

func (e *statEffects) apply(command string, change RawStatChange, r Record) error {
	key := change.key()
	switch command {
	case StatApprove:
		id, err := ParseStatItemID(change.Item)
		if err != nil {
			return err
		}
		if r.Principal != StatFeedPrincipal {
			e.baselines[key] = StatBaseline{Seq: id.Seq, Digest: id.Digest, Entry: r.Receipt.Entry}
		}
	case StatReject:
		id, err := ParseStatItemID(change.Item)
		if err != nil {
			return err
		}
		e.rejected[id] = true
	default:
		if e.corrections[key] == nil {
			e.corrections[key] = map[leaguerules.StatID]StatCorrection{}
		}
		e.corrections[key][change.Stat] = StatCorrection{Value: *change.Value, Source: change.Source, Entry: r.Receipt.Entry}
	}
	return nil
}

// countedVersion follows the newest feed evidence unless the commissioner chose a revision.
// A version the ledger approved but the held files no longer carry is an error, never a quiet fallback.
func countedVersion(versions []source.LineRevision, base StatBaseline) (*source.LineRevision, error) {
	if len(versions) == 0 {
		if base.Entry != "" || base.Digest != "" {
			return nil, fmt.Errorf("the statistics you approved are no longer in the held files")
		}
		return nil, nil
	}
	if base.Entry == "" && base.Digest == "" {
		return FeedVersion(versions), nil
	}
	if base.Seq < 0 || base.Seq >= len(versions) || !strings.HasPrefix(versions[base.Seq].Digest, base.Digest) {
		return nil, fmt.Errorf("the statistics you approved are no longer in the held files")
	}
	return &versions[base.Seq], nil
}

// withCorrections lays the entered values over a week's line; the original is not changed.
func withCorrections(week source.PlayerWeek, corrections map[leaguerules.StatID]StatCorrection) source.PlayerWeek {
	if len(corrections) == 0 {
		return week
	}
	line := week.Line
	line.Quantities = make(map[leaguerules.StatID]leaguerules.Decimal, len(week.Line.Quantities)+len(corrections))
	for id, quantity := range week.Line.Quantities {
		line.Quantities[id] = quantity
	}
	for id, correction := range corrections {
		line.Quantities[id] = correction.Value
	}
	week.Line = line
	return week
}

// orderChanges lists provenance entries in the order the ledger recorded them.
func orderChanges(position map[string]int, lists ...[]ScoreChange) []ScoreChange {
	var all []ScoreChange
	for _, list := range lists {
		all = append(all, list...)
	}
	slices.SortStableFunc(all, func(a, b ScoreChange) int { return position[a.Entry] - position[b.Entry] })
	if all == nil {
		return []ScoreChange{}
	}
	return all
}

// FeedVersion prefers the newest nflverse revision over any live provider.
func FeedVersion(versions []source.LineRevision) *source.LineRevision {
	var selected *source.LineRevision
	for i := range versions {
		v := &versions[i]
		if v.Held.Authority == source.MFLLive {
			continue
		}
		if selected == nil || v.Held.Authority == source.NFLVerse || selected.Held.Authority != source.NFLVerse {
			selected = v
		}
	}
	return selected
}
