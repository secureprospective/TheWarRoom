package commissioner

import (
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

// statStatus words a decision's standing: what it did while it counts, and that it was undone afterwards.
func statStatus(command string, active bool) string {
	words := map[string][2]string{
		StatApprove: {"Approved", "Approval undone"}, StatReject: {"Rejected", "Rejection undone"},
		StatCorrect: {"Entered", "Entry undone"},
	}[command]
	if active {
		return words[0]
	}
	return words[1]
}

func statKind(command string) string {
	switch command {
	case StatApprove:
		return "approve"
	case StatReject:
		return "reject"
	}
	return "correct"
}

// decisions lists what was decided about players' statistics, newest first, with who decided it and when.
func (b queueBuilder) decisions(weeks []StatWeek) ([]StatDecision, error) {
	versions := map[statKey][]source.LineRevision{}
	for _, week := range weeks {
		versions[statKey{week.Player, b.season, week.Week}] = week.Evidence.Versions()
	}
	active := Replay(b.records).Active
	counted := map[statKey]int{}
	out := []StatDecision{}
	for _, r := range b.records {
		command := r.Command.Command
		if !isStatCommand(command) {
			continue
		}
		change, err := DecodeStatChange(r.Command.Args, command)
		if err != nil {
			return nil, err
		}
		key := change.key()
		if key.season != b.season {
			continue
		}
		decision := StatDecision{Entry: r.Receipt.Entry, Kind: statKind(command), Status: statStatus(command, active[r.Receipt.Entry]),
			Player: key.player, Name: b.name(key.player), Season: key.season, Week: key.week,
			By: personWords(r.Principal), At: r.Receipt.RecordedAt, Reason: r.Command.Reason}
		decision.Words = b.decisionWords(command, change, versions[key], counted[key])
		out = append(out, decision)
		if command == StatApprove && active[r.Receipt.Entry] {
			id, _ := ParseStatItemID(change.Item)
			counted[key] = id.Seq
		}
	}
	slices.Reverse(out)
	return out, nil
}

// decisionWords says what a decision was about; the values come from the held versions it chose between.
func (b queueBuilder) decisionWords(command string, change RawStatChange, versions []source.LineRevision,
	countedSeq int) string {
	key := change.key()
	where := fmt.Sprintf("%s, week %d", b.name(key.player), key.week)
	if command == StatCorrect {
		return fmt.Sprintf("%s: %s set to %s (source: %s)", where, b.statLabel(change.Stat),
			scorePointsWords(*change.Value), change.Source)
	}
	id, _ := ParseStatItemID(change.Item)
	if countedSeq >= len(versions) || id.Seq >= len(versions) {
		return where + ": statistics changed"
	}
	diffs := b.differences(*versions[countedSeq].Held.Week, *versions[id.Seq].Held.Week)
	return fmt.Sprintf("%s: %s", where, changesWords(diffs))
}
