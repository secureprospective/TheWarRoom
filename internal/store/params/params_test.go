package params

import (
	"context"
	"math"
	"path/filepath"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// openStore returns a freshly seeded store over a temp SQLite DB plus the underlying
// pools (so a test can re-open a second store on the SAME database to prove restart
// behaviour).
func openStore(t *testing.T) (*Store, *db.Pools) {
	t.Helper()
	pools := openPools(t, filepath.Join(t.TempDir(), "params.db"))
	s := New(pools)
	if err := s.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	return s, pools
}

func openPools(t *testing.T, path string) *db.Pools {
	t.Helper()
	pools, err := db.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	return pools
}

func TestInitializeSeedsDefaults(t *testing.T) {
	s, _ := openStore(t)

	tiers, err := s.GetCapTiers()
	if err != nil {
		t.Fatalf("GetCapTiers: %v", err)
	}
	if tiers.ColdCeiling != 1.2 || tiers.HotFloor != 4.8 {
		t.Fatalf("cap tiers = %+v, want {1.2 4.8}", tiers)
	}
	decay, err := s.GetGlobal(KeyLayer3DecayRate)
	if err != nil {
		t.Fatalf("GetGlobal decay: %v", err)
	}
	if decay != 0.03 {
		t.Fatalf("decay = %g, want 0.03", decay)
	}
}

func TestDefinitionsShipUncalibrated(t *testing.T) {
	s, _ := openStore(t)
	defs := s.Definitions()
	if len(defs) != len(defaultParams()) {
		t.Fatalf("got %d defs, want %d", len(defs), len(defaultParams()))
	}
	for _, d := range defs {
		if d.IsCalibrated {
			t.Errorf("param %q shipped IsCalibrated=true, want false (OQ-006 not yet tuned)", d.Key)
		}
	}
	// Definitions is sorted by (key, position) for determinism.
	for i := 1; i < len(defs); i++ {
		if defs[i-1].Key > defs[i].Key {
			t.Fatalf("Definitions not sorted: %q before %q", defs[i-1].Key, defs[i].Key)
		}
	}
}

func TestSetOverrideAppliesAndIsRangeAware(t *testing.T) {
	s, _ := openStore(t)
	ctx := context.Background()

	if err := s.SetOverride(ctx, KeyCapTierColdCeiling, "", 2.0, "tuned up"); err != nil {
		t.Fatalf("SetOverride: %v", err)
	}
	tiers, err := s.GetCapTiers()
	if err != nil {
		t.Fatalf("GetCapTiers: %v", err)
	}
	if tiers.ColdCeiling != 2.0 {
		t.Fatalf("ColdCeiling = %g, want overridden 2.0", tiers.ColdCeiling)
	}
	// The un-overridden boundary still reads its default.
	if tiers.HotFloor != 4.8 {
		t.Fatalf("HotFloor = %g, want default 4.8", tiers.HotFloor)
	}
}

// TestOverrideOutOfRangeRejected is the M3 planted-failure gate: a value beyond the
// shipped [Min,Max] must be rejected AND must not be written.
func TestOverrideOutOfRangeRejected(t *testing.T) {
	s, _ := openStore(t)
	ctx := context.Background()

	// decay rate is bounded [0,1]; 1.5 is out of range.
	if err := s.SetOverride(ctx, KeyLayer3DecayRate, "", 1.5, "bad"); err == nil {
		t.Fatal("out-of-range override accepted, want rejection")
	}
	decay, err := s.GetGlobal(KeyLayer3DecayRate)
	if err != nil {
		t.Fatalf("GetGlobal: %v", err)
	}
	if decay != 0.03 {
		t.Fatalf("decay = %g after rejected override, want unchanged 0.03", decay)
	}
}

// TestOverrideBoundaryAccepted confirms the range gate is INCLUSIVE: a value exactly
// on either bound is valid.
func TestOverrideBoundaryAccepted(t *testing.T) {
	s, _ := openStore(t)
	ctx := context.Background()
	// decay rate bounds are [0,1]; both boundaries must be accepted.
	if err := s.SetOverride(ctx, KeyLayer3DecayRate, "", 0, "floor"); err != nil {
		t.Fatalf("lower-boundary override rejected: %v", err)
	}
	if err := s.SetOverride(ctx, KeyLayer3DecayRate, "", 1, "ceiling"); err != nil {
		t.Fatalf("upper-boundary override rejected: %v", err)
	}
}

// TestOverrideNonFiniteRejected is the M3 gate for NaN/±Inf: a NaN passes both range
// comparisons (unordered) and would round-trip silently into calibration, so it must
// be rejected explicitly and leave the shipped default untouched.
func TestOverrideNonFiniteRejected(t *testing.T) {
	s, _ := openStore(t)
	ctx := context.Background()
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if err := s.SetOverride(ctx, KeyCapTierColdCeiling, "", v, "bad"); err == nil {
			t.Fatalf("non-finite override %v accepted, want rejection", v)
		}
	}
	tiers, err := s.GetCapTiers()
	if err != nil {
		t.Fatalf("GetCapTiers: %v", err)
	}
	if tiers.ColdCeiling != 1.2 {
		t.Fatalf("ColdCeiling = %g after rejected non-finite, want unchanged 1.2", tiers.ColdCeiling)
	}
}

// TestOverrideUnknownParamRejected is the M3 gate for a non-existent target.
func TestOverrideUnknownParamRejected(t *testing.T) {
	s, _ := openStore(t)
	if err := s.SetOverride(context.Background(), "nope.not_a_param", "", 1, ""); err == nil {
		t.Fatal("override of unknown parameter accepted, want rejection")
	}
}

// TestInitializeDoesNotReseed proves the stability invariant: re-opening the store on
// an existing DB loads the persisted override and does NOT wipe it back to the default
// (and does not duplicate the seed).
func TestInitializeDoesNotReseed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "persist.db")
	ctx := context.Background()

	pools := openPools(t, path)
	s1 := New(pools)
	if err := s1.Initialize(ctx); err != nil {
		t.Fatalf("first Initialize: %v", err)
	}
	if err := s1.SetOverride(ctx, KeyCushionGuardRAS, "", 9.0, "tuned"); err != nil {
		t.Fatalf("SetOverride: %v", err)
	}

	// Re-open a fresh store on the SAME database (a restart).
	s2 := New(pools)
	if err := s2.Initialize(ctx); err != nil {
		t.Fatalf("second Initialize: %v", err)
	}
	ras, err := s2.GetGlobal(KeyCushionGuardRAS)
	if err != nil {
		t.Fatalf("GetGlobal: %v", err)
	}
	if ras != 9.0 {
		t.Fatalf("RAS = %g after restart, want surviving override 9.0", ras)
	}
	if got := len(s2.Definitions()); got != len(defaultParams()) {
		t.Fatalf("reseed duplicated defaults: %d defs, want %d", got, len(defaultParams()))
	}
}

// TestConcurrentReadsAndOverride exercises the two-lock idiom under -race.
func TestConcurrentReadsAndOverride(t *testing.T) {
	s, _ := openStore(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _ = s.GetCapTiers()
				_, _ = s.GetGlobal(KeyCushionGuardReduct)
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 50; j++ {
			_ = s.SetOverride(ctx, KeyCapTierHotFloor, "", 5.0, "")
		}
	}()
	wg.Wait()
}
