package state

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// leagueRows counts each league_id-keyed table's rows for one generation.
func leagueRows(t *testing.T, pools *db.Pools, league string) map[string]int {
	t.Helper()
	ctx := t.Context()
	tx, err := pools.Read().BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	tables, err := queryStrings(ctx, tx, `SELECT m.name FROM sqlite_master m, pragma_table_info(m.name) c
WHERE m.type = 'table' AND c.name = 'league_id' ORDER BY m.name`)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]int{}
	for _, table := range tables {
		var n int
		if err := pools.Read().QueryRowContext(ctx, `SELECT COUNT(*) FROM "`+table+`" WHERE league_id = ?`, league).Scan(&n); err != nil {
			t.Fatal(err)
		}
		out[table] = n
	}
	return out
}

func total(counts map[string]int) int {
	n := 0
	for _, c := range counts {
		n += c
	}
	return n
}

func TestTradeScratchCleanupDeletesOnlyTradeProjections(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "whatif.db")
	pools, err := db.Open(ctx, path, SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	generations := []string{testLeague, "whatif:0001:1", "trade-counterparty:2", "trade-preview:3"}
	for _, league := range generations {
		if err := New(pools, league, testSeason, nil).Initialize(ctx, &fakeSource{rosters: baseRosters(t)}); err != nil {
			t.Fatal(err)
		}
		// Rows in two append-only tables prove their delete guards are lifted and restored.
		if _, err := pools.Write().ExecContext(ctx, `INSERT INTO dead_cap_ledger VALUES (?, ?, '0001', 2026, '0001', 100, 'cut', 'now');
INSERT INTO contract_year_changes VALUES (?, ?, '0001', 2026, 0, 100, 'cut', 'test', 'now')`, league+"-d", league, league+"-c", league); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pools.Write().ExecContext(ctx, `INSERT INTO whatif_sessions (franchise_id, session) VALUES (?, ?)`, "0001", `{"mine":"0001","generation":"whatif:0001:1"}`); err != nil {
		t.Fatal(err)
	}
	before := map[string]map[string]int{}
	for _, league := range generations {
		before[league] = leagueRows(t, pools, league)
		if before[league]["rosters"] == 0 || before[league]["dead_cap_ledger"] != 1 || before[league]["contract_year_changes"] == 0 {
			t.Fatalf("%s seed: %v", league, before[league])
		}
	}
	// Back to the file as an older build left it: v2, scratch rows present.
	if _, err := pools.Write().ExecContext(ctx, "PRAGMA user_version = 2"); err != nil {
		t.Fatal(err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	pools, err = db.Open(ctx, path, SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	if err := InitializeWhatIf(ctx, pools); err != nil {
		t.Fatal(err)
	}
	for _, league := range generations {
		after := leagueRows(t, pools, league)
		scratch := league == "trade-counterparty:2" || league == "trade-preview:3"
		if scratch && total(after) != 0 {
			t.Fatalf("%s survived: %v", league, after)
		}
		if !scratch && total(after) != total(before[league]) {
			t.Fatalf("%s lost rows: %v -> %v", league, before[league], after)
		}
	}
	var saved string
	if err := pools.Read().QueryRowContext(ctx, `SELECT session FROM whatif_sessions WHERE franchise_id = ?`, "0001").Scan(&saved); err != nil || saved != `{"mine":"0001","generation":"whatif:0001:1"}` {
		t.Fatal("saved session lost", err)
	}
	for _, table := range []string{"dead_cap_ledger", "contract_year_changes", "season_phases"} {
		if _, err := pools.Write().ExecContext(ctx, `DELETE FROM "`+table+`"`); err == nil {
			t.Fatalf("%s delete guard not restored", table)
		}
	}
}
