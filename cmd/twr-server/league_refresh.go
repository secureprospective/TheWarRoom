package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leagueload"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type refreshOptions struct {
	dbPath     string
	mirrorPath string
	league     string
	mflLeague  string
	pinPath    string
	reason     string
	dryRun     bool
}

type heldRefresh struct {
	Source   func(mflLeague string, base leaguerules.RuleSet) (leaguemigration.Source, error)
	Football func(ctx context.Context, season int) ([]byte, string, error)
	Now      func() time.Time
}

// runLeagueRefresh is a local console command run with the hosted service stopped. It refreshes
// an already-loaded server league from a read-only desktop MFL mirror, preserving seats, roles,
// rules, and in-app commissioner edits while stopping if open in-app trade offers exist.
func runLeagueRefresh(ctx context.Context, args []string, input io.Reader, output io.Writer) error {
	opts, err := parseLeagueRefreshFlags(args)
	if err != nil {
		return err
	}
	raw, err := readConsoleCredentials(input)
	if err != nil {
		return err
	}
	if err := checkHostedDatabase(opts.dbPath, 0); err != nil {
		return err
	}
	return withStateDirectoryOwnership(filepath.Dir(opts.dbPath), func() error {
		return refreshFromMirror(ctx, opts, raw, output)
	})
}

func parseLeagueRefreshFlags(args []string) (refreshOptions, error) {
	flags := flag.NewFlagSet("twr-server league-refresh", flag.ContinueOnError)
	var opts refreshOptions
	flags.StringVar(&opts.dbPath, "db", "", "existing server database (service stopped)")
	flags.StringVar(&opts.mirrorPath, "mirror", "", "copy of desktop history.db (opened read-only)")
	flags.StringVar(&opts.league, "league", "", "league ID on this server (native.…)")
	flags.StringVar(&opts.mflLeague, "mfl-league", "", "optional MFL league number held in the mirror")
	flags.StringVar(&opts.pinPath, "football-pin", "", "optional file to receive the pinned football digest")
	flags.StringVar(&opts.reason, "reason", "", "optional reason recorded on refresh ledger entries")
	flags.BoolVar(&opts.dryRun, "dry-run", false, "plan and print summary without changing the server")
	if err := flags.Parse(args); err != nil {
		return refreshOptions{}, fmt.Errorf("parse league-refresh flags: %w", err)
	}
	if flags.NArg() != 0 || opts.dbPath == "" || opts.mirrorPath == "" || opts.league == "" {
		return refreshOptions{}, fmt.Errorf(
			"league-refresh needs -db, -mirror, and -league (optional: -mfl-league, -football-pin, -reason, -dry-run)")
	}
	return opts, nil
}

func refreshFromMirror(
	ctx context.Context, opts refreshOptions, raw accounts.RawCredentials, output io.Writer,
) error {
	mirrorPools, err := db.OpenReadOnly(ctx, opts.mirrorPath, history.SchemaVersion())
	if err != nil {
		return fmt.Errorf("open the mirror read-only: %w", err)
	}
	defer func() { _ = mirrorPools.Close() }()
	registry, err := measures.Embedded()
	if err != nil {
		return fmt.Errorf("load server measures: %w", err)
	}
	mirror := leagueload.Mirror{Store: history.New(mirrorPools, registry)}
	held := heldRefresh{
		Source: leagueload.ReplaySourceByLeague(ctx, mirror, time.Now),
		Football: func(ctx context.Context, season int) ([]byte, string, error) {
			return mirror.Football(ctx, season, time.Now())
		},
		Now: time.Now,
	}
	return refreshHeld(ctx, opts, held, raw, output)
}

func refreshHeld(
	ctx context.Context,
	opts refreshOptions,
	held heldRefresh,
	raw accounts.RawCredentials,
	output io.Writer,
) error {
	files, err := openStateFiles(ctx, opts.dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = files.Close() }()
	store, err := leagueStore(ctx, files, opts.league)
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
	signIns := signInStore(files)
	currentPin, err := currentLeaguePin(ctx, signIns, opts.league, opts.pinPath)
	if err != nil {
		return err
	}
	now := held.Now
	if now == nil {
		now = time.Now
	}
	res, err := leagueload.Refresh(ctx, leagueload.RefreshRequest{
		League:       opts.league,
		MFLLeague:    opts.mflLeague,
		Reason:       opts.reason,
		Commissioner: principal,
		Server:       store,
		CurrentPin:   currentPin,
		DryRun:       opts.dryRun,
		Source:       held.Source,
		Football:     held.Football,
		Now:          now,
	})
	if err != nil {
		return handleRefreshError(err, output)
	}
	if err := applyRefreshedPin(ctx, signIns, opts, res); err != nil {
		return err
	}
	return reportRefresh(output, res)
}

func handleRefreshError(err error, output io.Writer) error {
	var stopped *leagueload.StoppedTradesError
	if errors.As(err, &stopped) {
		if werr := reportStoppedTrades(output, stopped); werr != nil {
			return errors.Join(stopped, werr)
		}
		return stopped
	}
	return fmt.Errorf("refresh league: %w", err)
}

