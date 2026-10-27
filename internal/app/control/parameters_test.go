package control

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

type parameterStore struct {
	defs    []params.ParamDef
	set     params.Set
	failure error
	writes  int
}

func (s *parameterStore) Snapshot() params.Set           { return s.set }
func (s *parameterStore) Definitions() []params.ParamDef { return s.defs }
func (*parameterStore) Overridden(string, string) bool   { return true }
func (s *parameterStore) SetOverride(ctx context.Context, key, position string, value float64, reason string) error {
	s.writes++
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("missing deadline")
	}
	if key != "x" || position != "QB" || value != 3 || reason != "admin console" {
		return errors.New("wrong override")
	}
	return s.failure
}
func (s *parameterStore) ClearOverride(ctx context.Context, _, _ string) error {
	s.writes++
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("missing deadline")
	}
	return s.failure
}

func TestParameterReadsAndWritesPreserveAdmission(t *testing.T) {
	t.Parallel()
	failure := errors.New("startup")
	never := func() ParameterStore { t.Fatal("store read before startup"); return nil }
	if out := ReadParams(func() error { return failure }, never); out.OK || out.Error != "startup" {
		t.Fatal(out)
	}
	parent := func() context.Context { return t.Context() }
	if out := SetParameter(parent, func() error { return failure }, never, "x", "QB", 3); out.OK || out.Error != "startup" {
		t.Fatal(out)
	}
	if out := ResetParameter(parent, func() error { return failure }, never, "x", "QB"); out.OK {
		t.Fatal(out)
	}
	store := &parameterStore{defs: []params.ParamDef{{Key: "x", Position: "QB", Description: "X"}}, set: params.SetOf(map[string]float64{"x@QB": 3})}
	source := func() ParameterStore { return store }
	ready := func() error { return nil }
	out := ReadParams(ready, source)
	if !out.OK || len(out.Params) != 1 || out.Params[0].Value != 3 || !out.Params[0].Overridden {
		t.Fatal(out)
	}
	store.set = params.Set{}
	if out := ReadParams(ready, source); out.OK || out.Error == "" {
		t.Fatal("missing effective value hidden")
	}
	if out := SetParameter(parent, ready, source, "x", "QB", 3); !out.OK {
		t.Fatal(out)
	}
	store.failure = failure
	if out := ResetParameter(parent, ready, source, "x", "QB"); out.OK || out.Error != "startup" {
		t.Fatal(out)
	}
	if store.writes != 2 {
		t.Fatalf("unexpected writes %d", store.writes)
	}
}

func TestAccountingAdmissionBeforeHostAndOpen(t *testing.T) {
	t.Parallel()
	failure := errors.New("startup")
	host := func() string { t.Fatal("read host before ready"); return "" }
	if err := OpenAccounting(func() error { return failure }, host, nil, nil); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	opened := false
	if err := OpenAccounting(func() error { return nil }, func() string { return "" }, func() int { return 2026 }, func(string) { opened = true }); err == nil || opened {
		t.Fatal("unknown host opened")
	}
}

func TestAlertGatheringOrderAndEarlyRefusal(t *testing.T) {
	t.Parallel()
	failure := errors.New("lineup failed")
	if _, err := GatherAlerts(AlertSource{Lineup: func() error { return failure }}); err != failure {
		t.Fatal(err)
	}
	var calls []string
	at := time.Now()
	add := func(name string) func(*AlertReading, time.Time) {
		return func(_ *AlertReading, now time.Time) {
			if !now.Equal(at) {
				t.Fatal("clock resampled")
			}
			calls = append(calls, name)
		}
	}
	got, err := GatherAlerts(AlertSource{Lineup: func() error { calls = append(calls, "lineup"); return nil }, Trades: func() error { calls = append(calls, "trades"); return nil }, Now: func() time.Time { calls = append(calls, "now"); return at }, IR: func(*AlertReading) { calls = append(calls, "ir") }, SavedLineup: add("saved"), Byes: add("bye"), TradeOffers: add("offers")})
	if err != nil || got.Alerts == nil || got.Unavailable == nil {
		t.Fatal("empty slice contract lost")
	}
	want := []string{"lineup", "trades", "now", "ir", "saved", "bye", "offers"}
	for i, name := range want {
		if len(calls) != len(want) || calls[i] != name {
			t.Fatal(calls)
		}
	}
}
