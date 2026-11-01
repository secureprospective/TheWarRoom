package composition

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livescoring"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/weeklyresults"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
)

// migrationInputs wires the move from MFL: a read-only MFL reader per league, the league ledger, and the
// signed-in commissioner. Each reader uses only that MFL league's saved credential.
func (a *App) migrationInputs() *catalog.MigrationInputs {
	service := &leaguemigration.Service{Now: a.timeNow,
		Ledger: leagueservice.MigrationLedger{Store: a.core.History,
			Auth: func() commissioner.Authorizer { return a.core.CommissionerAuthorizer }},
		SourceWithBase: func(mflLeague string, base leaguerules.RuleSet) (leaguemigration.Source, error) {
			options := []mfl.Option{mfl.WithTransport(a.core.Fetches)}
			if a.core.KeyStore != nil {
				options = append(options, mfl.WithKeySource(func(ctx context.Context) (mflkey.Key, error) {
					return a.core.KeyStore.ForLeague(mflLeague).Get(ctx)
				}))
			}
			client, err := mfl.New("api", 2, options...)
			if err != nil {
				return nil, fmt.Errorf("open the MFL reader: %w", err)
			}
			return &mflsource.Source{Client: client, ID: mflLeague, Now: a.timeNow, Held: a.heldPlayerNames,
				Base: base}, nil
		}}
	return &catalog.MigrationInputs{Service: service, Principal: a.commissionerPrincipal,
		SaveKey: func(ctx context.Context, league string, key mflkey.Key) error {
			if a.core.KeyStore == nil {
				return mflkey.ErrUnavailable
			}
			return a.core.KeyStore.ForLeague(league).Set(ctx, key)
		}}
}

// heldPlayerNames is the player list the app holds, by MFL id.
func (a *App) heldPlayerNames() map[string]string {
	a.core.TargetSnapshotMu.Lock()
	defer a.core.TargetSnapshotMu.Unlock()
	out := make(map[string]string, len(a.core.TargetSnapshot.Players.Value))
	for _, p := range a.core.TargetSnapshot.Players.Value {
		out[p.ID.String()] = p.Name
	}
	return out
}

func (h controlHost) Migrate(command, body string) (json.RawMessage, error) {
	if err := h.app.ready(); err != nil {
		return nil, err
	}
	reply, err := catalog.RunMigration(h.app.core.Ctx, *h.app.migrationInputs(), command, json.RawMessage(body))
	if err != nil {
		return nil, fmt.Errorf("move the MFL league: %w", err)
	}
	return reply, nil
}

// syncLeagueSchedules is a reference-feed refresh. Failures retain the published ledger
// and retry on the next refresh; they never stop the rest of the board.
func (a *App) syncLeagueSchedules(ctx context.Context) {
	if a.core.History == nil {
		return
	}
	leagues, err := a.core.History.NativeLeagues(ctx)
	if err != nil {
		log.Printf("read leagues for schedule refresh: %v", err)
		return
	}
	ledger := leagueservice.MigrationLedger{Store: a.core.History,
		Auth: func() commissioner.Authorizer { return a.core.CommissionerAuthorizer }}
	for _, league := range leagues {
		records, err := ledger.Records(ctx, league.ID)
		if err != nil {
			log.Printf("read schedule refresh league: %v", err)
			continue
		}
		if err := a.fillScheduleDates(ctx, records); err != nil {
			log.Printf("refresh NFL dates for %s (held dates kept): %v", league.ID, err)
		}
		a.core.SerializeSourcesLoad(ctx, func() {
			if err := sourcesapp.SyncScheduledScoring(ctx, a.core.History, a.core.Fetches, league.ID, records); err != nil {
				log.Printf("refresh scoring evidence for %s (held data kept): %v", league.ID, err)
			}
		})
		source, season, principal, follows := mflsource.FollowingLeague(league.ID, records)
		if !follows {
			continue
		}
		raw, err := leagueschedule.Fetch(ctx, a.core.MFLClient, strconv.Itoa(season), source)
		if err == nil {
			_, err = mflsource.BackfillSchedule(ctx, ledger, league.ID, "Refresh the published schedule", principal, season, raw)
		}
		if err == nil {
			records, err = ledger.Records(ctx, league.ID)
			if err == nil {
				err = a.fillScheduleDates(ctx, records)
			}
		}
		if err != nil {
			log.Printf("refresh schedule for %s (held schedule kept): %v", league.ID, err)
		}
		a.syncFollowedLineups(ctx, ledger, league.ID, principal, season, source)
	}
}

