package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"filippo.io/age"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/offsite"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// parseRestoreInput reads the restore settings object the restore scripts send on stdin.
func parseRestoreInput(input io.Reader) (*age.X25519Identity, offsite.RawConfig, time.Time, error) {
	var raw rawRestoreInput
	decoder := json.NewDecoder(io.LimitReader(input, 8192))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return nil, offsite.RawConfig{}, time.Time{}, fmt.Errorf("restore input is not one valid settings object")
	}
	identity, err := age.ParseX25519Identity(strings.ToUpper(strings.Join(strings.Fields(raw.RestoreKey), "")))
	if err != nil {
		return nil, offsite.RawConfig{}, time.Time{}, fmt.Errorf("that restore key is not right: use the line beginning AGE-SECRET-KEY-1 from the sealed copy")
	}
	var before time.Time
	if raw.Before != "" {
		day, err := time.Parse(time.DateOnly, strings.TrimSpace(raw.Before))
		if err != nil {
			return nil, offsite.RawConfig{}, time.Time{}, fmt.Errorf("that date is not right: use YYYY-MM-DD, or leave it blank for the newest backup")
		}
		before = day.AddDate(0, 0, 1)
	}
	config := offsite.RawConfig{Endpoint: raw.Endpoint, Bucket: raw.Bucket, AccessKey: raw.AccessKey, SecretKey: raw.SecretKey, Recipient: identity.Recipient().String()}
	return identity, config, before, nil
}

// runRestoreLeague puts one league back from its own off-site backup on a stopped server (Christopher,
// 2026-11-14: restoring one league signs out only that league's owners). Every other league, its file
// and its owners' sign-ins are untouched. The league's file before the restore is kept in the archive
// for 30 days.
func runRestoreLeague(ctx context.Context, args []string, input io.Reader, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server restore-league", flag.ContinueOnError)
	path := flags.String("db", "", "existing server database (server stopped)")
	league := flags.String("league", "", "league ID (native.…)")
	stage := flags.String("stage", "", "new private working folder")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse restore-league flags: %w", err)
	}
	if flags.NArg() != 0 || *path == "" || *league == "" || *stage == "" {
		return fmt.Errorf("restore-league needs -db, -league and a new -stage folder, with the server stopped")
	}
	if !validLeagueID(*league) {
		return fmt.Errorf("%q is not a league id", *league)
	}
	identity, config, before, err := parseRestoreInput(input)
	if err != nil {
		return err
	}
	r2, err := offsite.NewR2(config)
	if err != nil {
		return fmt.Errorf("check the R2 settings: %w", err)
	}
	if err := checkHostedDatabase(*path, 0); err != nil {
		return err
	}
	files, err := openStateFiles(ctx, *path)
	if err != nil {
		return err
	}
	defer func() { _ = files.Close() }()
	return restoreOneLeague(ctx, r2, identity, files, *league, *stage, before, time.Now(), output)
}

func restoreOneLeague(ctx context.Context, sets recoverySets, identity *age.X25519Identity, files stateFiles, league, stage string,
	before, now time.Time, output io.Writer) error {
	byLeague, err := leagueSetNames(ctx, sets, before)
	if err != nil {
		return err
	}
	if len(byLeague[league]) == 0 {
		return fmt.Errorf("the bucket holds no backup of league %s; nothing was changed", league)
	}
	if err := os.Mkdir(stage, 0o700); err != nil {
		return fmt.Errorf("create restore folder: %w", err)
	}
	staged := filepath.Join(stage, leagueFileName)
	captured, err := restoreLeagueSet(ctx, sets, byLeague[league], league, identity, filepath.Join(stage, "sets"), staged, "", output)
	if err != nil {
		return fmt.Errorf("%w; nothing was changed", err)
	}
	put, err := completeLeagueReference(ctx, sets, identity, files, league, staged)
	if err != nil {
		return err
	}
	owners, err := leaguePrincipals(ctx, staged)
	if err != nil {
		return err
	}
	if current, ok := files.leagues.pools(league); ok {
		named, err := history.NamedPrincipals(ctx, current)
		if err != nil {
			return fmt.Errorf("read league %s's current owners: %w", league, err)
		}
		owners = append(owners, named...)
	}
	slices.Sort(owners)
	owners = slices.Compact(owners)
	// Signed out first: a stop after this leaves owners signed out, never a restored league with old sign-ins.
	signedOut, err := signInStore(files).SignOutPrincipals(ctx, owners, "league "+league+" restored from the backup saved "+captured.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("sign out league %s's owners: %w; nothing else was changed", league, err)
	}
	kept, err := installLeagueFile(ctx, files, league, staged, now)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(stage); err != nil {
		return fmt.Errorf("remove restore working folder: %w", err)
	}
	return reportLeagueRestore(output, league, captured, signedOut, put, kept, now)
}

