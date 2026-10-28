package catalog

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// TestAssistMovesALeagueFromOneAsk runs the AI assist end to end through the command registry, with no
// UI: one plain ask, then each confirm. Nothing is recorded before its confirm, and every recorded act
// says it was made via AI for the commissioner.
func TestAssistMovesALeagueFromOneAsk(t *testing.T) {
	t.Parallel()
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "history.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	store := history.New(pools, &measures.Registry{})
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	now := migrationtest.ReadAt.Add(time.Hour)
	store.SetClock(func() time.Time { return now })
	export := migrationtest.Synthetic("Assist league", migrationtest.Specs()["empty franchise no email"])
	src := &migrationtest.Source{Export: export}
	service := &leaguemigration.Service{Source: func(string) (leaguemigration.Source, error) { return src, nil },
		Ledger: leagueservice.MigrationLedger{Store: store}, Now: func() time.Time { return now }}
	registry := &Registry{entries: MigrationEntries(MigrationInputs{Service: service,
		Principal: func() (string, error) { return "commish", nil }})}
	if err := registry.Validate(); err != nil {
		t.Fatal(err)
	}
	executes := 0
	assistant := &leaguemigration.Assistant{Invoke: func(ctx context.Context, id, mode string,
		body json.RawMessage) (json.RawMessage, error) {
		if mode == "execute" {
			executes++
		}
		// The assist's transport names the AI helper; the ledger shows each act as made via AI.
		return registry.Invoke(commissioner.WithTokenAudit(ctx, "assist.migration"), AssistDoor, id, Mode(mode), body)
	}}
	league := leaguemigration.AppLeagueID(export.LeagueID, 2026)
	ledger := func() commissioner.Ledger {
		t.Helper()
		l, err := store.CommissionerLedger(t.Context(), league, "commish", nil)
		if err != nil && !strings.Contains(err.Error(), "created") {
			t.Fatal(err)
		}
		return l
	}

	turn, err := assistant.Ask(t.Context(), "Move my league over from MFL, it's league "+export.LeagueID)
	if err != nil || turn.Confirm == nil || executes != 0 || len(ledger().Entries) != 0 {
		t.Fatalf("the ask recorded something or offered no confirm: %v %+v executes=%d", err, turn, executes)
	}
	if _, err := assistant.Confirm(t.Context(), "confirm.99", "Move it"); err == nil || executes != 0 {
		t.Fatal("a confirm for an act never offered was recorded")
	}
	if _, err := assistant.Confirm(t.Context(), turn.Confirm.ID, " "); err == nil || executes != 0 {
		t.Fatal("a confirm without a typed reason was recorded")
	}
	turn, err = assistant.Confirm(t.Context(), turn.Confirm.ID, "Moving the league to the app")
	if err != nil || executes != 1 || turn.Confirm == nil || !strings.Contains(strings.Join(turn.Say, " "), "Everything matches") {
		t.Fatalf("import: %v %+v", err, turn)
	}
	if n := len(ledger().Entries); n != 2 {
		t.Fatalf("the import recorded %d entries, want the creation and its starting snapshot", n)
	}
	again, err := assistant.Review(t.Context())
	if err != nil || executes != 1 {
		t.Fatalf("re-running the review recorded something: %v", err)
	}
	turn, err = assistant.Confirm(t.Context(), again.Confirm.ID, "Everything matches")
	if err != nil || executes != 2 || turn.Confirm == nil || !strings.Contains(strings.Join(turn.Say, " "), "7 of 8 owners") {
		t.Fatalf("cutover: %v %+v", err, turn)
	}
	turn, err = assistant.Confirm(t.Context(), turn.Confirm.ID, "Welcome everyone")
	if err != nil || executes != 3 || !turn.Done {
		t.Fatalf("invites: %v %+v", err, turn)
	}
	entries := ledger().Entries
	if len(entries) != 2+1+7 {
		t.Fatalf("ledger entries=%d", len(entries))
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.Who, "via AI/CLI, acting for commish") {
			t.Errorf("%q was not recorded as via AI: %q", e.What, e.Who)
		}
	}
	if !strings.Contains(entries[7].What, "Moved the league over from MFL") {
		t.Errorf("cutover ledger words: %q", entries[7].What)
	}
}

func TestGenericExecuteRefusesARawCutover(t *testing.T) {
	t.Parallel()
	service := leagueservice.New(nil)
	_, err := service.Execute(t.Context(), commissioner.RawCommand{League: "native.x", Command: commissioner.Cutover,
		Expected: 2, Request: "r", Reason: "Ready", Effective: "now", Visibility: "league"})
	if err == nil || !strings.Contains(err.Error(), "from its review") {
		t.Fatalf("a raw cutover skipped the review: %v", err)
	}
}