// syncFollowedLineups records the week in play, then starts the played-week import on its own. The
// week in play goes first because it is what a live screen needs and it is one small read. The
// played-week import re-checks every franchise-week the reference league starts differently from
// the held lineup, which takes minutes on a big league; it runs as its own job, outside the refresh
// lock, so the player load and live points never wait behind it. A refused franchise-week never
// stops the other weeks, and held lineups are kept.
func (a *App) syncFollowedLineups(ctx context.Context, ledger leagueservice.MigrationLedger,
	league, principal string, season int, source string) {
	if err := a.syncCurrentWeekLineups(ctx, ledger, league, principal, season, source); err != nil {
		log.Printf("refresh this week's lineups for %s (held lineups kept): %v", league, err)
	}
	if a.core.WorkCtx == nil { // no worker lifetime yet: import inline
		a.syncPlayedLineups(ctx, ledger, league, principal, season, source)
		return
	}
	if !a.playedLineupsRunning.CompareAndSwap(false, true) {
		return
	}
	a.startWorker(func() { //nolint:contextcheck // Outlives the refresh by design; bound by the app worker lifetime.
		defer a.playedLineupsRunning.Store(false)
		work, cancel := context.WithTimeout(a.core.WorkCtx, playedLineupsTimeout)
		defer cancel()
		a.syncPlayedLineups(work, ledger, league, principal, season, source)
	})
}

// playedLineupsTimeout bounds one played-week import; what it does not reach is retried next refresh.
const playedLineupsTimeout = 10 * time.Minute

func (a *App) syncPlayedLineups(ctx context.Context, ledger leagueservice.MigrationLedger,
	league, principal string, season int, source string) {
	weeks, lineupErr := weeklyresults.Fetch(ctx, a.core.MFLClient, strconv.Itoa(season), source)
	if lineupErr == nil {
		var directory map[string]string
		directory, lineupErr = a.heldLineupPlayers(ctx)
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
func (a *App) syncCurrentWeekLineups(ctx context.Context, ledger leagueservice.MigrationLedger,
	league, principal string, season int, source string) error {
	raw, err := livescoring.Fetch(ctx, a.core.MFLClient, strconv.Itoa(season), source, 0)
	if err != nil {
		return fmt.Errorf("read this week's live lineups: %w", err)
	}
	live, err := livescoring.ToLineups(raw)
	if err != nil {
		return fmt.Errorf("read this week's live lineups: %w", err)
	}
	directory, err := a.heldLineupPlayers(ctx)
	if err != nil {
		return err
	}
	if _, err := mflsource.BackfillLiveLineups(ctx, ledger, league,
		"Record this week's starting lineups from the reference sync", principal, season, live, directory); err != nil {
		return fmt.Errorf("record this week's lineups: %w", err)
	}
	return nil
}

// Launch can precede the first target snapshot. Read the held directory, never
// spend MFL's once-a-day players request just to map historical lineups.
func (a *App) heldLineupPlayers(ctx context.Context) (map[string]string, error) {
	directory := a.heldPlayerNames()
	if len(directory) != 0 {
		return directory, nil
	}
	found, err := a.core.History.ArchivedBodies(ctx, "TYPE=players", func(_ string, body []byte, _ time.Time) bool {
		raw, parseErr := players.Parse(ctx, body)
		if parseErr != nil {
			return false
		}
		for _, p := range raw {
			directory[p.ID] = p.Name
		}
		return len(directory) != 0
	})
	if err != nil {
		return nil, fmt.Errorf("read held lineup player directory: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("no held player directory for lineup import")
	}
	return directory, nil
}
