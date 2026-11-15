package commissioner

import "time"

// scoreIndex holds a ledger's position overrides and points adjustments by player and franchise, in ledger
// order, so scoring one starter reads only that starter's changes. err is the first change that does not
// decode, which every reading of that ledger reports.
type scoreIndex struct {
	by  map[scoreOwner][]indexedScore
	err error
}

type scoreOwner struct{ player, franchise string }

type indexedScore struct {
	command string
	change  RawScoreChange
	entry   string
	reason  string
}

func newScoreIndex(records []Record) *scoreIndex {
	ix := &scoreIndex{by: map[scoreOwner][]indexedScore{}}
	for _, r := range records {
		switch r.Command.Command {
		case PlayerPosition, PlayerPositionClear, ScorePlayer, ScoreFranchise:
		default:
			continue
		}
		change, err := DecodeScoreChange(r.Command.Args, r.Command.Command)
		if err != nil {
			ix.err = err
			return ix
		}
		owner := scoreOwner{change.Player, change.Franchise}
		ix.by[owner] = append(ix.by[owner], indexedScore{command: r.Command.Command, change: change,
			entry: r.Receipt.Entry, reason: r.Command.Reason})
	}
	return ix
}

// changes is scoreChangesFrom over the indexed ledger (TestScoreIndexMatchesLedgerScan).
func (ix *scoreIndex) changes(active map[string]bool, player, franchise string, season, week int) (ScoreChanges, error) {
	if ix.err != nil {
		return ScoreChanges{}, ix.err
	}
	out := ScoreChanges{Changes: []ScoreChange{}}
	for _, s := range ix.by[scoreOwner{player, franchise}] {
		position := s.command == PlayerPosition || s.command == PlayerPositionClear
		if !position && (s.change.Season != season || s.change.Week != week) {
			continue
		}
		status := "Reversed"
		if active[s.entry] {
			status = "Applied"
			if position {
				out.Position = s.change.Position
			} else {
				var err error
				if out.Delta, err = addScoreDelta(out.Delta, *s.change.Delta); err != nil {
					return ScoreChanges{}, err
				}
			}
		}
		out.Changes = append(out.Changes, ScoreChange{Entry: s.entry, What: scoreChangeLabel(s.command, s.change),
			Reason: s.reason, Status: status})
	}
	return out, nil
}

// latestRecorded is the first latest RecordedAt in the ledger, zero for none.
func latestRecorded(records []Record) time.Time {
	var at time.Time
	for _, r := range records {
		if r.Receipt.RecordedAt.After(at) {
			at = r.Receipt.RecordedAt
		}
	}
	return at
}

// instantAfter is ReadingInstant given the ledger's latestRecorded.
func instantAfter(latest, now time.Time) time.Time {
	at := now
	if latest.After(at) {
		at = latest
	}
	if at.IsZero() {
		return time.Unix(1, 0).UTC()
	}
	return at
}
