package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leagueload"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// runLeagueLoad is a local console command, run with the hosted service stopped. It puts the league
// held in a desktop MFL mirror onto this server: the commissioner's account is created on the server
// (first account only, or the existing account after its password checks), the native league is
// built from held MFL reads only, then moved here by the confirmed seal/activate path. The mirror is
// opened read-only; no MFL call is made.
func runLeagueLoad(ctx context.Context, args []string, input io.Reader, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server league-load", flag.ContinueOnError)
	path := flags.String("db", "", "existing server database (service stopped)")
	mirrorPath := flags.String("mirror", "", "a copy of the commissioner's desktop history.db; opened read-only")
	mflLeague := flags.String("mfl-league", "", "MFL league number held in the mirror")
	pin := flags.String("football-pin", "", "file to receive the pinned football digest the service starts with; empty writes none")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse league load flags: %w", err)
	}
	if flags.NArg() != 0 || *mflLeague == "" {
		return fmt.Errorf("league-load accepts only -db, -mirror, -mfl-league and -football-pin")
	}
	if err := checkHostedDatabase(*path, 0); err != nil {
		return err
	}
	raw, err := readConsoleCredentials(input)
	if err != nil {
		return err
	}
	return loadLeague(ctx, *path, *mirrorPath, *mflLeague, *pin, raw, output)
}

func loadLeague(ctx context.Context, path, mirrorPath, mflLeague, pin string, raw accounts.RawCredentials, output io.Writer) error {
	mirrorPools, err := db.OpenReadOnly(ctx, mirrorPath, history.SchemaVersion())
	if err != nil {
		return fmt.Errorf("open the mirror read-only: %w", err)
	}
	defer func() { _ = mirrorPools.Close() }()
	pools, err := openHostedFile(ctx, path)
	if err != nil {
		return fmt.Errorf("open server database: %w", err)
	}
	defer func() { _ = pools.Close() }()
	registry, err := measures.Embedded()
	if err != nil {
		return fmt.Errorf("load server measures: %w", err)
	}
	server := history.New(pools, registry)
	if err := server.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize server history: %w", err)
	}
	commissioner, err := commissionerAccount(ctx, pools, raw)
	if err != nil {
		return err
	}
	principal, err := commissioner.Principal()
	if err != nil {
		return fmt.Errorf("commissioner identity: %w", err)
	}
	workspace, cleanup, err := scratchHistory(ctx, registry)
	if err != nil {
		return err
	}
	defer cleanup()
	mirror := leagueload.Mirror{Store: history.New(mirrorPools, registry)}
	result, err := leagueload.Load(ctx, leagueload.Request{MFLLeague: mflLeague, Commissioner: principal, Workspace: workspace, Server: server,
		Source: leagueload.ReplaySource(ctx, mirror, mflLeague, time.Now),
		Football: func(ctx context.Context, season int) ([]byte, string, error) {
			return mirror.Football(ctx, season, time.Now())
		}, Now: time.Now})
	if err != nil {
		return fmt.Errorf("load the league: %w", err)
	}
	if pin != "" {
		if err := writePin(pin, result.Football); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(output, "League %s is active on this server with %d franchises; %s is its commissioner. Football evidence %s.\n",
		result.League, result.Franchises, commissioner.Username, result.Football)
	if err != nil {
		return fmt.Errorf("report league load (the load has already committed): %w", err)
	}
	return nil
}

// commissionerAccount creates the server's first account, or signs in the existing one by password.
func commissionerAccount(ctx context.Context, pools *db.Pools, raw accounts.RawCredentials) (accounts.Account, error) {
	store := accountstore.New(pools)
	if err := store.Initialize(ctx); err != nil {
		return accounts.Account{}, fmt.Errorf("initialize server accounts: %w", err)
	}
	service := accounts.New(store)
	account, err := service.LocalBootstrap(ctx, raw)
	if errors.Is(err, accounts.ErrBootstrapClosed) {
		account, err = service.Authenticate(ctx, raw)
	}
	if err != nil {
		return accounts.Account{}, fmt.Errorf("commissioner account: %w", err)
	}
	return account, nil
}

// scratchHistory is the disposable workspace the native league is built in before it moves.
func scratchHistory(ctx context.Context, registry *measures.Registry) (*history.Store, func(), error) {
	pools, err := db.OpenScratch(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("open league workspace: %w", err)
	}
	store := history.New(pools, registry)
	if err := store.Initialize(ctx); err != nil {
		_ = pools.Close()
		return nil, nil, fmt.Errorf("initialize league workspace: %w", err)
	}
	return store, func() { _ = pools.Close() }, nil
}

func readConsoleCredentials(input io.Reader) (accounts.RawCredentials, error) {
	body, err := io.ReadAll(io.LimitReader(input, 4097))
	if err != nil || len(body) > 4096 {
		return accounts.RawCredentials{}, fmt.Errorf("commissioner input exceeds 4096 bytes or cannot be read")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	// RawCredentials keeps the password out of JSON on purpose; this console input is the one place it is read.
	var fields struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decoder.Decode(&fields); err != nil {
		return accounts.RawCredentials{}, fmt.Errorf("supply the commissioner's username and password as one JSON object on stdin")
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		return accounts.RawCredentials{}, fmt.Errorf("supply exactly one JSON object on stdin")
	}
	raw := accounts.RawCredentials{Username: fields.Username, Password: fields.Password}
	if err := raw.Validate(); err != nil {
		return accounts.RawCredentials{}, fmt.Errorf("commissioner credentials: %w", err)
	}
	return raw, nil
}

// writePin replaces the pin file atomically; it holds only a public digest.
func writePin(path, digest string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".football-pin-")
	if err != nil {
		return fmt.Errorf("write football pin: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.WriteString(digest + "\n"); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write football pin: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write football pin: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write football pin: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("write football pin: %w", err)
	}
	return nil
}
