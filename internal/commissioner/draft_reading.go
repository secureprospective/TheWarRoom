package commissioner

import (
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/secureprospective/TheWarRoom/internal/engine/rookiedraft"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

type RawNativeDraftScope struct {
	League    string    `json:"league"`
	Season    int       `json:"season"`
	Franchise string    `json:"franchise"`
	At        time.Time `json:"at"`
}

func (r RawNativeDraftScope) Validate() error {
	if !strings.HasPrefix(r.League, "native.") || r.League == "native." || strings.TrimSpace(r.League) != r.League ||
		r.Season < 1 || r.Season > 9999 || blank(r.Franchise) || r.At.IsZero() {
		return fmt.Errorf("choose an app league, draft season, franchise and reading time")
	}
	return nil
}

// NativeDraft is the rookie draft as a franchise sees it: the order with its owners, the picks made,
// who is on the clock until when (in the viewer's time) and the viewer's own next pick.
type NativeDraft struct {
	League   string                   `json:"league"`
	Season   int                      `json:"season"`
	Revision int64                    `json:"revision"`
	Rules    []RuleLine               `json:"rules"`
	Owners   []DraftOwner             `json:"owners"`
	Order    []rookiedraft.RoundOrder `json:"order"`
	Words    string                   `json:"words"`
	Draft    rookiedraft.Draft        `json:"draft"`
	Next     *rookiedraft.Pick        `json:"nextOwnedPick"`
}

type DraftOwner struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// draftBasis is one draft replayed from the ledger. words says why it cannot run yet, and is empty once
// the order is set and the clock can be read.
type draftBasis struct {
	rules   leaguerules.RuleSet
	owners  []DraftOwner
	order   []rookiedraft.RoundOrder
	start   time.Time
	started bool
	draft   rookiedraft.Draft
	words   string
}

// readDraft replays the ledger up to an instant. A draft that cannot be read yet is explained in words; an
// error means the ledger holds a pick that contradicts the draft's own order or clock.
func readDraft(records []Record, season int, at time.Time) (draftBasis, error) {
	records = recordsThrough(records, at)
	if len(records) == 0 {
		return draftBasis{}, fmt.Errorf("this app league has not been created")
	}
	rules, err := draftRulesFor(records, season, at)
	if err != nil {
		return draftBasis{}, err
	}
	basis := draftBasis{rules: rules, owners: draftOwners(records)}
	basis.start, err = draftCalendarStart(records, rules, season, at)
	if err != nil {
		basis.words = draftWords(err)
		return basis, nil
	}
	basis.started = !at.Before(basis.start)
	owned, err := draftOwnership(season, records)
	if err != nil {
		basis.words = draftWords(err)
		return basis, nil
	}
	owned = slices.DeleteFunc(owned, func(p rookiedraft.OwnedPick) bool {
		return rules.Draft.Rounds.State == leaguerules.IsKnown && p.Round > rules.Draft.Rounds.Value
	})
	basis.order, err = rookiedraft.Order(rules.Draft, records[0].Command.Args.Franchises, owned,
		rookiedraft.MissingFinalStandings{})
	if err != nil {
		basis.words = draftWords(err)
		return basis, nil
	}
	for _, round := range basis.order {
		if len(round.Picks) == 0 {
			basis.words = round.Words
			return basis, nil
		}
	}
	basis.draft, err = rookiedraft.Replay(rules.Draft, basis.order, basis.start, at, draftSelections(records, season))
	if err != nil {
		return basis, fmt.Errorf("replay the rookie draft: %w", err)
	}
	return basis, nil
}

// draftClockRuns says whether the draft's start has passed, so its clock is counting.
func draftClockRuns(records []Record, season int, at time.Time) bool {
	basis, err := readDraft(records, season, at)
	return err == nil && basis.started
}

func draftWords(err error) string {
	text := err.Error()
	first, size := utf8.DecodeRuneInString(text)
	return string(unicode.ToUpper(first)) + text[size:] + "."
}

// NativeDraftReading is public: every owner may read the order, the picks and the clock.
func NativeDraftReading(scope RawNativeDraftScope, records []Record, local *time.Location) (NativeDraft, error) {
	if err := scope.Validate(); err != nil {
		return NativeDraft{}, err
	}
	if local == nil {
		return NativeDraft{}, fmt.Errorf("choose your local time zone")
	}
	if len(records) == 0 || !leagueFranchise(records, scope.Franchise) {
		return NativeDraft{}, fmt.Errorf("choose a franchise in this app league")
	}
	basis, err := readDraft(records, scope.Season, scope.At)
	if err != nil {
		return NativeDraft{}, err
	}
	out := NativeDraft{League: scope.League, Season: scope.Season, Revision: int64(len(recordsThrough(records, scope.At))),
		Rules: ruleLines(basis.rules, draftReadingFields()...), Owners: basis.owners, Order: basis.order,
		Words: basis.words, Draft: basis.draft}
	if out.Order == nil {
		out.Order = []rookiedraft.RoundOrder{}
	}
	if basis.words != "" {
		out.Draft.Picks = []rookiedraft.Result{}
		return out, nil
	}
	out.Words = "The rookie draft is waiting to start."
	switch {
	case out.Draft.Exhausted:
		out.Words = "Every pick has been used; the draft has not been closed."
	case out.Draft.Started:
		out.Words = "The rookie draft is running."
	}
	if pick, ok := out.Draft.NextPick(scope.Franchise); ok {
		out.Next = &pick
	}
	localizeDraft(&out.Draft, local)
	return out, nil
}

func draftReadingFields() []string {
	return []string{"draft.mode", "draft.clock", "draft.onExpire", "draft.timeouts", "draft.rounds",
		"draft.maximumPicks", "draft.forfeitExtra", "draft.pricing", "draft.lengths", "draft.rfaAfter",
		"draft.orderSource", "draft.order", "draft.laterPattern", "draft.tieRule"}
}

func draftOwners(records []Record) []DraftOwner {
	seats := make([]PeopleSeat, len(records[0].Command.Args.Franchises))
	for i, id := range records[0].Command.Args.Franchises {
		seats[i] = PeopleSeat{Franchise: id, Name: fmt.Sprintf("Franchise %d", i+1)}
	}
	applyNativeNames(seats, records)
	out := make([]DraftOwner, len(seats))
	for i, seat := range seats {
		out[i] = DraftOwner{ID: seat.Franchise, Name: seat.Name}
	}
	return out
}

// draftOwnership lists who owns each of a draft's picks as the starting snapshot recorded it. A draft whose
// picks carry a held slot was run before this league began.
func draftOwnership(season int, records []Record) ([]rookiedraft.OwnedPick, error) {
	idx := slices.IndexFunc(records, func(r Record) bool { return r.Command.Command == NativeSeed })
	if idx < 0 {
		return nil, fmt.Errorf("record the starting snapshot's draft picks before ordering the draft")
	}
	owned := []rookiedraft.OwnedPick{}
	assets, err := NativeAssetsReading(records[idx].Command.League, records)
	if err != nil {
		return nil, err
	}
	for _, p := range assets.Snapshot.Picks {
		if p.Season != season {
			continue
		}
		if p.Original == "" {
			return nil, fmt.Errorf("this draft was run before the league began; it cannot be run again here")
		}
		owned = append(owned, rookiedraft.OwnedPick{Round: p.Round, Original: p.Original, Owner: p.Owner})
	}
	if len(owned) == 0 {
		return nil, fmt.Errorf("no picks were recorded for the %d draft", season)
	}
	return owned, nil
}

// draftSelections are the picks in force for one draft, in ledger order.
func draftSelections(records []Record, season int) []rookiedraft.Selection {
	active := Replay(records).Active
	out := []rookiedraft.Selection{}
	for _, r := range records {
		if r.Command.Command != DraftPick || !active[r.Receipt.Entry] || r.Command.Args.Draft.Season != season {
			continue
		}
		p := r.Command.Args.Draft.Pick
		terms := p.Terms
		out = append(out, rookiedraft.Selection{Round: p.Round, Original: p.Original, Owner: p.Franchise,
			Player: p.Player, At: r.Receipt.RecordedAt, Terms: &terms})
	}
	return out
}

func localizeDraft(d *rookiedraft.Draft, local *time.Location) {
	d.Start = d.Start.In(local)
	if d.OnClock != nil {
		d.OnClock.OpenedAt, d.OnClock.Deadline = d.OnClock.OpenedAt.In(local), d.OnClock.Deadline.In(local)
	}
	for i := range d.Picks {
		if d.Picks[i].MadeAt != nil {
			at := d.Picks[i].MadeAt.In(local)
			d.Picks[i].MadeAt = &at
		}
	}
}
