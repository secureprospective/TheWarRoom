package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
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
	take := flags.String("take", "", "open franchise ID the signed-in commissioner also owns")
	noLogin := flags.Bool("no-login", false, "with -take: seat the league's one commissioner without signing in (console only)")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse join link flags: %w", err)
	}
	if err := checkJoinFlags(flags.NArg(), *league, joinChoice{rotate: *rotate, revoke: *revoke, release: *release, take: *take}, *noLogin); err != nil {
		return err
	}
	if err := checkHostedDatabase(*path, 0); err != nil {
		return err
	}
	if *noLogin {
		return takeWithoutLogin(ctx, *path, *league, *take, output)
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
	store, err := leagueStore(ctx, files, *league)
	if err != nil {
		return err
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
	return joinAction(ctx, service, *league, joinChoice{rotate: *rotate, revoke: *revoke, release: *release, take: *take}, principal, output)
}

// joinChoice is the one console action asked for.
type joinChoice struct {
	rotate, revoke bool
	release, take  string
}

func joinAction(ctx context.Context, service *accounts.JoinService, league string, choice joinChoice, principal governance.Principal, output io.Writer) error {
	rotate, revoke, release := choice.rotate, choice.revoke, choice.release
	switch {
	case choice.take != "":
		if err := service.Take(ctx, league, choice.take, principal); err != nil {
			return err //nolint:wrapcheck // Take words its own failure
		}
		if _, err := fmt.Fprintf(output, "You own franchise %s now as well as holding the commissioner's seat; sign in again to open it.\n", choice.take); err != nil {
			return fmt.Errorf("report take (the seat is already recorded): %w", err)
		}
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

// takeWithoutLogin seats the league's one commissioner on franchise without a password. Only someone
// already at the server console, holding the database file, can run it.
func takeWithoutLogin(ctx context.Context, path, league, franchise string, output io.Writer) error {
	files, err := openStateFiles(ctx, path)
	if err != nil {
		return err
	}
	defer func() { _ = files.Close() }()
	store, err := leagueStore(ctx, files, league)
	if err != nil {
		return err
	}
	seats := leagueservice.JoinSeats{Store: store}
	principal, err := seats.Commissioner(ctx, league)
	if err != nil {
		return fmt.Errorf("find commissioner: %w", err)
	}
	service := accounts.NewJoin(signInStore(files), seats, nil)
	return joinAction(ctx, service, league, joinChoice{take: franchise}, principal, output)
}

// checkJoinFlags requires a league and exactly one action; -no-login goes only with -take.
func checkJoinFlags(extra int, league string, choice joinChoice, noLogin bool) error {
	chosen := 0
	for _, on := range []bool{choice.rotate, choice.revoke, choice.release != "", choice.take != ""} {
		if on {
			chosen++
		}
	}
	if extra != 0 || chosen != 1 || league == "" {
		return fmt.Errorf("join-link needs -db, -league and exactly one of -rotate, -revoke, -release FRANCHISE or -take FRANCHISE")
	}
	if noLogin && choice.take == "" {
		return fmt.Errorf("-no-login works only with -take FRANCHISE")
	}
	return nil
}
