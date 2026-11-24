package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

// createLeaguePrincipal is the creator principal the commissioner path records a new league under; the
// league's commissioner seat is taken later with join-link, so nobody is seated here.
const createLeaguePrincipal = "creator"

// runCreateLeague is the console command that makes a brand-new native league on a stopped server: its
// own file, the create command through the ordinary commissioner path (which records the starting
// snapshot with empty rosters), and its directory row. It seats nobody; empty seats play automatically.
func runCreateLeague(ctx context.Context, args []string, input io.Reader, output io.Writer) error {
	req, err := parseCreateLeague(args)
	if err != nil {
		return err
	}
	lock, err := lockStoppedServer(req.path)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()
	var raw accounts.RawCredentials
	if req.commissioner {
		if raw, err = readConsoleCredentials(input); err != nil {
			return err
		}
	}
	path, league, name, preset, season, teams := &req.path, &req.league, &req.name, &req.preset, &req.season, &req.teams
	if err := checkHostedDatabase(*path, 0); err != nil {
		return err
	}
	files, err := openStateFiles(ctx, *path)
	if err != nil {
		return err
	}
	defer func() { _ = files.Close() }()
	if _, ok := files.leagues.pools(*league); ok {
		return fmt.Errorf("league %s is already on this server; nothing was changed", *league)
	}
	principal, by, err := createLeagueCommissioner(ctx, files, req.commissioner, raw)
	if err != nil {
		return err
	}
	created := &createdLeague{files: files.leagues}
	store, err := created.create(ctx, *league)
	if err != nil {
		return errors.Join(fmt.Errorf("create league %s: %w", *league, err), files.leagues.discard(created.league))
	}
	command := commissioner.RawCommand{League: *league, Command: commissioner.Create,
		Args: commissioner.RawArgs{Name: *name, Preset: *preset, Source: commissioner.SourceNative,
			Season: *season, FranchiseCount: *teams},
		Request: "create", Reason: "Created from the server console", Effective: "now", Visibility: "league"}
	if _, err := store.ExecuteCommissionerWithFacts(ctx, command, string(principal), nil, nil, time.UTC); err != nil {
		return errors.Join(fmt.Errorf("the league was refused: %w; nothing was changed", err), files.leagues.discard(*league))
	}
	if err := recordDirectory(ctx, store, signInStore(files)); err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, `League %s ("%s") is on this server with %d franchises, empty rosters and a starting snapshot. Its commissioner is %s. Nobody is seated: every franchise plays automatically.
Next, with the server stopped unless noted:
  twr-server pin-football -db %s -league %s -football <digest>
  twr-server refresh-nfl-schedule -db %s -league %s -season %d   (after the schedule is published)
  twr-server join-link -db %s -league %s -take franchise.01 -no-login   (seats the commissioner)
`, *league, *name, *teams, by, *path, *league, *path, *league, *season, *path, *league)
	if err != nil {
		return fmt.Errorf("report create-league (the league is already created): %w", err)
	}
	return nil
}

// knownPreset refuses a preset the app does not list, in plain words naming the ones it does.
func knownPreset(id string) error {
	presets := commissioner.StartingPresets()
	ids := make([]string, len(presets))
	for i, p := range presets {
		ids[i] = p.ID
	}
	if !slices.Contains(ids, id) {
		return fmt.Errorf("preset %q is not one this app knows; choose one of: %s; nothing was changed", id, strings.Join(ids, ", "))
	}
	return nil
}

type createLeagueRequest struct {
	path, league, name, preset string
	season, teams              int
	commissioner               bool
}

// createLeagueCommissioner is the principal the new league is created under: the named account when
// -commissioner is given (made if the username is free, otherwise signed in by its password), else
// the console's own creator principal, which no account holds.
func createLeagueCommissioner(ctx context.Context, files stateFiles, named bool, raw accounts.RawCredentials) (governance.Principal, string, error) {
	if !named {
		return createLeaguePrincipal, "the console (no account)", nil
	}
	store := signInStore(files)
	if err := store.Initialize(ctx); err != nil {
		return "", "", fmt.Errorf("initialize server accounts: %w", err)
	}
	account, err := accounts.New(store).ConsoleAccount(ctx, raw)
	if err != nil {
		return "", "", fmt.Errorf("commissioner account: %w; nothing was changed", err)
	}
	principal, err := account.Principal()
	if err != nil {
		return "", "", fmt.Errorf("commissioner identity: %w", err)
	}
	return principal, account.Username, nil
}

func parseCreateLeague(args []string) (createLeagueRequest, error) {
	var r createLeagueRequest
	flags := flag.NewFlagSet("twr-server create-league", flag.ContinueOnError)
	flags.StringVar(&r.path, "db", "", "existing server database (server stopped)")
	flags.StringVar(&r.league, "league", "", "new league ID (native.…)")
	flags.StringVar(&r.name, "name", "", "the league's name")
	flags.StringVar(&r.preset, "preset", "", "starting preset ID, e.g. default_32")
	flags.IntVar(&r.season, "season", 0, "NFL season the league starts in")
	flags.IntVar(&r.teams, "teams", 32, "number of franchises")
	flags.BoolVar(&r.commissioner, "commissioner", false,
		"read the commissioner's username and password as JSON on stdin; the account is made if the name is free")
	if err := flags.Parse(args); err != nil {
		return r, fmt.Errorf("parse create-league flags: %w", err)
	}
	if flags.NArg() != 0 || r.path == "" || r.league == "" || strings.TrimSpace(r.name) == "" || r.preset == "" || r.season == 0 {
		return r, fmt.Errorf("create-league needs -db, -league, -name, -preset and -season, with the server stopped")
	}
	if !strings.HasPrefix(r.league, "native.") {
		return r, fmt.Errorf("a new league's ID starts with native. (for example native.rehearsal); nothing was changed")
	}
	return r, knownPreset(r.preset)
}
