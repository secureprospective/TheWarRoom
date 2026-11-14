package main

import (
	"bytes"
	"errors"
	"io"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/tools/hostedfixture"
)

// reset-password rotates the league link, releases the named team and prints a fresh page link
// opened on that team; the released team can then be claimed exactly once.
func TestResetPasswordReleasesTeamAndPrintsPageLink(t *testing.T) {
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
	credentials := accountstore.New(pools)
	principal, err := hostedfixture.Bootstrap(ctx, credentials, accounts.RawCredentials{Username: "commissioner", Password: "private-password"})
	if err != nil {
		t.Fatal(err)
	}
	if err := hostedfixture.CreateLeague(ctx, ledger, principal); err != nil {
		t.Fatal(err)
	}
	service := accounts.NewJoin(credentials, leagueservice.JoinSeats{Store: ledger}, nil)
	first, err := service.Rotate(ctx, "native.hosted", principal)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Claim(ctx, accounts.RawJoinClaim{
		RawJoin:   accounts.RawJoin{League: "native.hosted", Code: first.Code},
		Franchise: "franchise.01", Username: "owner", Password: "owner-private-password",
	}); err != nil {
		t.Fatal(err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}

	input := strings.NewReader(`{"username":"commissioner","password":"private-password"}`)
	var output bytes.Buffer
	if err := runResetPassword(ctx, []string{"-db", path, "-league", "native.hosted", "-team", "franchise.01"}, input, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "franchise.01") || strings.Contains(output.String(), "owner-private-password") {
		t.Fatalf("unsafe or missing output: %s", output.String())
	}
	open := regexp.MustCompile(`open\s+(\S+)`).FindStringSubmatch(output.String())
	if open == nil {
		t.Fatalf("no page link in output: %s", output.String())
	}
	link, err := url.Parse(open[1])
	if err != nil {
		t.Fatal(err)
	}
	fragment, err := url.ParseQuery(link.Fragment)
	if err != nil || fragment.Get("league") != "native.hosted" || fragment.Get("franchise") != "franchise.01" || len(fragment.Get("code")) != 43 {
		t.Fatalf("fragment=%q: %v", link.Fragment, err)
	}

	reopened, err := openSignInFiles(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reopened.Close() }()
	after := accounts.NewJoin(signInStore(reopened), leagueservice.JoinSeats{Store: history.New(reopened.league, registry)}, nil)
	list, err := after.Franchises(ctx, accounts.RawJoin{League: "native.hosted", Code: fragment.Get("code")})
	if err != nil {
		t.Fatal(err)
	}
	for _, franchise := range list {
		if franchise.ID == "franchise.01" && franchise.Taken {
			t.Fatal("released team is still taken")
		}
	}
	if _, err := after.Franchises(ctx, accounts.RawJoin{League: "native.hosted", Code: first.Code}); !errors.Is(err, accounts.ErrJoinRefused) {
		t.Fatalf("previous link survived reset: %v", err)
	}
	claim := accounts.RawJoinClaim{RawJoin: accounts.RawJoin{League: "native.hosted", Code: fragment.Get("code")},
		Franchise: "franchise.01", Username: "new.owner", Password: "new-owner-password"}
	if _, err := after.Claim(ctx, claim); err != nil {
		t.Fatalf("released team cannot be claimed: %v", err)
	}
	if _, err := after.Claim(ctx, claim); !errors.Is(err, accounts.ErrFranchiseTaken) {
		t.Fatalf("reset link claimed twice: %v", err)
	}
	if err := runResetPassword(ctx, []string{"-db", path, "-league", "native.hosted"}, strings.NewReader("{}"), io.Discard); err == nil {
		t.Fatal("missing -team accepted")
	}
}
