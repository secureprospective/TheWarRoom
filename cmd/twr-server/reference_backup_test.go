package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/offsite"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// importFootball puts one football evidence body into the reference file through the league store.
func importFootball(t *testing.T, files stateFiles, name string) (string, []byte) {
	t.Helper()
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	body, digest := footballEvidence(t, name)
	if err := history.New(files.league, registry).ImportFootball(t.Context(), body, digest); err != nil {
		t.Fatal(err)
	}
	return digest, body
}

// Two leagues pinned to different football; an unpinned league uses the server's pin; re-pinning one
// league changes only its own directory row and its own store.
func TestLeaguePinsAreSeparate(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	files := openTestServer(t, dir)
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	base := history.New(files.league, registry)
	if err := base.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	server, _ := importFootball(t, files, "Server pin")
	first, _ := importFootball(t, files, "League A pin")
	second, _ := importFootball(t, files, "League B pin")
	signIns := signInStore(files)
	if err := signIns.RecordLeagues(ctx, []accountstore.DirectoryEntry{
		{League: "native.a", State: "active", Created: "2026-11-14"}, {League: "native.b", State: "active", Created: "2026-11-14"},
		{League: "native.c", State: "active", Created: "2026-11-14"}}); err != nil {
		t.Fatal(err)
	}
	for league, digest := range map[string]string{"native.a": first, "native.b": second} {
		if err := signIns.PinFootball(ctx, league, digest); err != nil {
			t.Fatal(err)
		}
	}
	read := func() *leaguePins {
		directory, err := signIns.Directory(ctx)
		if err != nil {
			t.Fatal(err)
		}
		pins := newLeaguePins(base)
		if err := pins.selectPins(ctx, server, directory); err != nil {
			t.Fatal(err)
		}
		return pins
	}
	pins := read()
	a, b, c := pins.pin("native.a"), pins.pin("native.b"), pins.pin("native.c")
	if a.Digest != first || b.Digest != second || c.Digest != server || a.Store == b.Store || c.Store != base {
		t.Fatalf("pins: a=%s b=%s c=%s", a.Digest, b.Digest, c.Digest)
	}
	if store, _ := pins.League(ctx, "native.b"); store != b.Store {
		t.Fatal("the due worker would serve league b from another pin's store")
	}
	before, err := signIns.Directory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := files.Close(); err != nil {
		t.Fatal(err)
	}
	if err := runPinFootball(ctx, []string{"-db", filepath.Join(dir, "server.db"), "-league", "native.a", "-football", second}, &out); err != nil {
		t.Fatal(err)
	}
	files = openTestServer(t, dir)
	signIns = signInStore(files)
	after, err := signIns.Directory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for i := range before {
		changed := before[i] != after[i]
		if changed != (before[i].League == "native.a") || (before[i].League == "native.a" && after[i].Football != second) {
			t.Fatalf("re-pinning league a changed %+v to %+v", before[i], after[i])
		}
	}
	base = history.New(files.league, registry)
	pins = read()
	if pins.pin("native.b").Digest != second || pins.pin("native.c").Digest != server || pins.pin("native.a").Digest != second {
		t.Fatal("re-pinning one league moved another")
	}
	if err := runPinFootball(ctx, []string{"-db", filepath.Join(dir, "server.db"), "-league", "native.a", "-football", strings.Repeat("0", 64)}, &out); err == nil {
		t.Fatal("pinned football the reference file does not hold")
	}
}

