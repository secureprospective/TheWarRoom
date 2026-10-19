package commissioner

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// DraftClockPrincipal is who the app itself acts as when it closes a draft at its last pick.
const DraftClockPrincipal = "native.draft.clock"

// DraftPlayer is a rookie as the draft room names him.
type DraftPlayer struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Position string `json:"position"`
	Team     string `json:"team"`
}

// DraftCheckpoint is a point the commissioner may restore the draft to: the draft's start or any change since.
type DraftCheckpoint struct {
	Entry string    `json:"entry"`
	Words string    `json:"words"`
	At    time.Time `json:"at"`
}

// DraftClosed is a sealed draft: the rookies nobody took, now in the free-agent pool, and the picks it issued.
type DraftClosed struct {
	At        time.Time     `json:"at"`
	Automatic bool          `json:"automatic"`
	Pool      []DraftPlayer `json:"pool"`
	Issued    []DraftIssued `json:"issued"`
}

// DraftIssued is the picks issued for one season when a draft closed. Tradable says every one of them is
// in its franchise's hands, unused, and can be traded.
type DraftIssued struct {
	Season   int  `json:"season"`
	Rounds   int  `json:"rounds"`
	Picks    int  `json:"picks"`
	Tradable bool `json:"tradable"`
}

func playerNamed(id string, facts nativecheck.Facts) DraftPlayer {
	f := facts[id]
	name := f.Name
	if name == "" {
		name = "Player " + id
	}
	return DraftPlayer{ID: id, Name: name, Position: string(f.Position), Team: f.Team}
}

// pickTradable says whether a pick can be traded: it has not been used.
func pickTradable(p NativePick) bool { return p.Player == "" && !p.Skipped }

// addRoom adds what the draft room shows beside the order: the names, the class still to be drafted, the
// closed draft's outcome and the points the draft may be restored to.
func (d *NativeDraft) addRoom(basis draftBasis, records []Record, facts nativecheck.Facts) error {
	taken := map[string]bool{}
	for _, pick := range d.Draft.Picks {
		if pick.Player != "" {
			taken[pick.Player] = true
			d.Players[pick.Player] = playerNamed(pick.Player, facts)
		}
	}
	assets, err := NativeAssetsReading(d.League, records)
	if err != nil {
		return err
	}
	if assets.Seeded {
		for _, p := range assets.Snapshot.Players {
			taken[p.ID] = true
		}
		for _, p := range assets.Snapshot.Pool {
			taken[p.Player] = true
		}
	}
	if basis.closed != nil {
		d.Closed = closedReading(*basis.closed, assets, facts)
		for _, p := range d.Closed.Pool {
			d.Players[p.ID] = p
		}
	} else {
		d.Available = classRookies(facts, d.Season, taken)
		for _, p := range d.Available {
			d.Players[p.ID] = p
		}
	}
	d.Checkpoints = draftCheckpoints(records, d.Season, facts)
	return nil
}

// classRookies are the held players whose draft year is the season, minus those already taken.
func classRookies(facts nativecheck.Facts, season int, taken map[string]bool) []DraftPlayer {
	out := []DraftPlayer{}
	for id, f := range facts {
		if f.DraftYearHeld && f.DraftYear == season && !taken[id] {
			out = append(out, playerNamed(id, facts))
		}
	}
	slices.SortFunc(out, func(a, b DraftPlayer) int {
		return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.ID, b.ID))
	})
	return out
}

