package history

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

func TestInactiveImportCrashRecovery(t *testing.T) {
	if path := os.Getenv("TWR_IMPORT_CRASH_DB"); path != "" {
		crashImportTransaction(t, path, os.Getenv("TWR_IMPORT_CRASH_POINT"))
		return
	}
	source, _, _ := newStore(t)
	if _, err := source.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err != nil {
		t.Fatal(err)
	}
	events, _, _, err := source.CaptureTransfer(t.Context(), "native.test", nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(events[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, point := range []string{"event", "receipt"} {
		t.Run(point, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "target.db")
			pools, err := db.Open(t.Context(), path, SchemaVersion())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = pools.Close() })
			target, _, _ := reopen(t, pools, source.Registry())
			if err := os.WriteFile(path+".event", body, 0600); err != nil {
				t.Fatal(err)
			}
			child := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestInactiveImportCrashRecovery$") //nolint:gosec // The child is this test binary with a fixed test selector.
			child.Env = append(os.Environ(), "TWR_IMPORT_CRASH_DB="+path, "TWR_IMPORT_CRASH_POINT="+point)
			output, err := child.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 73 {
				t.Fatalf("child did not crash at checkpoint: %v %s", err, output)
			}
			if err := pools.Close(); err != nil {
				t.Fatal(err)
			}
			reopened, err := db.Open(t.Context(), path, SchemaVersion())
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = reopened.Close() }()
			store := New(reopened, target.Registry())
			if err := store.Initialize(t.Context()); err != nil {
				t.Fatal(err)
			}
			var count int
			if err := reopened.Read().QueryRowContext(t.Context(), `SELECT (SELECT count(*) FROM league_event)+(SELECT count(*) FROM league_import)+(SELECT count(*) FROM raw_archive)`).Scan(&count); err != nil || count != 0 {
				t.Fatal("crash left partial import", count, err)
			}
			head, err := store.VerifyAudit(t.Context())
			if err != nil || head.Size != 0 {
				t.Fatal("crash left partial audit", head, err)
			}
		})
	}
}

func crashImportTransaction(t *testing.T, path, point string) {
	t.Helper()
	pools, err := db.Open(t.Context(), path, SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	data, err := os.ReadFile(filepath.Clean(path) + ".event")
	if err != nil {
		t.Fatal(err)
	}
	var event RawTransferEvent
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatal(err)
	}
	tx, err := pools.Write().BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	// Spill dirty pages to WAL so recovery exercises more than a connection-local rollback.
	if _, err := tx.ExecContext(t.Context(), `PRAGMA cache_size=1; INSERT INTO raw_archive VALUES('crash',1048576,zeroblob(1048576))`); err != nil {
		t.Fatal(err)
	}
	if err := stageEvent(t.Context(), tx, event); err != nil {
		t.Fatal(err)
	}
	if point == "receipt" {
		if _, err := tx.ExecContext(t.Context(), `INSERT INTO league_import VALUES(?,?,?,?,?)`, event.League, "epoch", "content", event.RecordedAt, []byte("document")); err != nil {
			t.Fatal(err)
		}
	}
	os.Exit(73) //nolint:gocritic // Crash recovery must skip rollback and connection cleanup.
}