// The reference set carries no raw capture: every body goes out once as its own object and the set's
// copy holds only keys and sizes. The set is far smaller than the bodies it names.
func TestReferenceSetLeavesRawCapturesOut(t *testing.T) {
	ctx := t.Context()
	h := newTwoFileHost(t)
	importFootball(t, h.files, "Player 99")
	// One large, incompressible raw capture, as a fetched page would be (stored compressed already).
	body := make([]byte, 512<<10)
	if _, err := rand.Read(body); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	if _, err := h.files.reference.Write().ExecContext(ctx, `INSERT INTO raw_archive(sha256, size, body) VALUES (?, ?, ?)`, hex.EncodeToString(sum[:]), len(body), body); err != nil {
		t.Fatal(err)
	}
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	sets := &memorySets{objects: map[string][]byte{}}
	captureReferenceInto(t, sets, identity.Recipient().String(), h.files.reference, h.config)
	var setName string
	objects := 0
	for name := range sets.objects {
		switch {
		case strings.HasPrefix(name, offsite.ReferenceSet.Prefix):
			setName = name
		case strings.HasPrefix(name, offsite.EvidencePrefix):
			objects++
		}
	}
	proof, err := history.ProveReference(ctx, h.files.reference)
	if err != nil {
		t.Fatal(err)
	}
	if setName == "" || int64(objects) != proof.Archive.Rows {
		t.Fatalf("set %q, %d raw capture objects for %d bodies", setName, objects, proof.Archive.Rows)
	}
	dest := filepath.Join(t.TempDir(), "unlocked")
	manifest, err := offsite.Restore(bytes.NewReader(sets.objects[setName]), identity, dest)
	if err != nil {
		t.Fatal(err)
	}
	if !manifest.Holds(offsite.ReferenceSet.File) || manifest.Holds(offsite.LeagueSet.File) || manifest.Holds(offsite.SignInSet.File) {
		t.Fatalf("set members: %+v", manifest.Files)
	}
	staged, err := os.ReadFile(filepath.Join(dest, referenceFileName))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(staged, body[:min(len(body), 256)]) {
		t.Fatal("a raw capture rode in the reference set")
	}
	pools, err := db.OpenReadOnly(ctx, filepath.Join(dest, referenceFileName), history.ReferenceSchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	var bodies, held int64
	if err := pools.Read().QueryRowContext(ctx, `SELECT count(*), coalesce(sum(length(body)), 0) FROM raw_archive`).Scan(&held, &bodies); err != nil {
		t.Fatal(err)
	}
	if held != proof.Archive.Rows || bodies != 0 {
		t.Fatalf("set raw archive: %d rows holding %d body bytes", held, bodies)
	}
	if size := int64(len(sets.objects[setName])); size >= int64(len(body)) {
		t.Fatalf("reference set is %d bytes, not smaller than one %d-byte capture", size, len(body))
	}
}

// The three sets restore together; a league pinned to football the reference set lacks is reported
// and nothing is installed.
func TestRestoreThreeFilesReportsMissingPin(t *testing.T) {
	for _, tc := range []struct {
		name    string
		pinGone bool
	}{{"league, reference and sign-ins restore together", false}, {"a pinned version the reference set lacks is reported", true}} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			identity, err := age.GenerateX25519Identity()
			if err != nil {
				t.Fatal(err)
			}
			h := newTwoFileHost(t)
			h.seatOwner(t)
			digest, _ := importFootball(t, h.files, "Pinned")
			if tc.pinGone {
				digest = strings.Repeat("e", 64)
			}
			if err := recordDirectory(ctx, history.New(h.files.league, &measures.Registry{}), signInStore(h.files)); err != nil {
				t.Fatal(err)
			}
			if err := signInStore(h.files).PinFootball(ctx, "native.hosted", digest); err != nil {
				t.Fatal(err)
			}
			sets := &memorySets{objects: map[string][]byte{}}
			for _, kind := range []offsite.SetKind{offsite.LeagueSet, offsite.ReferenceSet, offsite.SignInSet} {
				h.capture(t, sets, identity.Recipient().String(), kind)
			}
			want, err := history.ProveReference(ctx, h.files.reference)
			if err != nil {
				t.Fatal(err)
			}
			target, _ := newHostFiles(t, bytes.Repeat([]byte{7}, 32))
			stage := filepath.Join(t.TempDir(), "restore")
			var output bytes.Buffer
			err = restoreSpareHost(ctx, sets, identity, restoreSettings(identity), filepath.Join(target, "server.db"), stage, time.Time{}, &output)
			if tc.pinGone {
				if err == nil || !strings.Contains(err.Error(), "no football reference backup matches") ||
					!strings.Contains(output.String(), "league native.hosted pins football version "+digest) {
					t.Fatalf("missing pin not reported: %v\n%s", err, output.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("%v\n%s", err, output.String())
			}
			ready := filepath.Join(stage, "ready", "server.db")
			files, err := openSignInFiles(ctx, ready)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = files.Close() }()
			got, err := history.ProveReference(ctx, files.reference)
			if err != nil {
				t.Fatal(err)
			}
			if got.Archive != want.Archive {
				t.Fatalf("restored raw archive %+v, want %+v", got.Archive, want.Archive)
			}
			if _, err := history.New(files.league, &measures.Registry{}).LoadFootball(ctx, digest); err != nil {
				t.Fatalf("restored league cannot read its pinned football: %v", err)
			}
		})
	}
}

// A due round captures league first, reference next, sign-ins last.
func TestBackupRoundOrderIsLeagueReferenceSignIns(t *testing.T) {
	league, reference, signIns := &offsite.Manager{}, &offsite.Manager{}, &offsite.Manager{}
	got := backupManagers{league: league, reference: reference, signIns: signIns}.inOrder()
	if len(got) != 3 || got[0] != league || got[1] != reference || got[2] != signIns {
		t.Fatal("capture order is not league, reference, sign-ins")
	}
	if one := (backupManagers{league: league}).inOrder(); len(one) != 1 || one[0] != league {
		t.Fatal("a one-file server captures more than its league set")
	}
}

// O5/O10: a ledger write commits to the league file only; the reference file's revision and bytes do
// not move, so its set renews without an upload. O10: capturing the reference file never waits on the
// league writer.
func TestLedgerWritesLeaveReferenceFileAlone(t *testing.T) {
	ctx := t.Context()
	h := newTwoFileHost(t)
	importFootball(t, h.files, "Before the ledger write")
	if _, err := h.files.reference.Write().ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatal(err)
	}
	referenceRev, err := h.files.reference.Revision(ctx)
	if err != nil {
		t.Fatal(err)
	}
	leagueRev, err := h.files.league.Revision(ctx)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(h.dir, referenceFileName)
	before := digestFile(t, path)
	h.seatOwner(t)
	if rev, err := h.files.league.Revision(ctx); err != nil || rev == leagueRev {
		t.Fatalf("the ledger write did not reach the league file (%v)", err)
	}
	if rev, err := h.files.reference.Revision(ctx); err != nil || rev != referenceRev {
		t.Fatalf("a ledger write changed the reference file (%v)", err)
	}
	if digestFile(t, path) != before {
		t.Fatal("a ledger write changed the reference file's bytes")
	}
	held, err := h.files.league.Write().Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = held.Close() }()
	capture, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := h.files.reference.CaptureHosted(capture, filepath.Join(t.TempDir(), "reference.db"), nil); err != nil {
		t.Fatalf("reference capture waited on the league writer: %v", err)
	}
}
