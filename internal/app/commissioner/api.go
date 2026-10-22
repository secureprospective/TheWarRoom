package commissioner

import (
	"context"
	"encoding/json"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// Host retains the session, football-fact composition and transactional gate at their existing owners.
type Host interface {
	SessionHost
	ActionHost
	LeagueHost
	ResultsHost
	RosterHost
	MutationHost
}
type SessionHost interface {
	Ready() error
	CommissionerContext() context.Context
	CommissionerPrincipal() (string, error)
	CommissionerAuthorization() commissioner.Authorizer
	CommissionerNow() time.Time
	CommissionerZone() *time.Location
	CommissionerPreviewRegistry() string
	CommissionerRecords(context.Context, string) ([]commissioner.Record, error)
	NativeFacts() nativecheck.Facts
	NativeScheduleDates(context.Context, int, time.Time) ([]leagueweek.Week, error)
}
type ActionHost interface {
	RunNativeOwnerAction(string, string, string, json.RawMessage) (commissioner.Receipt, error)
	SubmitNativeTrade(commissioner.RawCommand) (commissioner.Receipt, error)
	SubmitNativeWaiver(commissioner.RawCommand) (commissioner.Receipt, error)
	SubmitNativeBid(commissioner.RawCommand) (commissioner.Receipt, error)
	NativeSeedCommand(context.Context, RawNativeSeedRequest) (commissioner.RawCommand, error)
	CloseFinishedDrafts(string)
}

type MutationHost interface {
	ExecuteCommissioner(ctx context.Context, raw commissioner.RawCommand,
		principal string, auth commissioner.Authorizer) (commissioner.Receipt, error)
	ExecuteCommissionerWithFacts(ctx context.Context, raw commissioner.RawCommand,
		principal string, auth commissioner.Authorizer, facts nativecheck.Facts,
		zone *time.Location) (commissioner.Receipt, error)
	PreviewCommissionerWithFacts(ctx context.Context, raw commissioner.RawCommand,
		principal string, auth commissioner.Authorizer, catalogue commissioner.PreviewCatalogue,
		playerFacts nativecheck.Facts, zone *time.Location) (commissioner.Preview, error)
}

type LeagueHost interface {
	CommissionerLedgerFiltered(ctx context.Context, league, principal string,
		auth commissioner.Authorizer, filter commissioner.RawLedgerFilter) (commissioner.Ledger, error)
	CommissionerPeople(ctx context.Context, league, principal string) (commissioner.People, error)
	NativeLeagues(ctx context.Context) ([]commissioner.NativeLeague, error)
	CommissionerRules(ctx context.Context, league string) (commissioner.Rules, error)
	CommissionerJobs(ctx context.Context, league string) (commissioner.Jobs, error)
	CommissionerNotes(ctx context.Context, league string) (commissioner.LeagueNotes, error)
	CommissionerDigests(ctx context.Context, league, principal string,
		auth commissioner.Authorizer) (commissioner.Digests, error)
	Corrections(ctx context.Context, league string) (commissioner.Corrections, error)
	Events(ctx context.Context, league, stream string, after int64) ([]history.LeagueEvent, error)
}

type ResultsHost interface {
	NativeDraft(ctx context.Context, scope commissioner.RawNativeDraftScope,
		local *time.Location, facts nativecheck.Facts) (commissioner.NativeDraft, error)
	NativeLineup(ctx context.Context, league, franchise, principal string, week int,
		at time.Time) (commissioner.NativeLineup, error)
	NativePlayerScore(ctx context.Context, league string,
		scope source.RawNativeScope) (commissioner.NativePlayerScore, error)
	NativeResults(ctx context.Context, league string, season, week int) (commissioner.NativeResults, error)
	NativeStandings(ctx context.Context, league string, season int) (commissioner.NativeStandings, error)
	NativeScorePreview(ctx context.Context, raw commissioner.RawCommand) (commissioner.ScorePreview, error)
	NativeBracket(ctx context.Context, league string, season int) (commissioner.NativeBracket, error)
	StatQueue(ctx context.Context, league string, season int) (commissioner.StatQueue, error)
}

type RosterHost interface {
	NativeCapCheck(ctx context.Context, scope commissioner.RawNativeCapScope) (commissioner.NativeCapCheck,
		error)
	NativeRosterCheck(ctx context.Context, league string,
		facts nativecheck.Facts) (commissioner.NativeRosterCheck, error)
	NativeReserveCheck(ctx context.Context, league string,
		facts nativecheck.Facts) (commissioner.NativeReserveCheck, error)
	NativeLineupCheck(ctx context.Context, in commissioner.RawNativeLineup,
		facts nativecheck.Facts) (commissioner.NativeLineupCheck, error)
	NativeHouseRules(ctx context.Context, league string) (commissioner.NativeHouseRules, error)
	NativeDirectory(ctx context.Context, league string) (commissioner.NativeDirectory, error)
	NativeAssets(ctx context.Context, league string) (commissioner.NativeAssets, error)
	NativeWaivers(ctx context.Context, league, franchise, principal string, at time.Time,
		zone *time.Location, facts nativecheck.Facts) (commissioner.NativeWaivers, error)
	NativeBids(ctx context.Context, league, franchise, principal string, at time.Time,
		zone *time.Location, facts nativecheck.Facts) (commissioner.NativeBids, error)
}

type API struct{ host Host }

func New(host Host) *API { return &API{host: host} }

const refreshTimeout = 120 * time.Second
const draftPickName = "draft.pick"

func Commands() []catalog.Command {
	methods := []struct{ method, role, gravity string }{
		{"TargetCommissionerActor", "gm,commish", "G0"},
		{"TargetCommissionerControls", "gm,commish", "G0"},
		{"TargetCommissionerDigests", "gm,commish", "G0"},
		{"TargetCommissionerExecute", "commish", "G3"},
		{"TargetCommissionerJobs", "gm,commish", "G0"},
		{"TargetCommissionerLeagues", "gm,commish", "G0"},
		{"TargetCommissionerLedger", "gm,commish", "G0"},
		{"TargetCommissionerLedgerFiltered", "gm,commish", "G0"},
		{"TargetCommissionerMarket", "gm,commish", "G0"},
		{"TargetCommissionerNotices", "gm,commish", "G0"},
		{"TargetCommissionerPeople", "gm,commish", "G0"},
		{"TargetCommissionerPreview", "commish", "G0"},
		{"TargetCommissionerRules", "gm,commish", "G0"},
		{"TargetCommissionerSchedule", "commish", "G0"},
		{"TargetCorrections", "gm,commish", "G0"},
		{"TargetLeagueCalendar", "gm,commish", "G0"},
		{"TargetLeagueSchedule", "gm,commish", "G0"},
		{"TargetNativeAssets", "gm,commish", "G0"},
		{"TargetNativeBidAction", "gm,commish", "G3"},
		{"TargetNativeBids", "gm,commish", "G0"},
		{"TargetNativeBracket", "gm,commish", "G0"},
		{"TargetNativeCapCheck", "gm,commish", "G0"},
		{"TargetNativeContractQuote", "gm,commish,admin", "G0"},
		{"TargetNativeDirectory", "gm,commish,admin", "G0"},
		{"TargetNativeDraft", "gm,commish,admin", "G0"},
		{"TargetNativeDraftPick", "gm,commish", "G3"},
		{"TargetNativeFranchiseSchedule", "gm,commish,admin", "G0"},
		{"TargetNativeHouseRules", "gm,commish,admin", "G0"},
		{"TargetNativeLineup", "gm,commish,admin", "G0"},
		{"TargetNativeLineupCheck", "gm,commish,admin", "G0"},
		{"TargetNativeLineupSet", "gm,commish", "G2"},
		{"TargetNativePlayerScore", "gm,commish,admin", "G0"},
		{"TargetNativeReserveCheck", "gm,commish,admin", "G0"},
		{"TargetNativeResults", "gm,commish,admin", "G0"},
		{"TargetNativeRosterCheck", "gm,commish,admin", "G0"},
		{"TargetNativeScorePreview", "commish", "G0"},
		{"TargetNativeSeed", "commish", "G3"},
		{"TargetNativeStandings", "gm,commish,admin", "G0"},
		{"TargetNativeTradeAction", "gm,commish", "G3"},
		{"TargetNativeTrades", "gm,commish", "G0"},
		{"TargetNativeWaiverAction", "gm,commish", "G3"},
		{"TargetNativeWaivers", "gm,commish", "G0"},
		{"TargetNativeWeek", "gm,commish,admin", "G0"},
		{"TargetStatQueue", "commish", "G0"},
	}
	commands := make([]catalog.Command, 0, len(methods))
	for _, metadata := range methods {
		command := catalog.Command{ID: catalog.Unset, Method: metadata.method, Version: catalog.Unset,
			Role: metadata.role, Gravity: metadata.gravity, Availability: "both"}
		switch metadata.method {
		case "TargetCommissionerLedger", "TargetCommissionerLedgerFiltered":
			command.ID = "commish.ledger.read"
		case "TargetNativeLineupSet":
			command.ID = "lineup.set"
		}
		commands = append(commands, command)
	}
	return commands
}