// installLeagueFile replaces league's file with the staged one; the file it replaces goes to the archive.
func installLeagueFile(ctx context.Context, files stateFiles, league, staged string, now time.Time) (string, error) {
	root := files.leagues.root
	kept := ""
	if _, ok := files.leagues.pools(league); ok {
		if err := files.leagues.remove(league); err != nil {
			return "", err
		}
		var err error
		if kept, err = archiveLeague(root, league, now); err != nil {
			return "", err
		}
	}
	final, err := leagueFilePath(root, league)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(final), 0o700); err != nil {
		return "", fmt.Errorf("make league %s's folder: %w", league, err)
	}
	if err := os.Rename(staged, final); err != nil {
		return "", fmt.Errorf("put league %s's restored file in place: %w", league, err)
	}
	if err := errors.Join(syncPath(filepath.Dir(final)), syncPath(root)); err != nil {
		return "", err
	}
	store, err := files.leagues.adopt(ctx, league)
	if err != nil {
		return "", err
	}
	return kept, recordDirectory(ctx, store, signInStore(files))
}

// completeLeagueReference gives the server's reference file what the restored league needs from it:
// the football version the league pins, from its own evidence object, and the league's players from
// its own restore points. It reports what was put back.
func completeLeagueReference(ctx context.Context, sets recoverySets, identity *age.X25519Identity, files stateFiles, league, staged string) (referenceFill, error) {
	var out referenceFill
	directory, err := signInStore(files).Directory(ctx)
	if err != nil {
		return out, err //nolint:wrapcheck // names what was read
	}
	var pins []string
	for _, entry := range directory {
		if entry.League == league && entry.Football != "" {
			pins = append(pins, entry.Football)
		}
	}
	_, missing, err := history.MissingReference(ctx, files.reference, nil, pins)
	if err != nil {
		return out, err //nolint:wrapcheck // names what was read
	}
	for _, digest := range missing {
		stored, err := openEvidence(ctx, sets, identity, digest)
		if err != nil {
			continue // not off-site either; the league serves without it until it is back
		}
		if err := history.PutArchiveBody(ctx, files.reference, digest, stored); err != nil {
			return out, err //nolint:wrapcheck // names the capture
		}
		out.pins++
	}
	pools, err := db.OpenReadOnly(ctx, staged, history.LeagueSchemaVersion())
	if err != nil {
		return out, fmt.Errorf("open the restored league file: %w", err)
	}
	defer func() { _ = pools.Close() }()
	recovered, err := history.RecoverLeagueIdentities(ctx, pools, files.reference)
	if err != nil {
		return out, err //nolint:wrapcheck // names what was read
	}
	out.players, out.conflicts = recovered.Installed, recovered.Conflicts
	return out, nil
}

func reportLeagueRestore(output io.Writer, league string, captured time.Time, signedOut int, put referenceFill, kept string, now time.Time) error {
	var b strings.Builder
	fmt.Fprintf(&b, "League %s is restored from its backup saved %s.\n", league, captured.UTC().Format("2006-01-02 15:04:05 UTC"))
	fmt.Fprintf(&b, "Signed out its %d owner account(s); they sign in again with their passwords. Other leagues and their owners are untouched.\n", signedOut)
	if put.players+put.pins > 0 {
		fmt.Fprintf(&b, "Put back for it: %d player(s) and %d football version(s) the football reference file lacked.\n", put.players, put.pins)
	}
	for _, conflict := range put.conflicts {
		fmt.Fprintf(&b, "Left out player %s: another player already holds that outside id. Call Christopher.\n", conflict)
	}
	if kept != "" {
		fmt.Fprintf(&b, "Its file from before the restore is kept in %s until %s.\n", kept, now.Add(archiveDays*24*time.Hour).Format(time.DateOnly))
	}
	if _, err := io.WriteString(output, b.String()); err != nil {
		return fmt.Errorf("report restore (the league is already restored): %w", err)
	}
	return nil
}
