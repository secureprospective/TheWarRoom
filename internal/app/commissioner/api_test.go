package commissioner

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	commish "github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// Unimplemented ports panic, so an unexpected effect cannot make a refusal test pass.
type orderedHost struct {
	Host
	calls             []string
	startup, identity error
	raw               commish.RawCommand
	principal         string
}

func (h *orderedHost) Ready() error { h.calls = append(h.calls, "ready"); return h.startup }
func (h *orderedHost) CommissionerPrincipal() (string, error) {
	h.calls = append(h.calls, "principal")
	return "caller", h.identity
}
func (h *orderedHost) CommissionerContext() context.Context {
	h.calls = append(h.calls, "context")
	return context.Background()
}
func (h *orderedHost) CommissionerAuthorization() commish.Authorizer {
	h.calls = append(h.calls, "authorization")
	return nil
}
func (h *orderedHost) NativeFacts() nativecheck.Facts {
	h.calls = append(h.calls, "facts")
	return nil
}
func (h *orderedHost) CommissionerZone() *time.Location {
	h.calls = append(h.calls, "zone")
	return time.UTC
}
func (h *orderedHost) ExecuteCommissionerWithFacts(ctx context.Context, raw commish.RawCommand,
	principal string, _ commish.Authorizer, _ nativecheck.Facts, zone *time.Location) (commish.Receipt, error) {
	h.calls = append(h.calls, "execute")
	h.raw, h.principal = raw, principal
	if _, bounded := ctx.Deadline(); !bounded || zone != time.UTC {
		panic("lost operation context or zone")
	}
	return commish.Receipt{Entry: "recorded", Revision: 1}, nil
}
func createBody(t *testing.T) string {
	t.Helper()
	body, err := json.Marshal(commish.RawCommand{League: "native.test", Command: commish.Create,
		Args:    commish.RawArgs{Name: "Test league", Preset: commish.PresetEmpty, Season: 2026, FranchiseCount: 4},
		Request: "create", Reason: "Create this league", Effective: "now", Visibility: "league"})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
func TestExecuteKeepsIdentityAndOneTransactionalWriteInOrder(t *testing.T) {
	t.Parallel()
	h := &orderedHost{}
	receipt, err := New(h).TargetCommissionerExecute(createBody(t))
	want := []string{"ready", "principal", "context", "authorization", "facts", "zone", "execute"}
	if err != nil || receipt.Entry != "recorded" || h.principal != "caller" ||
		h.raw.Request != "create" || !reflect.DeepEqual(h.calls, want) {
		t.Fatal(receipt, err, h.calls, h.raw)
	}
}
func TestExecuteRefusesBeforeAnyLaterCapability(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name              string
		body              string
		startup, identity error
		want              []string
	}{
		{"decode", `{}`, nil, nil, nil},
		{"startup", createBody(t), errors.New("not started"), nil, []string{"ready"}},
		{"session", createBody(t), nil, errors.New("unreadable identity"), []string{"ready", "principal"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := &orderedHost{startup: tc.startup, identity: tc.identity}
			if _, err := New(h).TargetCommissionerExecute(tc.body); err == nil {
				t.Fatal("refusal lost")
			}
			if !reflect.DeepEqual(h.calls, tc.want) {
				t.Fatal(h.calls, tc.want)
			}
		})
	}
}
func TestReadRefusalsKeepScopeStartupAndIdentityOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		run  func(*API) error
		want []string
	}{
		{"ledger scope", func(a *API) error { _, e := a.TargetCommissionerLedger("mfl.1"); return e }, nil},
		{"people scope", func(a *API) error { _, e := a.TargetCommissionerPeople("native."); return e }, nil},
		{"rules scope", func(a *API) error { _, e := a.TargetCommissionerRules("mfl.1"); return e }, nil},
		{"filter decode", func(a *API) error {
			_, e := a.TargetCommissionerLedgerFiltered("native.x",
				`{`)
			return e
		}, nil},
		{"actor", func(a *API) error { _, e := a.TargetCommissionerActor(); return e }, []string{"ready",
			"principal"}},
		{"ledger", func(a *API) error { _, e := a.TargetCommissionerLedger("native."); return e },
			[]string{"ready", "principal"}},
		{"people", func(a *API) error { _, e := a.TargetCommissionerPeople("native.x"); return e },
			[]string{"ready", "principal"}},
		{"market", func(a *API) error {
			_, e := a.TargetCommissionerMarket("native.x",
				"waivers.process")
			return e
		}, []string{"ready", "principal"}},
		{"schedule", func(a *API) error { _, e := a.TargetCommissionerSchedule("native.x"); return e },
			[]string{"ready", "principal"}},
		{"digests", func(a *API) error { _, e := a.TargetCommissionerDigests("native.x"); return e },
			[]string{"ready", "principal"}},
		{"trades", func(a *API) error { _, e := a.TargetNativeTrades("native.x", "franchise.01"); return e },
			[]string{"ready", "principal"}},
		{"bids", func(a *API) error { _, e := a.TargetNativeBids("native.x", "franchise.01"); return e },
			[]string{"ready", "principal"}},
		{"waivers", func(a *API) error {
			_, e := a.TargetNativeWaivers("native.x", "franchise.01",
				time.Now())
			return e
		}, []string{"ready", "principal"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := &orderedHost{identity: errors.New("identity refused")}
			if err := tc.run(New(h)); err == nil {
				t.Fatal("refusal lost")
			}
			if !reflect.DeepEqual(h.calls, tc.want) {
				t.Fatal(h.calls, tc.want)
			}
		})
	}
}
