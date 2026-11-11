// Package params stores the engine's calibration parameters (cap-tier percentages and other
// tunables) as shipped defaults plus an admin override layer applied at read time. Storage is a
// generic (key, position) table with a [Min,Max] range per row, so adding a parameter is a data
// row, not code. The public surface includes GetGlobal.
//
// It holds percentages and rates only; the cap amount belongs to the rulebook. Writes are
// admin-only and never go through the transaction coordinator.
package params

import (
	"context"
	"fmt"
	"maps"
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

// Initialize ensures the schema, writes the shipped defaults, and loads defaults and overrides
// into memory. Overrides are never touched.
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

// Overridden reports whether an admin override is in effect for (key, position), even one equal
// to the default: an override pins the value against later shipped defaults.
func (s *Store) Overridden(key, position string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.overrides[defKey(key, position)]
	return ok
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

// The board and the measurables model read disjoint parts of a set: the model its fitted values
// and the dynasty horizon, the board everything else. A run records only its own part, so editing
// one side's setting never makes a new run of the other.

// IsModel reports whether key, in either form Values uses, belongs to the model.
func IsModel(key string) bool {
	return strings.HasPrefix(key, "model.") || strings.HasPrefix(key, "dynasty.")
}

// Board is Values without the model's part.
func (p Set) Board() map[string]float64 { return p.part(false) }

// Model is Values limited to the model's part.
func (p Set) Model() map[string]float64 { return p.part(true) }

func (p Set) part(model bool) map[string]float64 {
	out := p.Values()
	maps.DeleteFunc(out, func(k string, _ float64) bool { return IsModel(k) != model })
	return out
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

// ValueType says how a stored value is parsed and bounded. Only floats exist today.
type ValueType string

// TypeFloat is a float64 parameter validated against an inclusive [Min,Max] range.
const TypeFloat ValueType = "float"

// global is the Position of a league-wide parameter. A per-position parameter is one row per
// position under the same Key.
const global = ""

// Parameter keys. The engine reads by these constants.
const (
	// KeyLayer3DecayRate: the annual age-decay rate.
	KeyLayer3DecayRate = "layer3.decay_rate"
	// KeyDynastySeasons: how many seasons the dynasty measurable counts, this one first.
	KeyDynastySeasons = "dynasty.seasons"
	// KeyDynastyDiscount: each later season's weight in the dynasty measurable.
	KeyDynastyDiscount   = "dynasty.discount"
	KeyPickValueDiscount = "pickvalue.discount"
	KeyTradeKeep         = "trade.keep" // fitted per position (tools/fit)
	KeyPowerPickWeight   = "power.pick_weight"
	// KeyDynastyContractTilt: the most a contract's price and length move a dynasty value.
	KeyDynastyContractTilt = "dynasty.contract_tilt"
)

// ParamDef is one parameter: its identity (Key, Position), type, shipped default and the
// inclusive range an override must fall in. IsCalibrated is false while the value is still a
// placeholder.
type ParamDef struct {
	Key          string
	Position     string
	Type         ValueType
	Default      float64
	Min          float64
	Max          float64
	IsCalibrated bool
	Description  string
}

// Override is an admin value layered over a shipped default; a release may replace the default,
// never the override.
// Clearing an override is not implemented yet.
type Override struct {
	Key       string
	Position  string
	Value     float64
	Note      string
	UpdatedAt string
}

// defKey is the in-memory map key for a (parameter, position) pair.
func defKey(key, position string) string { return key + "\x00" + position }

// defaultParams returns the shipped settings and fitted values.
func defaultParams() []ParamDef {
	return append([]ParamDef{
		{
			Key: KeyLayer3DecayRate, Position: global, Type: TypeFloat,
			Default: 0.03, Min: 0, Max: 1,
			Description: "Layer 3: annual age-decay rate",
		},
		{
			Key: KeyDynastySeasons, Position: global, Type: TypeFloat,
			Default: 3, Min: 1, Max: 10,
			Description: "Dynasty: seasons counted, this one first (whole seasons)",
		},
		{
			Key: KeyDynastyDiscount, Position: global, Type: TypeFloat,
			Default: 0.75, Min: 0, Max: 1,
			Description: "Dynasty: weight of each later season against the one before",
		},
		{
			Key: KeyPickValueDiscount, Position: global, Type: TypeFloat,
			Default: 0.75, Min: 0, Max: 1,
			Description: "Pick Value: weight of each later draft against the one before",
		},
		{
			Key: KeyPowerPickWeight, Position: global, Type: TypeFloat,
			Default: 0.6, Min: 0, Max: 1,
			Description: "Power rankings: how much a future draft pick counts in the franchise view",
		},
		{
			Key: KeyDynastyContractTilt, Position: global, Type: TypeFloat,
			Default: 0.06, Min: 0, Max: 0.25,
			Description: "Dynasty: most a contract's price and length move a value, as a share",
		},
	}, fittedParams()...)
}
