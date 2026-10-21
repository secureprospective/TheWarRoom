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
	NativeLineup(ctx context.Context, league, franchise string, week int,
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
	methods := []string{
		"TargetCommissionerActor",
		"TargetCommissionerControls",
		"TargetCommissionerDigests",
		"TargetCommissionerExecute",
		"TargetCommissionerJobs",
		"TargetCommissionerLeagues",
		"TargetCommissionerLedger",
		"TargetCommissionerLedgerFiltered",
		"TargetCommissionerMarket",
		"TargetCommissionerNotices",
		"TargetCommissionerPeople",
		"TargetCommissionerPreview",
		"TargetCommissionerRules",
		"TargetCommissionerSchedule",
		"TargetCorrections",
		"TargetLeagueCalendar",
		"TargetLeagueSchedule",
		"TargetNativeAssets",
		"TargetNativeBidAction",
		"TargetNativeBids",
		"TargetNativeBracket",
		"TargetNativeCapCheck",
		"TargetNativeContractQuote",
		"TargetNativeDirectory",
		"TargetNativeDraft",
		"TargetNativeDraftPick",
		"TargetNativeFranchiseSchedule",
		"TargetNativeHouseRules",
		"TargetNativeLineup",
		"TargetNativeLineupCheck",
		"TargetNativeLineupSet",
		"TargetNativePlayerScore",
		"TargetNativeReserveCheck",
		"TargetNativeResults",
		"TargetNativeRosterCheck",
		"TargetNativeScorePreview",
		"TargetNativeSeed",
		"TargetNativeStandings",
		"TargetNativeTradeAction",
		"TargetNativeTrades",
		"TargetNativeWaiverAction",
		"TargetNativeWaivers",
		"TargetNativeWeek",
		"TargetStatQueue",
	}
	commands := make([]catalog.Command, 0, len(methods))
	for _, method := range methods {
		command := catalog.Command{ID: catalog.Unset, Method: method, Version: catalog.Unset,
			Role: catalog.Unset, Gravity: catalog.Unset, Availability: "both"}
		switch method {
		case "TargetCommissionerLedger", "TargetCommissionerLedgerFiltered":
			command.ID, command.Gravity = "commish.ledger.read", "G0"
		case "TargetNativeLineupSet":
			command.ID, command.Gravity = "lineup.set", "G3"
		}
		commands = append(commands, command)
	}
	return commands
}
