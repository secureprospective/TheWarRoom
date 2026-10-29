package commissioner

import (
	"slices"
	"time"
)

// PickValueDrafts replays rookie selections after corrections/restores, never startup drafts.
// Explicit orders are available even before a start date has been set.
func PickValueDrafts(records []Record, at time.Time, standings DraftStandings) ([]NativeDraft, error) {
	records = RecordsThrough(records, at)
	if standings == nil {
		standings = NoDraftStandings{}
	}
	active := Replay(records).Active
	keys := []DraftKey{}
	for _, r := range records {
		d := r.Command.Args.Draft
		if d == nil || d.Season < 1 || d.Startup || !active[r.Receipt.Entry] {
			continue
		}
		for _, key := range sideDraftKeys(records, d.key()) {
			if !slices.Contains(keys, key) {
				keys = append(keys, key)
			}
		}
	}
	out := []NativeDraft{}
	for _, key := range keys {
		basis, err := readDraft(records, key, at, standings)
		if err != nil {
			return nil, err
		}
		if len(basis.order) == 0 {
			rules, err := draftRulesFor(records, key.Season, at)
			if err != nil {
				return nil, err
			}
			// An unresolved order is unknown; it must not prevent reading the other drafts.
			if order, err := rookieOrder(records, key, rules.Draft, standings.Final(records, key.Season-1, at)); err == nil {
				basis.order = order
			}
		}
		out = append(out, NativeDraft{Season: key.Season, Side: key.Side, Order: basis.order, Draft: basis.draft})
	}
	return out, nil
}
