package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leagueload"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

const mirrorFetchTimeout = 5 * time.Minute

type mirrorFetchOptions struct {
	dbPath    string
	mflLeague string
	year      int
}

type mirrorFetchEnv struct {
	transport http.RoundTripper
	keySource func(ctx context.Context, league string, season int) (mflkey.Key, error)
	now       func() time.Time
	rps       float64
}

// runMirrorFetch pulls fresh MFL league exports into a mirror history.db using the same fetch
// and archive path as the desktop Prepare step. It writes to a temporary file beside -db and
// renames atomically only after the full fetch and offline verification succeed.
func runMirrorFetch(ctx context.Context, args []string, output io.Writer) error {
	return runMirrorFetchWithEnv(ctx, args, mirrorFetchEnv{}, output)
}

func runMirrorFetchWithEnv(
	ctx context.Context, args []string, env mirrorFetchEnv, output io.Writer,
) error {
	opts, err := parseMirrorFetchFlags(args)
	if err != nil {
		return err
	}
	if err := checkMirrorDestination(opts.dbPath); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, mirrorFetchTimeout)
	defer cancel()
	res, err := fetchMirrorToFile(ctx, opts, env)
	if err != nil {
		return err
	}
	return reportMirrorFetch(output, opts.dbPath, res)
}

func parseMirrorFetchFlags(args []string) (mirrorFetchOptions, error) {
	flags := flag.NewFlagSet("twr-server mirror-fetch", flag.ContinueOnError)
	var opts mirrorFetchOptions
	flags.StringVar(&opts.dbPath, "db", "", "destination path for the MFL mirror history.db")
	flags.StringVar(&opts.mflLeague, "mfl-league", "", "MFL league number to fetch")
	flags.IntVar(&opts.year, "year", 0, "optional 4-digit MFL season year (default: current season)")
	if err := flags.Parse(args); err != nil {
		return mirrorFetchOptions{}, fmt.Errorf("parse mirror-fetch flags: %w", err)
	}
	opts.dbPath = strings.TrimSpace(opts.dbPath)
	opts.mflLeague = strings.TrimSpace(opts.mflLeague)
	if flags.NArg() != 0 || opts.dbPath == "" || opts.mflLeague == "" {
		return mirrorFetchOptions{}, fmt.Errorf(
			"mirror-fetch needs -db and -mfl-league (optional: -year)")
	}
	if opts.year != 0 && (opts.year < 1920 || opts.year > 9999) {
		return mirrorFetchOptions{}, fmt.Errorf("mirror-fetch -year must be a 4-digit season year")
	}
	return opts, nil
}

func checkMirrorDestination(dbPath string) error {
	dir := filepath.Dir(dbPath)
	dirInfo, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("check mirror directory %s: %w", dir, err)
	}
	if !dirInfo.IsDir() {
		return fmt.Errorf("mirror directory %s is not a directory", dir)
	}
	if info, err := os.Stat(dbPath); err == nil && info.IsDir() {
		return fmt.Errorf("mirror destination %s is a directory, not a file", dbPath)
	}
	return nil
}

func fetchMirrorToFile(
	ctx context.Context, opts mirrorFetchOptions, env mirrorFetchEnv,
) (res leagueload.MirrorFetchResult, retErr error) {
	tmpPath, err := createTempMirrorPath(filepath.Dir(opts.dbPath))
	if err != nil {
		return leagueload.MirrorFetchResult{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = discardPending(tmpPath)
		}
	}()
	res, err = populateTempMirror(ctx, tmpPath, opts, env)
	if err != nil {
		return leagueload.MirrorFetchResult{}, err
	}
	if err := finalizeMirrorFile(tmpPath, opts.dbPath); err != nil {
		return leagueload.MirrorFetchResult{}, err
	}
	committed = true
	return res, nil
}

