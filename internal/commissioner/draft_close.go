package commissioner

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine/rookiedraft"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// issuedSeasonsAhead is how many seasons out a closed draft leaves picks in every franchise's hands: ruling
// 9c, new picks "for two seasons out", and the rulebook's "picks may be traded up to two years in advance".
const issuedSeasonsAhead = 2

// PrepareDraftAppend fixes what Go owns on a live draft change before it is recorded: the contract a rookie
// signs at his pick's slot, and for a close the rookies released to the pool and the picks issued. The rookie
// must be an undrafted, unrostered player of the draft's class.
func PrepareDraftAppend(raw *RawCommand, records []Record, now time.Time, facts nativecheck.Facts,
	standings DraftStandings) error {
	switch raw.Command {
	case DraftAssign, DraftInsert, DraftChange:
		return prepareRookie(raw, records, now, facts, standings)
	case DraftArchive:
		return prepareClose(raw, records, now, facts, standings)
	case DraftStartupOpen:
		return prepareStartupOpen(raw, records, now)
	}
	return nil
}

// draftHolds is the draft as it stands, with the assets it moves.
type draftHolds struct {
	basis  draftBasis
	assets NativeAssets
	taken  map[string]bool
}

func holdsOf(league string, records []Record, key DraftKey, now time.Time, standings DraftStandings) (draftHolds, error) {
	basis, err := readDraft(records, key, now, standings)
	if err != nil {
		return draftHolds{}, err
	}
	if basis.words != "" {
		return draftHolds{}, fmt.Errorf("the draft is not ready: %s", strings.TrimSuffix(basis.words, "."))
	}
	assets, err := NativeAssetsReading(league, RecordsThrough(records, now))
	if err != nil {
		return draftHolds{}, err
	}
	out := draftHolds{basis: basis, assets: assets, taken: map[string]bool{}}
	for _, pick := range basis.draft.Picks {
		out.taken[pick.Player] = true
	}
	if assets.Seeded {
		for _, p := range assets.Snapshot.Players {
			if assets.Snapshot.SideOf(p.Franchise) == key.Side {
				out.taken[p.ID] = true
			}
		}
		for _, p := range assets.Snapshot.Pool {
			if p.Side == key.Side {
				out.taken[p.Player] = true
			}
		}
	}
	return out, nil
}

func prepareRookie(raw *RawCommand, records []Record, now time.Time, facts nativecheck.Facts,
	standings DraftStandings) error {
	d := raw.Args.Draft
	holds, err := holdsOf(raw.League, records, d.key(), now, standings)
	if err != nil {
		return err
	}
	if err := inPool(d, facts); err != nil {
		return err
	}
	if holds.taken[d.Act.Player] {
		return fmt.Errorf("this player already belongs to a team or has been drafted")
	}
	if raw.Command == DraftChange {
		return holds.keepsReplaced(d.Act.Replaces)
	}
	target, ok := holds.pick(d.Act.ref())
	if !ok {
		return fmt.Errorf("choose a pick in this draft")
	}
	terms, err := rookiedraft.SlotTerms(holds.basis.rules.Draft, target.Round, target.Slot)
	if err != nil {
		return fmt.Errorf("price the pick: %w", err)
	}
	franchise := target.Owner
	if raw.Command == DraftInsert {
		franchise = d.Act.Franchise
	}
	raw.Args.DraftOutcome = &DraftOutcome{Terms: &terms, Franchise: franchise, Slot: target.Slot}
	return nil
}

// keepsReplaced holds a change to a rookie who is still on the roster that drafted him: once he has left it,
// his pick can no longer be swapped for another player.
func (h draftHolds) keepsReplaced(player string) error {
	if !h.assets.Seeded || !h.assets.Snapshot.OnSide(h.basis.key.Side, player) {
		return fmt.Errorf("this rookie is no longer on the roster that drafted him, so his pick cannot be changed")
	}
	return nil
}

func (h draftHolds) pick(ref rookiedraft.PickRef) (rookiedraft.Result, bool) {
	i := slices.IndexFunc(h.basis.draft.Picks, func(p rookiedraft.Result) bool {
		return p.Round == ref.Round && p.Original == ref.Original && p.Extra == ref.Extra
	})
	if i < 0 {
		return rookiedraft.Result{}, false
	}
	return h.basis.draft.Picks[i], true
}

