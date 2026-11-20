package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

const nflScheduleRefreshTimeout = 2 * time.Minute

// runRefreshNFLSchedule imports only missing NFL dates for one published league schedule. The server
// must be stopped; the command writes a new immutable reference object and changes that league's pin.
func runRefreshNFLSchedule(ctx context.Context, args []string, output io.Writer) (retErr error) {
	flags := flag.NewFlagSet("twr-server refresh-nfl-schedule", flag.ContinueOnError)
	path := flags.String("db", "", "existing server database (server stopped)")
	league := flags.String("league", "", "league ID (native.…)")
	season := flags.Int("season", 0, "NFL season to refresh")
	pinFile := flags.String("football-pin-file", "/etc/twr/football.sha256", "server-level football pin file used when this league has no individual pin")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse refresh-nfl-schedule flags: %w", err)
	}
	if flags.NArg() != 0 || *path == "" || *league == "" || *season < 1920 || *season > 9999 {
		return fmt.Errorf("refresh-nfl-schedule needs -db, -league, and a valid -season; stop the server first")
	}
	if err := checkHostedDatabase(*path, 0); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, nflScheduleRefreshTimeout)
	defer cancel()
	files, err := openStateFiles(ctx, *path)
	if err != nil {
		return err
	}
	defer func() {
		if err := files.Close(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("close server state: %w", err))
		}
	}()
	store, err := leagueStore(ctx, files, *league)
	if err != nil {
		return err
	}
	weeks, err := publishedNFLWeeks(ctx, store, *league, *season)
	if err != nil {
		return err
	}
	signIns := signInStore(files)
	digest, err := selectedFootballDigest(ctx, signIns, *league, *pinFile)
	if err != nil {
		return err
	}
	return refreshNFLWeekEvidence(ctx, store, signIns, *league, *season, digest, weeks, newNFLWeekFetcher(store, nil), output)
}

func publishedNFLWeeks(ctx context.Context, store *history.Store, league string, season int) ([]int, error) {
	records, err := store.SharedCommissionerRecords(ctx, league)
	if err != nil {
		return nil, fmt.Errorf("read published league schedule: %w", err)
	}
	published, err := commissioner.ScheduleReading(league, records)
	if err != nil {
		return nil, fmt.Errorf("read published league schedule: %w", err)
	}
	if published.Season != season {
		return nil, fmt.Errorf("league %s publishes season %d, not requested season %d", league, published.Season, season)
	}
	if len(published.Weeks) == 0 {
		return nil, fmt.Errorf("league %s has no published schedule; nothing was changed", league)
	}
	weeks := make([]int, 0, len(published.Weeks))
	for _, week := range published.Weeks {
		weeks = append(weeks, week.Week)
	}
	return weeks, nil
}

func refreshNFLWeekEvidence(ctx context.Context, store *history.Store, signIns *accountstore.Store, league string, season int, digest string, expected []int, fetch nflWeekFetcher, output io.Writer) error {
	football, err := store.LoadFootball(ctx, digest)
	if err != nil {
		return fmt.Errorf("load pinned football evidence: %w", err)
	}
	held, err := football.ScheduleDates(ctx, season)
	if err != nil {
		return fmt.Errorf("read pinned NFL dates: %w", err)
	}
	missing, err := fetchMissingNFLWeeks(ctx, season, expected, held, fetch)
	if err != nil {
		return err
	}
	stale, err := refetchStaleNFLWeeks(ctx, season, time.Now().UTC(), held, fetch)
	if err != nil {
		return err
	}
	if len(stale) > 0 {
		if _, err := fmt.Fprintf(output, "Re-fetched results for %d past week(s) that were not final: %s.\n", len(stale), joinWeekNumbers(stale)); err != nil {
			return fmt.Errorf("report schedule refresh: %w", err)
		}
	}
	if len(missing) == 0 {
		if _, err := fmt.Fprintf(output, "All %d published league weeks already have NFL kickoff dates; no changes made.\n", len(expected)); err != nil {
			return fmt.Errorf("report schedule refresh: %w", err)
		}
		return nil
	}
	next, err := store.ExtendFootballSchedule(ctx, digest, missing, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("import refreshed NFL schedule: %w", err)
	}
	if _, err := store.LoadFootball(ctx, next); err != nil {
		return fmt.Errorf("verify imported football evidence %s: %w", next, err)
	}
	if err := signIns.PinFootball(ctx, league, next); err != nil {
		return fmt.Errorf("football evidence %s was imported, but updating league %s's pin failed: %w", next, league, err)
	}
	if _, err := fmt.Fprintf(output, "Added NFL kickoff dates for %d missing published week(s) in %s. League history was not changed; restart the server to load the new pin.\n", len(missing), league); err != nil {
		return fmt.Errorf("report schedule refresh: %w", err)
	}
	return nil
}

