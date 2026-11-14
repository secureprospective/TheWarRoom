package history

import (
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// guardLeagueIdentity makes any insert into canonical_player or player_external_id abort when the same
// transaction has already written a league row. The temp objects live on the single writer
// connection; reset clears the marker a committed league write leaves behind.
func guardLeagueIdentity(t *testing.T, pools *db.Pools) (reset func()) {
	t.Helper()
	ddl := `CREATE TEMP TABLE league_write(n INTEGER);`
	for _, table := range []string{"league_event", "league_import", "player_ownership", "league_roster_restore"} {
		ddl += `CREATE TEMP TRIGGER mark_` + table + ` AFTER INSERT ON main.` + table + ` BEGIN INSERT INTO league_write VALUES(1); END;`
	}
	for _, table := range []string{"canonical_player", "player_external_id"} {
		ddl += `CREATE TEMP TRIGGER guard_` + table + ` BEFORE INSERT ON main.` + table +
			` WHEN EXISTS(SELECT 1 FROM temp.league_write) BEGIN SELECT RAISE(ABORT,'league transaction minted player identity'); END;`
	}
	if _, err := pools.Write().ExecContext(t.Context(), ddl); err != nil {
		t.Fatal(err)
	}
	return func() {
		if _, err := pools.Write().ExecContext(t.Context(), `DELETE FROM temp.league_write`); err != nil {
			t.Fatal(err)
		}
	}
}

func identityCounts(t *testing.T, pools *db.Pools) string {
	t.Helper()
	return historyJSON(t, pools.Read(), `SELECT json_array((SELECT count(*) FROM canonical_player),(SELECT count(*) FROM player_external_id),(SELECT count(*) FROM player_ownership))`)
}

// Players the file has never seen are minted and committed before the league transaction, which only
// reads their ids. The guard fails the write if any league transaction inserts identity.
func TestLeagueTransactionNeverMintsPlayerIdentity(t *testing.T) {
	s, _, pools := newStore(t)
	reset := guardLeagueIdentity(t, pools)
	reset()
	if _, err := s.StandUpRosters(t.Context(), perSidePoint()); err != nil {
		t.Fatal(err)
	}
	if got := identityCounts(t, pools); got != "[2,2,3]" {
		t.Fatalf("identities, links, ownership = %s", got)
	}
}

// A crash after the identity commit and before the league commit leaves only identity rows; the next
// attempt reuses them and completes with the same ids.
func TestIdentityCommitThenLeagueCrashCompletesOnRetry(t *testing.T) {
	s, _, pools := newStore(t)
	if _, err := pools.Write().ExecContext(t.Context(), `CREATE TEMP TRIGGER crash_league BEFORE INSERT ON main.league_import BEGIN SELECT RAISE(ABORT,'injected crash'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.StandUpRosters(t.Context(), perSidePoint()); err == nil {
		t.Fatal("injected crash did not stop the league write")
	}
	if got := identityCounts(t, pools); got != "[2,2,0]" {
		t.Fatalf("after crash: identities, links, ownership = %s", got)
	}
	var events int
	if err := pools.Read().QueryRowContext(t.Context(), `SELECT count(*) FROM league_event`).Scan(&events); err != nil || events != 0 {
		t.Fatalf("league rows survived the crash: %d %v", events, err)
	}
	minted := historyJSON(t, pools.Read(), `SELECT json_group_array(id) FROM (SELECT id FROM canonical_player ORDER BY id)`)
	if _, err := pools.Write().ExecContext(t.Context(), `DROP TRIGGER temp.crash_league`); err != nil {
		t.Fatal(err)
	}
	reset := guardLeagueIdentity(t, pools)
	reset()
	if _, err := s.StandUpRosters(t.Context(), perSidePoint()); err != nil {
		t.Fatal(err)
	}
	if got := identityCounts(t, pools); got != "[2,2,3]" {
		t.Fatalf("after retry: identities, links, ownership = %s", got)
	}
	if again := historyJSON(t, pools.Read(), `SELECT json_group_array(id) FROM (SELECT id FROM canonical_player ORDER BY id)`); again != minted {
		t.Fatalf("retry minted new ids: %s -> %s", minted, again)
	}
}

func TestHostedIdentityCommitThenLeagueCrashCompletesOnRetry(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "league.db")
	original, err := db.Open(ctx, path, SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = original.Close() }()
	s, _, _ := reopen(t, original, registry(t, ""))
	reference, err := db.Open(ctx, filepath.Join(t.TempDir(), "reference.db"), SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reference.Close() }()
	if err := PrepareReferenceFile(ctx, reference); err != nil {
		t.Fatal(err)
	}
	if err := CopyReference(ctx, original, reference); err != nil {
		t.Fatal(err)
	}
	if err := CommitReferenceMove(ctx, original, ReferenceMove{File: "reference.db"}); err != nil {
		t.Fatal(err)
	}
	league, err := db.OpenWithReference(ctx, path, SchemaVersion(), reference)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = league.Close() }()
	s = New(league, s.reg)
	if err := s.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := league.Write().ExecContext(ctx, `CREATE TEMP TRIGGER crash_league BEFORE INSERT ON main.league_import BEGIN SELECT RAISE(ABORT,'injected crash'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.StandUpRosters(ctx, perSidePoint()); err == nil {
		t.Fatal("injected crash did not stop the league write")
	}
	if got := identityCounts(t, league); got != "[2,2,0]" {
		t.Fatalf("after crash: %s", got)
	}
	minted := historyJSON(t, reference.Read(), `SELECT json_group_array(id) FROM (SELECT id FROM canonical_player ORDER BY id)`)
	if _, err := league.Write().ExecContext(ctx, `DROP TRIGGER temp.crash_league`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.StandUpRosters(ctx, perSidePoint()); err != nil {
		t.Fatal(err)
	}
	if got := identityCounts(t, league); got != "[2,2,3]" {
		t.Fatalf("after retry: %s", got)
	}
	if again := historyJSON(t, reference.Read(), `SELECT json_group_array(id) FROM (SELECT id FROM canonical_player ORDER BY id)`); again != minted {
		t.Fatal("retry replaced identities")
	}
}
