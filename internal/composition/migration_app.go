package composition

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	migrationapp "github.com/secureprospective/TheWarRoom/internal/app/mflref/migration"
	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"

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
		SaveKey: func(ctx context.Context, league string, key string) error {
			if a.core.KeyStore == nil {
				return mflkey.ErrUnavailable
			}
			return a.core.KeyStore.ForLeague(league).Set(ctx, mflkey.Key(key))
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

func (a *App) scheduleSync() migrationapp.ScheduleSync {
	return migrationapp.ScheduleSync{History: a.core.History,
		Authorizer: func() commissioner.Authorizer { return a.core.CommissionerAuthorizer },
		Client:     a.core.MFLClient, Now: a.timeNow,
		FillScheduleDates: a.fillScheduleDates, SyncScoring: a.syncScheduledScoring,
		LineupPlayers: a.heldLineupPlayers, WorkContext: func() context.Context { return a.core.WorkCtx }, StartWorker: a.startWorker,
		Running: &a.playedLineupsRunning, Started: &a.playedLineupsStarted}
}

func (a *App) syncScheduledScoring(ctx context.Context, league string, records []commissioner.Record) error {
	var err error
	a.core.SerializeSourcesLoad(ctx, func() {
		err = sourcesapp.SyncScheduledScoring(ctx, a.core.History, a.core.Fetches, league, records)
	})
	return err //nolint:wrapcheck // The service supplies scoring context.
}

func (a *App) syncLeagueSchedules(ctx context.Context) {
	a.scheduleSyncMu.Lock()
	defer a.scheduleSyncMu.Unlock()
	a.scheduleSync().Sync(ctx)
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

func (a *App) backfillScheduleFollowing(ctx context.Context) {
	if a.core.History == nil {
		return
	}
	leagues, err := a.core.History.NativeLeagues(ctx)
	if err != nil {
		log.Printf("read leagues for reference settings: %v", err)
		return
	}
	ledger := leagueservice.MigrationLedger{Store: a.core.History, Auth: func() commissioner.Authorizer { return a.core.CommissionerAuthorizer }}
	for _, league := range leagues {
		if err := leaguemigration.BackfillFollowing(ctx, ledger, league.ID); err != nil {
			log.Printf("backfill schedule setting for %s: %v", league.ID, err)
		}
	}
}