// prepareClose records the rookies nobody took, who go to the free-agent pool, and the picks issued for the
// seasons ahead.
func prepareClose(raw *RawCommand, records []Record, now time.Time, facts nativecheck.Facts,
	standings DraftStandings) error {
	season := raw.Args.Draft.Season
	holds, err := holdsOf(raw.League, records, raw.Args.Draft.key(), now, standings)
	if err != nil {
		return err
	}
	outcome := &DraftOutcome{Pool: []NativePoolEntry{}, Issued: []NativePick{}}
	// A startup draft moves nobody to the pool and issues no picks: they are the rookie draft's.
	if raw.Args.Draft.Startup {
		raw.Args.DraftOutcome = outcome
		return nil
	}
	for _, p := range classRookies(facts, season, holds.taken) {
		name := ""
		if facts[p.ID].Name != "" {
			name = p.Name
		}
		outcome.Pool = append(outcome.Pool, NativePoolEntry{Player: p.ID, Name: name, Season: season, Side: holds.basis.key.Side})
	}
	franchises, err := draftFranchises(records, holds.basis.key)
	if err != nil {
		return err
	}
	outcome.Issued = issuedPicks(holds, franchises, season)
	raw.Args.DraftOutcome = outcome
	return nil
}

// issuedPicks are the picks a closed draft puts in every franchise's hands for the seasons ahead, as many
// rounds as the draft has, leaving alone any a franchise already holds.
func issuedPicks(holds draftHolds, franchises []string, season int) []NativePick {
	out := []NativePick{}
	if !holds.assets.Seeded || holds.basis.rules.Draft.Rounds.State != leaguerules.IsKnown {
		return out
	}
	for ahead := 1; ahead <= issuedSeasonsAhead && season+ahead <= lastSeason; ahead++ {
		for round := 1; round <= holds.basis.rules.Draft.Rounds.Value; round++ {
			for _, franchise := range franchises {
				held := slices.ContainsFunc(holds.assets.Snapshot.Picks, func(p NativePick) bool {
					return p.Season == season+ahead && p.Round == round && p.Original == franchise
				})
				if !held {
					out = append(out, NativePick{Season: season + ahead, Round: round, Original: franchise,
						Owner: franchise})
				}
			}
		}
	}
	return out
}

// DueDrafts are the seasons whose draft has used every pick and has not been closed: the app closes them.
// A league with no live draft change recorded has no draft to look at.
func DueDrafts(records []Record, at time.Time, standings DraftStandings) []DraftKey {
	keys := []DraftKey{}
	for _, key := range draftSeasonsKnown(records) {
		basis, err := readDraft(records, key, at, standings)
		if err == nil && basis.words == "" && basis.draft.Exhausted && basis.closed == nil {
			keys = append(keys, key)
		}
	}
	return keys
}

// NextDraftDeadline is the nearest instant after at when a draft's clock starts or runs out, so a due
// worker can sleep until then; no one records a pick the clock used up. A paused clock waits for reconcile.
func NextDraftDeadline(records []Record, at time.Time, standings DraftStandings) time.Time {
	var next time.Time
	for _, key := range draftSeasonsKnown(records) {
		basis, err := readDraft(records, key, at, standings)
		if err != nil || basis.words != "" || basis.closed != nil || basis.draft.Exhausted {
			continue
		}
		if !basis.started {
			next = NearerFutureDeadline(next, basis.start, at)
		} else if clock := basis.draft.OnClock; clock != nil && !clock.Paused {
			next = NearerFutureDeadline(next, clock.Deadline, at)
		}
	}
	return next
}

// draftSeasonsKnown are the drafts with a start, an opening, a pick or any other live change recorded.
func draftSeasonsKnown(records []Record) []DraftKey {
	keys := draftsWithPicks(records)
	active := Replay(records).Active
	for _, r := range records {
		started := r.Command.Command == DraftStart || r.Command.Command == DraftStartupOpen
		if started && active[r.Receipt.Entry] {
			for _, key := range sideDraftKeys(records, r.Command.Args.Draft.key()) {
				if !slices.Contains(keys, key) {
					keys = append(keys, key)
				}
			}
		}
	}
	return keys
}

// inPool holds the player a commissioner gives a pick to the draft's pool: the season's rookie class, or for
// a startup draft any held player but an O-Line card.
func inPool(d *RawDraftArgs, facts nativecheck.Facts) error {
	f, held := facts[d.Act.Player]
	_, card := nativecheck.LineCardTeam(d.Act.Player)
	switch {
	case d.Startup && (!held || card):
		return fmt.Errorf("choose a player from the startup draft's pool")
	case !d.Startup && (!f.DraftYearHeld || f.DraftYear != d.Season):
		return fmt.Errorf("choose a rookie from the %d player pool", d.Season)
	}
	return nil
}