func applyRefreshedPin(
	ctx context.Context, signIns *accountstore.Store, opts refreshOptions, res leagueload.RefreshResult,
) error {
	if opts.dryRun || !res.FootballChanged {
		return nil
	}
	if err := signIns.PinFootball(ctx, opts.league, res.Football); err != nil {
		return fmt.Errorf("pin refreshed football evidence: %w", err)
	}
	if opts.pinPath != "" {
		if err := writePin(opts.pinPath, res.Football); err != nil {
			return err
		}
	}
	return nil
}

func currentLeaguePin(
	ctx context.Context, signIns *accountstore.Store, league, pinPath string,
) (string, error) {
	directory, err := signIns.Directory(ctx)
	if err != nil {
		return "", fmt.Errorf("read server league directory: %w", err)
	}
	for _, entry := range directory {
		if entry.League == league && entry.Football != "" {
			return entry.Football, nil
		}
	}
	if pinPath == "" {
		return "", nil
	}
	body, err := os.ReadFile(pinPath)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read football pin file: %w", err)
	}
	digest := strings.TrimSpace(string(body))
	if len(digest) != 64 {
		return "", fmt.Errorf("football pin file must contain one SHA-256 digest")
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return "", fmt.Errorf("football pin file has an invalid SHA-256 digest: %w", err)
	}
	return digest, nil
}

func reportStoppedTrades(output io.Writer, stopped *leagueload.StoppedTradesError) error {
	noun := "open trade offers"
	if len(stopped.Offers) == 1 {
		noun = "open trade offer"
	}
	header := fmt.Sprintf(
		"Refresh stopped for %s (%d season) — no changes were saved.\n"+
			"Settle %d %s before refreshing from MFL:\n",
		stopped.LeagueName, stopped.Season, len(stopped.Offers), noun)
	if _, err := io.WriteString(output, header); err != nil {
		return fmt.Errorf("report stopped trades: %w", err)
	}
	for _, offer := range stopped.Offers {
		line := fmt.Sprintf("  - %s: %s\n", joinTeamNames(offer.Franchises), offer.Status)
		if _, err := io.WriteString(output, line); err != nil {
			return fmt.Errorf("report stopped trades: %w", err)
		}
	}
	return nil
}

func reportRefresh(output io.Writer, res leagueload.RefreshResult) error {
	s := res.Summary
	lines := []string{
		refreshHeadline(res),
		fmt.Sprintf("  MFL transactions replayed: %d", s.Transactions),
		fmt.Sprintf("  Roster additions: %d", s.RosterAdds),
		fmt.Sprintf("  Roster removals: %d", s.RosterRemoves),
		fmt.Sprintf("  Roster status changes: %d", s.RosterStatusChanges),
		fmt.Sprintf("  Contract updates: %d", s.Contracts),
		fmt.Sprintf("  Salary cap adjustments: %d", s.Adjustments),
		fmt.Sprintf("  Draft pick updates: %d", s.Picks),
		fmt.Sprintf("  Franchise name updates: %d", s.Franchises),
		fmt.Sprintf("  Schedule and MFL scores: %s", scheduleWords(s)),
		fmt.Sprintf("  Weekly lineup updates: %d", s.Lineups),
		fmt.Sprintf("  Player list and NFL schedule: %s", changedWord(res.FootballChanged)),
		"  Open trade offers blocking refresh: none",
	}
	if _, err := io.WriteString(output, strings.Join(lines, "\n")+"\n"); err != nil {
		return fmt.Errorf("report league refresh: %w", err)
	}
	return nil
}

func refreshHeadline(res leagueload.RefreshResult) string {
	noChanges := res.Commands == 0 && !res.FootballChanged
	switch {
	case res.DryRun && noChanges:
		return fmt.Sprintf(
			"Dry run for %s (%d season) — already up to date; no changes needed.",
			res.LeagueName, res.Season)
	case res.DryRun:
		return fmt.Sprintf(
			"Dry run for %s (%d season) — no changes were saved.",
			res.LeagueName, res.Season)
	case noChanges:
		return fmt.Sprintf(
			"%s (%d season) is already up to date; no changes were made.",
			res.LeagueName, res.Season)
	default:
		return fmt.Sprintf("Refreshed %s (%d season) from MFL.", res.LeagueName, res.Season)
	}
}

func scheduleWords(s leaguemigration.RefreshSummary) string {
	status := changedWord(s.ScheduleUpdated)
	if s.RecordedWeeks == 0 && s.CutoverWeek <= 1 {
		return status + " (no completed MFL weeks yet)"
	}
	weekWord := "completed weeks"
	if s.RecordedWeeks == 1 {
		weekWord = "completed week"
	}
	through := s.CutoverWeek - 1
	if through < 1 {
		through = s.RecordedWeeks
	}
	return fmt.Sprintf(
		"%s (%d %s; MFL scores count through week %d)",
		status, s.RecordedWeeks, weekWord, through)
}

func changedWord(changed bool) string {
	if changed {
		return "updated"
	}
	return "unchanged"
}

func joinTeamNames(names []string) string {
	switch len(names) {
	case 0:
		return "unknown teams"
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	default:
		return strings.Join(names[:len(names)-1], ", ") + ", and " + names[len(names)-1]
	}
}
