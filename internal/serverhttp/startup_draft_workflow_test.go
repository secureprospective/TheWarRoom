package serverhttp_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestHostedStartupDraftRead(t *testing.T) {
	f := newHostedFixture(t)
	bearer, proof := f.directCommissioner(t)
	client := hostedWorkflowClient{t: t, f: f, bearer: bearer, proof: proof}
	facts := nativecheck.Facts{"1000": {Name: "Pool receiver", Position: "WR"}}
	create := commissioner.RawCommand{League: "native.startup", Command: commissioner.Create,
		Args: commissioner.RawArgs{Name: "Startup test", Preset: commissioner.PresetDefault32,
			Source: commissioner.SourceNative, Season: 2026, FranchiseCount: 32},
		Request: "create", Reason: "Test setup", Effective: "now", Visibility: "league"}
	if _, err := f.history.ExecuteCommissionerWithFacts(t.Context(), create, f.principal, nil, facts, time.UTC); err != nil {
		t.Fatal(err)
	}
	records, err := f.history.SharedCommissionerRecords(t.Context(), create.League)
	if err != nil {
		t.Fatal(err)
	}
	open := commissioner.RawCommand{League: create.League, Command: commissioner.DraftStartupOpen,
		Args:     commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{Season: 2026, Startup: true}},
		Expected: int64(len(records)), Request: "open", Reason: "Test setup", Effective: "now", Visibility: "league"}
	if _, err := f.history.ExecuteCommissionerWithFacts(t.Context(), open, f.principal, nil, facts, time.UTC); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, args string }{
		{"room opens", `{"league":"native.startup","franchise":"franchise.01","season":0,"at":"0001-01-01T00:00:00Z"}`},
		{"room refreshes", `{"league":"native.startup","franchise":"franchise.01","season":2026,"at":"0001-01-01T00:00:00Z","startup":true}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := client
			client.t = t
			response := client.invoke("draft.read", "read", []byte(tc.args), 200)
			var draft commissioner.NativeDraft
			if err := json.Unmarshal(response.Data, &draft); err != nil {
				t.Fatal(err)
			}
			if draft.League != create.League || !draft.Startup || draft.Season != 2026 ||
				!draft.Draft.Started || len(draft.Draft.Picks) < 32 {
				t.Fatalf("expected an open startup draft, got startup=%t season=%d picks=%d: %s",
					draft.Startup, draft.Season, len(draft.Draft.Picks), draft.Words)
			}
		})
	}
}
