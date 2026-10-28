package commissioner

import (
	"fmt"
	"slices"
)

// setAsideByDraftRestores marks inactive every live draft entry a restore compensated: those of the same
// draft recorded after its checkpoint and before the restore. Nothing is deleted; the entries stay in the
// ledger, and undoing the restore's own effect is reading the ledger as it stood before it.
func setAsideByDraftRestores(records []Record, active map[string]bool) {
	index := map[string]int{}
	for i, r := range records {
		index[r.Receipt.Entry] = i
	}
	for j, r := range records {
		if r.Command.Command != DraftRevert || !active[r.Receipt.Entry] {
			continue
		}
		checkpoint, ok := index[r.Command.Args.Draft.Act.To]
		if !ok || checkpoint >= j {
			continue
		}
		for _, later := range records[checkpoint+1 : j] {
			if liveDraftEntry(later.Command.Command) && ofDraft(later, r.Command.Args.Draft.key()) {
				active[later.Receipt.Entry] = false
			}
		}
	}
}

// draftRestoreLabel says what a restore put the draft back to, by the label of its checkpoint.
func draftRestoreLabel(records []Record, i int) string {
	d := records[i].Command.Args.Draft
	at := slices.IndexFunc(records[:i], func(r Record) bool { return r.Receipt.Entry == d.Act.To })
	if at < 0 {
		return fmt.Sprintf("Restored the %s to an earlier point", d.noun())
	}
	return fmt.Sprintf("Restored the %s to how it stood after: %s", d.noun(), recordLabel(records[at]))
}

// checkpointOf is the recorded entry a restore names, when it is a change to this draft that is still in
// force: a draft's start, its order, or one of its live entries. Its place in the ledger is its index.
func checkpointOf(records []Record, key DraftKey, entry string) (int, error) {
	at := slices.IndexFunc(records, func(r Record) bool { return r.Receipt.Entry == entry })
	if at < 0 {
		return 0, fmt.Errorf("choose a point in this draft's history to restore it to")
	}
	r := records[at]
	if !draftCommand(r.Command.Command) || !ofDraft(r, key) || standingRule(r.Command.Command) ||
		!Replay(records).Active[entry] {
		return 0, fmt.Errorf("choose a point in this draft's history that is still part of it")
	}
	return at, nil
}
