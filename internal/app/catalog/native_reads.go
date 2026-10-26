package catalog

import (
	"reflect"
	"strings"

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
		{"draft.read", "TargetNativeDraft", "", "league-view", "ledger,player-facts"},
		{"schedule.franchise.read", "TargetNativeFranchiseSchedule", "", "league-view", "ledger,schedule"},
		{"native.rules.read", "TargetNativeHouseRules", "league:string", "league-view", "ledger"},
		{"lineup.read", "TargetNativeLineup", "league:string franchise:string week:integer at:string", "franchise-view", "ledger,schedule"},
		{"lineup.check", "TargetNativeLineupCheck", "", "franchise-view", "ledger,player-facts"},
		{"score.player.read", "TargetNativePlayerScore", "league:string player:string season:integer week:integer", "league-view", "ledger,stat-evidence"},
		{"native.reserve.check", "TargetNativeReserveCheck", "league:string", "league-view", "ledger,player-facts"},
		{"results.read", "TargetNativeResults", "league:string season:integer week:integer", "league-view", "ledger,stat-evidence"},
		{"native.roster.check", "TargetNativeRosterCheck", "league:string", "league-view", "ledger,player-facts"},
		{"standings.read", "TargetNativeStandings", "league:string season:integer", "league-view", "ledger,stat-evidence"},
		{"trades.read", "TargetNativeTrades", "league:string viewer:string", "franchise-view", "ledger"},
		{"waivers.read", "TargetNativeWaivers", "league:string franchise:string at:string", "franchise-view", "ledger,player-facts"},
		{"week.read", "TargetNativeWeek", "", "league-view", "ledger,schedule"},
		{"stats.queue.read", "TargetStatQueue", "league:string season:integer", "statistics.manage", "ledger,stat-evidence"},
	}
	out := make([]Command, 0, len(rows))
	for _, row := range rows {
		out = append(out, Command{ID: row.id, Method: row.method, Version: "1", Role: "domain",
			Gravity: "G0", Availability: "both", Arguments: NativeReadShape(row.method, row.fields),
			Permission: row.permission, Modes: []Mode{Read}, Undo: "none", Facts: row.facts, Safeguards: "none"})
	}
	return out
}
func NativeReadShape(method, fields string) Schema {
	switch method {
	case "TargetLeagueCalendar":
		return Shape(reflect.TypeFor[leagueservice.RawLeagueCalendarScope](), "RawLeagueCalendarScope.Validate")
	case "TargetLeagueSchedule":
		return Shape(reflect.TypeFor[leagueservice.RawLeagueScheduleScope](), "RawLeagueScheduleScope.Validate")
	case "TargetNativeContractQuote":
		return Shape(reflect.TypeFor[commissioner.RawContractQuote](), "RawContractQuote.Validate")
	case "TargetNativeDraft":
		return Shape(reflect.TypeFor[commissioner.RawNativeDraftScope](), "RawNativeDraftScope.Validate")
	case "TargetNativeFranchiseSchedule":
		return Shape(reflect.TypeFor[leagueservice.RawNativeScheduleScope](), "RawNativeScheduleScope.Validate")
	case "TargetNativeLineupCheck":
		return Shape(reflect.TypeFor[commissioner.RawNativeLineup](), "RawNativeLineup.Validate")
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
