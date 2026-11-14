package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// runJoinLink is the commissioner's local console door to the league join link: print a new link
// (which revokes the old one), revoke it, or release a claimed franchise. The commissioner signs in
// with username and password as JSON on stdin; nothing is accepted from the network here.
func runJoinLink(ctx context.Context, args []string, input io.Reader, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server join-link", flag.ContinueOnError)
	path := flags.String("db", "", "existing server database")
	league := flags.String("league", "", "league ID (native.…)")
	rotate := flags.Bool("rotate", false, "print a new join link; the previous link stops working")
	revoke := flags.Bool("revoke", false, "stop the join link from working")
	release := flags.String("release", "", "franchise ID whose owner is removed so it can be claimed again")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse join link flags: %w", err)
	}
	chosen := 0
	for _, on := range []bool{*rotate, *revoke, *release != ""} {
		if on {
			chosen++
		}
	}
	if flags.NArg() != 0 || chosen != 1 || *league == "" {
		return fmt.Errorf("join-link needs -db, -league and exactly one of -rotate, -revoke or -release FRANCHISE")
	}
	if err := checkHostedDatabase(*path, 0); err != nil {
		return err
	}
	raw, err := readConsoleCredentials(input)
	if err != nil {
		return err
	}
	files, err := openStateFiles(ctx, *path)
	if err != nil {
		return err
	}
	defer func() { _ = files.Close() }()
	registry, err := measures.Embedded()
	if err != nil {
		return fmt.Errorf("load server measures: %w", err)
	}
	store := history.New(files.league, registry)
	if err := store.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize server history: %w", err)
	}
	account, err := commissionerSignIn(ctx, files, raw)
	if err != nil {
		return err
	}
	principal, err := account.Principal()
	if err != nil {
		return fmt.Errorf("commissioner identity: %w", err)
	}
	service := accounts.NewJoin(signInStore(files), leagueservice.JoinSeats{Store: store}, nil)
	return joinAction(ctx, service, *league, *rotate, *revoke, *release, principal, output)
}

func joinAction(ctx context.Context, service *accounts.JoinService, league string, rotate, revoke bool, release string, principal governance.Principal, output io.Writer) error {
	switch {
	case rotate:
		code, err := service.Rotate(ctx, league, principal)
		if err != nil {
			return fmt.Errorf("new join link: %w", err)
		}
		_, err = fmt.Fprintf(output, "Join link for %s (private: anyone holding it can claim an open franchise; the previous link no longer works):\n  league %s\n  code   %s\n  open   %s\nOwners can use the link, or POST it to %s to see open franchises, then to %s with their pick, username and password.\n",
			league, league, code.Code, serverhttp.JoinPageLink(league, code.Code, ""), serverhttp.JoinFranchisesPath, serverhttp.JoinClaimPath)
		if err != nil {
			return fmt.Errorf("print join link (the new link is already live): %w", err)
		}
	case revoke:
		if err := service.Revoke(ctx, league, principal); err != nil {
			return fmt.Errorf("revoke join link: %w", err)
		}
		if _, err := fmt.Fprintln(output, "The join link no longer works. Claimed franchises keep their owners."); err != nil {
			return fmt.Errorf("report revoke: %w", err)
		}
	default:
		if err := service.Release(ctx, league, release, principal); err != nil {
			return err //nolint:wrapcheck // Release words its own failure
		}
		if _, err := fmt.Fprintf(output, "Franchise %s has no owner now and can be claimed with the join link.\n", release); err != nil {
			return fmt.Errorf("report release: %w", err)
		}
	}
	return nil
}

// commissionerSignIn checks an existing account's password; it never creates one.
func commissionerSignIn(ctx context.Context, files stateFiles, raw accounts.RawCredentials) (accounts.Account, error) {
	store := signInStore(files)
	if err := store.Initialize(ctx); err != nil {
		return accounts.Account{}, fmt.Errorf("initialize server accounts: %w", err)
	}
	account, err := accounts.New(store).Authenticate(ctx, raw)
	if err != nil {
		return accounts.Account{}, fmt.Errorf("commissioner sign-in: %w", err)
	}
	return account, nil
}
