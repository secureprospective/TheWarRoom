package history

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

func legacyHistory(t *testing.T) (string, string, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "history.db")
	fixture, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = fixture.Close() }()
	tx, err := fixture.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := adoptHistory(t.Context(), tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	// Dev-data shape: a model run references an absent param set, but owns real score rows.
	if _, err := fixture.ExecContext(t.Context(), `
 INSERT INTO param_sets VALUES ('held','params-sha','{}','2026-10-01');
 INSERT INTO model_runs VALUES (41,'model',2026,'2026-10-01','missing','engine-a','input-a','score-a','[]','2026-10-01');
 INSERT INTO model_runs VALUES (42,'model',2026,'2026-10-02','held','engine-b','input-b','score-b','[]','2026-10-02');
 INSERT INTO model_scores VALUES (41,'0001','DE',1.25,2.5,3.75,4.5,5.25,6.5,7.75,8.25,9.5,10.75,'{"held":true}');
 INSERT INTO model_scores VALUES (41,'0002','LB',11.25,12.5,13.75,14.5,15.25,16.5,17.75,18.25,19.5,20.75,'{}');
 INSERT INTO model_scores VALUES (42,'0003','S',21.25,22.5,23.75,24.5,25.25,26.5,27.75,28.25,29.5,30.75,'{}');
 INSERT INTO model_scores VALUES (999,'0004','CB',31.25,32.5,33.75,34.5,35.25,36.5,37.75,38.25,39.5,40.75,'{}');
 INSERT INTO fetch_log VALUES (1,'fixture','fixture://held',200,'absent-body','','2026-10-01');
 `); err != nil {
		t.Fatal(err)
	}
	run := historyJSON(t, fixture, `SELECT json_array(run_id,kind,season,as_of,param_set_id,engine,inputs_hash,scores_hash,missing_measures,created_at) FROM model_runs WHERE run_id=41`)
	scores := historyJSON(t, fixture, `SELECT json_group_array(json_array(run_id,mfl_id,position,now,now_ppg,dynasty,dynasty_ppg,prior,past_games,season_games,z_past,z_now,on_field,inputs)) FROM (SELECT * FROM model_scores WHERE run_id=41 ORDER BY mfl_id)`)
	return path, run, scores
}