func newNFLWeekFetcher(store *history.Store, base http.RoundTripper) nflWeekFetcher {
	var client *mfl.Client
	return func(ctx context.Context, season, week int) (leagueweek.Week, error) {
		if client == nil {
			transport := &archive.Transport{Base: base, Sink: store, Judge: ingestion.MFLRefusal}
			var err error
			client, err = mfl.New("api", 2, mfl.WithTransport(transport))
			if err != nil {
				return leagueweek.Week{}, fmt.Errorf("create MFL API client: %w", err)
			}
		}
		return fetchNFLWeek(ctx, client, season, week)
	}
}

func selectedFootballDigest(ctx context.Context, signIns *accountstore.Store, league, pinFile string) (string, error) {
	directory, err := signIns.Directory(ctx)
	if err != nil {
		return "", fmt.Errorf("read server league directory: %w", err)
	}
	for _, entry := range directory {
		if entry.League != league {
			continue
		}
		if entry.Football != "" {
			return entry.Football, nil
		}
		body, err := os.ReadFile(pinFile)
		if err != nil {
			return "", fmt.Errorf("read server football pin file: %w", err)
		}
		digest := strings.TrimSpace(string(body))
		if len(digest) != 64 {
			return "", fmt.Errorf("server football pin file must contain one SHA-256 digest")
		}
		if _, err := hex.DecodeString(digest); err != nil {
			return "", fmt.Errorf("server football pin file has an invalid SHA-256 digest: %w", err)
		}
		return digest, nil
	}
	return "", fmt.Errorf("league %s is not recorded in the server directory", league)
}

type nflWeekFetcher func(context.Context, int, int) (leagueweek.Week, error)

func fetchNFLWeek(ctx context.Context, client *mfl.Client, season, week int) (leagueweek.Week, error) {
	raw, err := nflschedule.Fetch(ctx, client, strconv.Itoa(season), week)
	if err != nil {
		return leagueweek.Week{}, fmt.Errorf("fetch NFL week %d: %w", week, err)
	}
	if raw.Week != strconv.Itoa(week) {
		return leagueweek.Week{}, fmt.Errorf("fetch NFL week %d: source returned week %q", week, raw.Week)
	}
	converted, err := nflschedule.ToWeek(raw, time.Now().UTC())
	if err != nil {
		return leagueweek.Week{}, fmt.Errorf("validate NFL week %d: %w", week, err)
	}
	return converted, nil
}

// fetchMissingNFLWeeks collects and validates every requested addition before the caller persists any.
func fetchMissingNFLWeeks(ctx context.Context, season int, expected []int, held []leagueweek.Week, fetch nflWeekFetcher) ([]leagueweek.Week, error) {
	present := make(map[int]struct{}, len(held)+len(expected))
	for _, week := range held {
		present[week.Number] = struct{}{}
	}
	seen := make(map[int]struct{}, len(expected))
	missing := make([]leagueweek.Week, 0, len(expected))
	for _, number := range expected {
		if _, duplicate := seen[number]; duplicate {
			return nil, fmt.Errorf("published league schedule repeats week %d", number)
		}
		seen[number] = struct{}{}
		if _, ok := present[number]; ok {
			continue
		}
		week, err := fetch(ctx, season, number)
		if err != nil {
			return nil, fmt.Errorf("refresh NFL dates: %w", err)
		}
		if week.Number != number {
			return nil, fmt.Errorf("refresh NFL dates: requested week %d but source returned week %d", number, week.Number)
		}
		missing = append(missing, week)
		present[number] = struct{}{}
	}
	return missing, nil
}

// refetchStaleNFLWeeks re-fetches each held week whose games have all kicked off but are not all
// final, so the newest archived copy carries results. It changes no pin and extends no schedule.
func refetchStaleNFLWeeks(ctx context.Context, season int, now time.Time, held []leagueweek.Week, fetch nflWeekFetcher) ([]int, error) {
	var stale []int
	var errs []error
	for _, week := range held {
		if !pastNotFinal(now, week) {
			continue
		}
		stale = append(stale, week.Number)
		if _, err := fetch(ctx, season, week.Number); err != nil {
			errs = append(errs, fmt.Errorf("re-fetch NFL week %d: %w", week.Number, err))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("refresh past NFL results: %w", err)
	}
	return stale, nil
}

// pastNotFinal holds when every game has kicked off and at least one game is not yet final.
func pastNotFinal(now time.Time, week leagueweek.Week) bool {
	if len(week.Games) == 0 {
		return false
	}
	allFinal := true
	for _, game := range week.Games {
		if now.Before(game.Kickoff) {
			return false
		}
		if !game.Final {
			allFinal = false
		}
	}
	return !allFinal
}

func joinWeekNumbers(weeks []int) string {
	parts := make([]string, len(weeks))
	for i, week := range weeks {
		parts[i] = strconv.Itoa(week)
	}
	return strings.Join(parts, ", ")
}
