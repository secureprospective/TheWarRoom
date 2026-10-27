package sources

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSettingsPersistenceAndFailures(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "settings.json")
	if got, err := ReadSettings(path); err != nil || got != (Settings{}) {
		t.Fatalf("missing defaults: %+v %v", got, err)
	}
	want := Settings{LeagueArchiveDir: "archive", PhotosOff: true}
	if err := WriteSettings(path, want); err != nil {
		t.Fatal(err)
	}
	if got, err := ReadSettings(path); err != nil || got != want {
		t.Fatalf("round trip: %+v %v", got, err)
	}
	stat, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if stat.Mode().Perm() != 0o600 {
		t.Fatalf("public settings mode: %v", stat.Mode())
	}
	if err := os.WriteFile(path, []byte(`{`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSettings(path); err == nil {
		t.Fatal("malformed config hidden")
	}
	if err := WriteSettings(filepath.Join(path, "child"), want); err == nil {
		t.Fatal("blocked path accepted")
	}
	if err := os.Mkdir(path+".tmp", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := WriteSettings(path, want); err == nil {
		t.Fatal("blocked temp overwritten")
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != "{" {
		t.Fatal("failed write changed prior file")
	}
}

func TestCacheScopesOrderedAndDispatchOnlySelected(t *testing.T) {
	t.Parallel()
	failure := errors.New("refresh failed")
	calls := 0
	refresh := func(force bool) error {
		calls++
		if !force {
			t.Fatal("refresh not forced")
		}
		return failure
	}
	scopes := CacheScopes(CacheRefreshers{Pool: refresh, Research: refresh, Race: refresh, Comms: refresh, Draft: refresh})
	var ids []string
	for _, s := range CacheScopeReadings(scopes) {
		ids = append(ids, s.ID)
	}
	if !reflect.DeepEqual(ids, []string{"pool", "research", "race", "comms", "draft"}) {
		t.Fatal(ids)
	}
	if label, found, err := ClearCache(scopes, "unknown"); label != "" || found || err != nil || calls != 0 {
		t.Fatal("unknown cache had effects")
	}
	if label, found, err := ClearCache(scopes, "research"); label != "Player research" || !found || err != failure || calls != 1 {
		t.Fatalf("dispatch lost result %q %v %v", label, found, err)
	}
}
