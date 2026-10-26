package offsite

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"filippo.io/age"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestR2RosterContractRecoveryRoundTrip(t *testing.T) {
	pools, config, checkpoint := testCapture(t)
	ledger := history.New(pools, &measures.Registry{})
	body, err := os.ReadFile("../commissioner/testdata/native-assets.wire.json")
	if err != nil {
		t.Fatal(err)
	}
	var held commissioner.NativeAssets
	if err := json.Unmarshal(body, &held); err != nil {
		t.Fatal(err)
	}
	if held.Snapshot == nil {
		t.Fatal("missing held snapshot")
	}
	var franchises []string
	for _, f := range held.Snapshot.Franchises {
		franchises = append(franchises, f.ID)
	}
	commands := []commissioner.RawCommand{
		{League: "native.r2", Command: commissioner.Create, Args: commissioner.RawArgs{Name: "R2 recovery fixture", Preset: commissioner.PresetLegacy, Season: held.Snapshot.Season, Franchises: franchises, FranchiseCount: 32}, Request: "create", Reason: "Recovery proof", Effective: "now", Visibility: "league"},
		{League: "native.r2", Command: commissioner.NativeSeed, Args: commissioner.RawArgs{Native: held.Snapshot}, Expected: 1, Request: "seed", Reason: "Recovery proof", Effective: "now", Visibility: "league"},
	}
	for _, command := range commands {
		if _, err := ledger.ExecuteCommissioner(t.Context(), command, "creator", nil); err != nil {
			t.Fatal(err)
		}
	}
	original, err := ledger.RosterRestorePoint(t.Context(), "native.r2")
	if err != nil {
		t.Fatal(err)
	}
	key, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	path, at, err := Capture(t.Context(), pools, config, t.TempDir(), key.Recipient().String(), checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	r2, fake := testR2(t)
	name := "sets/" + at.Format(objectFormat) + ".tar.gz.age"
	if err := r2.upload(t.Context(), name, path); err != nil {
		t.Fatal(err)
	}
	if err := r2.verify(t.Context(), name, path); err != nil {
		t.Fatal(err)
	}
	fake.mu.Lock()
	encrypted := bytes.Clone(fake.objects[name])
	fake.mu.Unlock()
	if bytes.Contains(encrypted, []byte("native.r2")) {
		t.Fatal("R2 received plaintext league data")
	}
	stage := filepath.Join(t.TempDir(), "decrypted")
	if _, err := Restore(bytes.NewReader(encrypted), key, stage); err != nil {
		t.Fatal(err)
	}
	backup, err := db.Open(t.Context(), filepath.Join(stage, "server.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = backup.Close() }()
	point, err := history.New(backup, &measures.Registry{}).RosterRestorePoint(t.Context(), "native.r2")
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "fresh.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = fresh.Close() }()
	recovered := history.New(fresh, &measures.Registry{})
	if err := recovered.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := recovered.StandUpRosters(t.Context(), point); err != nil {
		t.Fatal(err)
	}
	got, err := recovered.RosterRestorePoint(t.Context(), point.League)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Snapshot, original.Snapshot) || !reflect.DeepEqual(got.Identities, original.Identities) {
		t.Fatal("encrypted R2 round trip changed rosters, exact contracts or canonical identity")
	}
}
