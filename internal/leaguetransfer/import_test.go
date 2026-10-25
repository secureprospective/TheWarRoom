package leaguetransfer

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func importTarget(t *testing.T, path string) (*history.Store, *db.Pools) {
	t.Helper()
	pools, err := db.Open(t.Context(), path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	store := history.New(pools, registry)
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	return store, pools
}

func encryptedImport(t *testing.T, raw RawExport) []byte {
	t.Helper()
	var data bytes.Buffer
	if err := Encrypt(t.Context(), &data, raw, [32]byte{7}); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func TestInactiveImportPreservesHistoryAndReplayAcrossRestart(t *testing.T) {
	_, _, raw := transferFixture(t)
	path := filepath.Join(t.TempDir(), "target.db")
	target, pools := importTarget(t, path)
	unrelated := commissioner.RawCommand{League: "native.unrelated", Command: commissioner.Create, Args: commissioner.RawArgs{Name: "Keep this league", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 2}, Request: "create", Reason: "Create a separate league", Effective: "now", Visibility: "league"}
	if _, err := target.ExecuteCommissioner(t.Context(), unrelated, "other.creator", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := target.NativeLeagues(t.Context()); err != nil {
		t.Fatal(err)
	}
	data := encryptedImport(t, raw)
	receipt, err := Import(t.Context(), target, bytes.NewReader(data), [32]byte{7})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.League != raw.League || receipt.Content != raw.Content || receipt.SourceEpoch != raw.SourceEpoch || receipt.RecordedAt.IsZero() {
		t.Fatalf("bad receipt: %+v", receipt)
	}
	events, blobs, head, err := target.CaptureTransfer(t.Context(), raw.League, []string{raw.Football})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(events, raw.Events) || !reflect.DeepEqual(blobs, raw.Blobs) {
		t.Fatal("original event or evidence bytes changed")
	}
	if head.Size != int64(len(raw.Events)+1) {
		t.Fatal("target audit was not built from imported rows")
	}
	leagues, err := target.NativeLeagues(t.Context())
	if err != nil || len(leagues) != 1 || leagues[0].ID != unrelated.League {
		t.Fatal("inactive league offered as active", leagues, err)
	}
	assertImportFenced(t, target, pools, raw)
	document, err := target.ImportDocument(t.Context(), raw.League)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := decode(t.Context(), document)
	if err != nil || !reflect.DeepEqual(checked, raw) {
		t.Fatal("saved comparison evidence changed", err)
	}
	imported := raw
	imported.Events, imported.Blobs = events, blobs
	projection, err := imported.replay(t.Context())
	if err != nil || projection != raw.Projection {
		t.Fatal("import replay mismatch", err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, _ := importTarget(t, path)
	retry, err := Import(t.Context(), reopened, bytes.NewReader(data), [32]byte{7})
	if err != nil || retry != receipt {
		t.Fatal("retry did not return original receipt", retry, err)
	}
}

func assertImportFenced(t *testing.T, target *history.Store, pools *db.Pools, raw RawExport) {
	t.Helper()
	records, err := history.TransferRecords(raw.Events)
	if err != nil {
		t.Fatal(err)
	}
	command := records[0].Command
	command.Command = commissioner.RulesSet
	command.Args = commissioner.RawArgs{Field: "cap.amount", Value: json.RawMessage(`1234567`)}
	command.Expected = int64(len(records))
	command.Request = "must.not.execute"
	if _, err := target.ExecuteCommissioner(t.Context(), command, "creator", nil); err == nil {
		t.Fatal("inactive league accepted command")
	}
	at := time.Now()
	if err := target.SetAuthorityPolicy(t.Context(), history.AuthorityPolicy{League: raw.League, Domain: "roster", CutoverAt: &at}); err == nil {
		t.Fatal("inactive league gained authority")
	}
	if _, err := pools.Write().ExecContext(t.Context(), `DELETE FROM league_import WHERE league=?`, raw.League); err == nil {
		t.Fatal("inactive guard was removable")
	}
	events, _, _, err := target.CaptureTransfer(t.Context(), raw.League, []string{raw.Football})
	if err != nil || !reflect.DeepEqual(events, raw.Events) {
		t.Fatal("rejected writes changed history", err)
	}
}

func TestImportConcurrentRetryAndConflictingContent(t *testing.T) {
	_, _, raw := transferFixture(t)
	path := filepath.Join(t.TempDir(), "target.db")
	first, _ := importTarget(t, path)
	second, _ := importTarget(t, path)
	data := encryptedImport(t, raw)
	var receipts [2]history.ImportReceipt
	var errs [2]error
	var group sync.WaitGroup
	for i, store := range []*history.Store{first, second} {
		group.Go(func() { receipts[i], errs[i] = Import(t.Context(), store, bytes.NewReader(data), [32]byte{7}) })
	}
	group.Wait()
	if errs[0] != nil || errs[1] != nil || receipts[0] != receipts[1] {
		t.Fatal(receipts, errs)
	}
	for _, change := range []func(*RawExport){
		func(r *RawExport) { r.FrozenAt = r.FrozenAt.Add(time.Microsecond) },
		func(r *RawExport) { r.SourceEpoch = "bbbbbbbb-bbbb-4ccc-8ddd-eeeeeeeeeeee" },
	} {
		changed := raw
		change(&changed)
		var err error
		changed.Content, err = changed.contentDigest()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Import(t.Context(), second, bytes.NewReader(encryptedImport(t, changed)), [32]byte{7}); err == nil {
			t.Fatal("conflicting import accepted")
		}
	}
	saved, err := first.ImportDocument(t.Context(), raw.League)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decode(t.Context(), saved)
	if err != nil || !reflect.DeepEqual(got, raw) {
		t.Fatal("conflict replaced original", err)
	}
}

func TestImportRollbackOnEventFailureAndArchiveCollision(t *testing.T) {
	_, _, raw := transferFixture(t)
	data := encryptedImport(t, raw)
	for _, failure := range []string{"event", "receipt", "archive", "existing league", "existing policy", "wrong key"} {
		t.Run(failure, func(t *testing.T) {
			target, pools := importTarget(t, filepath.Join(t.TempDir(), "target.db"))
			key := [32]byte{7}
			switch failure {
			case "event":
				if _, err := pools.Write().ExecContext(t.Context(), `CREATE TRIGGER fail_import BEFORE INSERT ON league_event WHEN NEW.revision=2 BEGIN SELECT RAISE(ABORT,'injected event failure'); END`); err != nil {
					t.Fatal(err)
				}
			case "receipt":
				if _, err := pools.Write().ExecContext(t.Context(), `CREATE TRIGGER fail_import BEFORE INSERT ON league_import BEGIN SELECT RAISE(ABORT,'injected receipt failure'); END`); err != nil {
					t.Fatal(err)
				}
			case "archive":
				if _, err := pools.Write().ExecContext(t.Context(), `INSERT INTO raw_archive VALUES(?,?,?)`, raw.Football, 3, []byte("bad")); err != nil {
					t.Fatal(err)
				}
			case "existing league":
				records, err := history.TransferRecords(raw.Events)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := target.ExecuteCommissioner(t.Context(), records[0].Command, "creator", nil); err != nil {
					t.Fatal(err)
				}
			case "existing policy":
				if err := target.SetAuthorityPolicy(t.Context(), history.AuthorityPolicy{League: raw.League, Domain: "roster"}); err != nil {
					t.Fatal(err)
				}
			case "wrong key":
				key = [32]byte{8}
			}
			if _, err := Import(t.Context(), target, bytes.NewReader(data), key); err == nil {
				t.Fatal("failure accepted")
			}
			assertImportCounts(t, pools, failure)
		})
	}
}

func assertImportCounts(t *testing.T, pools *db.Pools, failure string) {
	t.Helper()
	var events, receipts, archives int
	if err := pools.Read().QueryRowContext(t.Context(), `SELECT (SELECT count(*) FROM league_event),(SELECT count(*) FROM league_import),(SELECT count(*) FROM raw_archive)`).Scan(&events, &receipts, &archives); err != nil {
		t.Fatal(err)
	}
	wantEvents, wantArchives := 0, 0
	if failure == "existing league" {
		wantEvents = 1
	}
	if failure == "archive" {
		wantArchives = 1
	}
	if events != wantEvents || receipts != 0 || archives != wantArchives {
		t.Fatal("failed import left partial state", events, receipts, archives)
	}
}

func TestArchiveGuardChecksWinnerUnderIndependentWriter(t *testing.T) {
	_, _, raw := transferFixture(t)
	path := filepath.Join(t.TempDir(), "target.db")
	target, pools := importTarget(t, path)
	_, other := importTarget(t, path)
	writer, err := other.Write().BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = writer.Rollback() }()
	if _, err := writer.ExecContext(t.Context(), `INSERT INTO raw_archive VALUES(?,?,?)`, raw.Football, 3, []byte("bad")); err != nil {
		t.Fatal(err)
	}
	data := encryptedImport(t, raw)
	started := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		close(started)
		_, err := Import(context.WithoutCancel(t.Context()), target, bytes.NewReader(data), [32]byte{7})
		finished <- err
	}()
	<-started
	if err := writer.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := <-finished; err == nil {
		t.Fatal("racing damaged archive accepted")
	}
	assertImportCounts(t, pools, "archive")
}
