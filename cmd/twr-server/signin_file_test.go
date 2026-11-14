package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/tools/hostedfixture"
)

// oneFile is a server as it was before sign-ins moved: everything in server.db.
type oneFile struct {
	path     string
	proofs   map[string]accountstore.MoveProof
	security db.AuditHead
}

// newOneFileServer builds a one-file server with a commissioner, an owner seated by join link, a
// sign-in reset in the security history and a live join link, then records what must move.
func newOneFileServer(t *testing.T) oneFile {
	t.Helper()
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "server.db")
	pools, err := db.Open(ctx, path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	ledger := history.New(pools, registry)
	if err := ledger.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	store := accountstore.New(pools)
	principal, err := hostedfixture.Bootstrap(ctx, store, accounts.RawCredentials{Username: "commissioner", Password: "private-password"})
	if err != nil {
		t.Fatal(err)
	}
	if err := hostedfixture.CreateLeague(ctx, ledger, principal); err != nil {
		t.Fatal(err)
	}
	join := accounts.NewJoin(store, leagueservice.JoinSeats{Store: ledger}, nil)
	code, err := join.Rotate(ctx, "native.hosted", principal)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := join.Claim(ctx, accounts.RawJoinClaim{RawJoin: accounts.RawJoin{League: "native.hosted", Code: code.Code},
		Franchise: "franchise.01", Username: "owner", Password: "owner-private-password"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ResetSignIns(ctx, "conversion drill"); err != nil {
		t.Fatal(err)
	}
	if _, err := join.Rotate(ctx, "native.hosted", principal); err != nil {
		t.Fatal(err)
	}
	s := oneFile{path: path}
	if s.proofs, err = accountstore.MoveProofs(ctx, pools); err != nil {
		t.Fatal(err)
	}
	if len(s.proofs) != len(accountstore.SignInTables())+2 || s.proofs["accounts"].Rows != 2 || s.proofs["league_join_links"].Rows != 1 {
		t.Fatalf("fixture too thin: %+v", s.proofs)
	}
	if s.security, err = store.VerifyAudit(ctx); err != nil || s.security.Size == 0 {
		t.Fatalf("security chain %+v: %v", s.security, err)
	}
	if _, err := pools.Write().ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatal(err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	return s
}

func digestFile(t *testing.T, path string) [32]byte {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(body)
}

// assertConverted proves the two-file layout is complete: every moved set equal, the chain intact.
func (s oneFile) assertConverted(t *testing.T) {
	t.Helper()
	ctx := t.Context()
	files, err := openSignInFiles(ctx, s.path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = files.Close() }()
	got, err := accountstore.MoveProofs(ctx, files.signIns)
	if err != nil {
		t.Fatal(err)
	}
	for set, want := range s.proofs {
		if got[set] != want {
			t.Errorf("%s: got %+v want %+v", set, got[set], want)
		}
	}
	for _, table := range accountstore.SignInTables() {
		if held, err := hasTable(ctx, files.shared, table); err != nil || held {
			t.Errorf("league file still holds %s (%v)", table, err)
		}
	}
	head, err := accountstore.NewServerFile(files.signIns).VerifyAudit(ctx)
	if err != nil || head != s.security {
		t.Errorf("security chain in its new home: %+v want %+v (%v)", head, s.security, err)
	}
	if info, err := os.Stat(signInPath(s.path)); err != nil || info.Mode().Perm()&0o077 != 0 {
		t.Errorf("sign-in file is not owner-only: %v %v", info.Mode(), err)
	}
}

func TestFreshServerCreatesBothFilesAndEachRefusesNewer(t *testing.T) {
	for _, file := range []string{"server.db", signInFileName} {
		t.Run(file, func(t *testing.T) {
			dir := t.TempDir()
			files, err := openSignInFiles(t.Context(), filepath.Join(dir, "server.db"))
			if err != nil {
				t.Fatal(err)
			}
			if !fileExists(filepath.Join(dir, signInFileName)) {
				t.Fatal("no sign-in file")
			}
			pools := map[string]*db.Pools{"server.db": files.shared, signInFileName: files.signIns}[file]
			if _, err := pools.Write().ExecContext(t.Context(), `PRAGMA user_version=999`); err != nil {
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

func TestConversionMovesSignInsAndSecondStartIsNoOp(t *testing.T) {
	s := newOneFileServer(t)
	s.assertConverted(t)
	before := digestFile(t, signInPath(s.path))
	s.assertConverted(t)
	if digestFile(t, signInPath(s.path)) != before {
		t.Fatal("second start changed the sign-in file")
	}
}

// A copy that does not match refuses to start and leaves the league file byte-identical.
func TestConversionMismatchRefusesAndChangesNothing(t *testing.T) {
	for _, tamper := range []string{
		`UPDATE accounts SET password_hash = password_hash || 'x' WHERE username = 'owner'`,
		`DELETE FROM league_join_links`,
		`UPDATE account_auth_epoch SET epoch = CASE WHEN substr(epoch, 1, 1) = 'a' THEN 'b' ELSE 'a' END || substr(epoch, 2)`,
		`INSERT INTO audit_chain VALUES ('account_security_audit', 999999, 'forged', 'd', 'p', 'h')`,
	} {
		t.Run(tamper, func(t *testing.T) {
			s := newOneFileServer(t)
			original := digestFile(t, s.path)
			t.Cleanup(func() { conversionStep = func(string, string) {} })
			conversionStep = func(step, pending string) {
				if step != "copied" {
					return
				}
				pools, err := db.Open(t.Context(), pending, accountstore.ServerVersion())
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = pools.Close() }()
				if _, err := pools.Write().ExecContext(t.Context(), tamper); err != nil {
					t.Fatal(err)
				}
			}
			files, err := openSignInFiles(t.Context(), s.path)
			if err == nil {
				_ = files.Close()
				t.Fatal("a mismatched copy was accepted")
			}
			if digestFile(t, s.path) != original {
				t.Fatal("league file changed after a refused move")
			}
			if fileExists(signInPath(s.path)) || fileExists(pendingSignIns(s.path)) {
				t.Fatal("a refused move left a sign-in file behind")
			}
			conversionStep = func(string, string) {}
			s.assertConverted(t)
		})
	}
}

type crash struct{ step string }

// A crash at any step leaves the old layout intact or the new one complete; the next start finishes.
func TestConversionCrashAtEachStepRecovers(t *testing.T) {
	for _, step := range []struct {
		name      string
		oldIntact bool
	}{{"backup", true}, {"copied", true}, {"proved", true}, {"synced", true}, {"committed", false}} {
		t.Run(step.name, func(t *testing.T) {
			s := newOneFileServer(t)
			t.Cleanup(func() { conversionStep = func(string, string) {} })
			conversionStep = func(at, _ string) {
				if at == step.name {
					panic(crash{at})
				}
			}
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
			held, herr := hasTable(t.Context(), league, "accounts")
			moved, merr := hasTable(t.Context(), league, signInMoveTable)
			_ = league.Close()
			if herr != nil || merr != nil || held != step.oldIntact || moved == step.oldIntact || fileExists(signInPath(s.path)) {
				t.Fatalf("after crash: accounts held=%v moved=%v sign-in file=%v (%v %v)", held, moved, fileExists(signInPath(s.path)), herr, merr)
			}
			s.assertConverted(t)
		})
	}
}

// failingSeats fails the first seat, as a crash between the join's two commits would.
type failingSeats struct {
	leagueservice.JoinSeats
	failed bool
}

func (f *failingSeats) Seat(ctx context.Context, league, franchise string, owner, issuer governance.Principal) error {
	if !f.failed {
		f.failed = true
		return errors.New("stopped between the two commits")
	}
	return f.JoinSeats.Seat(ctx, league, franchise, owner, issuer)
}

// The account commits in the sign-in file first; if the seat never commits the account stands
// alone with the franchise still open, and a retry seats the owner.
func TestJoinStoppedBetweenCommitsLeavesAccountWithoutSeat(t *testing.T) {
	ctx := t.Context()
	files := openTestServer(t, t.TempDir())
	_, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	ledger := hostedLeague(t, files)
	if err := ledger.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	store := signInStore(files)
	if err := store.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	principal, err := hostedfixture.Bootstrap(ctx, store, accounts.RawCredentials{Username: "commissioner", Password: "private-password"})
	if err != nil {
		t.Fatal(err)
	}
	if err := hostedfixture.CreateLeague(ctx, ledger, principal); err != nil {
		t.Fatal(err)
	}
	seats := &failingSeats{JoinSeats: leagueservice.JoinSeats{Store: ledger}}
	join := accounts.NewJoin(store, seats, nil)
	code, err := join.Rotate(ctx, "native.hosted", principal)
	if err != nil {
		t.Fatal(err)
	}
	claim := func(username string) error {
		_, err := join.Claim(ctx, accounts.RawJoinClaim{RawJoin: accounts.RawJoin{League: "native.hosted", Code: code.Code},
			Franchise: "franchise.01", Username: username, Password: "owner-private-password"})
		return err
	}
	if err := claim("owner"); err == nil {
		t.Fatal("the stopped claim reported success")
	}
	var accountsHeld int
	if err := files.signIns.Read().QueryRowContext(ctx, `SELECT count(*) FROM accounts WHERE username='owner'`).Scan(&accountsHeld); err != nil || accountsHeld != 1 {
		t.Fatalf("account after the stop: %d (%v)", accountsHeld, err)
	}
	if taken := franchiseTaken(t, join, code.Code); taken {
		t.Fatal("seat committed although the second commit never ran")
	}
	// The same name is taken (the message tells the owner to ask the commissioner); a retry seats.
	if err := claim("owner"); !errors.Is(err, accounts.ErrUsernameTaken) {
		t.Fatalf("retry with the stranded name: %v", err)
	}
	if err := claim("owner.retry"); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if !franchiseTaken(t, join, code.Code) {
		t.Fatal("retry did not seat the owner")
	}
}

func franchiseTaken(t *testing.T, join *accounts.JoinService, code string) bool {
	t.Helper()
	list, err := join.Franchises(t.Context(), accounts.RawJoin{League: "native.hosted", Code: code})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range list {
		if f.ID == "franchise.01" {
			return f.Taken
		}
	}
	t.Fatal(fmt.Sprint("no franchise.01 in ", list))
	return false
}