func historyJSON(t *testing.T, pool *sql.DB, query string) string {
	t.Helper()
	var result string
	if err := pool.QueryRowContext(t.Context(), query).Scan(&result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestHistoryMigrationQuarantinesOnlyReportedOrphans(t *testing.T) {
	path, wantRun, wantScores := legacyHistory(t)
	pools, err := db.Open(t.Context(), path, SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	store := New(pools, registry(t, ""))
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	findings := pools.MigrationFindings()
	if len(findings.Before.ForeignKeys) != 3 || len(findings.After.ForeignKeys) != 1 || findings.After.ForeignKeys[0].Table != "fetch_log" {
		t.Fatalf("findings = %+v", findings)
	}
	if got := historyJSON(t, pools.Read(), `SELECT json_array(run_id,kind,season,as_of,param_set_id,engine,inputs_hash,scores_hash,missing_measures,created_at) FROM quarantine_model_runs WHERE run_id=41`); got != wantRun {
		t.Fatalf("run changed: %s != %s", got, wantRun)
	}
	if got := historyJSON(t, pools.Read(), `SELECT json_group_array(json_array(run_id,mfl_id,position,now,now_ppg,dynasty,dynasty_ppg,prior,past_games,season_games,z_past,z_now,on_field,inputs)) FROM (SELECT * FROM quarantine_model_scores WHERE run_id=41 ORDER BY mfl_id)`); got != wantScores {
		t.Fatalf("scores changed: %s != %s", got, wantScores)
	}
	if got := historyJSON(t, pools.Read(), `SELECT json_array((SELECT count(*) FROM model_runs),(SELECT count(*) FROM model_scores),(SELECT count(*) FROM param_sets),(SELECT count(*) FROM quarantine_model_runs),(SELECT count(*) FROM quarantine_model_scores),(SELECT count(*) FROM quarantine_model_scores WHERE reason<>'' AND datetime(found_at) IS NOT NULL))`); got != "[1,1,1,1,3,3]" {
		t.Fatalf("counts = %s", got)
	}
	if got := historyJSON(t, pools.Read(), `SELECT json_array(run_id,mfl_id,now) FROM model_scores`); got != "[42,\"0003\",21.25]" {
		t.Fatalf("valid score changed: %s", got)
	}
	if got := historyJSON(t, pools.Read(), `SELECT json_array(reason,datetime(found_at) IS NOT NULL) FROM quarantine_model_runs`); !strings.Contains(got, "missing param_sets") || !strings.HasSuffix(got, ",1]") {
		t.Fatalf("quarantine provenance = %s", got)
	}
	for _, query := range []string{"DELETE FROM model_runs", "DELETE FROM model_scores", "DELETE FROM quarantine_model_runs", "DELETE FROM quarantine_model_scores"} {
		if _, err := pools.Write().ExecContext(t.Context(), query); err == nil {
			t.Fatalf("append-only protection missing: %s", query)
		}
	}
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatalf("reopen: %v", err)
	}
	backup, err := sql.Open("sqlite", "file:"+path+".pre-v0.bak?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = backup.Close() }()
	if got := historyJSON(t, backup, `SELECT json_array((SELECT count(*) FROM model_runs),(SELECT count(*) FROM model_scores))`); got != "[2,4]" {
		t.Fatalf("backup lost orphan evidence: %s", got)
	}
	var version int
	if err := pools.Read().QueryRowContext(t.Context(), "PRAGMA user_version").Scan(&version); err != nil || version != SchemaVersion() {
		t.Fatalf("version=%d err=%v", version, err)
	}
}

func TestHistoryQuarantineFailureRestoresRowsAndTriggers(t *testing.T) {
	path, wantRun, _ := legacyHistory(t)
	pools, err := db.Open(t.Context(), path, SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	injected := errors.New("injected after quarantine")
	_, err = pools.Migrate(t.Context(), adoptHistory, func(ctx context.Context, tx *sql.Tx) error {
		if err := quarantineOrphans(ctx, tx); err != nil {
			return err
		}
		return injected
	})
	if !errors.Is(err, injected) {
		t.Fatalf("failure=%v", err)
	}
	if got := historyJSON(t, pools.Read(), `SELECT json_array(run_id,kind,season,as_of,param_set_id,engine,inputs_hash,scores_hash,missing_measures,created_at) FROM model_runs WHERE run_id=41`); got != wantRun {
		t.Fatal("failed quarantine changed run")
	}
	if got := historyJSON(t, pools.Read(), `SELECT json_array((SELECT count(*) FROM model_scores),(SELECT count(*) FROM sqlite_master WHERE name LIKE 'quarantine_model_%'))`); got != "[4,0]" {
		t.Fatalf("rollback=%s", got)
	}
	var version int
	if err := pools.Read().QueryRowContext(t.Context(), "PRAGMA user_version").Scan(&version); err != nil || version != 1 {
		t.Fatalf("version=%d err=%v", version, err)
	}
	if _, err := pools.Write().ExecContext(t.Context(), "DELETE FROM model_scores WHERE run_id=41"); err == nil {
		t.Fatal("rollback lost append-only trigger")
	}
}

func TestHistoryAdoptionPreservesCurrentRows(t *testing.T) {
	path, wantRun, wantScores := legacyHistory(t)
	pools, err := db.Open(t.Context(), path, SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	if _, err := pools.Migrate(t.Context(), adoptHistory); err != nil {
		t.Fatal(err)
	}
	if got := historyJSON(t, pools.Read(), `SELECT json_array(run_id,kind,season,as_of,param_set_id,engine,inputs_hash,scores_hash,missing_measures,created_at) FROM model_runs WHERE run_id=41`); got != wantRun {
		t.Fatal("adoption changed run")
	}
	if got := historyJSON(t, pools.Read(), `SELECT json_group_array(json_array(run_id,mfl_id,position,now,now_ppg,dynasty,dynasty_ppg,prior,past_games,season_games,z_past,z_now,on_field,inputs)) FROM (SELECT * FROM model_scores WHERE run_id=41 ORDER BY mfl_id)`); got != wantScores {
		t.Fatal("adoption changed scores")
	}
}

// An older guard under the v3 name passes `{}` (its verdict is NULL); v3 must replace it.
func TestHistoryGuardMigrationReplacesOlderPayloadGuard(t *testing.T) {
	path, _, _ := legacyHistory(t)
	pools, err := db.Open(t.Context(), path, SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	if _, err := pools.Migrate(t.Context(), adoptHistory, quarantineOrphans); err != nil {
		t.Fatal(err)
	}
	if _, err := pools.Write().ExecContext(t.Context(), `CREATE TRIGGER league_event_payload_json_insert BEFORE INSERT ON league_event
WHEN NOT (json_valid(NEW.payload) AND json_extract(NEW.payload, '$.version') > 0)
BEGIN SELECT RAISE(ABORT, 'old guard'); END`); err != nil {
		t.Fatal(err)
	}
	if err := New(pools, registry(t, "")).Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	_, err = pools.Write().ExecContext(t.Context(), `INSERT INTO league_event (id,league,stream,revision,effective_at,recorded_at,actor,scope,kind,payload,input_hash,correlation_id) VALUES ('e','legacy','s',1,'','','actor','league','test','{}',?, 'c')`, strings.Repeat("a", 64))
	if err == nil || !strings.Contains(err.Error(), "invalid JSON payload") {
		t.Fatalf("versionless payload: %v", err)
	}
}
