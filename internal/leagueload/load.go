package leagueload

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/leaguetransfer"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

const reason = "Load the league onto its hosted server from the commissioner's held MFL copy"

// Request names one league move onto a server. Workspace is a fresh, private, disposable database
// the native league is built in before it moves; Server is the hosted league database.
type Request struct {
	MFLLeague    string
	Commissioner governance.Principal
	Workspace    *history.Store
	Server       *history.Store
	// Source reads the MFL league. ReplaySource is the production reader: held copies only.
	Source func(base leaguerules.RuleSet) (leaguemigration.Source, error)
	// Football is the season's football evidence, built from the same held copies.
	Football func(ctx context.Context, season int) ([]byte, string, error)
	Now      func() time.Time
}

// Result is what the server now holds: the active league and its pinned football evidence.
type Result struct {
	League     string `json:"league"`
	Franchises int    `json:"franchises"`
	Football   string `json:"football"`
}

// ReplaySource reads the MFL league from mirror's held copies through the normal MFL reader. Its
// transport opens no connection, so the read cannot reach MFL at all.
func ReplaySource(ctx context.Context, mirror Mirror, mflLeague string, now func() time.Time) func(leaguerules.RuleSet) (leaguemigration.Source, error) {
	return func(base leaguerules.RuleSet) (leaguemigration.Source, error) {
		client, err := mfl.New("api", 50, mfl.WithTransport(archive.Replay{Lookup: mirror.Lookup}))
		if err != nil {
			return nil, fmt.Errorf("open the held MFL reader: %w", err)
		}
		list, err := mirror.Players(ctx)
		if err != nil {
			return nil, err
		}
		names := Names(list)
		return &mflsource.Source{Client: client, ID: mflLeague, Now: now, Held: func() map[string]string { return names }, Base: base}, nil
	}
}

// Load builds the native league in the workspace (import, then the commissioner's cutover from MFL),
// pins its football evidence, and moves it to the server through the existing confirmed path:
// capture → encrypted export → inactive import → seal → activate. The server's copy is the only
// writable one afterwards. Nothing is written to MFL or to the mirror.
func Load(ctx context.Context, r Request) (Result, error) {
	if r.Workspace == nil || r.Server == nil || r.Source == nil || r.Football == nil || r.Now == nil {
		return Result{}, fmt.Errorf("league load needs a workspace, a server, a source and football evidence")
	}
	principal := string(r.Commissioner)
	service := &leaguemigration.Service{Now: r.Now, Ledger: leagueservice.MigrationLedger{Store: r.Workspace},
		SourceWithBase: func(_ string, base leaguerules.RuleSet) (leaguemigration.Source, error) { return r.Source(base) }}
	receipts, err := service.Import(ctx, r.MFLLeague, reason, principal)
	if err != nil {
		return Result{}, fmt.Errorf("build the native league: %w", err)
	}
	if len(receipts) == 0 {
		return Result{}, fmt.Errorf("build the native league: nothing was recorded")
	}
	league := receipts[0].League
	if err := cutover(ctx, service, r.MFLLeague, league, principal); err != nil {
		return Result{}, err
	}
	records, err := leagueservice.Records(ctx, r.Workspace, league, "load")
	if err != nil {
		return Result{}, fmt.Errorf("read the native league: %w", err)
	}
	season := records[0].Command.Args.Season
	body, digest, err := r.Football(ctx, season)
	if err != nil {
		return Result{}, fmt.Errorf("build football evidence: %w", err)
	}
	for _, store := range []*history.Store{r.Workspace, r.Server} {
		if err := store.ImportFootball(ctx, body, digest); err != nil {
			return Result{}, fmt.Errorf("pin football evidence: %w", err)
		}
	}
	if err := move(ctx, r, league, digest); err != nil {
		return Result{}, err
	}
	return Result{League: league, Franchises: len(commissioner.LeagueFranchises(records)), Football: digest}, nil
}

func cutover(ctx context.Context, service *leaguemigration.Service, mflLeague, league, principal string) error {
	request := leaguemigration.Request{MFLLeague: mflLeague, League: league}
	review, err := service.Review(ctx, request)
	if err != nil {
		return fmt.Errorf("review the native league against MFL: %w", err)
	}
	if review.CutOver {
		return nil
	}
	if _, err := service.Cutover(ctx, request, review.Digest, reason, principal); err != nil {
		return fmt.Errorf("move the league over from MFL: %w", err)
	}
	return nil
}

func move(ctx context.Context, r Request, league, football string) error {
	raw, err := leaguetransfer.Capture(ctx, r.Workspace, league, uuid.NewString(), football, r.Now().UTC(), nil)
	if err != nil {
		return fmt.Errorf("capture the league: %w", err)
	}
	var key [32]byte
	if _, err := rand.Read(key[:]); err != nil {
		return fmt.Errorf("league move key: %w", err)
	}
	var sealed bytes.Buffer
	if err := leaguetransfer.Encrypt(ctx, &sealed, raw, key); err != nil {
		return fmt.Errorf("encrypt the league: %w", err)
	}
	if _, err := leaguetransfer.Import(ctx, r.Server, bytes.NewReader(sealed.Bytes()), key); err != nil {
		return fmt.Errorf("import the league inactive: %w", err)
	}
	for _, action := range []string{"seal", "activate"} {
		if err := leaguetransfer.Move(ctx, r.Workspace, r.Server, raw, action, string(r.Commissioner), true); err != nil {
			return fmt.Errorf("%s the league move: %w", action, err)
		}
	}
	return nil
}
