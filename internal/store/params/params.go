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

// Initialize ensures the schema, seeds the shipped defaults on a fresh database, and loads
// defaults and overrides into memory.
func (s *Store) Initialize(ctx context.Context) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if err := s.initSchema(ctx); err != nil {
		return err
	}
	seeded, err := s.hasDefaults(ctx)
	if err != nil {
		return err
	}
	if !seeded {
		if err := s.seedDefaults(ctx); err != nil {
			return err
		}
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

// value returns the effective value for a parameter under a single read lock.
func (s *Store) value(key, position string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.effectiveLocked(key, position)
}

// GetCapTiers returns the cap-tier boundaries, override applied. Both are read under one lock
// so the pair always comes from one snapshot.
func (s *Store) GetCapTiers() (CapTiers, error) {
	s.mu.RLock()
	cold, coldOK := s.effectiveLocked(KeyCapTierColdCeiling, global)
	hot, hotOK := s.effectiveLocked(KeyCapTierHotFloor, global)
	s.mu.RUnlock()
	if !coldOK {
		return CapTiers{}, fmt.Errorf("params: missing %s", KeyCapTierColdCeiling)
	}
	if !hotOK {
		return CapTiers{}, fmt.Errorf("params: missing %s", KeyCapTierHotFloor)
	}
	return CapTiers{ColdCeiling: cold, HotFloor: hot}, nil
}

// GetGlobal returns a league-wide parameter, override applied. An unknown key is an error,
// never a silent 0.
func (s *Store) GetGlobal(key string) (float64, error) {
	v, ok := s.value(key, global)
	if !ok {
		return 0, fmt.Errorf("params: unknown global parameter %q", key)
	}
	return v, nil
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
