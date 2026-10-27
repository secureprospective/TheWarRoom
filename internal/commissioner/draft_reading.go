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
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// RawNativeDraftScope asks for one franchise's view of a draft. A season of zero asks for the draft that is
// next: the first one nobody has closed.
type RawNativeDraftScope struct {
	League    string    `json:"league"`
	Season    int       `json:"season"`
	Franchise string    `json:"franchise"`
	At        time.Time `json:"at"`
}

func (r RawNativeDraftScope) Validate() error {
	if !strings.HasPrefix(r.League, "native.") || r.League == "native." || strings.TrimSpace(r.League) != r.League ||
		r.Season < 0 || r.Season > 9999 || blank(r.Franchise) || r.At.IsZero() {
		return fmt.Errorf("choose an app league, draft season, franchise and reading time")
	}
	return nil
}

// NativeDraft is the rookie draft as a franchise sees it: the order with its owners, the picks made,
// who is on the clock until when (in the viewer's time) and the viewer's own next pick.
type NativeDraft struct {
	Corrections []Correction             `json:"corrections,omitempty"`
	League      string                   `json:"league"`
	Season      int                      `json:"season"`
	Revision    int64                    `json:"revision"`
	Rules       []RuleLine               `json:"rules"`
	Owners      []DraftOwner             `json:"owners"`
	Order       []rookiedraft.RoundOrder `json:"order"`
	Words       string                   `json:"words"`
	Draft       rookiedraft.Draft        `json:"draft"`
	Next        *rookiedraft.Pick        `json:"nextOwnedPick"`
	// Closed is set once the draft is sealed. Players names every rookie the reading mentions, Available
	// lists those still undrafted, and Checkpoints are the points the commissioner may restore the draft to.
	Closed      *DraftClosed           `json:"closed"`
	Players     map[string]DraftPlayer `json:"players"`
	Available   []DraftPlayer          `json:"available"`
	Checkpoints []DraftCheckpoint      `json:"checkpoints"`
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
	closed  *Record
}

// readDraft replays the ledger up to an instant. A draft that cannot be read yet is explained in words; an
// error means the ledger holds a pick that contradicts the draft's own order or clock.
func readDraft(records []Record, season int, at time.Time, standings DraftStandings) (draftBasis, error) {
	if standings == nil {
		standings = NoDraftStandings{}
	}
	records = RecordsThrough(records, at)
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
	// A first round that begins reads the standings as they stood when it began, so a result corrected
	// afterwards cannot move it; before then it follows the results as they are recorded.
	through := at
	if basis.started {
		through = basis.start
	}
	basis.order, err = rookiedraft.Order(rules.Draft, LeagueFranchises(records), owned,
		standings.Final(records, season-1, through))
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
	basis.draft, err = rookiedraft.ReplayActs(rules.Draft, basis.order, basis.start, at, draftActs(records, season))
	if err != nil {
		return basis, fmt.Errorf("replay the rookie draft: %w", err)
	}
	if closed, ok := closeRecord(records, season); ok {
		basis.closed = &closed
	}
	return basis, nil
}

// draftClockRuns says whether the draft's start has passed, so its clock is counting.
func draftClockRuns(records []Record, season int, at time.Time, standings DraftStandings) bool {
	basis, err := readDraft(records, season, at, standings)
	return err == nil && basis.started
}

func draftWords(err error) string {
	text := err.Error()
	first, size := utf8.DecodeRuneInString(text)
	return string(unicode.ToUpper(first)) + text[size:] + "."
}

