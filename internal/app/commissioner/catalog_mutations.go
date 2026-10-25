package commissioner

import (
	"reflect"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// NativeCommands lists logical reads and individual mutations, not generic desktop dispatch doors.
// A completed declaration is not a live route; hosted registration and credential admission belong
// to the service. Schemas describe canonical requests, which existing desktop adapters may compose.
func NativeCommands() []catalog.Command {
	out := nativeReads()
	for _, row := range nativeMutations() {
		out = append(out, nativeMutation(row.id, row.args))
	}
	return out
}

func nativeMutations() []struct{ id, args string } {
	return slices.Concat(nativeLeagueMutations(), nativeMarketMutations(), nativePeopleMutations())
}

func nativeLeagueMutations() []struct{ id, args string } {
	return []struct{ id, args string }{
		{commissioner.Create, "name preset franchiseCount season"},
		{commissioner.Revert, "entry"},
		{commissioner.RulesSet, "field value"},
		{commissioner.ContractActionSet, "contractAction"},
		{commissioner.NativeSeed, ""},
		{commissioner.SeasonRollover, "rollover"},
		{commissioner.CalendarException, "exception"},
		{commissioner.CorrectRoster, "correction"},
		{commissioner.CorrectTransaction, "correction"},
		{commissioner.CorrectProvenance, "correction"},
		{commissioner.CorrectPick, "correction"},
		{commissioner.RuleException, "correction"},
		{commissioner.DraftOrderAuto, "draft"},
		{commissioner.DraftOrderSet, "draft"},
		{commissioner.DraftOrderPattern, "draft"},
		{commissioner.DraftOrderTiebreak, "draft"},
		{commissioner.DraftStart, "draft"},
		{commissioner.DraftPick, "draft"},
		{commissioner.DraftSkip, "draft"},
		{commissioner.DraftAssign, "draft"},
		{commissioner.DraftInsert, "draft"},
		{commissioner.DraftChange, "draft"},
		{commissioner.DraftTimer, "draft"},
		{commissioner.DraftRevert, "draft"},
		{commissioner.DraftArchive, "draft"},
		{commissioner.WindowSet, "window"},
		{commissioner.WindowMove, "window"},
		{commissioner.WindowRemove, "window"},
		{commissioner.ContractEdit, "override"},
		{commissioner.CapAdjust, "override"},
		{commissioner.CapRelief, "override"},
		{commissioner.ReserveException, "override"},
		{commissioner.ScheduleGenerate, "schedule"},
		{commissioner.ScheduleEdit, "schedule"},
		{commissioner.SchedulePublish, "schedule"},
		{commissioner.BracketBuild, "bracket"},
		{commissioner.BracketEdit, "bracket"},
		{commissioner.BracketSeed, "bracket"},
		{commissioner.BracketPresetSave, "bracket"},
		{commissioner.BracketAdvance, "bracket"},
		{commissioner.StandingsRecordAdjust, "standings"},
		{commissioner.StandingsOverride, "standings"},
		{commissioner.StandingsOverrideClear, "standings"},
		{commissioner.StatApprove, "stat"},
		{commissioner.StatReject, "stat"},
		{commissioner.StatCorrect, "stat"},
		{commissioner.PlayerPosition, "score"},
		{commissioner.PlayerPositionClear, "score"},
		{commissioner.ScorePlayer, "score"},
		{commissioner.ScoreFranchise, "score"},
		{commissioner.LineupSet, "lineup"},
		{commissioner.LineupCorrect, "lineup"},
		{commissioner.TitleSet, "title"},
		{commissioner.HistoryLink, "link"},
		{commissioner.NoticePublish, "note"},
		{commissioner.NotePost, "note"},
	}
}

func nativeMarketMutations() []struct{ id, args string } {
	return []struct{ id, args string }{
		{commissioner.TradeOffer, "trade"},
		{commissioner.TradeAccept, "trade"},
		{commissioner.TradeDecline, "trade"},
		{commissioner.TradeWithdraw, "trade"},
		{commissioner.TradeReview, "trade"},
		{commissioner.TradeRecuse, "trade"},
		{commissioner.TradeAppeal, "trade"},
		{commissioner.TradeAppealRule, "trade"},
		{commissioner.TradeOverride, "trade"},
		{commissioner.RosterReserve, "waiver"},
		{commissioner.RosterCut, "waiver"},
		{commissioner.WaiverClaim, "waiver"},
		{commissioner.WaiverProcess, "waiver"},
		{commissioner.WaiverOrderSet, "waiver"},
		{commissioner.WaiverOrderAuto, "waiver"},
		{commissioner.WaiverOverride, "waiver"},
		{commissioner.BidPlace, "bid"},
		{commissioner.BidRaise, "bid"},
		{commissioner.BidWithdraw, "bid"},
		{commissioner.BidsProcess, "bid"},
		{commissioner.BidsOverride, "bid"},
	}
}

func nativePeopleMutations() []struct{ id, args string } {
	return []struct{ id, args string }{
		{commissioner.CaretakerSeat, "people"},
		{"commish.franchise.edit", "people"},
		{"commish.seat.invite", "people"},
		{"commish.seat.invite.revoke", "people"},
		{"commish.seat.invite.redeem", "people"},
		{"commish.seat.remove", "people"},
		{"commish.seat.replace", "people"},
		{"commish.abilities.set", "people"},
		{"commish.visibility.set", "people"},
		{"commish.role.grant", "people"},
		{"commish.role.revoke", "people"},
		{commissioner.DeputyOrder, "people"},
		{"commish.succession.stepdown", "people"},
		{"commish.succession.accept", "people"},
		{"commish.succession.decline", "people"},
		{"commish.succession.inactivity.set", "people"},
		{"commish.succession.activity", "people"},
		{"commish.succession.takeover", "people"},
		{"commish.succession.voteout.open", "people"},
		{"commish.succession.election.open", "people"},
		{"commish.succession.periods.set", "people"},
		{"commish.vote.cast", "people"},
		{"commish.vote.close", "people"},
		{"commish.vote.tiebreak", "people"},
	}
}

func nativeMutation(id, args string) catalog.Command {
	gravity := "argument-dependent"
	if definition, err := commissioner.DefinitionFor(id); err == nil && id != commissioner.ContractActionSet {
		gravity = definition.Gravity
	}
	command := catalog.Command{ID: id, Method: nativeMutationMethod(id), Version: "1", Role: "domain",
		Gravity: gravity, Availability: "both", Permission: "command:" + id,
		Modes: []catalog.Mode{catalog.Preview, catalog.Execute}, Undo: "checked-reversal",
		Facts: "ledger", Safeguards: "RawCommand.EffectiveDefinition+MissingReason",
		Arguments: mutationShape(id, args)}
	if id == commissioner.DraftPick {
		command.Arguments = catalog.Shape(reflect.TypeFor[commissioner.DraftPickInput](), "DraftPickInput.Validate")
		command.Modes = []catalog.Mode{catalog.Execute}
		command.Safeguards = "on-clock owner selection; server fixes slot and contract"
	}
	if id == commissioner.NativeSeed {
		command.Availability, command.Undo = "desktop", "none"
		command.Arguments = catalog.Shape(reflect.TypeFor[RawNativeSeedRequest](), "RawNativeSeedRequest.Validate/ValidatePreview")
		command.Facts = "held-mirror"
	}

	if id == commissioner.Create || strings.HasPrefix(id, "draft.") &&
		!strings.HasPrefix(id, "draft.order.") && id != commissioner.DraftStart {
		command.Undo = "none"
	}
	switch args {
	case "bid", "waiver", "trade", "draft", "correction", "override":
		command.Facts = "ledger,player-facts"
	case "score", "standings", "stat", "bracket", "title":
		command.Facts = "ledger,stat-evidence"
	case "lineup", "schedule", "rollover":
		command.Facts = "ledger,player-facts,schedule,stat-evidence"
	}
	return command
}

func mutationShape(id, args string) catalog.Schema {
	schema := catalog.Shape(reflect.TypeFor[commissioner.RawCommand](), "RawCommand.Validate/ValidatePreview:"+id)
	allowed := strings.Fields(args)
	for i := range schema.Fields {
		if schema.Fields[i].Name == "command" {
			schema.Fields[i].Schema.Enum = []string{id}
		}
		if schema.Fields[i].Name != "args" {
			continue
		}
		fields := schema.Fields[i].Schema.Fields
		schema.Fields[i].Schema.Fields = slices.DeleteFunc(fields, func(field catalog.Field) bool {
			return !slices.Contains(allowed, field.Name)
		})
	}
	return schema
}

func nativeMutationMethod(id string) string {
	switch {
	case commissioner.OwnerTradeAction(id) != "":
		return "TargetNativeTradeAction"
	case commissioner.OwnerWaiverAction(id) != "":
		return "TargetNativeWaiverAction"
	case commissioner.OwnerBidAction(id) != "":
		return "TargetNativeBidAction"
	case id == commissioner.LineupSet:
		return "TargetNativeLineupSet"
	case id == commissioner.DraftPick:
		return "TargetNativeDraftPick"
	case id == commissioner.NativeSeed:
		return "TargetNativeSeed"
	default:
		return "TargetCommissionerExecute"
	}
}
