package commissioner

import (
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

func feedReason(version source.LineRevision) string {
	if version.Held.Authority == source.NFLVerse {
		return "feed: nflverse final"
	}
	return "feed: newest live"
}

// FeedStatMoves records the currently selected feed revision through the ordinary approval ledger.
// Commissioner choices pin the revision; per-stat corrections still overlay the selected line.
func FeedStatMoves(league string, records []Record, weeks []StatWeek, season int) ([]RawCommand, error) {
	effects, err := foldStatEffects(records)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, r := range records {
		if r.Principal == StatFeedPrincipal && r.Command.Command == StatApprove {
			seen[r.Command.Args.Stat.Item] = true
		}
	}
	out := []RawCommand{}
	for _, week := range weeks {
		key := statKey{week.Player, season, week.Week}
		if effects.baselines[key].Entry != "" {
			continue
		}
		versions := week.Evidence.Versions()
		selected := FeedVersion(versions)
		if selected == nil {
			continue
		}
		version := *selected
		id := StatItemID{Player: week.Player, Season: season, Week: week.Week, Seq: version.Seq, Digest: version.Digest[:statDigestLen]}.String()
		if seen[id] {
			continue
		}
		out = append(out, RawCommand{League: league, Command: StatApprove, Args: RawArgs{Stat: &RawStatChange{Item: id}},
			Request: "feed." + id, Reason: feedReason(version), Effective: "now", Visibility: "league"})
	}
	return out, nil
}

// CheckFeedStatAppend rechecks automatic approvals under the same append transaction as manual changes.
func CheckFeedStatAppend(raw RawCommand, records []Record, facts StatFacts) error {
	check, err := FeedStatAppendChecker(records)
	if err != nil {
		return err
	}
	return check(raw, facts)
}

// FeedStatAppendChecker shares the immutable pin fold across one locked feed batch.
// The caller must retain the same ledger snapshot until every check and append completes.
func FeedStatAppendChecker(records []Record) (func(RawCommand, StatFacts) error, error) {
	effects, err := foldStatEffects(records)
	if err != nil {
		return nil, err
	}
	started := seeded(records)
	return func(raw RawCommand, facts StatFacts) error {
		return checkFeedStatAppend(raw, facts, effects, started)
	}, nil
}

func checkFeedStatAppend(raw RawCommand, facts StatFacts, effects statEffects, started bool) error {
	if raw.Command != StatApprove {
		return Refused(fmt.Errorf("the stat feed may only count feed revisions"))
	}
	change, err := DecodeStatChange(raw.Args, raw.Command)
	if err != nil {
		return err
	}
	id, err := ParseStatItemID(change.Item)
	if err != nil {
		return err
	}
	if !started || id.Seq >= len(facts.Versions) {
		return Refused(fmt.Errorf("that feed revision is not held"))
	}
	version := facts.Versions[id.Seq]
	if !strings.HasPrefix(version.Digest, id.Digest) || FeedVersion(facts.Versions[:id.Seq+1]).Seq != id.Seq || raw.Reason != feedReason(version) {
		return Refused(fmt.Errorf("that revision is not the feed authority"))
	}
	if effects.baselines[change.key()].Entry != "" {
		return Refused(fmt.Errorf("the commissioner chose the counted revision"))
	}
	return nil
}
