package composition

import (
	"errors"
	"net/http"
	"path/filepath"
	"reflect"
	"testing"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/mflref/contracts"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

type noMFLTransport struct{ calls, writes int }

func (s *noMFLTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	s.calls++
	if r.Method != http.MethodGet || r.URL.Path != "/2026/export" {
		s.writes++
	}
	return nil, errors.New("local plans and native commands must not contact MFL")
}

func TestMFLWhatIfAndNativeRulesRemainSeparateWithoutMFLWrites(t *testing.T) {
	t.Parallel()
	f := quoteApp(t)
	a := f.app
	a.core.MyFranchise = "0002"
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "whatif.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	}()
	a.core.WhatifPools = pools
	transport := &noMFLTransport{}
	a.core.MFLClient, err = mfl.New("api", 10000, mfl.WithTransport(transport))
	if err != nil {
		t.Fatal(err)
	}
	before := readNativeTest(t, a, legalityLeague)
	mirror, stamp, used := a.core.League.ContractsView()
	simulated := a.contracts().TargetWhatIf(contractsapp.RawWhatIfRequest{Mine: "0002", Action: "cut", Player: "0099"})
	if simulated.Status != report.ReadingOK || len(simulated.Value.Moves) != 1 {
		t.Fatalf("simulation: %+v", simulated)
	}
	if after := readNativeTest(t, a, legalityLeague); !reflect.DeepEqual(before, after) {
		t.Fatal("MFL simulation changed native assets")
	}
	f.set(t, "cap.amount", `2000000000`, "now")
	reread := a.contracts().TargetWhatIf(contractsapp.RawWhatIfRequest{Mine: "0002", Action: "read"})
	if reread.Status != report.ReadingOK || !reflect.DeepEqual(simulated.Value, reread.Value) {
		t.Fatalf("native rule changed MFL simulation: %+v", reread)
	}
	after, afterStamp, afterUsed := a.core.League.ContractsView()
	if !reflect.DeepEqual(mirror, after) || stamp != afterStamp || !reflect.DeepEqual(used, afterUsed) {
		t.Fatal("local work changed source mirror")
	}
	if transport.calls != 0 || transport.writes != 0 {
		t.Fatalf("MFL calls %d; writes %d", transport.calls, transport.writes)
	}
}
