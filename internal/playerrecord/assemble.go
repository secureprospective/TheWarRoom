package playerrecord

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

// Listing is MFL's players database as the record reads it. normalize.Lookup satisfies it.
type Listing interface {
	Facts(mflID string) (normalize.PlayerFacts, bool)
	IDs() []string
}

// identitySlack is how many seasons history's first NFL season may sit from the year MFL has the
// player entering the league. On the 2026 roster 1,439 of 1,440 sit within one; the one outside
// sat 48 away.
const identitySlack = 2

// noDraftYear is MFL's placeholder draft year: 1970 (and 0) mean MFL does not know the year.
const noDraftYear = 1970

// Assemble resolves every player's record from what the feeds hold: MFL's listing and history's
// facts (nflverse, through the crosswalk), both keyed by MFL link as the feeds stored them. ids
// gives our ID for an MFL link; it never mints a second one for a link it already holds.
//
// Which source wins a field:
//   - position: MFL's, the league's own label; nflverse's for a player MFL does not list.
//   - birth: nflverse's date; MFL's when nflverse has none.
//   - rookie season: nflverse's; MFL's draft year when nflverse has none.
//   - draft: MFL's year, round, pick and team; the overall pick is nflverse's, else derived from
//     MFL's round and pick by the round's first overall pick in that draft, as the players
//     nflverse does know place it.
//
// A history record whose first NFL season is far from MFL's draft year is another player's: the
// crosswalk tied him to the wrong IDs (DynastyProcess gave a 2026 rookie DT the IDs of a 1978
// receiver). MFL wins: that record's birth, rookie season, draft slot and measurements are set
// aside and the record says Mislinked.
func Assemble(listing Listing, history map[string]*model.Player, ids func(mfl string) (string, error)) ([]Record, error) {
	links := map[string]bool{}
	for _, id := range listing.IDs() {
		links[id] = true
	}
	for id := range history {
		links[id] = true
	}
	out := make([]Record, 0, len(links))
	for _, mfl := range slices.Sorted(maps.Keys(links)) {
		id, err := ids(mfl)
		if err != nil {
			return nil, fmt.Errorf("playerrecord: player %s: %w", mfl, err)
		}
		facts, listed := listing.Facts(mfl)
		out = append(out, resolve(id, mfl, facts, listed, history[mfl]))
	}
	deriveOverall(out)
	return out, nil
}

func resolve(id, mfl string, f normalize.PlayerFacts, listed bool, h *model.Player) Record {
	r := Record{ID: id, MFL: mfl, Listed: listed, Sources: map[string]string{}}
	if listed && h != nil && f.HasDraftYear && h.Rookie != 0 && abs(h.Rookie-f.DraftYear) > identitySlack {
		r.Mislinked, h = true, nil
	}
	if listed {
		r.Name, r.College, r.Position = f.Name, f.College, f.Position
		r.Sources[FieldBio], r.Sources[FieldPosition] = SourceMFL, SourceMFL
	} else if h != nil {
		r.Position, r.Sources[FieldPosition] = h.Position, SourceNFLverse
	}
	year := 0
	if listed && f.HasDraftYear && f.DraftYear > noDraftYear {
		year = f.DraftYear
	}
	r.resolveAge(year, f, listed, h)
	r.resolveDraft(year, f, listed, h)
	return r
}

// resolveAge sets birth and rookie season: nflverse's, else MFL's birth date and draft year.
func (r *Record) resolveAge(year int, f normalize.PlayerFacts, listed bool, h *model.Player) {
	switch {
	case h != nil && !h.Birth.IsZero():
		r.Birth, r.Sources[FieldBirth] = h.Birth, SourceNFLverse
	case listed && f.HasBirthdate:
		r.Birth, r.Sources[FieldBirth] = time.Unix(f.Birthdate, 0).UTC(), SourceMFL
	}
	switch {
	case h != nil && h.Rookie != 0:
		r.Rookie, r.Sources[FieldRookie] = h.Rookie, SourceNFLverse
	case year != 0:
		r.Rookie, r.Sources[FieldRookie] = year, SourceMFL
	}
}

// resolveDraft sets the draft capital: MFL's selection, nflverse's overall pick, and undrafted
// when MFL has a year he entered the league with no selection.
func (r *Record) resolveDraft(year int, f normalize.PlayerFacts, listed bool, h *model.Player) {
	r.Draft = Draft{Status: DraftUnknown, Year: year}
	if listed && f.DraftRound > 0 {
		r.Draft.Round, r.Draft.Pick, r.Draft.Team = f.DraftRound, f.DraftPick, f.DraftTeam
		r.Sources[FieldDraft] = SourceMFL
	}
	if h != nil && h.DraftPick >= 1 {
		r.Draft.Overall, r.Sources[FieldOverall] = int(h.DraftPick), SourceNFLverse
	}
	switch {
	case r.Draft.Round > 0 || r.Draft.Overall > 0:
		r.Draft.Status = Drafted
	case year != 0:
		r.Draft.Status, r.Sources[FieldDraft] = Undrafted, SourceMFL
	}
}

type draftRound struct{ year, round int }

// deriveOverall fills the overall pick of each drafted player nflverse does not know, from MFL's
// round and pick: a round's first overall pick in a draft is the same for every player in it, so
// the players nflverse does know give it. Where they disagree (a forfeited pick) the most common
// answer wins, the earlier on a tie.
func deriveOverall(records []Record) {
	starts := map[draftRound]map[int]int{}
	for _, r := range records {
		d := r.Draft
		if d.Overall > 0 && d.Round > 0 && d.Pick > 0 && d.Year > 0 && r.Sources[FieldOverall] == SourceNFLverse {
			k := draftRound{d.Year, d.Round}
			if starts[k] == nil {
				starts[k] = map[int]int{}
			}
			starts[k][d.Overall-d.Pick+1]++
		}
	}
	for i := range records {
		d := &records[i].Draft
		if d.Overall != 0 || d.Round == 0 || d.Pick == 0 || d.Year == 0 {
			continue
		}
		votes := starts[draftRound{d.Year, d.Round}]
		if len(votes) == 0 {
			continue
		}
		best := slices.MaxFunc(slices.Sorted(maps.Keys(votes)), func(a, b int) int {
			return cmp.Or(cmp.Compare(votes[a], votes[b]), cmp.Compare(b, a))
		})
		d.Overall = best + d.Pick - 1
		records[i].Sources[FieldOverall] = SourceDerived
	}
}

func abs(n int) int { return max(n, -n) }
