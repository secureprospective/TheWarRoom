package composition

import (
	"context"
	"fmt"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
)

// migrationInputs wires the move from MFL: a read-only MFL reader per league, the league ledger, and the
// signed-in commissioner. Each move reads through its own client, so the app's own league's host and
// pacing are never disturbed; the saved MFL key is offered only for the app's own league.
func (a *App) migrationInputs() *catalog.MigrationInputs {
	service := &leaguemigration.Service{Now: a.timeNow,
		Ledger: leagueservice.MigrationLedger{Store: a.core.History,
			Auth: func() commissioner.Authorizer { return a.core.CommissionerAuthorizer }},
		Source: func(mflLeague string) (leaguemigration.Source, error) {
			options := []mfl.Option{mfl.WithTransport(a.core.Fetches)}
			if mflLeague == ingestion.LeagueID && a.core.KeyStore != nil {
				options = append(options, mfl.WithKeySource(func(ctx context.Context) (mflkey.Key, error) {
					return a.core.KeyStore.Get(ctx)
				}))
			}
			client, err := mfl.New("api", 2, options...)
			if err != nil {
				return nil, fmt.Errorf("open the MFL reader: %w", err)
			}
			return &mflsource.Source{Client: client, ID: mflLeague, Now: a.timeNow, Held: a.heldPlayerNames}, nil
		}}
	return &catalog.MigrationInputs{Service: service, Principal: a.commissionerPrincipal}
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
