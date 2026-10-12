package facts

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"slices"
	"strconv"
)

type RawNoteReading struct {
	TxnID      int64
	FromFID    string
	ToFID      string
	Kind       string
	PickYear   *int64
	PickRound  *int64
	PickSlot   *int64
	PickOrig   string
	CapM       *float64
	CapSeasons string
	Player     string
	Clarity    string
	Reading    string
}

func ParseCurated(body []byte) ([]RawNoteReading, error) {
	rows, err := csv.NewReader(bytes.NewReader(body)).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("curated readings: %w", err)
	}
	header := []string{"txn_id", "from_fid", "to_fid", "kind", "pick_year", "pick_round", "pick_slot", "pick_orig", "cap_m", "cap_seasons", "player", "clarity", "reading"}
	if len(rows) == 0 || !slices.Equal(rows[0], header) {
		return nil, fmt.Errorf("invalid curated reading columns")
	}
	var out []RawNoteReading
	for i, cells := range rows[1:] {
		r, err := parseNote(cells)
		if err != nil {
			return nil, fmt.Errorf("curated row %d: %w", i+2, err)
		}
		out = append(out, r)
	}
	return out, nil
}

func parseNote(cells []string) (RawNoteReading, error) {
	if len(cells) != 13 {
		return RawNoteReading{}, fmt.Errorf("expected 13 columns")
	}
	r := RawNoteReading{FromFID: cells[1], ToFID: cells[2], Kind: cells[3], PickOrig: cells[7], CapSeasons: cells[9], Player: cells[10], Clarity: cells[11], Reading: cells[12]}
	for _, field := range []struct {
		index  int
		target **int64
	}{{0, nil}, {4, &r.PickYear}, {5, &r.PickRound}, {6, &r.PickSlot}} {
		if cells[field.index] == "" {
			continue
		}
		n, err := strconv.ParseInt(cells[field.index], 10, 64)
		if err != nil {
			return r, fmt.Errorf("integer column %d: %w", field.index, err)
		}
		if field.target == nil {
			r.TxnID = n
		} else {
			*field.target = &n
		}
	}
	if cells[8] != "" {
		n, err := strconv.ParseFloat(cells[8], 64)
		if err != nil {
			return r, fmt.Errorf("cap amount: %w", err)
		}
		r.CapM = &n
	}
	return r, r.Validate()
}

func (r RawNoteReading) Validate() error {
	if r.CapM != nil && (math.IsNaN(*r.CapM) || math.IsInf(*r.CapM, 0)) {
		return fmt.Errorf("nonfinite curated cap amount")
	}
	if r.TxnID <= 0 || r.Reading == "" || !slices.Contains([]string{"pick", "cap", "player", "none", "unreadable"}, r.Kind) || !slices.Contains([]string{"clear", "inferred", "ambiguous", "unreadable"}, r.Clarity) {
		return fmt.Errorf("invalid curated reading for transaction %d", r.TxnID)
	}
	if r.Kind == "pick" && r.PickOrig != "" && r.PickYear != nil && r.FromFID != "" && r.PickRound == nil {
		return fmt.Errorf("pick reading %d lacks round", r.TxnID)
	}
	return nil
}

type pickKey struct {
	year, round int64
	orig        string
}
type pickSlot struct{ year, round, slot int64 }
type transfer struct {
	ts       int64
	from, to string
	reading  int
}
type noteDependencies struct {
	users        map[pickKey]string
	moves        map[pickKey][]transfer
	trades       map[int64]int64
	transactions map[int64]bool
}

func (c *compiler) notes(ctx context.Context, notes []RawNoteReading) error {
	dep, err := c.noteDependencies(ctx)
	if err != nil {
		return err
	}
	noted := map[pickKey][]transfer{}
	for i, r := range notes {
		if err := r.Validate(); err != nil {
			return err
		}
		if !dep.transactions[r.TxnID] {
			return fmt.Errorf("curated reading names absent transaction %d", r.TxnID)
		}
		if r.Kind == "pick" && r.PickOrig != "" && r.PickYear != nil && r.FromFID != "" {
			ts, exists := dep.trades[r.TxnID]
			if !exists {
				return fmt.Errorf("curated pick names non-trade %d", r.TxnID)
			}
			key := pickKey{*r.PickYear, *r.PickRound, r.PickOrig}
			noted[key] = append(noted[key], transfer{ts, r.FromFID, r.ToFID, i})
		}
	}
	verdict := noteVerdicts(dep, noted)
	for i, r := range notes {
		c.facts.NoteReading = append(c.facts.NoteReading, NoteReading{
			TxnID: r.TxnID, FromFid: nonempty(&r.FromFID), ToFid: nonempty(&r.ToFID), Kind: r.Kind,
			PickYear: copyPointer(r.PickYear), PickRound: copyPointer(r.PickRound), PickSlot: copyPointer(r.PickSlot), PickOrig: nonempty(&r.PickOrig),
			CapM: copyPointer(r.CapM), CapSeasons: nonempty(&r.CapSeasons), Player: nonempty(&r.Player), Clarity: r.Clarity,
			Reading: r.Reading, DraftCheck: verdict[i], Confirmed: 0,
		})
	}
	return nil
}

func noteVerdicts(dep noteDependencies, noted map[pickKey][]transfer) map[int]*string {
	verdict := map[int]*string{}
	for key, notes := range noted {
		used, exists := dep.users[key]
		if !exists {
			continue
		}
		moves := append(slices.Clone(dep.moves[key]), notes...)
		slices.SortStableFunc(moves, func(a, b transfer) int { return cmpInt(a.ts, b.ts) })
		holder := key.orig
		held := map[int]bool{}
		for _, move := range moves {
			ok := holder == move.from
			if move.reading >= 0 {
				held[move.reading] = ok
			}
			if ok {
				holder = move.to
			}
		}
		for index, ok := range held {
			status := "agrees"
			if !ok {
				status = "sender did not hold it"
			} else if holder != used {
				status = "chain ends with another team"
			}
			verdict[index] = &status
		}
	}
	return verdict
}

func copyPointer[T int64 | float64](p *T) *T {
	if p == nil {
		return nil
	}
	return ptr(*p)
}
