package main

import (
	"bytes"
	"errors"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

var (
	moveFileLine = regexp.MustCompile(`move file  (\S+)`)
	moveCodeLine = regexp.MustCompile(`move code  (AGE-SECRET-KEY-1\S+)`)
	claimLink    = regexp.MustCompile(`/join#(\S+)`)
)

// exportFixture seals the fixture league for a move and returns its move file and code.
func exportFixture(t *testing.T, h *twoFileHost) (string, *age.X25519Identity) {
	t.Helper()
	var output bytes.Buffer
	if err := exportLeague(t.Context(), h.files, fixtureLeague, filepath.Join(t.TempDir(), "move"), time.Now(), &output); err != nil {
		t.Fatalf("export: %v", err)
	}
	file, code := moveFileLine.FindStringSubmatch(output.String()), moveCodeLine.FindStringSubmatch(output.String())
	if file == nil || code == nil {
		t.Fatalf("export report:\n%s", output.String())
	}
	identity, err := age.ParseX25519Identity(code[1])
	if err != nil {
		t.Fatal(err)
	}
	return file[1], identity
}

// newEmptyServer is a fresh hosted server holding no league.
func newEmptyServer(t *testing.T) stateFiles {
	t.Helper()
	files := openTestServer(t, t.TempDir())
	if err := signInStore(files).Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	return files
}

func principalOf(t *testing.T, files stateFiles, username, password string) string {
	t.Helper()
	account, err := accounts.New(signInStore(files)).Authenticate(t.Context(), accounts.RawCredentials{Username: username, Password: password})
	if err != nil {
		t.Fatal(err)
	}
	p, err := account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	return string(p)
}

// A league moves to another server as its own file: the old server keeps it readable and unchangeable,
// the new server holds exactly the same league, each person claims their own seat once under the same
// identity, and finishing archives the old copy.
func TestLeagueMovesToAnotherServerAndPeopleClaimTheirSeats(t *testing.T) {
	ctx := t.Context()
	source := newTwoFileHost(t)
	source.seatOwner(t)
	owner, commish := principalOf(t, source.files, "owner", "owner-private-password"), principalOf(t, source.files, "commissioner", "private-password")
	sourcePools := hostedPools(t, source.files)
	want, err := history.LeagueProofs(ctx, sourcePools, fixtureLeague, false)
	if err != nil {
		t.Fatal(err)
	}
	file, code := exportFixture(t, source)
	guards, _, stop, err := startHostedBackup(ctx, source.files, "", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	stop()
	if err := guards.league(fixtureLeague); !errors.Is(err, errLeagueMoving) {
		t.Fatalf("a moving league's changes were not refused: %v", err)
	}
	if _, err := history.LeagueProofs(ctx, sourcePools, fixtureLeague, false); err != nil {
		t.Fatalf("a moving league could not be read: %v", err)
	}

	dest := newEmptyServer(t)
	var output bytes.Buffer
	if err := importLeague(ctx, dest, file, code, filepath.Join(t.TempDir(), "stage"), time.Now(), &output); err != nil {
		t.Fatalf("import: %v\n%s", err, output.String())
	}
	destPools, ok := dest.leagues.pools(fixtureLeague)
	if !ok {
		t.Fatal("the moved league is not served")
	}
	got, err := history.LeagueProofs(ctx, destPools, fixtureLeague, false)
	if err != nil {
		t.Fatal(err)
	}
	for table, proof := range want {
		if got[table] != proof {
			t.Errorf("moved league's %s: %+v, want %+v", table, got[table], proof)
		}
	}
	links := claimLink.FindAllStringSubmatch(output.String(), -1)
	if len(links) != 2 {
		t.Fatalf("want a claim link for the owner and the commissioner:\n%s", output.String())
	}
	destStore, _ := dest.leagues.store(fixtureLeague)
	seats := leagueservice.JoinSeats{Store: destStore}
	join := accounts.NewJoin(signInStore(dest), seats, nil)
	claimed := map[string]string{}
	for i, link := range links {
		values, err := url.ParseQuery(link[1])
		if err != nil {
			t.Fatal(err)
		}
		raw := accounts.RawJoin{League: values.Get("league"), Code: values.Get("code")}
		list, err := join.Franchises(ctx, raw)
		if err != nil || len(list) != 1 || list[0].ID != values.Get("franchise") || list[0].Taken {
			t.Fatalf("claim %d lists %+v (%v)", i, list, err)
		}
		claim := accounts.RawJoinClaim{RawJoin: raw, Franchise: values.Get("franchise"), Username: "person" + string(rune('a'+i)), Password: "a-new-private-password"}
		account, err := join.Claim(ctx, claim)
		if err != nil {
			t.Fatalf("claim %d: %v", i, err)
		}
		p, _ := account.Principal()
		claimed[values.Get("franchise")] = string(p)
		claim.Username += "x"
		if _, err := join.Claim(ctx, claim); !errors.Is(err, accounts.ErrJoinRefused) {
			t.Fatalf("a claim worked twice: %v", err)
		}
	}
	if claimed["franchise.01"] != owner || claimed[accounts.CommissionerSeat] != commish {
		t.Fatalf("claimed identities %v, want owner %s and commissioner %s", claimed, owner, commish)
	}
	if yes, err := seats.IsCommissioner(ctx, fixtureLeague, governance.Principal(commish)); err != nil || !yes {
		t.Fatalf("the commissioner's new sign-in is not the league's commissioner: %v %v", yes, err)
	}

	if err := source.files.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(source.dir, "server.db")
	output.Reset()
	if err := runFinishMove(ctx, []string{"-db", path, "-league", fixtureLeague}, &output); err != nil {
		t.Fatalf("finish: %v", err)
	}
	if !leagueArchived(leaguesRoot(path), fixtureLeague) || !strings.Contains(output.String(), "has moved") {
		t.Fatalf("finish did not archive the old copy: %s", output.String())
	}
}

// A cancelled move lets the league change on the old server again; a wrong move code opens nothing
// and changes nothing on the new server.
func TestLeagueMoveCancelsAndRefusesAWrongCode(t *testing.T) {
	ctx := t.Context()
	source := newTwoFileHost(t)
	file, _ := exportFixture(t, source)
	other, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	dest := newEmptyServer(t)
	if err := importLeague(ctx, dest, file, other, filepath.Join(t.TempDir(), "stage"), time.Now(), &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "does not open") {
		t.Fatalf("a wrong move code: %v", err)
	}
	if got := dest.leagues.leagues(); len(got) != 0 {
		t.Fatalf("a refused import left leagues %v", got)
	}
	if err := source.files.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(source.dir, "server.db")
	var output bytes.Buffer
	if err := runCancelMove(ctx, []string{"-db", path, "-league", fixtureLeague}, &output); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	files := openTestServer(t, source.dir)
	if files.leagues.moving(fixtureLeague) {
		t.Fatal("the league is still moving after cancel")
	}
	if err := runCancelMove(ctx, []string{"-db", path, "-league", fixtureLeague}, &output); err == nil {
		t.Fatal("cancelling a league that is not moving was accepted")
	}
}
