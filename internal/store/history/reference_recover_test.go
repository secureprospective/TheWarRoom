package history

import (
	"bytes"
	"compress/gzip"
	"path/filepath"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

func freshReference(t *testing.T) *db.Pools {
	t.Helper()
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "reference.db"), ReferenceSchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	if err := PrepareReferenceFile(t.Context(), pools); err != nil {
		t.Fatal(err)
	}
	return pools
}

const identityRows = `SELECT json_group_array(json_array(player,provider,value)) FROM (SELECT player,provider,value FROM player_external_id ORDER BY player,provider,value)`

// A reference file that lost a league's players gets them back from the league's own restore points,
// with the same permanent ids and outside ids; a second run changes nothing; a player whose outside id
// the reference file already gives to someone else is reported and left out, never merged or minted.
func TestRecoverLeagueIdentitiesFromRestorePoints(t *testing.T) {
	s, _, league := newStore(t)
	if _, err := s.StandUpRosters(t.Context(), perSidePoint()); err != nil {
		t.Fatal(err)
	}
	want := historyJSON(t, league.Read(), identityRows)
	for _, tc := range []struct {
		name      string
		taken     bool
		installed int
		conflicts int
	}{{"lost players come back unchanged", false, 2, 0}, {"an outside id held by another player is reported", true, 1, 1}} {
		t.Run(tc.name, func(t *testing.T) {
			reference := freshReference(t)
			var taken string
			if tc.taken {
				if err := league.Read().QueryRowContext(t.Context(), `SELECT value FROM player_external_id WHERE provider='mfl' ORDER BY value LIMIT 1`).Scan(&taken); err != nil {
					t.Fatal(err)
				}
				other := "player." + strings.Repeat("ab", 16)
				if _, err := reference.Write().ExecContext(t.Context(), `INSERT INTO canonical_player(id) VALUES(?)`, other); err != nil {
					t.Fatal(err)
				}
				if _, err := reference.Write().ExecContext(t.Context(), `INSERT INTO player_external_id(provider,value,player) VALUES('mfl',?,?)`, taken, other); err != nil {
					t.Fatal(err)
				}
			}
			got, err := RecoverLeagueIdentities(t.Context(), league, reference)
			if err != nil {
				t.Fatal(err)
			}
			if got.Installed != tc.installed || len(got.Conflicts) != tc.conflicts {
				t.Fatalf("recovery = %+v, want %d installed, %d conflicts", got, tc.installed, tc.conflicts)
			}
			if !tc.taken {
				if rows := historyJSON(t, reference.Read(), identityRows); rows != want {
					t.Fatalf("recovered identities\n%s\nwant\n%s", rows, want)
				}
			} else if !strings.Contains(got.Conflicts[0], "mfl "+taken) {
				t.Fatalf("conflict does not name the outside id: %q", got.Conflicts[0])
			}
			again, err := RecoverLeagueIdentities(t.Context(), league, reference)
			if err != nil || again.Installed != 0 {
				t.Fatalf("second run = %+v, %v; want nothing installed", again, err)
			}
		})
	}
}

// A raw capture comes back from its evidence object only when it unpacks to content with its name.
func TestPutArchiveBodyChecksItsName(t *testing.T) {
	reference := freshReference(t)
	body := []byte(`{"football":"evidence"}`)
	var stored bytes.Buffer
	writer := gzip.NewWriter(&stored)
	if _, err := writer.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := PutArchiveBody(t.Context(), reference, strings.Repeat("0", 64), stored.Bytes()); err == nil {
		t.Fatal("a capture under the wrong name was accepted")
	}
	key := footballDigest(body)
	if err := PutArchiveBody(t.Context(), reference, key, stored.Bytes()); err != nil {
		t.Fatal(err)
	}
	var size int
	if err := reference.Read().QueryRowContext(t.Context(), `SELECT size FROM raw_archive WHERE sha256=?`, key).Scan(&size); err != nil || size != len(body) {
		t.Fatalf("stored capture size %d, %v", size, err)
	}
}
