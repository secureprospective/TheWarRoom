package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/offsite"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// secondLeague is a league that, in the one-file fixture, has only a recovery log row.
const secondLeague = "native.other"

// newTwoLeagueServer is a one-file server (the sign-in fixture) holding two leagues.
func newTwoLeagueServer(t *testing.T) oneFile {
	t.Helper()
	ctx := t.Context()
	s := newOneFileServer(t)
	pools, err := db.Open(ctx, s.path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pools.Write().ExecContext(ctx, `INSERT INTO league_recovery_log(sequence,league,revision,recorded_at,message)
		VALUES(999,'`+secondLeague+`',1,'2026-11-14T00:00:00Z','other league');
		PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatal(err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	return s
}

// assertLeaguesSplit proves the split layout: each league in its own named file holding only its rows,
// and server.db holding no league tables, only the record of the split.
func assertLeaguesSplit(t *testing.T, path string) {
	t.Helper()
	ctx := t.Context()
	files, err := openSignInFiles(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = files.Close() }()
	if got := files.leagues.leagues(); !slices.Equal(got, []string{fixtureLeague, secondLeague}) {
		t.Fatalf("league files %v", got)
	}
	for league, logs := range map[string]int{fixtureLeague: 0, secondLeague: 1} {
		pools, _ := files.leagues.pools(league)
		file, err := history.ReadLeagueFile(ctx, pools)
		if err != nil || file.League != league {
			t.Fatalf("league file %s: %+v %v", league, file, err)
		}
		var foreign, own int
		if err := pools.Read().QueryRowContext(ctx, `SELECT (SELECT count(*) FROM league_recovery_log WHERE league!=?),
			(SELECT count(*) FROM league_recovery_log WHERE league=?)`, league, league).Scan(&foreign, &own); err != nil {
			t.Fatal(err)
		}
		if foreign != 0 || own != logs {
			t.Fatalf("league %s: %d foreign and %d own recovery rows, want 0 and %d", league, foreign, own, logs)
		}
		if info, err := os.Stat(filepath.Dir(mustLeaguePath(t, files.leagues.root, league))); err != nil || info.Mode().Perm() != 0o700 {
			t.Fatalf("league %s folder: %v %v", league, info, err)
		}
	}
	held, herr := hasTable(ctx, files.shared, "league_event")
	split, serr := hasTable(ctx, files.shared, history.LeagueSplitTable)
	if herr != nil || serr != nil || held || !split {
		t.Fatalf("server.db: league tables=%v split record=%v (%v %v)", held, split, herr, serr)
	}
}

func mustLeaguePath(t *testing.T, root, league string) string {
	t.Helper()
	path, err := leagueFilePath(root, league)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLeagueSplitGivesEachLeagueItsFileAndSecondStartIsNoOp(t *testing.T) {
	s := newTwoLeagueServer(t)
	assertLeaguesSplit(t, s.path)
	root := leaguesRoot(s.path)
	before := map[string][32]byte{}
	for _, league := range []string{fixtureLeague, secondLeague} {
		before[league] = digestFile(t, mustLeaguePath(t, root, league))
	}
	assertLeaguesSplit(t, s.path)
	for league, digest := range before {
		if digestFile(t, mustLeaguePath(t, root, league)) != digest {
			t.Fatalf("second start changed league %s's file", league)
		}
	}
}

// A crash at any split step leaves server.db holding every league or the split complete; the next
// start finishes it.
func TestLeagueSplitCrashAtEachStepRecovers(t *testing.T) {
	for _, step := range []struct {
		name      string
		oldIntact bool
	}{{"split-backup", true}, {"split-proved", true}, {"split-committed", false}} {
		t.Run(step.name, func(t *testing.T) {
			s := newTwoLeagueServer(t)
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
			shared, err := db.OpenReadOnly(t.Context(), s.path, history.LeagueSchemaVersion())
			if err != nil {
				t.Fatal(err)
			}
			held, herr := hasTable(t.Context(), shared, "league_event")
			split, serr := hasTable(t.Context(), shared, history.LeagueSplitTable)
			_ = shared.Close()
			if herr != nil || serr != nil || held != step.oldIntact || split == step.oldIntact {
				t.Fatalf("after crash: league tables=%v split record=%v (%v %v)", held, split, herr, serr)
			}
			if step.oldIntact && fileExists(mustLeaguePath(t, leaguesRoot(s.path), fixtureLeague)) {
				t.Fatal("a league file is in place before the split committed")
			}
			assertLeaguesSplit(t, s.path)
		})
	}
}

// A deleted league leaves the server at once but its file is kept 30 days, then purged; other leagues
// and anything else in the archive folder are untouched.
func TestDeletedLeagueIsArchivedThenPurgedAfter30Days(t *testing.T) {
	ctx := t.Context()
	h := newTwoFileHost(t)
	leagueOn(t, h.files, secondLeague)
	if err := h.files.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(h.dir, "server.db")
	root := leaguesRoot(path)
	var output bytes.Buffer
	if err := runDeleteLeague(ctx, []string{"-db", path, "-league", fixtureLeague}, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "until") || fileExists(mustLeaguePath(t, root, fixtureLeague)) || !leagueArchived(root, fixtureLeague) {
		t.Fatalf("delete: %q archived=%v", output.String(), leagueArchived(root, fixtureLeague))
	}
	if !fileExists(mustLeaguePath(t, root, secondLeague)) {
		t.Fatal("deleting one league removed another")
	}
	if err := runDeleteLeague(ctx, []string{"-db", path, "-league", fixtureLeague}, &output); err == nil || !strings.Contains(err.Error(), "not on this server") {
		t.Fatalf("second delete: %v", err)
	}
	files := openTestServer(t, h.dir)
	if got := files.leagues.leagues(); !slices.Equal(got, []string{secondLeague}) {
		t.Fatalf("leagues after delete %v", got)
	}
	if err := files.Close(); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(archiveRoot(root), "notes")
	if err := os.Mkdir(keep, 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if gone, err := purgeArchive(root, now.Add(29*24*time.Hour)); err != nil || len(gone) != 0 || !leagueArchived(root, fixtureLeague) {
		t.Fatalf("purge at 29 days: %v %v", gone, err)
	}
	if gone, err := purgeArchive(root, now.Add(31*24*time.Hour)); err != nil || !slices.Equal(gone, []string{fixtureLeague}) || leagueArchived(root, fixtureLeague) {
		t.Fatalf("purge at 31 days: %v %v", gone, err)
	}
	if !fileExists(keep) || !fileExists(mustLeaguePath(t, root, secondLeague)) {
		t.Fatal("purge removed something that was not an expired archived league")
	}
}

// leagueEnrollment lets the test enroll an account that sits in no league.
type leagueEnrollment struct {
	store  *history.Store
	issuer governance.Principal
}

func (a leagueEnrollment) EnrollmentPolicy(ctx context.Context, league string) (governance.State, governance.Principal, error) {
	events, err := a.store.Events(ctx, league, commissioner.Stream, 0)
	if err != nil {
		return governance.State{}, "", err
	}
	records := make([]commissioner.Record, 0, len(events))
	for _, event := range events {
		records = append(records, *event.Payload.Commissioner)
	}
	policy, err := commissioner.GovernanceState(records)
	return policy, a.issuer, err
}

// Restoring one league signs out that league's owners and no one else, and keeps the file it replaced.
func TestRestoreOneLeagueSignsOutOnlyItsOwners(t *testing.T) {
	ctx := t.Context()
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	h := newTwoFileHost(t)
	h.seatOwner(t)
	store := signInStore(h.files)
	passwords := accounts.New(store)
	commish, err := passwords.Authenticate(ctx, accounts.RawCredentials{Username: "commissioner", Password: "private-password"})
	if err != nil {
		t.Fatal(err)
	}
	issuer, err := commish.Principal()
	if err != nil {
		t.Fatal(err)
	}
	enroll := accounts.NewEnrollment(store, leagueEnrollment{hostedLeague(t, h.files), issuer})
	code, err := enroll.Issue(ctx, fixtureLeague, "bystander")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := enroll.Redeem(ctx, accounts.RawEnrollment{Code: code.Code, Username: "bystander", Password: "bystander-private-password"}); err != nil {
		t.Fatal(err)
	}
	leagueOn(t, h.files, secondLeague)
	sessions := accounts.NewSessions(passwords, store)
	signedIn := map[string]accounts.RawSession{}
	for user, password := range map[string]string{"commissioner": "private-password", "owner": "owner-private-password", "bystander": "bystander-private-password"} {
		session, err := sessions.Login(ctx, accounts.RawCredentials{Username: user, Password: password}, "desktop")
		if err != nil {
			t.Fatal(err)
		}
		signedIn[user] = session.RawSession
	}
	sets := &memorySets{objects: map[string][]byte{}}
	h.capture(t, sets, identity.Recipient().String(), offsite.LeagueSet)
	h.capture(t, sets, identity.Recipient().String(), offsite.ReferenceSet)
	var output bytes.Buffer
	now := time.Now()
	if err := restoreOneLeague(ctx, sets, identity, h.files, fixtureLeague, filepath.Join(t.TempDir(), "restore"), time.Time{}, now, &output); err != nil {
		t.Fatalf("restore one league: %v\n%s", err, output.String())
	}
	for user, out := range map[string]bool{"commissioner": true, "owner": true, "bystander": false} {
		_, err := sessions.Authenticate(ctx, signedIn[user])
		if (err != nil) != out {
			t.Errorf("%s: signed out=%v, want %v (%v)", user, err != nil, out, err)
		}
	}
	root := h.files.leagues.root
	if !leagueArchived(root, fixtureLeague) {
		t.Fatal("the replaced league file was not kept")
	}
	if _, ok := h.files.leagues.store(fixtureLeague); !ok {
		t.Fatal("the restored league is not served")
	}
	if _, ok := h.files.leagues.store(secondLeague); !ok || !fileExists(mustLeaguePath(t, root, secondLeague)) {
		t.Fatal("restoring one league disturbed another")
	}
}
