// Package params stores the engine's calibration parameters (cap-tier percentages and other
// tunables) as shipped defaults plus an admin override layer applied at read time. Storage is a
// generic (key, position) table with a [Min,Max] range per row, so adding a parameter is a data
// row, not code. The public surface is typed (GetCapTiers, GetGlobal).
//
// It holds percentages and rates only; the cap amount belongs to the rulebook. Writes are
// admin-only and never go through the transaction coordinator.
package params

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// Store is the parameter store. Construct with New, seed with Initialize. Reads take mu; admin
// writes take wmu first, so each DB write and its in-memory reload are one step.
type Store struct {
	pools *db.Pools

	wmu sync.Mutex // serializes admin mutations (seed + SetOverride) end to end

	mu        sync.RWMutex
	defs      map[string]ParamDef // keyed defKey(key,position); the immutable shipped set
	overrides map[string]Override // keyed defKey(key,position); the admin layer
}

// New constructs an unseeded store over the given SQLite pools. Call Initialize
// before any read.
func New(pools *db.Pools) *Store {
	return &Store{
		pools:     pools,
		defs:      map[string]ParamDef{},
		overrides: map[string]Override{},
	}
}

// Initialize ensures the schema, adds any shipped default the database lacks, and loads
// defaults and overrides into memory. Existing rows are never rewritten.
func (s *Store) Initialize(ctx context.Context) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if err := s.initSchema(ctx); err != nil {
		return err
	}
	if err := s.seedDefaults(ctx); err != nil {
		return err
	}
	return s.load(ctx)
}

// effectiveLocked returns the override if set, else the default. The caller holds mu. ok is
// false for an unknown (key, position).
func (s *Store) effectiveLocked(key, position string) (float64, bool) {
	k := defKey(key, position)
	if o, ok := s.overrides[k]; ok {
		return o.Value, true
	}
	if d, ok := s.defs[k]; ok {
		return d.Default, true
	}
	return 0, false
}

// Snapshot freezes every effective value, overrides applied. A scoring run reads params only
// through one Snapshot, so the values it used are exactly the values it records.
func (s *Store) Snapshot() Set {
	s.mu.RLock()
	defer s.mu.RUnlock()
	values := make(map[string]float64, len(s.defs))
	for k, d := range s.defs {
		v, _ := s.effectiveLocked(d.Key, d.Position)
		values[k] = v
	}
	return Set{values: values}
}

// Set is a frozen copy of the effective parameters. The zero Set holds nothing.
type Set struct {
	values map[string]float64 // keyed defKey(key, position)
}

// SetOf builds a Set from Values-style keys. It is how a stored or proposed param set is scored.
func SetOf(values map[string]float64) Set {
	out := make(map[string]float64, len(values))
	for k, v := range values {
		key, pos, _ := strings.Cut(k, "@")
		out[defKey(key, pos)] = v
	}
	return Set{values: out}
}

// Values returns the set keyed "key" for a league-wide parameter and "key@POS" for a
// per-position one: the form a scoring run stores.
func (p Set) Values() map[string]float64 {
	out := make(map[string]float64, len(p.values))
	for k, v := range p.values {
		key, pos, _ := strings.Cut(k, "\x00")
		if pos != global {
			key += "@" + pos
		}
		out[key] = v
	}
	return out
}

// GetCapTiers returns the cap-tier boundaries.
func (p Set) GetCapTiers() (CapTiers, error) {
	cold, coldOK := p.values[defKey(KeyCapTierColdCeiling, global)]
	hot, hotOK := p.values[defKey(KeyCapTierHotFloor, global)]
	if !coldOK {
		return CapTiers{}, fmt.Errorf("params: missing %s", KeyCapTierColdCeiling)
	}
	if !hotOK {
		return CapTiers{}, fmt.Errorf("params: missing %s", KeyCapTierHotFloor)
	}
	return CapTiers{ColdCeiling: cold, HotFloor: hot}, nil
}

// GetGlobal returns a league-wide parameter. An unknown key is an error, never a silent 0.
func (p Set) GetGlobal(key string) (float64, error) {
	v, ok := p.values[defKey(key, global)]
	if !ok {
		return 0, fmt.Errorf("params: unknown global parameter %q", key)
	}
	return v, nil
}

// GetPosition returns a parameter at a position, "" for a league-wide one. An unknown key or
// position is an error.
func (p Set) GetPosition(key, position string) (float64, error) {
	v, ok := p.values[defKey(key, position)]
	if !ok {
		return 0, fmt.Errorf("params: unknown parameter %q at %q", key, position)
	}
	return v, nil
}

// DefaultSet is the shipped defaults with no overrides, as a fresh database holds them.
func DefaultSet() Set {
	values := map[string]float64{}
	for _, d := range defaultParams() {
		values[defKey(d.Key, d.Position)] = d.Default
	}
	return Set{values: values}
}

// Definitions returns every shipped parameter, sorted, for the admin console and the
// "still on placeholder defaults" report.
func (s *Store) Definitions() []ParamDef {
	s.mu.RLock()
	out := make([]ParamDef, 0, len(s.defs))
	for _, d := range s.defs {
		out = append(out, d)
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].Key != out[j].Key {
			return out[i].Key < out[j].Key
		}
		return out[i].Position < out[j].Position
	})
	return out
}
