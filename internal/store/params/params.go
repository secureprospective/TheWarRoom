// Package params is the B4 Admin Parameter Store: the third Layer-2 store and the
// last piece of the Layer-2 config floor. It holds the engine's CALIBRATION
// parameters (cap-tier percentages and other tunables) SQLite-backed, served to the
// scoring engine and tuned by the admin calibration surface (M9a).
//
// It is a CONFIG store like the rulebook (B3b) — versioned-stability is replaced by
// the simpler shipped-defaults model: there is NO external fetch source. Initialize
// seeds built-in Go DEFAULTS once on a fresh DB and loads what exists on restart
// (stability, like B3b/B3c). An admin OVERRIDE layer is layered over a default at
// read time; the override write path is admin-only and is NEVER routed through B7
// (AD-05).
//
// Shape (GLM-5.2 review): storage is a GENERIC (key, position) typed key/value table
// with a per-row [Min,Max] range; the public surface is TYPED (GetCapTiers, GetGlobal).
// Adding a parameter is a data row, not new code — so per-position tables fold in with
// their engine layer without reshaping the store.
//
// It is PURE DATA ACCESS: it stores and serves calibration values and runs NO engine
// logic. AD-21: it holds cap-tier PERCENTAGES (calibration); the cap AMOUNT is the
// rulebook store's (config) — never duplicated here.
package params

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// Store is the admin parameter store. Construct with New; seed with Initialize. It is
// safe for concurrent reads via mu (the reader/snapshot lock); admin mutations
// serialize under wmu (the outer write lock), so each DB write and its in-memory
// reload are one atomic step — the two-lock idiom proven in B3b/B3c.
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

// Initialize ensures the schema, seeds the shipped defaults ONCE on a fresh database,
// and loads defaults + overrides into memory. On an existing database it loads what is
// there WITHOUT reseeding — an operator's calibration is never overwritten by a newer
// shipped default on restart (the B3b/B3c stability tradeoff; adding params in a later
// version is a migration, not a silent reseed).
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

// effectiveLocked returns the effective value for a parameter — the admin override if
// present, otherwise the shipped default — assuming the caller already holds mu (read
// or write). ok is false for an unknown (key, position).
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

// GetCapTiers returns the Layer-5 cap-scaling boundaries (AD-21), override-aware. Both
// boundaries are read under ONE lock hold so a concurrent load() swap can never return
// a Cold/Hot pair straddling two snapshots (the boundaries move as a pair).
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

// GetGlobal returns a global (non-per-position) scalar parameter, override-aware. It
// is the generic typed accessor the engine uses for single-value calibration params
// (the engine passes a Key* constant). An unknown key is an error, never a silent 0.
func (s *Store) GetGlobal(key string) (float64, error) {
	v, ok := s.value(key, global)
	if !ok {
		return 0, fmt.Errorf("params: unknown global parameter %q", key)
	}
	return v, nil
}

// Definitions returns a copy of every shipped parameter def (with its IsCalibrated
// flag), sorted for determinism. It backs the M9a admin calibration surface and the
// engine's "how much input is still on placeholder defaults" report.
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