func createTempMirrorPath(dir string) (string, error) {
	f, err := os.CreateTemp(dir, ".mirror-fetch-*.db.tmp")
	if err != nil {
		return "", fmt.Errorf("create temporary mirror file: %w", err)
	}
	path := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("prepare temporary mirror file: %w", err)
	}
	return path, nil
}

func populateTempMirror(
	ctx context.Context, tmpPath string, opts mirrorFetchOptions, env mirrorFetchEnv,
) (leagueload.MirrorFetchResult, error) {
	pools, err := db.Open(ctx, tmpPath, history.SchemaVersion())
	if err != nil {
		return leagueload.MirrorFetchResult{}, fmt.Errorf("open temporary mirror: %w", err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = pools.Close()
		}
	}()
	registry, err := measures.Embedded()
	if err != nil {
		return leagueload.MirrorFetchResult{}, fmt.Errorf("load server measures: %w", err)
	}
	store := history.New(pools, registry)
	if err := store.Initialize(ctx); err != nil {
		return leagueload.MirrorFetchResult{}, fmt.Errorf("initialize temporary mirror: %w", err)
	}
	res, err := leagueload.FetchMirror(ctx, leagueload.MirrorFetchRequest{
		MFLLeague: opts.mflLeague,
		Year:      opts.year,
		Store:     store,
		Transport: env.transport,
		KeySource: env.keySource,
		Now:       env.now,
		RPS:       env.rps,
	})
	if err != nil {
		return leagueload.MirrorFetchResult{}, fmt.Errorf("mirror-fetch: %w", err)
	}
	if err := checkpointLeague(ctx, pools); err != nil {
		return leagueload.MirrorFetchResult{}, fmt.Errorf("flush temporary mirror: %w", err)
	}
	closed = true
	if err := pools.Close(); err != nil {
		return leagueload.MirrorFetchResult{}, fmt.Errorf("close temporary mirror: %w", err)
	}
	return res, nil
}

func finalizeMirrorFile(tmpPath, dbPath string) error {
	if err := removeBuildBackups(tmpPath); err != nil {
		return err
	}
	if err := removeMirrorSidecars(tmpPath); err != nil {
		return err
	}
	if err := syncPath(tmpPath); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, dbPath); err != nil {
		return fmt.Errorf("put mirror file in place: %w", err)
	}
	if err := removeMirrorSidecars(dbPath); err != nil {
		return err
	}
	return syncPath(filepath.Dir(dbPath))
}

func removeMirrorSidecars(path string) error {
	var errs []error
	for _, name := range []string{path + "-wal", path + "-shm", path + "-journal"} {
		if err := os.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, fmt.Errorf("remove %s: %w", filepath.Base(name), err))
		}
	}
	return errors.Join(errs...)
}

func reportMirrorFetch(output io.Writer, dbPath string, res leagueload.MirrorFetchResult) error {
	nextCmd := fmt.Sprintf(
		"twr-server league-refresh -db /var/lib/twr/server.db -mirror %s -league %s"+
			" -dry-run < commissioner.json",
		dbPath, res.League)
	lines := []string{
		fmt.Sprintf(
			"Fetched MFL mirror for %s (league %s, %d season) into %s.",
			res.LeagueName, res.MFLLeague, res.Season, dbPath),
		fmt.Sprintf("  Fetched at: %s", res.FetchedAt.UTC().Format("2006-01-02 15:04:05 UTC")),
		fmt.Sprintf("  Franchises: %d", res.Franchises),
		fmt.Sprintf("  Players: %d", res.Players),
		fmt.Sprintf("  Contracts: %d", res.Contracts),
		fmt.Sprintf("  Transactions: %d", res.Transactions),
		fmt.Sprintf("  Weeks of scores: %d", res.ScoreWeeks),
		"Next step (dry run first):",
		"  " + nextCmd,
	}
	if _, err := io.WriteString(output, strings.Join(lines, "\n")+"\n"); err != nil {
		return fmt.Errorf("report mirror fetch: %w", err)
	}
	return nil
}
