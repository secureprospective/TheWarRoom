package catalog

import (
	"reflect"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

func queryShape(method, fields string) Schema {
	schema := Schema{Type: "object", Validator: method, Fields: []Field{}}
	for _, token := range strings.Fields(fields) {
		name, kind, _ := strings.Cut(token, ":")
		schema.Fields = append(schema.Fields, Field{Name: name, Schema: Schema{Type: kind}})
	}
	return schema
}
func nativeReadAvailability(facts string) string {
	if facts == "private-player-data" {
		return "server"
	}
	return "both"
}

func nativeReads() []Command {
	rows := []struct{ id, method, fields, permission, facts string }{
		{"commish.actor.read", "TargetCommissionerActor", "", "session", "none"},
		{"commish.controls.read", "TargetCommissionerControls", "", "session", "none"},
		{"commish.digests.read", "TargetCommissionerDigests", "league:string", "league-view", "ledger"},
		{"commish.jobs.read", "TargetCommissionerJobs", "league:string", "league-view", "ledger"},
		{"commish.leagues.read", "TargetCommissionerLeagues", "", "league-memberships", "ledger"},
		{"commish.ledger.read", "TargetCommissionerLedger", "league:string", "league-view", "ledger"},
		{"commish.ledger.filtered.read", "TargetCommissionerLedgerFiltered", "league:string", "league-view", "ledger"},
		{"commish.market.read", "TargetCommissionerMarket", "league:string command:string", "command-policy", "ledger,player-facts"},
		{"commish.notices.read", "TargetCommissionerNotices", "league:string", "league-view", "ledger"},
		{"commish.people.read", "TargetCommissionerPeople", "league:string", "league-view", "ledger"},
		{"commish.rules.read", "TargetCommissionerRules", "league:string", "league-view", "ledger"},
		{"commish.rules.preset.read", "TargetCommissionerPresetCompare", "league:string preset:string", "league-view", "ledger"},
		{"commish.schedule.read", "TargetCommissionerSchedule", "league:string", "schedule.manage", "ledger"},
		{"commish.corrections.read", "TargetCorrections", "league:string", "league-view", "ledger"},
		{"calendar.read", "TargetLeagueCalendar", "", "league-view", "ledger"},
		{"schedule.read", "TargetLeagueSchedule", "", "league-view", "ledger"},
		{"native.assets.read", "TargetNativeAssets", "league:string", "league-view", "ledger"},
		{"bids.read", "TargetNativeBids", "league:string franchise:string", "franchise-view", "ledger,player-facts"},
		{"bracket.read", "TargetNativeBracket", "league:string season:integer", "league-view", "ledger"},
		{"native.cap.check", "TargetNativeCapCheck", "league:string viewerZone:string", "league-view", "ledger"},
		{"contract.quote", "TargetNativeContractQuote", "", "league-view", "ledger,player-facts"},
		{"native.directory.read", "TargetNativeDirectory", "league:string", "league-view", "ledger"},
		{"nfl.game.read", "TargetNFLGame", "league:string game:string", "league-view", "stat-evidence"},
		{"nfl.games.read", "TargetNFLGames", "league:string", "league-view", "stat-evidence"},
		{"draft.read", "TargetNativeDraft", "", "league-view", "ledger,player-facts"},
		{"schedule.franchise.read", "TargetNativeFranchiseSchedule", "", "league-view", "ledger,schedule"},
		{"native.rules.read", "TargetNativeHouseRules", "league:string", "league-view", "ledger"},
		{"lineup.read", "TargetNativeLineup", "league:string franchise:string week:integer at:string", "franchise-view", "ledger,schedule"},
		{"lineup.check", "TargetNativeLineupCheck", "", "franchise-view", "ledger,player-facts"},
		{"lineupdelta.read", "TargetNativeLineupDelta", "", "franchise-view", "ledger,stat-evidence,player-facts"},
		{"player.usage.read", "TargetNativePlayerUsage", "league:string player:string", "league-view", "stat-evidence,player-record"},
		{"player.agearc.read", "TargetNativePlayerAgeArc", "league:string player:string", "league-view", "stat-evidence,player-record,fitted-params"},
		{leagueservice.PlayerNoteRead, "TargetPlayerNotes", "league:string player:string", "franchise-owner", "private-player-data"},
		{"value.read", "TargetNativeValues", "", "league-view", "ledger,stat-evidence,player-record,fitted-params"},
		{"tradelens.read", "TargetNativeTradeLens", "", "franchise-view", "ledger,stat-evidence,player-record,fitted-params"},
		{"projections.read", "TargetNativeProjections", "league:string players:string season:integer week:integer", "league-view", "ledger,stat-evidence"},
		{"score.player.read", "TargetNativePlayerScore", "league:string player:string season:integer week:integer", "league-view", "ledger,stat-evidence"},
		{"native.reserve.check", "TargetNativeReserveCheck", "league:string", "league-view", "ledger,player-facts"},
		{"results.read", "TargetNativeResults", "league:string season:integer week:integer", "league-view", "ledger,stat-evidence"},
		{"native.roster.check", "TargetNativeRosterCheck", "league:string", "league-view", "ledger,player-facts"},
		{"odds.read", "TargetNativeOdds", "league:string season:integer", "league-view", "ledger,stat-evidence,player-record,fitted-params"},
		{"standings.read", "TargetNativeStandings", "league:string season:integer", "league-view", "ledger,stat-evidence"},
		{"franchise.moves.history.read", "TargetFranchiseMovesHistory", "", "league-view", "ledger"},
		{"moves.read", "TargetNativeMoves", "league:string franchise:string", "franchise-view", "ledger,player-facts"},
		{"trades.read", "TargetNativeTrades", "league:string viewer:string", "franchise-view", "ledger"},
		{"waivers.read", "TargetNativeWaivers", "league:string franchise:string at:string", "franchise-view", "ledger,player-facts"},
		{"week.read", "TargetNativeWeek", "", "league-view", "ledger,schedule"},
		{"stats.queue.read", "TargetStatQueue", "league:string season:integer request:string", "statistics.manage", "ledger,stat-evidence"},
	}
	out := make([]Command, 0, len(rows))
	for _, row := range rows {
		out = append(out, Command{ID: row.id, Method: row.method, Version: "1", Role: "domain",
			Gravity: "G0", Availability: nativeReadAvailability(row.facts), Arguments: NativeReadShape(row.method, row.fields),
			Permission: row.permission, Modes: []Mode{Read}, Undo: "none", Facts: row.facts, Safeguards: "none"})
	}
	return out
}
func playerPersonalNativeReads() []Command {
	return []Command{{ID: leagueservice.PlayerWatchRead, Method: "TargetPlayerWatches", Version: "1",
		Role: "domain", Gravity: "G0", Availability: "server",
		Arguments: Shape(reflect.TypeFor[leagueservice.RawPlayerWatchReadScope](),
			"RawPlayerWatchReadScope.Validate"), Permission: "franchise-owner",
		Modes: []Mode{Read}, Undo: "none", Facts: "private-player-data",
		Safeguards: "authenticated seated franchise; no request franchise; uncached private entries"}}
}

func nativeDraftQueueCommands() []Command {
	return []Command{
		{ID: "draft.queue.read", Method: "TargetDraftQueue", Version: "1", Role: "domain", Gravity: "G0",
			Availability: "server", Arguments: Shape(reflect.TypeFor[leagueservice.RawDraftQueueScope](), "RawDraftQueueScope.Validate"),
			Permission: "franchise-view", Modes: []Mode{Read}, Undo: "none", Facts: "private-entries", Safeguards: "authenticated-franchise"},
		{ID: "draft.queue.set", Method: "SetDraftQueue", Version: "1", Role: "domain", Gravity: "G1",
			Availability: "server", Arguments: Shape(reflect.TypeFor[leagueservice.RawDraftQueueSet](), "RawDraftQueueSet.Validate"),
			Permission: "franchise-owner", Modes: []Mode{Execute}, Undo: "replace-own-private-queue", Facts: "private-entries", Safeguards: "authenticated-franchise"},
	}
}

func NativeReadShape(method, fields string) Schema {
	switch method {
	case "TargetLeagueCalendar":
		return Shape(reflect.TypeFor[leagueservice.RawLeagueCalendarScope](), "RawLeagueCalendarScope.Validate")
	case "TargetLeagueSchedule":
		return Shape(reflect.TypeFor[leagueservice.RawLeagueScheduleScope](), "RawLeagueScheduleScope.Validate")
	case "TargetNativeContractQuote":
		return Shape(reflect.TypeFor[market.RawContractQuote](), "RawContractQuote.Validate")
	case "TargetNativeDraft":
		return Shape(reflect.TypeFor[commissioner.RawNativeDraftScope](), "RawNativeDraftScope.Validate")
	case "TargetNativeFranchiseSchedule":
		return Shape(reflect.TypeFor[leagueservice.RawNativeScheduleScope](), "RawNativeScheduleScope.Validate")
	case "TargetNativeValues":
		return Shape(reflect.TypeFor[leagueservice.RawValueScope](), "RawValueScope.Validate")
	case "TargetNativeTradeLens":
		return Shape(reflect.TypeFor[leagueservice.RawTradeLensScope](), "RawTradeLensScope.Validate")
	case "TargetNativeLineupDelta":
		return Shape(reflect.TypeFor[leagueservice.RawLineupDeltaScope](), "RawLineupDeltaScope.Validate")
	case "TargetNativePlayerUsage", "TargetNativePlayerAgeArc":
		return Shape(reflect.TypeFor[leagueservice.RawPlayerReadScope](), "RawPlayerReadScope.Validate")
	case "TargetPlayerNotes":
		return Shape(reflect.TypeFor[leagueservice.RawPlayerNotesScope](), "RawPlayerNotesScope.Validate")
	case "TargetNativeLineupCheck":
		return Shape(reflect.TypeFor[commissioner.RawNativeLineup](), "RawNativeLineup.Validate")
	case "TargetFranchiseMovesHistory":
		return Shape(reflect.TypeFor[commissioner.RawFranchiseMovesHistory](), "RawFranchiseMovesHistory.Validate")
	case "TargetNativeWeek":
		return Shape(reflect.TypeFor[leagueservice.RawNativeWeekScope](), "RawNativeWeekScope.Validate")
	case "TargetCommissionerLedgerFiltered":
		schema := queryShape(method, fields)
		schema.Fields = append(schema.Fields, Field{Name: "filter", Schema: Shape(reflect.TypeFor[commissioner.RawLedgerFilter](), "RawLedgerFilter.Validate")})
		return schema
	default:
		return queryShape(method, fields)
	}
}
