package commissioner

import (
	"context"
	"encoding/json"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leagueservice"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// Host retains the session, football-fact composition and transactional gate at their existing owners.
type Host interface {
	SessionHost
	ActionHost
	LeagueHost
	ResultsHost
	RosterHost
	MutationHost
	SeatHost
}
type SessionHost = leagueservice.SessionHost
type ActionHost interface {
	RunNativeOwnerAction(string, string, string, json.RawMessage) (commissioner.Receipt, error)
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

type LeagueHost = leagueservice.LeagueHost

type ResultsHost = leagueservice.ResultsHost

type RosterHost = leagueservice.RosterHost

type API struct {
	host  Host
	reads *leagueservice.Service
}

func New(host Host) *API { return &API{host: host, reads: leagueservice.New(host)} }

const refreshTimeout = 120 * time.Second
const draftPickName = "draft.pick"

func Commands() []catalog.Command {
	methods := []struct{ method, role, gravity string }{
		{"TargetBackupNow", "gm,commish,admin", "G1"},
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
		{"TargetCommissionerSeat", "gm,commish,admin", "G0"},
		{"TargetCommissionerSeatSignIn", "gm,commish,admin", "G1"},
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
	return desktopCommands(methods)
}

func desktopCommands(methods []struct{ method, role, gravity string }) []catalog.Command {
	reads := map[string]catalog.Command{}
	for _, read := range nativeReads() {
		reads[read.Method] = read
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
		case "TargetCommissionerSeat", "TargetCommissionerSeatSignIn":
			command.Availability = "desktop" // a hosted league signs in on the server
		case "TargetBackupNow":
			command.ID, command.Availability = "backup.now", "desktop" // a hosted league backs up on the server
		}
		if read, ok := reads[command.Method]; ok {
			read.Role = command.Role
			command = read
		}
		commands = append(commands, command)
	}
	return commands
}
