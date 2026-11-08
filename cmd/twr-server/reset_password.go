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

// runResetPassword is the commissioner's console door for a lost owner password. It rotates the
// league link, releases the team, and prints a fresh link opened on that team: the owner claims it
// again with a new sign-in. Nothing here sets a password chosen by the commissioner.
func runResetPassword(ctx context.Context, args []string, input io.Reader, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server reset-password", flag.ContinueOnError)
	path := flags.String("db", "", "existing server database")
	league := flags.String("league", "", "league ID (native.…)")
	team := flags.String("team", "", "franchise ID released for a fresh claim")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse reset-password flags: %w", err)
	}
	if flags.NArg() != 0 || *league == "" || *team == "" {
		return fmt.Errorf("reset-password needs -db, -league and -team FRANCHISE")
	}
	if err := checkHostedDatabase(*path, 0); err != nil {
		return err
	}
	raw, err := readConsoleCredentials(input)
	if err != nil {
		return err
	}
	pools, err := openHostedFile(ctx, *path)
	if err != nil {
		return fmt.Errorf("open server database: %w", err)
	}
	defer func() { _ = pools.Close() }()
	registry, err := measures.Embedded()
	if err != nil {
		return fmt.Errorf("load server measures: %w", err)
	}
	store := history.New(pools, registry)
	if err := store.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize server history: %w", err)
	}
	account, err := commissionerSignIn(ctx, pools, raw)
	if err != nil {
		return err
	}
	principal, err := account.Principal()
	if err != nil {
		return fmt.Errorf("commissioner identity: %w", err)
	}
	service := accounts.NewJoin(accountStore(pools), leagueservice.JoinSeats{Store: store}, nil)
	return resetPasswordAction(ctx, service, *league, *team, principal, output)
}

func resetPasswordAction(ctx context.Context, service *accounts.JoinService, league, team string, principal governance.Principal, output io.Writer) error {
	if err := service.Release(ctx, league, team, principal); err != nil {
		return fmt.Errorf("release %s: %w", team, err)
	}
	code, err := service.Rotate(ctx, league, principal)
	if err != nil {
		return fmt.Errorf("%s was released, but the new join link failed: %w", team, err)
	}
	link := serverhttp.JoinPageLink(league, code.Code, team)
	_, err = fmt.Fprintf(output, "Reset link for %s: its owner must claim the team again with this one-time link. The previous league join link no longer works.\n  league %s\n  code   %s\n  open   %s\nShare it only with the new owner of %s. The old account keeps its password but no longer owns the team.\n",
		team, league, code.Code, link, team)
	if err != nil {
		return fmt.Errorf("print reset link (the new link is already live): %w", err)
	}
	return nil
}
