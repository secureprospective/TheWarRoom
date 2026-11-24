package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/injuries"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

const (
	nflInjuryRefreshCadence = 4 * time.Hour
	nflInjuryRefreshTimeout = 2 * time.Minute
)

type nflInjuryFeed struct {
	store *history.Store
	now   func() time.Time
}

func newNFLInjuryFeed(store *history.Store) nflInjuryFeed {
	return nflInjuryFeed{store: store, now: time.Now}
}

func (f nflInjuryFeed) ReadNFLInjuryFeed(ctx context.Context) (leagueservice.NFLInjuryFeed, error) {
	season := nflSeasonAt(f.now())
	feed := leagueservice.NFLInjuryFeed{Season: season}
	health, err := f.store.FetchHealth(ctx, "mfl")
	if err != nil {
		return feed, fmt.Errorf("read NFL injury fetch history: %w", err)
	}
	for _, entry := range health {
		if !ingestion.SameLeaguelessExport(entry.URL, injuries.Export, season) {
			continue
		}
		attempt := entry.LastSuccess
		if entry.LastFailure.After(attempt) {
			attempt = entry.LastFailure
		}
		if attempt.After(feed.LastAttemptAt) {
			feed.LastAttemptAt = attempt
		}
	}
	found, err := f.store.ArchivedBodies(ctx, "TYPE="+injuries.Export, func(rawURL string, body []byte, fetchedAt time.Time) bool {
		if !ingestion.SameLeaguelessExport(rawURL, injuries.Export, season) {
			return false
		}
		rows, parseErr := injuries.ParseToInjuries(ctx, body)
		if parseErr != nil {
			return false
		}
		feed.Injuries = rows
		feed.FetchedAt = fetchedAt
		feed.Available = true
		return true
	})
	if err != nil {
		return leagueservice.NFLInjuryFeed{}, fmt.Errorf("read held NFL injuries: %w", err)
	}
	if !found {
		feed.Injuries = []leaguefeed.Injury{}
	}
	return feed, nil
}

func (f nflInjuryFeed) Refresh(ctx context.Context, season int, base http.RoundTripper, output io.Writer) error {
	transport := &archive.Transport{Base: base, Sink: f.store, Judge: ingestion.MFLRefusal}
	client, err := mfl.New("api", 2, mfl.WithTransport(transport))
	if err != nil {
		return fmt.Errorf("create MFL API client: %w", err)
	}
	raw, err := injuries.Fetch(ctx, client, strconv.Itoa(season))
	if err != nil {
		return fmt.Errorf("fetch NFL injuries: %w", err)
	}
	rows, err := injuries.ToInjuries(raw)
	if err != nil {
		return fmt.Errorf("validate NFL injuries: %w", err)
	}
	if output != nil {
		if _, err := fmt.Fprintf(output, "Stored %d NFL injury status fact(s) for season %d.\n", len(rows), season); err != nil {
			return fmt.Errorf("report NFL injury refresh: %w", err)
		}
	}
	return nil
}

func nflSeasonAt(now time.Time) int {
	if now.Month() < time.July {
		return now.Year() - 1
	}
	return now.Year()
}

func startHostedInjuryRefresh(ctx context.Context, store *history.Store) func() {
	workerCtx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		feed := newNFLInjuryFeed(store)
		refresh := func() {
			attempt, cancel := context.WithTimeout(workerCtx, nflInjuryRefreshTimeout)
			defer cancel()
			season := nflSeasonAt(time.Now())
			if err := feed.Refresh(attempt, season, nil, nil); err != nil {
				log.Printf("hosted NFL injuries: %v", err)
				return
			}
			log.Printf("hosted NFL injuries: refreshed season %d", season)
		}
		refresh()
		ticker := time.NewTicker(nflInjuryRefreshCadence)
		defer ticker.Stop()
		for {
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
				refresh()
			}
		}
	}()
	return func() { stop(); <-done }
}

func runRefreshNFLInjuries(ctx context.Context, args []string, output io.Writer) (retErr error) {
	flags := flag.NewFlagSet("twr-server refresh-nfl-injuries", flag.ContinueOnError)
	path := flags.String("db", "", "existing server database (server stopped)")
	season := flags.Int("season", 0, "NFL season to refresh")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse refresh-nfl-injuries flags: %w", err)
	}
	if flags.NArg() != 0 || *path == "" || *season < 1920 || *season > 9999 {
		return fmt.Errorf("refresh-nfl-injuries needs -db and a valid -season; stop the server first")
	}
	lock, err := lockStoppedServer(*path)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()
	if err := checkHostedDatabase(*path, 0); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, nflInjuryRefreshTimeout)
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
	return newNFLInjuryFeed(files.leagues.family.Over(files.reference)).Refresh(ctx, *season, nil, output)
}

// ResolveNFLReturnWeeks maps expected return dates onto the league's pinned NFL weeks.
func (f nflInjuryFeed) ResolveNFLReturnWeeks(rows []leaguefeed.Injury, weeks []leagueweek.Week) []leaguefeed.Injury {
	raw := make([]nflschedule.RawWeek, 0, len(weeks))
	for _, week := range weeks {
		held := nflschedule.RawWeek{Week: strconv.Itoa(week.Number), Matchups: make([]nflschedule.RawMatchup, 0, len(week.Games))}
		for _, game := range week.Games {
			held.Matchups = append(held.Matchups, nflschedule.RawMatchup{Kickoff: strconv.FormatInt(game.Kickoff.Unix(), 10)})
		}
		raw = append(raw, held)
	}
	return injuries.ResolveReturnWeeks(rows, raw)
}
