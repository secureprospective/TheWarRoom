package mflwhatif_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/store/mflref/tradeplans"

	"github.com/secureprospective/TheWarRoom/internal/db"
	mflwhatif "github.com/secureprospective/TheWarRoom/internal/mflref/whatif"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestPreRelocationSessionsAndPlansSurviveFileRestart(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "whatif.db")
	pools, err := db.Open(ctx, path, state.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	if err := state.InitializeWhatIf(ctx, pools); err != nil {
		t.Fatal(err)
	}
	// Literal pre-relocation wire data, not the new types' serializer.
	session := `{"generation":"whatif:0001:1","mine":"0001","stamps":{"rosters":"2026-10-01T00:00:00Z","contract":"held","deadCap":"held"},"snapshot":{"season":2026},"seedUsed":12345,"taxiPercent":0.2,"irPercent":0.5,"used":12445,"cap":99999,"moves":[{"id":"cut-0042","kind":"cut","player":"0042","name":"Held player","charge":100,"explanation":"held charge"}]}`
	plan := `{"id":"trade-old-1","mine":"0001","other":"0002","give":["0042"],"get":["0043"],"rationale":"saved before relocation"}`
	if _, err := pools.Write().ExecContext(ctx, `INSERT INTO whatif_sessions VALUES (?, ?)`, "0001", session); err != nil {
		t.Fatal(err)
	}
	if _, err := pools.Write().ExecContext(ctx, `INSERT INTO trade_plans VALUES (?, ?, ?)`, "trade-old-1", "0001", plan); err != nil {
		t.Fatal(err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	pools, err = db.Open(ctx, path, state.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	}()
	store := mflwhatif.NewWhatIfSessions(pools)
	saved, ok, err := store.Load(ctx, "0001")
	if err != nil || !ok || saved.Generation != "whatif:0001:1" || saved.SeedUsed != 12345 || saved.Cap == nil || *saved.Cap != 99999 || len(saved.Moves) != 1 || saved.Moves[0].Player != "0042" || saved.Moves[0].Charge != 100 {
		t.Fatalf("session: %+v %v %v", saved, ok, err)
	}
	if !(state.MirrorStamps{Rosters: "2026-10-02T00:00:00Z"}).NewerThan(saved.Stamps) {
		t.Fatal("lost stale baseline")
	}
	plans, err := tradeplans.NewTradePlans(pools, state.InitializeWhatIf).List(ctx, "0001")
	if err != nil || len(plans) != 1 || plans[0].ID != "trade-old-1" || plans[0].Give[0] != "0042" || plans[0].Rationale != "saved before relocation" {
		t.Fatalf("plans: %+v %v", plans, err)
	}
	if err := store.Save(ctx, saved); err != nil {
		t.Fatal(err)
	}
	again, ok, err := store.Load(ctx, "0001")
	if err != nil || !ok || !reflect.DeepEqual(saved, again) {
		t.Fatalf("session roundtrip: %+v %v", again, err)
	}
}
