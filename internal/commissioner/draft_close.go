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
	}
	return nil
}

// draftHolds is the draft as it stands, with the assets it moves.
type draftHolds struct {
	basis  draftBasis
	assets NativeAssets
	taken  map[string]bool
}

func holdsOf(league string, records []Record, season int, now time.Time, standings DraftStandings) (draftHolds, error) {
	basis, err := readDraft(records, season, now, standings)
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
			out.taken[p.ID] = true
		}
		for _, p := range assets.Snapshot.Pool {
			out.taken[p.Player] = true
		}
	}
	return out, nil
}

func prepareRookie(raw *RawCommand, records []Record, now time.Time, facts nativecheck.Facts,
	standings DraftStandings) error {
	d := raw.Args.Draft
	holds, err := holdsOf(raw.League, records, d.Season, now, standings)
	if err != nil {
		return err
	}
	if f := facts[d.Act.Player]; !f.DraftYearHeld || f.DraftYear != d.Season {
		return fmt.Errorf("choose a rookie from the %d player pool", d.Season)
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
	if !h.assets.Seeded || !slices.ContainsFunc(h.assets.Snapshot.Players, func(p NativePlayer) bool { return p.ID == player }) {
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
	holds, err := holdsOf(raw.League, records, season, now, standings)
	if err != nil {
		return err
	}
	outcome := &DraftOutcome{Pool: []NativePoolEntry{}, Issued: []NativePick{}}
	for _, p := range classRookies(facts, season, holds.taken) {
		name := ""
		if facts[p.ID].Name != "" {
			name = p.Name
		}
		outcome.Pool = append(outcome.Pool, NativePoolEntry{Player: p.ID, Name: name, Season: season})
	}
	outcome.Issued = issuedPicks(holds, records[0].Command.Args.Franchises, season)
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

// applyDraftClose puts a closed draft's outcome into the native assets: the released rookies join the
// free-agent pool and the issued picks join the franchises' picks.
func applyDraftClose(seed *RawNativeSnapshot, record Record) {
	outcome := record.Command.Args.DraftOutcome
	if outcome == nil {
		return
	}
	seed.Pool = append(seed.Pool, outcome.Pool...)
	seed.Picks = append(seed.Picks, outcome.Issued...)
}

// DueDrafts are the seasons whose draft has used every pick and has not been closed: the app closes them.
// A league with no live draft change recorded has no draft to look at.
func DueDrafts(records []Record, at time.Time, standings DraftStandings) []int {
	seasons := []int{}
	for _, season := range draftSeasonsKnown(records) {
		basis, err := readDraft(records, season, at, standings)
		if err == nil && basis.words == "" && basis.draft.Exhausted && basis.closed == nil {
			seasons = append(seasons, season)
		}
	}
	return seasons
}

// NextDraftDeadline is the nearest instant after at when a draft's clock starts or runs out, so a due
// worker can sleep until then; no one records a pick the clock used up. A paused clock waits for reconcile.
func NextDraftDeadline(records []Record, at time.Time, standings DraftStandings) time.Time {
	var next time.Time
	for _, season := range draftSeasonsKnown(records) {
		basis, err := readDraft(records, season, at, standings)
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

// draftSeasonsKnown are the drafts with a start, a pick or any other live change recorded.
func draftSeasonsKnown(records []Record) []int {
	seasons := draftsWithPicks(records)
	active := Replay(records).Active
	for _, r := range records {
		if r.Command.Command == DraftStart && active[r.Receipt.Entry] && !slices.Contains(seasons, r.Command.Args.Draft.Season) {
			seasons = append(seasons, r.Command.Args.Draft.Season)
		}
	}
	return seasons
}
