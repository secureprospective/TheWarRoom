package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // RFC 6238 authenticator codes use HMAC-SHA1.
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/offsite"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// memorySets stands in for the R2 bucket: only encrypted objects survive the host.
type memorySets struct {
	objects map[string][]byte
	opened  int
}

func (m *memorySets) Sets(context.Context) ([]string, error) {
	var names []string
	for name := range m.objects {
		names = append(names, name)
	}
	slices.Sort(names)
	slices.Reverse(names)
	return names, nil
}

func (m *memorySets) Open(_ context.Context, name string) (io.ReadCloser, error) {
	m.opened++
	return io.NopCloser(bytes.NewReader(m.objects[name])), nil
}

type leagueHost struct {
	dir, config string
	pools       *db.Pools
	ledger      *history.Store
	store       *accountstore.Store
	passwords   *accounts.Service
	totp        *accounts.TOTPService
}

func openLeagueHost(t *testing.T, dir, config string, now func() time.Time) *leagueHost {
	t.Helper()
	key, err := readOfflineKey(filepath.Join(config, "authenticator.key"))
	if err != nil {
		t.Fatal(err)
	}
	pools, err := db.Open(t.Context(), filepath.Join(dir, "server.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	h := &leagueHost{dir: dir, config: config, pools: pools, ledger: history.New(pools, registry), store: accountstore.New(pools)}
	if err := h.ledger.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := h.store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	h.store.SetClock(now)
	h.passwords = accounts.New(h.store)
	if h.totp, err = accounts.NewTOTP(h.passwords, h.store, key); err != nil {
		t.Fatal(err)
	}
	return h
}

func newHostFiles(t *testing.T, key []byte) (string, string) {
	t.Helper()
	dir, config := t.TempDir(), t.TempDir()
	files := map[string][]byte{"authenticator.key": key, "origin.pem": []byte("tls"), "origin.key": []byte("tls key"), "endpoint-registry.csv": []byte("registry"), "backup.json": []byte("old R2 token")}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(config, name), body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "server.db"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	return dir, config
}

func (h *leagueHost) capture(t *testing.T, sets *memorySets, recipient string) {
	t.Helper()
	sealed, at, err := offsite.Capture(t.Context(), h.pools, h.config, t.TempDir(), recipient, hostedCheckpoint)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(sealed)
	if err != nil {
		t.Fatal(err)
	}
	sets.objects["sets/"+at.Format("20060102T150405.000000000Z")+".tar.gz.age"] = body
}

func authenticatorCode(t *testing.T, seed string, at time.Time) accounts.RawTOTP {
	t.Helper()
	secret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(seed)
	if err != nil {
		t.Fatal(err)
	}
	var message [8]byte
	binary.BigEndian.PutUint64(message[:], uint64(at.Unix()/30)) //nolint:gosec // Test clock is after 1970.
	mac := hmac.New(sha1.New, secret)
	_, _ = mac.Write(message[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 15
	return accounts.RawTOTP{Code: fmt.Sprintf("%06d", (binary.BigEndian.Uint32(sum[offset:offset+4])&0x7fffffff)%1000000)}
}

func restoreSettings(identity *age.X25519Identity) offsite.RawConfig {
	return offsite.RawConfig{Endpoint: strings.Repeat("0", 32) + ".r2.cloudflarestorage.com", Bucket: "league-backups",
		AccessKey: strings.Repeat("a", 32), SecretKey: strings.Repeat("b", 64), Recipient: identity.Recipient().String()}
}

// The slice-37 drill: a league host is lost after its last backup, a freshly installed
// spare restores it, the league fingerprints match and nothing signed in before comes back.
func TestSpareHostRestoreDrill(t *testing.T) {
	ctx := t.Context()
	now := time.Now().UTC()
	clock := func() time.Time { return now }
	restoreKey, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	key := bytes.Repeat([]byte{7}, 32)
	oldDir, oldConfig := newHostFiles(t, key)
	old := openLeagueHost(t, oldDir, oldConfig, clock)
	records, err := leagueservice.FixtureRecords()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := old.ledger.ExecuteCommissioner(ctx, records[0].Command, "fixture.creator", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := old.pools.Write().ExecContext(ctx, `INSERT INTO raw_archive(sha256,size,body) VALUES (?,?,?)`, strings.Repeat("e", 64), 8, []byte("evidence")); err != nil {
		t.Fatal(err)
	}
	owner := accounts.RawCredentials{Username: "commissioner", Password: "drill-commissioner-password"}
	if _, err := old.passwords.LocalBootstrap(ctx, owner); err != nil {
		t.Fatal(err)
	}
	setup, err := old.totp.Begin(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if err := old.totp.Confirm(ctx, owner, authenticatorCode(t, setup.Seed, now)); err != nil {
		t.Fatal(err)
	}
	now = now.Add(30 * time.Second)
	proof, err := old.totp.StepUp(ctx, owner, authenticatorCode(t, setup.Seed, now))
	if err != nil {
		t.Fatal(err)
	}
	codes, err := accounts.NewRecovery(old.passwords, old.store, [32]byte{}).PrintCodes(ctx, owner, proof)
	if err != nil {
		t.Fatal(err)
	}
	session, err := accounts.NewSessions(old.passwords, old.store).Login(ctx, owner, "old desktop")
	if err != nil {
		t.Fatal(err)
	}
	sets := &memorySets{objects: map[string][]byte{}}
	old.capture(t, sets, restoreKey.Recipient().String())
	want, err := readRestoreReport(ctx, old.ledger, old.store)
	if err != nil {
		t.Fatal(err)
	}
	// After the last backup the owner uses a printed code; the copy will still list it as unused.
	if err := accounts.NewRecovery(old.passwords, old.store, [32]byte{}).Recover(ctx, accounts.RawRecovery{Username: owner.Username, Code: codes.Codes[0], Password: "after-backup-password"}); err != nil {
		t.Fatal(err)
	}
	// A spare installed by mistake and backed up before the restore must be skipped, not restored.
	strayDir, strayConfig := newHostFiles(t, bytes.Repeat([]byte{9}, 32))
	stray := openLeagueHost(t, strayDir, strayConfig, clock)
	time.Sleep(time.Millisecond)
	stray.capture(t, sets, restoreKey.Recipient().String())

	start := time.Now()
	if err := old.pools.Close(); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{old.dir, old.config} {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	}
	freshDir, freshConfig := newHostFiles(t, bytes.Repeat([]byte{8}, 32))
	fresh := openLeagueHost(t, freshDir, freshConfig, clock)
	if err := fresh.pools.Close(); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(freshDir, "server.db")
	stage := filepath.Join(t.TempDir(), "restore")
	wrong, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	if err := restoreSpareHost(ctx, sets, wrong, restoreSettings(wrong), target, filepath.Join(t.TempDir(), "wrong"), io.Discard); !errors.Is(err, offsite.ErrRestoreKey) {
		t.Fatalf("wrong key: %v", err)
	}
	var report bytes.Buffer
	if err := restoreSpareHost(ctx, sets, restoreKey, restoreSettings(restoreKey), target, stage, &report); err != nil {
		t.Fatal(err)
	}
	// What restore.sh does as root: install the ready files, drop the fresh database's sidecars.
	ready := filepath.Join(stage, "ready")
	if err := os.Rename(filepath.Join(ready, "server.db"), target); err != nil {
		t.Fatal(err)
	}
	for _, sidecar := range []string{"-wal", "-shm"} {
		_ = os.Remove(target + sidecar)
	}
	if err := os.Rename(filepath.Join(ready, "authenticator.key"), filepath.Join(freshConfig, "authenticator.key")); err != nil {
		t.Fatal(err)
	}
	settings, err := offsite.ReadConfig(filepath.Join(ready, "backup.json"))
	if err != nil || settings.Recipient != restoreKey.Recipient().String() || settings.AccessKey != strings.Repeat("a", 32) {
		t.Fatalf("new backup settings: %v", err)
	}
	elapsed := time.Since(start)
	text := report.String()
	if !strings.Contains(text, "Skipping") || !strings.Contains(text, "holds no league") || !strings.Contains(text, "matches the checkpoint") || !strings.Contains(text, "Signed out all 1 accounts") {
		t.Fatalf("report:\n%s", text)
	}
	for _, secret := range []string{restoreKey.String(), strings.Repeat("b", 64)} {
		if strings.Contains(text, secret) {
			t.Fatal("report printed a secret")
		}
	}
	now = now.Add(2 * time.Minute)
	restored := openLeagueHost(t, freshDir, freshConfig, clock)
	got, err := readRestoreReport(ctx, restored.ledger, restored.store)
	if err != nil {
		t.Fatal(err)
	}
	if got.ledger != want.ledger || !sameDigests(got.digests, want.digests) || len(got.digests.Leagues) != 1 || got.digests.EvidenceFiles != 1 {
		t.Fatalf("league fingerprints changed:\nwant %+v\ngot  %+v", want, got)
	}
	if got.security.Size != want.security.Size+1 {
		t.Fatalf("restore reset missing from sign-in history: %+v", got.security)
	}
	for _, line := range []string{want.ledger.Hash, want.digests.Evidence, want.digests.Leagues[0].Replay} {
		if !strings.Contains(text, line) {
			t.Fatalf("report lacks fingerprint %s", line)
		}
	}
	sessions := accounts.NewSessions(restored.passwords, restored.store)
	if _, err := sessions.Authenticate(ctx, session.RawSession); !errors.Is(err, accounts.ErrSessionRefused) {
		t.Fatalf("pre-loss session revived: %v", err)
	}
	recovery := accounts.NewRecovery(restored.passwords, restored.store, [32]byte{})
	for _, code := range codes.Codes[:2] {
		err := recovery.Recover(ctx, accounts.RawRecovery{Username: owner.Username, Code: code, Password: "attacker-password"})
		if !errors.Is(err, accounts.ErrRecoveryRefused) {
			t.Fatalf("printed recovery code revived: %v", err)
		}
	}
	fresher, err := sessions.Login(ctx, owner, "signed in again")
	if err != nil {
		t.Fatal("owner cannot sign in again:", err)
	}
	if _, err := sessions.Authenticate(ctx, fresher.RawSession); err != nil {
		t.Fatal(err)
	}
	// The restored authenticator key still opens the owner's authenticator.
	if _, err := restored.totp.StepUp(ctx, owner, authenticatorCode(t, setup.Seed, now)); err != nil {
		t.Fatal("authenticator not restored:", err)
	}
	if elapsed >= 4*time.Hour {
		t.Fatalf("restore took %s", elapsed)
	}
	t.Logf("host loss to restored, signed-out league: %s (RTO 4h)", elapsed)
}

func TestRestoreRefusesServerHoldingLeague(t *testing.T) {
	restoreKey, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	dir, config := newHostFiles(t, bytes.Repeat([]byte{7}, 32))
	host := openLeagueHost(t, dir, config, time.Now)
	if _, err := host.passwords.LocalBootstrap(t.Context(), accounts.RawCredentials{Username: "commissioner", Password: "live-league-password"}); err != nil {
		t.Fatal(err)
	}
	sets := &memorySets{objects: map[string][]byte{}}
	host.capture(t, sets, restoreKey.Recipient().String())
	if err := host.pools.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(dir, "server.db"))
	if err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(t.TempDir(), "restore")
	err = restoreSpareHost(t.Context(), sets, restoreKey, restoreSettings(restoreKey), filepath.Join(dir, "server.db"), stage, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "freshly installed") || sets.opened != 0 {
		t.Fatalf("restore over a live league: %v (opened %d)", err, sets.opened)
	}
	if _, err := os.Stat(stage); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("refused restore created output")
	}
	after, err := os.ReadFile(filepath.Join(dir, "server.db"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("refused restore changed the live database")
	}
}