// NativeDraftReading is public: every owner may read the order, the picks and the clock. The facts name the
// rookies; without them a rookie reads as his number.
func NativeDraftReading(scope RawNativeDraftScope, records []Record, local *time.Location,
	standings DraftStandings, facts nativecheck.Facts) (NativeDraft, error) {
	if err := scope.Validate(); err != nil {
		return NativeDraft{}, err
	}
	if local == nil {
		return NativeDraft{}, fmt.Errorf("choose your local time zone")
	}
	if len(records) == 0 || !leagueFranchise(records, scope.Franchise) {
		return NativeDraft{}, fmt.Errorf("choose a franchise in this app league")
	}
	if scope.Season == 0 {
		season, err := draftSeasonNext(scope.League, records, scope.At)
		if err != nil {
			return NativeDraft{}, err
		}
		scope.Season = season
	}
	basis, err := readDraft(records, scope.Season, scope.At, standings)
	if err != nil {
		return NativeDraft{}, err
	}
	through := RecordsThrough(records, scope.At)
	out := NativeDraft{League: scope.League, Season: scope.Season, Revision: int64(len(through)),
		Rules: namedRules(RuleLines(basis.rules, draftReadingFields()...), basis.owners), Owners: basis.owners, Order: basis.order,
		Words: basis.words, Draft: basis.draft, Players: map[string]DraftPlayer{}, Available: []DraftPlayer{},
		Checkpoints: []DraftCheckpoint{}}
	changes, err := CorrectionsReading(scope.League, records, scope.At)
	if err != nil {
		return NativeDraft{}, err
	}
	out.Corrections = changes.Items
	if out.Order == nil {
		out.Order = []rookiedraft.RoundOrder{}
	}
	if basis.words != "" {
		out.Draft.Picks = []rookiedraft.Result{}
		return out, nil
	}
	out.Words = draftStatusWords(basis)
	if pick, ok := out.Draft.NextPick(scope.Franchise); ok {
		out.Next = &pick
	}
	if err := out.addRoom(basis, through, facts); err != nil {
		return NativeDraft{}, err
	}
	localizeDraft(&out.Draft, local)
	return out, nil
}

// namedRules writes a franchise's name wherever a rule's words give its id, so the draft room never shows
// a code. The longest ids go first, so one id is never read as the start of another.
func namedRules(lines []RuleLine, owners []DraftOwner) []RuleLine {
	byLength := slices.SortedFunc(slices.Values(owners), func(a, b DraftOwner) int {
		return len(b.ID) - len(a.ID)
	})
	for i := range lines {
		for _, o := range byLength {
			lines[i].Words = strings.ReplaceAll(lines[i].Words, o.ID, o.Name)
		}
	}
	return lines
}

// draftStatusWords says where a readable draft stands.
func draftStatusWords(basis draftBasis) string {
	switch {
	case basis.closed != nil:
		return "The rookie draft is closed."
	case basis.draft.Exhausted:
		return "Every pick has been used; the draft has not been closed."
	case basis.draft.OnClock != nil && basis.draft.OnClock.Paused:
		return "The rookie draft is running, and its clock is paused."
	case basis.draft.Started:
		return "The rookie draft is running."
	}
	return "The rookie draft is waiting to start."
}

func draftReadingFields() []string {
	return []string{"draft.mode", "draft.clock", "draft.onExpire", "draft.timeouts", "draft.rounds",
		"draft.maximumPicks", "draft.forfeitExtra", "draft.pricing", "draft.lengths", "draft.rfaAfter",
		"draft.orderSource", "draft.order", "draft.laterPattern", "draft.tieRule"}
}

func draftOwners(records []Record) []DraftOwner {
	ids := LeagueFranchises(records)
	seats := make([]PeopleSeat, len(ids))
	for i, id := range ids {
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
	for _, p := range records[idx].Command.Args.Native.Picks {
		if p.Season == season && p.Slot != 0 {
			return nil, fmt.Errorf("this draft was run before the league began; it cannot be run again here")
		}
	}
	owned := []rookiedraft.OwnedPick{}
	assets, err := NativeAssetsReading(records[0].Command.League, records)
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

// draftActs are the happenings in force for one draft, in ledger order: the picks, the tray, the restores
// and the close. A restore's checkpoint has set aside what came after it (see setAsideByDraftRestores).
func draftActs(records []Record, season int) []rookiedraft.Act {
	active := Replay(records).Active
	out := []rookiedraft.Act{}
	for _, r := range records {
		if !liveDraftEntry(r.Command.Command) || !active[r.Receipt.Entry] || r.Command.Args.Draft.Season != season {
			continue
		}
		if r.Command.Command == DraftPick {
			p := r.Command.Args.Draft.Pick
			terms := p.Terms
			out = append(out, rookiedraft.Act{Kind: rookiedraft.Select, At: r.Receipt.RecordedAt,
				Pick: rookiedraft.PickRef{Round: p.Round, Original: p.Original}, Owner: p.Franchise, Player: p.Player,
				Terms: &terms})
			continue
		}
		if act, ok := trayAct(r); ok {
			if act.Kind == rookiedraft.Restart {
				act.Since = checkpointTime(records, r.Command.Args.Draft.Act.To)
			}
			out = append(out, act)
		}
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

// checkpointTime is when the entry a restore names was recorded.
func checkpointTime(records []Record, entry string) time.Time {
	for _, r := range records {
		if r.Receipt.Entry == entry {
			return r.Receipt.RecordedAt
		}
	}
	return time.Time{}
}
