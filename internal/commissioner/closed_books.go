package commissioner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

// RawClosedBooks is a rebuildable fold through a recorded rollover, inclusive.
// Rules already have their frozen before/after states in that receipt.
type RawClosedBooks struct {
	Version      int
	Revision     int
	Season       int
	Entry        string
	SourceHash   string
	Projection   Projection
	People       governance.RawCheckpoint
	PeopleEvents int
	Native       NativeAssets
	Waivers      waiverState
}

func (r RawClosedBooks) Validate() error {
	if r.Version != 1 || r.Revision < 1 || r.Season < 1 || r.Entry == "" || len(r.SourceHash) != 64 || r.Projection.Active == nil ||
		r.Projection.Creator == "" || len(r.Projection.Active) != r.Revision || r.PeopleEvents < 1 {
		return fmt.Errorf("invalid closed season summary")
	}
	if err := r.People.Validate(); err != nil {
		return fmt.Errorf("invalid closed season people: %w", err)
	}
	return r.validateNative()
}

// ClosedBooksHash binds the summary to the authoritative prefix, including receipts.
func ClosedBooksHash(records []Record) (string, error) {
	body, err := json.Marshal(records)
	if err != nil {
		return "", fmt.Errorf("hash closed season: %w", err)
	}
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:]), nil
}

func CloseBooks(records []Record) (RawClosedBooks, error) {
	if len(records) == 0 || !recordedRollover(records[len(records)-1]) {
		return RawClosedBooks{}, fmt.Errorf("close books only at a recorded rollover")
	}
	state, err := GovernanceState(records)
	if err != nil {
		return RawClosedBooks{}, err
	}
	hash, err := ClosedBooksHash(records)
	if err != nil {
		return RawClosedBooks{}, err
	}
	projection := Replay(records)
	events, err := governanceEvents(records, projection)
	if err != nil {
		return RawClosedBooks{}, err
	}
	assets, err := seedNativeAssets(records[0].Command.League, records)
	if err != nil {
		return RawClosedBooks{}, err
	}
	waivers := waiverState{Windows: []waiverWindow{}, Order: []string{}}
	if err := projectNativeOverrides(&assets, records, &waivers, nil); err != nil {
		return RawClosedBooks{}, err
	}
	last := records[len(records)-1]
	raw := RawClosedBooks{Version: 1, Revision: len(records), Season: last.Receipt.Plan.From, Entry: last.Receipt.Entry,
		SourceHash: hash, Projection: projection, People: state.Checkpoint(), PeopleEvents: len(events), Native: assets, Waivers: waivers}
	if err := raw.Validate(); err != nil {
		return RawClosedBooks{}, err
	}
	return raw, nil
}

type closedBooks struct {
	raw        RawClosedBooks
	keys       []foldKey
	franchises []string
}

// foldKey guards the folds against callers editing their private record copies.
// Rule bodies do not affect identity/compensation or governance; they still resolve separately.
type foldKey struct {
	entry, command, principal, name, target, correction, restore string
	at                                                           time.Time
	people                                                       RawPeople
	hasPeople                                                    bool
}

func keyForFold(r Record) foldKey {
	k := foldKey{entry: r.Receipt.Entry, command: r.Command.Command, principal: r.Principal, at: r.Receipt.RecordedAt,
		name: r.Command.Args.Name, target: r.Command.Args.Entry}
	if p := r.Command.Args.People; p != nil {
		k.people, k.hasPeople = *p, true
		k.people.Order = slices.Clone(p.Order)
	}
	if c := r.Command.Args.Correction; c != nil {
		k.correction = c.Entry
	}
	if d := r.Command.Args.Draft; d != nil && d.Act != nil {
		k.restore = d.Act.To
	}
	return k
}

// WithClosedBooks returns an independently owned outer slice. A rejected summary
// leaves full replay in place; it can never replace the authoritative records.
func WithClosedBooks(records []Record, raw RawClosedBooks) ([]Record, bool) {
	out := WithoutClosedBooks(records)
	if raw.Validate() != nil || raw.Revision > len(records) {
		return out, false
	}
	prefix := out[:raw.Revision]
	last := prefix[len(prefix)-1]
	if !recordedRollover(last) || last.Receipt.Entry != raw.Entry || last.Receipt.Plan.From != raw.Season {
		return out, false
	}
	hash, err := ClosedBooksHash(prefix)
	if err != nil || hash != raw.SourceHash {
		return out, false
	}
	// The decoded payload belongs to this immutable memo, not its caller.
	state, err := governance.Resume(raw.People, nil, 0)
	if err != nil {
		return out, false
	}
	raw.People = state.Checkpoint()
	raw.Projection.Active = maps.Clone(raw.Projection.Active)
	native, waivers, err := cloneClosedNative(raw)
	if err != nil {
		return out, false
	}
	raw.Native, raw.Waivers = native, waivers
	b := &closedBooks{raw: raw, keys: make([]foldKey, len(prefix)), franchises: slices.Clone(prefix[0].Command.Args.Franchises)}
	for i, r := range prefix {
		b.keys[i] = keyForFold(r)
	}
	out[0].books = b
	return out, true
}

// WithoutClosedBooks selects authoritative replay without changing the caller's slice.
func WithoutClosedBooks(records []Record) []Record {
	out := slices.Clone(records)
	if len(out) > 0 {
		out[0].books = nil
	}
	return out
}

func usableClosedBooks(records []Record) *closedBooks {
	if len(records) == 0 {
		return nil
	}
	b := records[0].books
	if b == nil || len(records) < len(b.keys) || !slices.Equal(records[0].Command.Args.Franchises, b.franchises) {
		return nil
	}
	for i, key := range b.keys {
		if !reflect.DeepEqual(keyForFold(records[i]), key) {
			return nil
		}
	}
	for _, r := range records[len(b.keys):] {
		// A later compensation may change a folded event, including a reversal of
		// a reversal. Rebuild rather than infer that seasons isolate every link.
		switch r.Command.Command {
		case Revert:
			if _, ok := b.raw.Projection.Active[r.Command.Args.Entry]; ok {
				return nil
			}
		case CorrectTransaction:
			if r.Command.Args.Correction != nil {
				if _, ok := b.raw.Projection.Active[r.Command.Args.Correction.Entry]; ok {
					return nil
				}
			}
		case DraftRevert:
			return nil
		}
	}
	return b
}

func replayClosedBooks(records []Record, b *closedBooks) Projection {
	p := b.raw.Projection
	p.Active = maps.Clone(p.Active)
	tail := records[len(b.keys):]
	next := replayFull(tail)
	maps.Copy(p.Active, next.Active)
	for _, r := range tail {
		if r.Command.Command == Create {
			p.Creator, p.Name = next.Creator, next.Name
		}
	}
	return p
}
