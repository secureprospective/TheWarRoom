package leagueload

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// RefreshRequest names an already-loaded server league to refresh from held MFL reads.
type RefreshRequest struct {
	League       string
	MFLLeague    string
	Reason       string
	Commissioner governance.Principal
	Server       *history.Store
	CurrentPin   string
	DryRun       bool
	Source       func(mflLeague string, base leaguerules.RuleSet) (leaguemigration.Source, error)
	Football     func(ctx context.Context, season int) ([]byte, string, error)
	Now          func() time.Time
}

// StoppedTrade names one unsettled in-app trade blocking a refresh in owner-readable words.
type StoppedTrade struct {
	Status     string   `json:"status"`
	Franchises []string `json:"franchises"`
}

// StoppedTradesError is returned when open in-app trade offers stop a league refresh.
type StoppedTradesError struct {
	LeagueName string
	Season     int
	DryRun     bool
	Offers     []StoppedTrade
}

func (e *StoppedTradesError) Error() string {
	parts := make([]string, len(e.Offers))
	for i, o := range e.Offers {
		parts[i] = fmt.Sprintf("%s (%s)", joinNames(o.Franchises), o.Status)
	}
	return fmt.Sprintf(
		"refresh stopped for %s: %d open trade offer(s) must be settled first: %s",
		e.LeagueName, len(e.Offers), strings.Join(parts, "; "))
}

// RefreshResult reports what a league refresh planned (dry run) or committed on the server.
type RefreshResult struct {
	League          string                         `json:"league"`
	LeagueName      string                         `json:"leagueName"`
	MFLLeague       string                         `json:"mflLeague"`
	Season          int                            `json:"season"`
	DryRun          bool                           `json:"dryRun"`
	Summary         leaguemigration.RefreshSummary `json:"summary"`
	Commands        int                            `json:"commands"`
	OldFootball     string                         `json:"oldFootball,omitempty"`
	Football        string                         `json:"football"`
	FootballChanged bool                           `json:"footballChanged"`
}

// ReplaySourceByLeague builds a held-only MFL reader for the MFL league resolved at refresh time.
func ReplaySourceByLeague(
	ctx context.Context, mirror Mirror, now func() time.Time,
) func(string, leaguerules.RuleSet) (leaguemigration.Source, error) {
	return func(mflLeague string, base leaguerules.RuleSet) (leaguemigration.Source, error) {
		return ReplaySource(ctx, mirror, mflLeague, now)(base)
	}
}

// Refresh reads the mirror without network calls, plans and trial-verifies the atomic refresh batch,
// and commits ledger changes and football evidence unless DryRun is set.
func Refresh(ctx context.Context, r RefreshRequest) (RefreshResult, error) {
	if err := checkRefreshRequest(r); err != nil {
		return RefreshResult{}, err
	}
	records, err := leagueservice.Records(ctx, r.Server, r.League, "refresh")
	if err != nil {
		return RefreshResult{}, fmt.Errorf("read the server league: %w", err)
	}
	if len(records) == 0 {
		return RefreshResult{}, fmt.Errorf("league %s has no history on this server; load it first", r.League)
	}
	app, err := leaguemigration.ReadApp(r.League, records, r.Now())
	if err != nil {
		return RefreshResult{}, fmt.Errorf("read current league state: %w", err)
	}
	leagueName, season := leagueIdentity(r.League, records, app)
	fb, err := prepareMirrorFootball(ctx, r, season)
	if err != nil {
		return RefreshResult{}, err
	}
	sibling := r.Server.Sibling()
	sibling.SetLineupEnvironment(fb.env)
	report, err := planServerRefresh(ctx, r, sibling, fb.env, leagueName, season, app.Directory.Seats())
	if err != nil {
		return RefreshResult{}, err
	}
	res := RefreshResult{
		League:          r.League,
		LeagueName:      leagueName,
		MFLLeague:       report.MFLLeague,
		Season:          report.Season,
		DryRun:          r.DryRun,
		Summary:         report.Summary,
		Commands:        len(report.Commands),
		OldFootball:     fb.oldDigest,
		Football:        fb.digest,
		FootballChanged: fb.changed,
	}
	if r.DryRun {
		return res, nil
	}
	if err := commitServerRefresh(ctx, r, sibling, fb, report.Commands); err != nil {
		return RefreshResult{}, err
	}
	return res, nil
}

