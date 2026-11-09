package commissioner

import "strings"

// lineupSyncRequest prefixes every lineup the reference sync records, so the ledger alone says
// whether a league has a lineup sync source on and which weeks it has reached.
const lineupSyncRequest = "lineup.sync."

// LineupSyncPending reports that the league records lineups from a sync source that has not
// reached this week yet. A franchise is not a late franchise while the app has not yet read the
// week it is playing, so a missed lineup is not flagged to the commissioner until the sync has
// recorded at least one lineup for that week.
func LineupSyncPending(records []Record, season, week int) bool {
	on, reached := false, false
	var active map[string]bool
	for _, r := range records {
		if !isLineupCommand(r.Command.Command) || !strings.HasPrefix(r.Command.Request, lineupSyncRequest) {
			continue
		}
		if active == nil {
			active = Replay(records).Active
		}
		if !active[r.Receipt.Entry] {
			continue
		}
		change, err := DecodeLineupChange(r.Command.Args, r.Command.Command)
		if err != nil {
			continue
		}
		on = true
		reached = reached || (change.Season == season && change.Week == week)
	}
	return on && !reached
}
