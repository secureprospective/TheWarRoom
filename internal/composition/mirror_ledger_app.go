package composition

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/mflref/seed"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// MirrorLeague is the MFL-mirror league's app ID: native-shaped so every read takes the same path.
const MirrorLeague = "native.mfl." + ingestion.LeagueID

// mirrorCreator owns the in-memory mirror ledger; no session principal can carry this name.
const mirrorCreator = "mfl:held"

// mirrorLedger builds the MFL-mirror league's read-only records from the held data, once per refresh.
type mirrorLedger struct {
	app     *App
	mu      sync.Mutex
	at      string
	records []commissioner.Record
}

func (a *App) installMirrorLedger() {
	m := &mirrorLedger{app: a}
	a.core.History.SetMirrorLedger(history.MirrorLedger{League: MirrorLeague, Records: m.read})
}

func (m *mirrorLedger) read(_ context.Context) ([]commissioner.Record, error) {
	source := m.app.heldSource()
	if !source.Available() {
		return nil, fmt.Errorf("hold the league rosters and settings first")
	}
	_, at := source.Roster()
	m.mu.Lock()
	defer m.mu.Unlock()
	if at == m.at && m.records != nil {
		return m.records, nil
	}
	records, err := mirrorRecords(source)
	if err != nil {
		return nil, err
	}
	m.at, m.records = at, records
	return records, nil
}

// mirrorRecords is the league as a migration would start it: Create from the held settings, then
// Seed from the held rosters, both recorded at the held roster time.
func mirrorRecords(source nativeapp.SeedSource) ([]commissioner.Record, error) {
	seed, err := nativeapp.CaptureSeed(source)
	if err != nil {
		return nil, fmt.Errorf("mirror league rosters: %w", err)
	}
	config := source.Config()
	rules, _, _ := mflsource.Settings(config)
	args, problems := mflsource.CreateArgs(config.Name, seed.Season, len(config.Franchises), rules)
	if len(problems) > 0 {
		return nil, fmt.Errorf("mirror league settings: %s", strings.Join(problems, "; "))
	}
	create, err := (commissioner.RawCommand{League: MirrorLeague, Command: commissioner.Create, Args: args,
		Reason: "MFL mirror", Request: "mirror-create", Effective: "now", Visibility: "league"}).Prepared()
	if err != nil {
		return nil, fmt.Errorf("mirror league settings: %w", err)
	}
	at := seed.RosterAt.UTC()
	seeded := commissioner.RawCommand{League: MirrorLeague, Command: commissioner.NativeSeed,
		Args: commissioner.RawArgs{Native: &seed}, Request: "mirror-seed", Reason: "MFL mirror", Effective: "now",
		Visibility: "league"}
	return []commissioner.Record{
		{Command: create, Principal: mirrorCreator, Role: "commissioner",
			Receipt: commissioner.Receipt{Entry: "mirror-create", Revision: 1, RecordedAt: at.Add(-time.Second)}},
		{Command: seeded, Principal: mirrorCreator, Role: "commissioner",
			Receipt: commissioner.Receipt{Entry: "mirror-seed", Revision: 2, RecordedAt: at}},
	}, nil
}