func closedReading(r Record, assets NativeAssets, facts nativecheck.Facts) *DraftClosed {
	out := &DraftClosed{At: r.Receipt.RecordedAt, Automatic: r.Principal == DraftClockPrincipal,
		Pool: []DraftPlayer{}, Issued: []DraftIssued{}}
	outcome := r.Command.Args.DraftOutcome
	if outcome == nil {
		return out
	}
	for _, p := range outcome.Pool {
		player := playerNamed(p.Player, facts)
		if p.Name != "" {
			player.Name = p.Name
		}
		out.Pool = append(out.Pool, player)
	}
	seasons := map[int]*DraftIssued{}
	for _, pick := range outcome.Issued {
		row, ok := seasons[pick.Season]
		if !ok {
			row = &DraftIssued{Season: pick.Season, Tradable: true}
			seasons[pick.Season] = row
		}
		row.Picks++
		row.Rounds = max(row.Rounds, pick.Round)
		row.Tradable = row.Tradable && issuedTradable(pick, assets)
	}
	for _, row := range seasons {
		out.Issued = append(out.Issued, *row)
	}
	slices.SortFunc(out.Issued, func(a, b DraftIssued) int { return cmp.Compare(a.Season, b.Season) })
	return out
}

// issuedTradable says whether an issued pick is, in the franchise's assets now, an unused pick it holds.
func issuedTradable(issued NativePick, assets NativeAssets) bool {
	if !assets.Seeded {
		return false
	}
	return slices.ContainsFunc(assets.Snapshot.Picks, func(p NativePick) bool {
		return p.Season == issued.Season && p.Round == issued.Round && p.Original == issued.Original &&
			p.Owner == issued.Owner && pickTradable(p)
	})
}

// namedLabel is a live draft entry's ledger label with each rookie it names written by name, as the draft
// room shows him, where the ledger only holds his number.
func namedLabel(r Record, facts nativecheck.Facts) string {
	label := recordLabel(r)
	d := r.Command.Args.Draft
	ids := []string{}
	if d != nil && d.Pick != nil {
		ids = append(ids, d.Pick.Player)
	}
	if d != nil && d.Act != nil {
		ids = append(ids, d.Act.Player, d.Act.Replaces)
	}
	for _, id := range ids {
		if id != "" {
			label = strings.ReplaceAll(label, "player "+id, playerNamed(id, facts).Name)
		}
	}
	return label
}

// draftCheckpoints lists, newest first, the draft's start and each change in force since.
func draftCheckpoints(records []Record, season int, facts nativecheck.Facts) []DraftCheckpoint {
	active := Replay(records).Active
	out := []DraftCheckpoint{}
	for _, r := range records {
		d := r.Command.Args.Draft
		if d == nil || d.Season != season || !active[r.Receipt.Entry] || standingRule(r.Command.Command) ||
			(!liveDraftEntry(r.Command.Command) && r.Command.Command != DraftStart) {
			continue
		}
		out = append(out, DraftCheckpoint{Entry: r.Receipt.Entry, Words: namedLabel(r, facts), At: r.Receipt.RecordedAt})
	}
	slices.Reverse(out)
	return out
}

// draftSeasonNext is the season of the first rookie draft nobody has closed that this league has yet to run:
// one with a pick still waiting to be ordered. The picks of a draft run before the league began are history.
func draftSeasonNext(league string, records []Record, at time.Time) (int, error) {
	through := recordsThrough(records, at)
	assets, err := NativeAssetsReading(league, through)
	if err != nil {
		return 0, err
	}
	if !assets.Seeded {
		return 0, fmt.Errorf("record the starting snapshot's draft picks before opening the draft room")
	}
	if season, ok := draftSeasonPending(assets, through); ok {
		return season, nil
	}
	return 0, fmt.Errorf("this league has no rookie draft left to run")
}

// draftSeasonPending is the earliest season with a pick still waiting to be ordered and no close recorded.
func draftSeasonPending(assets NativeAssets, through []Record) (int, bool) {
	seasons := []int{}
	for _, p := range assets.Snapshot.Picks {
		if p.Slot == 0 && !slices.Contains(seasons, p.Season) {
			seasons = append(seasons, p.Season)
		}
	}
	slices.Sort(seasons)
	for _, season := range seasons {
		if _, closed := closeRecord(through, season); !closed {
			return season, true
		}
	}
	return 0, false
}