func checkRefreshRequest(r RefreshRequest) error {
	if r.Server == nil || r.Source == nil || r.Football == nil || r.Now == nil {
		return fmt.Errorf("league refresh needs a server store, source, football evidence, and clock")
	}
	if strings.TrimSpace(r.League) == "" || strings.TrimSpace(string(r.Commissioner)) == "" {
		return fmt.Errorf("league refresh needs a league ID and commissioner identity")
	}
	return nil
}

func leagueIdentity(
	league string, records []commissioner.Record, app leaguemigration.AppLeague,
) (string, int) {
	name := strings.TrimSpace(app.Directory.Name)
	if name == "" {
		name = league
	}
	season := app.Snapshot.Season
	if season == 0 {
		season = records[0].Command.Args.Season
	}
	return name, season
}

func planServerRefresh(
	ctx context.Context,
	r RefreshRequest,
	sibling *history.Store,
	env footballEnv,
	leagueName string,
	season int,
	seats commissioner.SeatIndex,
) (leaguemigration.RefreshReport, error) {
	service := &leaguemigration.Service{
		Now:            r.Now,
		Ledger:         leagueservice.MigrationLedger{Store: sibling, Facts: env.PlayerFacts},
		SourceWithBase: r.Source,
	}
	report, err := service.Refresh(ctx, leaguemigration.RefreshRequest{
		MFLLeague: r.MFLLeague,
		League:    r.League,
		Reason:    r.Reason,
		Principal: string(r.Commissioner),
		DryRun:    true,
	})
	if err != nil {
		var openErr *leaguemigration.OpenTradesError
		if errors.As(err, &openErr) {
			return leaguemigration.RefreshReport{}, &StoppedTradesError{
				LeagueName: leagueName,
				Season:     season,
				DryRun:     r.DryRun,
				Offers:     namedStoppedTrades(openErr.Offers, seats),
			}
		}
		return leaguemigration.RefreshReport{}, fmt.Errorf("plan league refresh: %w", err)
	}
	return report, nil
}

func commitServerRefresh(
	ctx context.Context,
	r RefreshRequest,
	sibling *history.Store,
	fb preparedFootball,
	commands []commissioner.RawCommand,
) error {
	if len(commands) > 0 {
		ledger := leagueservice.MigrationLedger{Store: sibling, Facts: fb.env.PlayerFacts}
		if _, err := ledger.Batch(ctx, commands, string(r.Commissioner)); err != nil {
			return fmt.Errorf("record league refresh (nothing was changed): %w", err)
		}
	}
	if fb.changed {
		if _, err := ensureFootballStored(ctx, r.Server, fb.body, fb.digest); err != nil {
			if len(commands) > 0 {
				return fmt.Errorf("the league refresh has already committed, but %w", err)
			}
			return err
		}
	}
	return nil
}

func namedStoppedTrades(
	offers []leaguemigration.OpenTradeRefusal, seats commissioner.SeatIndex,
) []StoppedTrade {
	out := make([]StoppedTrade, len(offers))
	for i, offer := range offers {
		names := make([]string, len(offer.Franchises))
		for j, fid := range offer.Franchises {
			names[j] = seats.Of(fid).Name
		}
		out[i] = StoppedTrade{
			Status:     tradeStatusWords(offer.Status),
			Franchises: names,
		}
	}
	return out
}

func tradeStatusWords(status string) string {
	switch status {
	case "offered":
		return "open trade offer waiting for a response"
	case "review":
		return "accepted trade waiting for league review"
	case "appealed":
		return "vetoed trade under appeal"
	default:
		return status
	}
}

func joinNames(names []string) string {
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
