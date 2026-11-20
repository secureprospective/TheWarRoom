package leagueload

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
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
// and commits football evidence and ledger changes unless DryRun is set.
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
		OldFootball:     r.CurrentPin,
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

type preparedFootball struct {
	body    []byte
	digest  string
	changed bool
	env     footballEnv
}

func prepareMirrorFootball(ctx context.Context, r RefreshRequest, season int) (preparedFootball, error) {
	body, digest, err := r.Football(ctx, season)
	if err != nil {
		return preparedFootball{}, fmt.Errorf("build football evidence from mirror: %w", err)
	}
	var raw history.RawFootballEvidence
	if err := json.Unmarshal(body, &raw); err != nil {
		return preparedFootball{}, fmt.Errorf("decode mirror football evidence: %w", err)
	}
	if err := raw.Validate(); err != nil {
		return preparedFootball{}, fmt.Errorf("validate mirror football evidence: %w", err)
	}
	changed := !sameFootballContent(ctx, r.Server, r.CurrentPin, raw, digest)
	finalDigest := digest
	if !changed {
		finalDigest = r.CurrentPin
	}
	return preparedFootball{
		body:    body,
		digest:  finalDigest,
		changed: changed,
		env:     footballEnv{raw: raw},
	}, nil
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
	if fb.changed {
		evidence, err := ensureFootballStored(ctx, r.Server, fb.body, fb.digest)
		if err != nil {
			return err
		}
		sibling.SetPinnedFootball(evidence)
	} else if r.CurrentPin != "" {
		evidence, err := r.Server.LoadFootball(ctx, r.CurrentPin)
		if err != nil {
			return fmt.Errorf("load pinned football evidence: %w", err)
		}
		sibling.SetPinnedFootball(evidence)
	}
	if len(commands) == 0 {
		return nil
	}
	ledger := leagueservice.MigrationLedger{Store: sibling, Facts: fb.env.PlayerFacts}
	if _, err := ledger.Batch(ctx, commands, string(r.Commissioner)); err != nil {
		return fmt.Errorf("record league refresh (nothing was changed): %w", err)
	}
	return nil
}

func ensureFootballStored(
	ctx context.Context, server *history.Store, body []byte, digest string,
) (*history.PinnedFootball, error) {
	evidence, err := server.LoadFootball(ctx, digest)
	if err == nil {
		return evidence, nil
	}
	if !errors.Is(err, history.ErrNoBody) {
		return nil, fmt.Errorf("check held football evidence: %w", err)
	}
	if err := server.ImportFootball(ctx, body, digest); err != nil {
		return nil, fmt.Errorf("import football evidence: %w", err)
	}
	evidence, err = server.LoadFootball(ctx, digest)
	if err != nil {
		return nil, fmt.Errorf("verify imported football evidence: %w", err)
	}
	return evidence, nil
}

func sameFootballContent(
	ctx context.Context,
	store *history.Store,
	currentPin string,
	want history.RawFootballEvidence,
	wantDigest string,
) bool {
	if currentPin == "" {
		return false
	}
	if currentPin == wantDigest {
		return true
	}
	body, err := store.Body(ctx, currentPin)
	if err != nil {
		return false
	}
	var have history.RawFootballEvidence
	if err := json.Unmarshal(body, &have); err != nil {
		return false
	}
	return have.Season == want.Season &&
		reflect.DeepEqual(have.Players, want.Players) &&
		reflect.DeepEqual(have.Weeks, want.Weeks)
}

type footballEnv struct {
	raw history.RawFootballEvidence
}

func (e footballEnv) PlayerFacts() nativecheck.Facts {
	facts := make(nativecheck.Facts, len(e.raw.Players))
	for id, player := range e.raw.Players {
		f := player.Facts
		if player.NFLRosterSeasons != nil {
			f.NFLSeasons, f.NFLSeasonsHeld = len(*player.NFLRosterSeasons), true
		}
		facts[id] = f
	}
	nativecheck.AddLineCards(facts)
	return facts
}

func (e footballEnv) LineupFacts(ctx context.Context, season, week int) (commissioner.LineupFacts, error) {
	if err := ctx.Err(); err != nil {
		return commissioner.LineupFacts{}, fmt.Errorf("read mirror football schedule: %w", err)
	}
	if season != e.raw.Season {
		return commissioner.LineupFacts{}, fmt.Errorf(
			"mirror football data is for season %d, not %d", e.raw.Season, season)
	}
	for _, held := range e.raw.Weeks {
		if held.Number == week {
			return commissioner.LineupFacts{
				Facts: e.PlayerFacts(), Locks: leagueweek.Locks(held),
				ScheduleHeld: true, Zone: time.UTC,
			}, nil
		}
	}
	return commissioner.LineupFacts{}, fmt.Errorf("mirror football data does not hold NFL week %d", week)
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
