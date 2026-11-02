package migration

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livescoring"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/transactions"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/weeklyresults"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// ScheduleSync refreshes reference schedules and lineups using the app's held resources.
// Running and Started belong to the app lifetime, shared across refreshes.
type ScheduleSync struct {
	History           *history.Store
	Authorizer        func() commissioner.Authorizer
	Client            *mfl.Client
	Now               func() time.Time
	FillScheduleDates func(context.Context, []commissioner.Record) error
	SyncScoring       func(context.Context, string, []commissioner.Record) error
	LineupPlayers     func(context.Context) (map[string]string, error)
	WorkContext       func() context.Context
	StartWorker       func(func())
	Running, Started  *atomic.Bool
}

// syncLeagueSchedules is a reference-feed refresh. Failures retain the published ledger
// and retry on the next refresh; they never stop the rest of the board.
func (s ScheduleSync) Sync(ctx context.Context) {
	if s.History == nil {
		return
	}
	leagues, err := s.History.NativeLeagues(ctx)
	if err != nil {
		log.Printf("read leagues for schedule refresh: %v", err)
		return
	}
	ledger := leagueservice.MigrationLedger{Store: s.History,
		Auth: func() commissioner.Authorizer { return s.Authorizer() }}
	var followed []followedLeague
	for _, league := range leagues {
		records, err := ledger.Records(ctx, league.ID)
		if err != nil {
			log.Printf("read schedule refresh league: %v", err)
			continue
		}
		if err := s.FillScheduleDates(ctx, records); err != nil {
			log.Printf("refresh NFL dates for %s (held dates kept): %v", league.ID, err)
		}
		if err := s.SyncScoring(ctx, league.ID, records); err != nil {
			log.Printf("refresh scoring evidence for %s (held data kept): %v", league.ID, err)
		}
		source, season, principal, follows := mflsource.FollowingLeague(league.ID, records)
		if !follows {
			continue
		}
		raw, err := leagueschedule.Fetch(ctx, s.Client, strconv.Itoa(season), source)
		if err == nil {
			_, err = mflsource.BackfillSchedule(ctx, ledger, league.ID, "Refresh the published schedule", principal, season, raw)
		}
		if err == nil {
			records, err = ledger.Records(ctx, league.ID)
			if err == nil {
				err = s.FillScheduleDates(ctx, records)
			}
		}
		if err != nil {
			log.Printf("refresh schedule for %s (held schedule kept): %v", league.ID, err)
		}
		if err := s.syncCurrentWeekLineups(ctx, ledger, league.ID, principal, season, source); err != nil {
			log.Printf("refresh this week’s lineups for %s (held lineups kept): %v", league.ID, err)
		}
		followed = append(followed, followedLeague{league.ID, principal, season, source})
	}
	s.syncFollowedLineups(ctx, ledger, followed)
}

type followedLeague struct {
	league, principal string
	season            int
	source            string
}

// syncFollowedLineups starts one bounded history run for all eligible leagues.
func (s ScheduleSync) syncFollowedLineups(ctx context.Context, ledger leagueservice.MigrationLedger,
	leagues []followedLeague) {
	if len(leagues) == 0 {
		return
	}
	run := func(work context.Context) {
		for _, league := range leagues {
			if work.Err() != nil {
				return
			}
			s.syncPlayedLineups(work, ledger, league.league, league.principal, league.season, league.source)
		}
	}
	if s.WorkContext() == nil { // no worker lifetime yet: import inline
		run(ctx)
		return
	}
	if !s.Running.CompareAndSwap(false, true) {
		return
	}
	s.StartWorker(func() { //nolint:contextcheck // Outlives the refresh by design; bound by the app worker lifetime.
		defer s.Running.Store(false)
		if !s.Started.Swap(true) { // first import of this launch: let startup settle
			select {
			case <-s.WorkContext().Done():
				return
			case <-time.After(playedLineupsLaunchDelay):
			}
		}
		work, cancel := context.WithTimeout(s.WorkContext(), playedLineupsTimeout)
		defer cancel()
		run(work)
	})
}

// playedLineupsTimeout bounds one played-week import; what it does not reach is retried next refresh.
const playedLineupsTimeout = 10 * time.Minute

// playedLineupsLaunchDelay holds the first played-week import of a launch back while the launch
// refresh, player load and live points land. Played weeks are history; each re-checked franchise-week
// holds the league write lock for seconds, which would otherwise time out every other startup read.
const playedLineupsLaunchDelay = 5 * time.Minute

func (s ScheduleSync) syncPlayedLineups(ctx context.Context, ledger leagueservice.MigrationLedger,
	league, principal string, season int, source string) {
	records, err := ledger.Records(ctx, league)
	if err != nil {
		log.Printf("read transaction refresh league %s: %v", league, err)
		return
	}
	rules, err := commissioner.RulesNow(records, s.Now())
	if err != nil {
		log.Printf("read transaction refresh rules %s: %v", league, err)
		return
	}
	if rules.Transactions.Follow.State == leaguerules.IsKnown && rules.Transactions.Follow.Value {
		raw, err := transactions.Fetch(ctx, s.Client, strconv.Itoa(season), source)
		if err == nil {
			var directory map[string]string
			directory, err = s.LineupPlayers(ctx)
			if err == nil {
				_, err = mflsource.BackfillHeldTransactions(ctx, ledger, s.History, league, "Import reference transaction history", principal, season, raw, directory)
			}
		}
		if err != nil {
			log.Printf("refresh transactions for %s (held history kept): %v", league, err)
			return
		}
	}
	weeks, lineupErr := weeklyresults.Fetch(ctx, s.Client, strconv.Itoa(season), source)
	if lineupErr == nil {
		var directory map[string]string
		directory, lineupErr = s.LineupPlayers(ctx)
		if lineupErr == nil {
			_, lineupErr = mflsource.BackfillLineups(ctx, ledger, league, "Import played starting lineups from the reference sync", principal, season, weeks, directory)
		}
	}
	if lineupErr != nil {
		log.Printf("refresh lineups for %s (refused franchise-weeks kept unchanged): %v", league, lineupErr)
	}
}

// syncCurrentWeekLineups records the starters the reference league holds for the week in play.
// The played-week import only reads weeks MFL has scored, so without this a week in progress has
// no lineups at all and every franchise reads as a missed lineup while its game is being played.
func (s ScheduleSync) syncCurrentWeekLineups(ctx context.Context, ledger leagueservice.MigrationLedger,
	league, principal string, season int, source string) error {
	raw, err := livescoring.Fetch(ctx, s.Client, strconv.Itoa(season), source, 0)
	if err != nil {
		return fmt.Errorf("read this week's live lineups: %w", err)
	}
	live, err := livescoring.ToLineups(raw)
	if err != nil {
		return fmt.Errorf("read this week's live lineups: %w", err)
	}
	directory, err := s.LineupPlayers(ctx)
	if err != nil {
		return err
	}
	if _, err := mflsource.BackfillLiveLineups(ctx, ledger, league,
		"Record this week's starting lineups from the reference sync", principal, season, live, directory); err != nil {
		return fmt.Errorf("record this week's lineups: %w", err)
	}
	return nil
}
