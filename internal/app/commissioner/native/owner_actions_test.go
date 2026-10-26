package native

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestOwnerFamiliesRefuseBeforeExecuting(t *testing.T) {
	t.Parallel()
	for _, family := range []OwnerFamily{TradeFamily(), WaiverFamily(), BidFamily()} {
		for _, body := range []string{`{`, `{}`, `{"league":"other","command":"trade.offer","args":{"trade":{"franchise":"other"}}}`} {
			calls := 0
			_, err := RecordOwnerAction(family)(t.Context(), func(context.Context, commissioner.RawCommand) (commissioner.Receipt, error) {
				calls++
				return commissioner.Receipt{}, nil
			}, commissioner.Permit{}, json.RawMessage(body))
			if err == nil || calls != 0 {
				t.Fatalf("unpermitted owner command executed: %s %d %v", body, calls, err)
			}
		}
	}
}

func TestOwnerActionAdmissionOrder(t *testing.T) {
	t.Parallel()
	failure := errors.New("not ready")
	calls := 0
	source := OwnerGate{Ready: func() error { calls++; return failure }}
	actions := []OwnerAction{{Name: "known"}}
	if _, err := RunOwnerAction(t.Context(), actions, "l", "f", "missing", nil, source); err == nil || calls != 0 {
		t.Fatalf("unknown action consulted startup: %v %d", err, calls)
	}
	if _, err := RunOwnerAction(t.Context(), actions, "l", "f", "known", nil, source); !errors.Is(err, failure) || calls != 1 {
		t.Fatalf("startup refusal lost: %v %d", err, calls)
	}
	source.Ready = func() error { return nil }
	source.Records = func(context.Context, string) ([]commissioner.Record, error) { return nil, failure }
	if _, err := RunOwnerAction(t.Context(), actions, "l", "f", "known", nil, source); err != failure {
		t.Fatalf("calendar refusal changed: %v", err)
	}
	for _, action := range ContractOwnerActions() {
		window, ok := commissioner.OwnerContractWindow(action.Action)
		if !ok || window != action.Opens || action.Name != "contract."+string(action.Action) {
			t.Fatalf("ungated contract: %+v", action)
		}
	}
}

func TestOwnerRequestsValidateBeforeIdentity(t *testing.T) {
	t.Parallel()
	principal := func() (string, error) { t.Fatal("invalid input read identity"); return "", nil }
	if _, err := RecordDraftPick(t.Context(), commissioner.Permit{}, json.RawMessage(`{}`), principal, nil); err == nil {
		t.Fatal("empty pick accepted")
	}
	if _, err := RecordLineup(t.Context(), commissioner.Permit{}, json.RawMessage(`{}`), principal, nil); err == nil {
		t.Fatal("empty lineup accepted")
	}
	var logged string
	CloseFinishedDrafts(t.Context(), "league", func(context.Context, string) error { return errors.New("close failed") }, func(s string) { logged = s })
	if !strings.Contains(logged, "league: close failed") {
		t.Fatalf("close error lost: %q", logged)
	}
}

func TestHeldLineupFactsAndOwnerZone(t *testing.T) {
	t.Parallel()
	zone := time.FixedZone("owner", 3600)
	if OwnerZone(zone) != zone || OwnerZone(nil) != time.Local {
		t.Fatal("zone fallback changed")
	}
	facts := nativecheck.Facts{"0099": {Name: "Player"}}
	got := HeldLineupFacts([]leagueweek.Week{{Number: 2}}, 1, facts, zone)
	if got.ScheduleHeld || got.Zone != zone || got.Facts["0099"].Name != "Player" {
		t.Fatalf("invented held week: %+v", got)
	}
	got = HeldLineupFacts([]leagueweek.Week{{Number: 2}}, 2, facts, zone)
	if !got.ScheduleHeld {
		t.Fatal("selected week was not held")
	}
}

func TestBidEnvironmentStopsAtRecordFailure(t *testing.T) {
	t.Parallel()
	failure := errors.New("record read")
	_, err := ReadBidEnvironment(t.Context(), BidSource{Records: func(context.Context, string) ([]commissioner.Record, error) { return nil, failure }}, "l", time.Now())
	if err != failure {
		t.Fatalf("record failure changed: %v", err)
	}
}
