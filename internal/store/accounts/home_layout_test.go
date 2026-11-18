package accounts

import (
	"encoding/json"
	"path/filepath"
	"testing"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
)

const secondAccountID = "aaaaaaaa-1234-4234-8234-123456789abc"

func openServerStore(t *testing.T, path string) (*Store, *db.Pools) {
	t.Helper()
	pools, err := db.Open(t.Context(), path, ServerVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	store := NewServerFile(pools)
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	return store, pools
}

func addHomeLayoutTestAccounts(t *testing.T, store *Store) {
	t.Helper()
	first := fixtureCredential(t)
	if err := store.Insert(t.Context(), first, false); err != nil {
		t.Fatal(err)
	}
	second := first
	second.ID, second.Username = secondAccountID, "other"
	if err := store.Insert(t.Context(), second, false); err != nil {
		t.Fatal(err)
	}
}

func TestHomeLayoutIsRevisionedAndScopedToAuthenticatedAccount(t *testing.T) {
	store, _ := openServerStore(t, filepath.Join(t.TempDir(), "accounts.db"))
	addHomeLayoutTestAccounts(t, store)
	const owner = "12345678-1234-4234-8234-123456789abc"
	layout := json.RawMessage(`{"mode":"custom","tiles":[["news","1x2"],["wire","1x1"]],"over":[],"pins":{"cap":true}}`)

	before, err := store.ReadHomeLayout(t.Context(), owner)
	if err != nil || before.Revision != 0 || string(before.Layout) != `{"mode":"default","tiles":[],"over":[],"pins":{}}` {
		t.Fatalf("initial layout = %+v err=%v", before, err)
	}
	saved, err := store.SetHomeLayout(t.Context(), owner,
		identity.RawHomeLayoutSet{Layout: layout, ExpectedRevision: before.Revision})
	if err != nil || saved.Conflict || saved.Revision != 1 || string(saved.Layout) != string(layout) {
		t.Fatalf("saved layout = %+v err=%v", saved, err)
	}
	other, err := store.ReadHomeLayout(t.Context(), secondAccountID)
	if err != nil || other.Revision != 0 || string(other.Layout) != string(before.Layout) {
		t.Fatalf("other account saw preference: %+v err=%v", other, err)
	}

	stale, err := store.SetHomeLayout(t.Context(), owner,
		identity.RawHomeLayoutSet{Layout: before.Layout, ExpectedRevision: 0})
	if err != nil || !stale.Conflict || stale.Revision != 1 || string(stale.Layout) != string(saved.Layout) {
		t.Fatalf("stale write did not return current server copy: %+v err=%v", stale, err)
	}
	after, err := store.ReadHomeLayout(t.Context(), owner)
	if err != nil || after.Revision != 1 || string(after.Layout) != string(saved.Layout) {
		t.Fatalf("conflict changed stored layout: %+v err=%v", after, err)
	}
}

func TestHomeLayoutRejectsInvalidWriteWithoutChangingSavedCopy(t *testing.T) {
	store, _ := openServerStore(t, filepath.Join(t.TempDir(), "accounts.db"))
	addHomeLayoutTestAccounts(t, store)
	const owner = "12345678-1234-4234-8234-123456789abc"
	layout := json.RawMessage(`{"mode":"custom","tiles":[["cap","1x1"]],"over":[],"pins":{}}`)
	if _, err := store.SetHomeLayout(t.Context(), owner,
		identity.RawHomeLayoutSet{Layout: layout, ExpectedRevision: 0}); err != nil {
		t.Fatal(err)
	}
	bad := json.RawMessage(`{"mode":"custom","tiles":[["unknown","1x1"]],"over":[],"pins":{}}`)
	if _, err := store.SetHomeLayout(t.Context(), owner,
		identity.RawHomeLayoutSet{Layout: bad, ExpectedRevision: 1}); err == nil {
		t.Fatal("unknown board id accepted")
	}
	stored, err := store.ReadHomeLayout(t.Context(), owner)
	if err != nil || stored.Revision != 1 || string(stored.Layout) != string(layout) {
		t.Fatalf("invalid write changed saved layout: %+v err=%v", stored, err)
	}
}

func TestServerHomeLayoutMigrationFreshAndUpgrade(t *testing.T) {
	if got := ServerVersion(); got != 4 {
		t.Fatalf("server migration version = %d, want 4", got)
	}
	t.Run("fresh", func(t *testing.T) {
		store, pools := openServerStore(t, filepath.Join(t.TempDir(), "fresh.db"))
		var version int
		if err := pools.Read().QueryRowContext(t.Context(), `PRAGMA user_version`).Scan(&version); err != nil || version != 4 {
			t.Fatalf("fresh schema version = %d err=%v", version, err)
		}
		var table string
		if err := pools.Read().QueryRowContext(t.Context(), `SELECT name FROM sqlite_master WHERE type='table' AND name='account_home_layouts'`).Scan(&table); err != nil {
			t.Fatalf("fresh schema has no layout table: %v", err)
		}
		layout, err := store.ReadHomeLayout(t.Context(), "12345678-1234-4234-8234-123456789abc")
		if err != nil || layout.Revision != 0 {
			t.Fatalf("read fresh layout = %+v err=%v", layout, err)
		}
	})
	t.Run("upgrade v3", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "upgrade.db")
		pools, err := db.Open(t.Context(), path, ServerVersion())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pools.Migrate(t.Context(), serverMigrations()[:3]...); err != nil {
			t.Fatal(err)
		}
		legacy := NewServerFile(pools)
		credential := fixtureCredential(t)
		if err := legacy.Insert(t.Context(), credential, false); err != nil {
			t.Fatal(err)
		}
		if err := pools.Close(); err != nil {
			t.Fatal(err)
		}

		upgraded, pools := openServerStore(t, path)
		stored, err := upgraded.Credential(t.Context(), credential.Username)
		if err != nil || stored != credential {
			t.Fatalf("upgrade changed account: %+v err=%v", stored, err)
		}
		var version int
		if err := pools.Read().QueryRowContext(t.Context(), `PRAGMA user_version`).Scan(&version); err != nil || version != 4 {
			t.Fatalf("upgrade schema version = %d err=%v", version, err)
		}
		if _, err := upgraded.ReadHomeLayout(t.Context(), credential.ID); err != nil {
			t.Fatalf("read upgraded preference: %v", err)
		}
	})
}
