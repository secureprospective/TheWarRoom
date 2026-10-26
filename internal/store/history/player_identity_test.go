package history

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

func TestPlayerIdentityMigrationAndSinglePoolOwnership(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "legacy.db")
	p, err := db.Open(t.Context(), path, SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	if _, err := p.Migrate(t.Context(), historyMigrations()[:9]...); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Write().ExecContext(t.Context(), `INSERT INTO player_ids VALUES('gsis','00-0023459','0099'),('sleeper','000123','0099'),('espn','456','1234'),('pfr','Someone00','1234')`); err != nil {
		t.Fatal(err)
	}
	before := historyJSON(t, p.Read(), `SELECT json_group_array(json_array(id_type,id_value,player_id)) FROM (SELECT * FROM player_ids ORDER BY id_type,id_value)`)
	s := New(p, registry(t, ""))
	if err := s.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	after := historyJSON(t, p.Read(), `SELECT json_group_array(json_array(id_type,id_value,player_id)) FROM (SELECT * FROM player_ids ORDER BY id_type,id_value)`)
	if before != after {
		t.Fatal("outside links or leading zeros changed")
	}
	if got := historyJSON(t, p.Read(), `SELECT json_array((SELECT count(*) FROM canonical_player),(SELECT count(*) FROM player_external_id))`); got != "[2,6]" {
		t.Fatal(got)
	}
	var id string
	if err := p.Read().QueryRowContext(t.Context(), `SELECT player FROM player_external_id WHERE provider='mfl' AND value='0099'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	for _, side := range []string{"", "AFC", "NFC"} {
		if _, err := p.Write().ExecContext(t.Context(), `INSERT INTO player_ownership VALUES('native.test',?,?,'franchise.01')`, side, id); err != nil {
			t.Fatal(err)
		}
		if _, err := p.Write().ExecContext(t.Context(), `INSERT INTO player_ownership VALUES('native.test',?,?,'franchise.02')`, side, id); err == nil {
			t.Fatal("two franchises own the same side/player")
		}
	}
	if _, err := p.Write().ExecContext(t.Context(), `UPDATE canonical_player SET id='changed'`); err == nil {
		t.Fatal("canonical identity changed")
	}
}

// The kept growth fixture is private, intentionally not committed. Only a copy
// is migrated, so this proof cannot mutate the retained benchmark artifact.
func TestKeptBigLeagueIdentityMigration(t *testing.T) {
	source := os.Getenv("TWR_KEPT_BIGLEAGUE")
	if source == "" {
		t.Skip("set TWR_KEPT_BIGLEAGUE to the kept big-league SQLite fixture")
	}
	root, err := os.OpenRoot(filepath.Dir(source))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	file, err := root.Open(filepath.Base(source))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	path := filepath.Join(t.TempDir(), "bigleague.db")
	copyFile, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.New()
	if _, err := io.Copy(copyFile, io.TeeReader(file, hash)); err != nil {
		t.Fatal(err)
	}
	if err := copyFile.Close(); err != nil {
		t.Fatal(err)
	}
	p, err := db.Open(t.Context(), path, SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	if _, err := p.Migrate(t.Context(), historyMigrations()[:9]...); err != nil {
		t.Fatal(err)
	}
	counts := historyJSON(t, p.Read(), `SELECT json_array((SELECT count(*) FROM league_event),(SELECT count(*) FROM observations),(SELECT count(*) FROM run_scores),(SELECT count(*) FROM model_scores),(SELECT count(*) FROM player_ids))`)
	events := historyJSON(t, p.Read(), `SELECT json_group_array(json_array(id,payload)) FROM (SELECT id,payload FROM league_event ORDER BY id)`)
	links := historyJSON(t, p.Read(), `SELECT json_group_array(json_array(id_type,id_value,player_id)) FROM (SELECT * FROM player_ids ORDER BY id_type,id_value)`)
	s := New(p, registry(t, ""))
	before, err := s.NativeAssets(t.Context(), "native.bigleague")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	after, err := s.NativeAssets(t.Context(), "native.bigleague")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("identity migration changed league assets")
	}
	if got := historyJSON(t, p.Read(), `SELECT json_array((SELECT count(*) FROM league_event),(SELECT count(*) FROM observations),(SELECT count(*) FROM run_scores),(SELECT count(*) FROM model_scores),(SELECT count(*) FROM player_ids))`); got != counts {
		t.Fatalf("data counts changed: %s -> %s", counts, got)
	}
	if got := historyJSON(t, p.Read(), `SELECT json_group_array(json_array(id,payload)) FROM (SELECT id,payload FROM league_event ORDER BY id)`); got != events {
		t.Fatal("original event bytes changed")
	}
	if got := historyJSON(t, p.Read(), `SELECT json_group_array(json_array(id_type,id_value,player_id)) FROM (SELECT * FROM player_ids ORDER BY id_type,id_value)`); got != links {
		t.Fatal("provider links changed")
	}
	var owners int
	if err := p.Read().QueryRowContext(t.Context(), `SELECT count(*) FROM player_ownership WHERE league='native.bigleague' AND side=''`).Scan(&owners); err != nil {
		t.Fatal(err)
	}
	if after.Snapshot == nil || owners != len(after.Snapshot.Players) {
		t.Fatal("ownership count differs from replayed league")
	}
	identityCounts := historyJSON(t, p.Read(), `SELECT json_array((SELECT count(*) FROM canonical_player),(SELECT count(*) FROM player_external_id),(SELECT count(*) FROM player_ownership))`)
	t.Logf("fixture SHA256=%s; preserved [events,observations,run_scores,model_scores,provider_links]=%s; new [canonical,external_links,owners]=%s", hex.EncodeToString(hash.Sum(nil)), counts, identityCounts)
	target, _, _ := newStore(t)
	point, err := s.RosterRestorePoint(t.Context(), "native.bigleague")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := target.StandUpRosters(t.Context(), point); err != nil {
		t.Fatal(err)
	}
	restored, err := target.RosterRestorePoint(t.Context(), point.League)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored.Snapshot, point.Snapshot) || !reflect.DeepEqual(restored.Identities, point.Identities) {
		t.Fatal("big-league restore changed contracts or permanent identities")
	}
	t.Logf("big-league round trip: %d rosters/contracts and canonical identities identical", owners)
}
