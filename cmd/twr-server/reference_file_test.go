package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// footballEvidence is a minimal football evidence body and its digest; name makes each one distinct.
func footballEvidence(t *testing.T, name string) ([]byte, string) {
	t.Helper()
	raw := history.RawFootballEvidence{Version: 1, Season: 2026, Source: "reference move fixture", AsOf: time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC),
		Players: map[string]history.RawFootballPlayer{"0099": {Identity: "reference.99", Facts: nativecheck.PlayerFacts{Name: name, Position: leaguerules.QB, Team: "DAL", DesignationHeld: true}}},
		Weeks:   []leagueweek.Week{{Number: 1, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: time.Date(2026, 10, 25, 17, 0, 0, 0, time.UTC)}}}}}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	return body, hex.EncodeToString(sum[:])
}

// oneFileWithReference is a one-file server (the sign-in fixture) that also holds football reference
// data: two football imports in the raw archive, identities and a roster that owns a player.
type oneFileWithReference struct {
	oneFile
	digests []string
	proof   history.ReferenceProof
}

func newReferenceServer(t *testing.T) oneFileWithReference {
	t.Helper()
	ctx := t.Context()
	s := oneFileWithReference{oneFile: newOneFileServer(t)}
	pools, err := db.Open(ctx, s.path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	store := history.New(pools, registry)
	for _, name := range []string{"Player 99", "Player 99 renamed"} {
		body, digest := footballEvidence(t, name)
		if err := store.ImportFootball(ctx, body, digest); err != nil {
			t.Fatal(err)
		}
		s.digests = append(s.digests, digest)
	}
	if s.proof, err = history.ProveReference(ctx, pools); err != nil {
		t.Fatal(err)
	}
	if s.proof.Archive.Rows < 2 || s.proof.Tables["measures"].Rows == 0 {
		t.Fatalf("fixture too thin: %+v", s.proof)
	}
	if _, err := pools.Write().ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatal(err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	return s
}

// onlyReference runs step for the reference move's steps only; the sign-in move shares the seam.
func onlyReference(step func(at, pending string)) func(string, string) {
	return func(at, pending string) {
		if strings.HasSuffix(pending, referenceFileName+".converting") {
			step(at, pending)
		}
	}
}

// assertReferenceMoved proves the three-file layout is complete: every table equal in its new home,
// none left in the league file, the reference file owner-only, and stores reading through the attachment.
func (s oneFileWithReference) assertReferenceMoved(t *testing.T) {
	t.Helper()
	ctx := t.Context()
	files, err := openSignInFiles(ctx, s.path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = files.Close() }()
	got, err := history.ProveReference(ctx, files.reference)
	if err != nil {
		t.Fatal(err)
	}
	// The measure mirror is rebuilt from the registry on every start; every other table must be exact.
	got.Tables["measures"], got.Tables["sources"], got.Tables["source_fields"] = s.proof.Tables["measures"], s.proof.Tables["sources"], s.proof.Tables["source_fields"]
	if err := history.SameReference(s.proof, got); err != nil {
		t.Error(err)
	}
	alone, err := db.OpenReadOnly(ctx, s.path, history.LeagueSchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = alone.Close() }()
	for _, table := range history.ReferenceTables() {
		if held, err := hasTable(ctx, alone, table); err != nil || held {
			t.Errorf("league file still holds %s (%v)", table, err)
		}
	}
	if info, err := os.Stat(referencePath(s.path)); err != nil || info.Mode().Perm()&0o077 != 0 {
		t.Errorf("reference file is not owner-only: %v", err)
	}
	_, err = measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := hostedLeague(t, files).LoadFootball(ctx, s.digests[0]); err != nil {
		t.Errorf("league store cannot read football through the reference file: %v", err)
	}
}

func TestFreshServerCreatesThreeFilesAndEachRefusesNewer(t *testing.T) {
	for _, file := range []string{"server.db", signInFileName, referenceFileName} {
		t.Run(file, func(t *testing.T) {
			dir := t.TempDir()
			files, err := openSignInFiles(t.Context(), filepath.Join(dir, "server.db"))
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{signInFileName, referenceFileName} {
				if !fileExists(filepath.Join(dir, name)) {
					t.Fatalf("no %s", name)
				}
			}
			pools := map[string]*db.Pools{"server.db": files.shared, signInFileName: files.signIns, referenceFileName: files.reference}[file]
			if _, err := pools.Write().ExecContext(t.Context(), `PRAGMA main.user_version=999`); err != nil {
				t.Fatal(err)
			}
			if err := files.Close(); err != nil {
				t.Fatal(err)
			}
			if again, err := openSignInFiles(t.Context(), filepath.Join(dir, "server.db")); err == nil {
				_ = again.Close()
				t.Fatalf("a newer %s was opened", file)
			}
		})
	}
}

func TestReferenceMoveProvesAndSecondStartIsNoOp(t *testing.T) {
	s := newReferenceServer(t)
	s.assertReferenceMoved(t)
	reference, league := digestFile(t, referencePath(s.path)), digestFile(t, s.path)
	s.assertReferenceMoved(t)
	if digestFile(t, referencePath(s.path)) != reference || digestFile(t, s.path) != league {
		t.Fatal("second start changed a file")
	}
}

// A copy that does not match refuses to start and leaves the league file byte-identical.
func TestReferenceMoveMismatchRefusesAndChangesNothing(t *testing.T) {
	for _, tamper := range []string{
		`DELETE FROM measures WHERE rowid IN (SELECT rowid FROM measures LIMIT 1)`,
		`DROP TRIGGER raw_archive_no_update; UPDATE raw_archive SET body = body || X'00' WHERE rowid IN (SELECT rowid FROM raw_archive LIMIT 1)`,
		`PRAGMA foreign_keys=OFF; DROP TRIGGER raw_archive_no_delete; DELETE FROM raw_archive WHERE rowid IN (SELECT rowid FROM raw_archive LIMIT 1)`,
		`INSERT INTO canonical_player VALUES ('player.` + strings.Repeat("ab", 16) + `')`,
	} {
		t.Run(tamper, func(t *testing.T) {
			s := newReferenceServer(t)
			// The sign-in move runs first in the same start; the baseline is the league file as the
			// reference move found it, after its pre-change backup.
			var original [32]byte
			t.Cleanup(func() { conversionStep = func(string, string) {} })
			conversionStep = onlyReference(func(at, pending string) {
				if at == "backup" {
					original = digestFile(t, s.path)
				}
				if at != "copied" {
					return
				}
				pools, err := db.Open(t.Context(), pending, history.ReferenceSchemaVersion())
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = pools.Close() }()
				if _, err := pools.Write().ExecContext(t.Context(), tamper); err != nil {
					t.Fatal(err)
				}
			})
			if files, err := openSignInFiles(t.Context(), s.path); err == nil {
				_ = files.Close()
				t.Fatal("a mismatched copy was accepted")
			} else if !strings.Contains(err.Error(), "did not copy exactly") {
				t.Fatalf("refused for another reason: %v", err)
			}
			if digestFile(t, s.path) != original {
				t.Fatal("league file changed after a refused move")
			}
			if fileExists(referencePath(s.path)) || fileExists(referenceMove.pending(s.path)) {
				t.Fatal("a refused move left a reference file behind")
			}
			conversionStep = func(string, string) {}
			s.assertReferenceMoved(t)
		})
	}
}

// A crash at any step leaves the old layout intact or the new one complete; the next start finishes.
func TestReferenceMoveCrashAtEachStepRecovers(t *testing.T) {
	for _, step := range []struct {
		name      string
		oldIntact bool
	}{{"backup", true}, {"copied", true}, {"proved", true}, {"synced", true}, {"committed", false}} {
		t.Run(step.name, func(t *testing.T) {
			s := newReferenceServer(t)
			t.Cleanup(func() { conversionStep = func(string, string) {} })
			conversionStep = onlyReference(func(at, _ string) {
				if at == step.name {
					panic(crash{at})
				}
			})
			func() {
				defer func() {
					if r := recover(); r != (crash{step.name}) {
						t.Fatalf("no crash at %s: %v", step.name, r)
					}
				}()
				_, _ = openSignInFiles(t.Context(), s.path)
			}()
			conversionStep = func(string, string) {}
			league, err := db.OpenReadOnly(t.Context(), s.path, history.LeagueSchemaVersion())
			if err != nil {
				t.Fatal(err)
			}
			held, herr := hasTable(t.Context(), league, "raw_archive")
			moved, merr := hasTable(t.Context(), league, history.ReferenceMoveTable)
			_ = league.Close()
			if herr != nil || merr != nil || held != step.oldIntact || moved == step.oldIntact || fileExists(referencePath(s.path)) {
				t.Fatalf("after crash: reference held=%v moved=%v reference file=%v (%v %v)", held, moved, fileExists(referencePath(s.path)), herr, merr)
			}
			s.assertReferenceMoved(t)
		})
	}
}

func TestLeagueAttachmentCannotWriteReference(t *testing.T) {
	files := openTestServer(t, t.TempDir())
	ctx := t.Context()
	league := hostedPools(t, files)
	for _, table := range history.ReferenceTables() {
		t.Run(table, func(t *testing.T) {
			if _, err := league.Write().ExecContext(ctx, `DELETE FROM reference.`+table+` WHERE 0`); err == nil || !strings.Contains(err.Error(), "readonly") {
				t.Fatalf("reference write was not refused as read-only: %v", err)
			}
		})
	}
	tx, err := hostedPools(t, files).BeginWrite(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	id := "player." + strings.Repeat("ab", 16)
	if _, err := tx.ExecContext(ctx, `INSERT INTO league_player VALUES(?)`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO reference.canonical_player VALUES(?)`, id); err == nil || !strings.Contains(err.Error(), "readonly") {
		t.Fatalf("cross-file transaction was not refused: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	for _, pools := range []*db.Pools{hostedPools(t, files), files.reference} {
		table := "league_player"
		if pools == files.reference {
			table = "canonical_player"
		}
		var n int
		if err := pools.Read().QueryRowContext(ctx, `SELECT count(*) FROM `+table+` WHERE id=?`, id).Scan(&n); err != nil || n != 0 {
			t.Fatalf("refused cross-file write left rows: %d (%v)", n, err)
		}
	}
}
