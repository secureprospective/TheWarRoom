package commissioner

import (
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

type DeputyCommandClass string

const (
	DeputyDelegable        DeputyCommandClass = "delegable"
	DeputyCommissionerOnly DeputyCommandClass = "commissioner-only"
	DeputyMemberAction     DeputyCommandClass = "member-action"
)

type DeputyCommandRule struct {
	Command    string
	Class      DeputyCommandClass
	Permission governance.DeputyPermission
}

// DeputyCommandRules is the one command-to-permission policy table for the native mutation catalog.
func DeputyCommandRules() []DeputyCommandRule {
	return slices.Concat(delegableCommandRules(), commissionerOnlyCommandRules(), memberActionCommandRules())
}

func delegableCommandRules() []DeputyCommandRule {
	return []DeputyCommandRule{
		{StatApprove, DeputyDelegable, governance.ApproveStatFixes},
		{StatReject, DeputyDelegable, governance.ApproveStatFixes},
		{StatCorrect, DeputyDelegable, governance.ApproveStatFixes},
		{ScorePlayer, DeputyDelegable, governance.ApproveStatFixes},
		{ScoreFranchise, DeputyDelegable, governance.ApproveStatFixes},
		{TradeReview, DeputyDelegable, governance.RuleOnTrades},
		{TradeOverride, DeputyDelegable, governance.RuleOnTrades},
		{TradeRecuse, DeputyDelegable, governance.RuleOnTrades},
		{TradeAppealRule, DeputyDelegable, governance.RuleOnTrades},
		{LineupSet, DeputyDelegable, governance.EditLineups},
		{LineupCorrect, DeputyDelegable, governance.EditLineups},
		{CorrectPick, DeputyDelegable, governance.MovePicks},
		{DraftOrderAuto, DeputyDelegable, governance.MovePicks},
		{DraftOrderSet, DeputyDelegable, governance.MovePicks},
		{DraftOrderPattern, DeputyDelegable, governance.MovePicks},
		{DraftOrderTiebreak, DeputyDelegable, governance.MovePicks},
		{DraftAssign, DeputyDelegable, governance.MovePicks},
		{DraftChange, DeputyDelegable, governance.MovePicks},
		{DraftInsert, DeputyDelegable, governance.MovePicks},
		{DraftSkip, DeputyDelegable, governance.MovePicks},
		{RulesSet, DeputyDelegable, governance.ChangeRules},
		{WindowSet, DeputyDelegable, governance.ChangeRules},
		{WindowMove, DeputyDelegable, governance.ChangeRules},
		{WindowRemove, DeputyDelegable, governance.ChangeRules},
		{CalendarException, DeputyDelegable, governance.ChangeRules},
		{SeasonRollover, DeputyDelegable, governance.RunRollover},
	}
}

func commissionerOnlyCommandRules() []DeputyCommandRule {
	return slices.Concat(commissionerOnlyLeagueRules(), commissionerOnlyPeopleRules(), commissionerOnlyMarketRules())
}

func commissionerOnlyLeagueRules() []DeputyCommandRule {
	return []DeputyCommandRule{
		{Create, DeputyCommissionerOnly, ""},
		{Revert, DeputyCommissionerOnly, ""},
		{BackupNowCommand, DeputyCommissionerOnly, ""},
		{ContractActionSet, DeputyCommissionerOnly, ""},
		{NativeSeed, DeputyCommissionerOnly, ""},
		{CorrectRoster, DeputyCommissionerOnly, ""},
		{CorrectTransaction, DeputyCommissionerOnly, ""},
		{CorrectProvenance, DeputyCommissionerOnly, ""},
		{RuleException, DeputyCommissionerOnly, ""},
		{DraftStart, DeputyCommissionerOnly, ""},
		{DraftTimer, DeputyCommissionerOnly, ""},
		{DraftRevert, DeputyCommissionerOnly, ""},
		{DraftArchive, DeputyCommissionerOnly, ""},
		{DraftStartupOpen, DeputyCommissionerOnly, ""},
		{ContractEdit, DeputyCommissionerOnly, ""},
		{CapAdjust, DeputyCommissionerOnly, ""},
		{CapRelief, DeputyCommissionerOnly, ""},
		{ReserveException, DeputyCommissionerOnly, ""},
		{ScheduleGenerate, DeputyCommissionerOnly, ""},
		{ScheduleEdit, DeputyCommissionerOnly, ""},
		{SchedulePublish, DeputyCommissionerOnly, ""},
		{BracketBuild, DeputyCommissionerOnly, ""},
		{BracketEdit, DeputyCommissionerOnly, ""},
		{BracketSeed, DeputyCommissionerOnly, ""},
		{BracketPresetSave, DeputyCommissionerOnly, ""},
		{BracketAdvance, DeputyCommissionerOnly, ""},
		{StandingsRecordAdjust, DeputyCommissionerOnly, ""},
		{StandingsOverride, DeputyCommissionerOnly, ""},
		{StandingsOverrideClear, DeputyCommissionerOnly, ""},
		{PlayerPosition, DeputyCommissionerOnly, ""},
		{PlayerPositionClear, DeputyCommissionerOnly, ""},
		{TitleSet, DeputyCommissionerOnly, ""},
		{HistoryLink, DeputyCommissionerOnly, ""},
		{NoticePublish, DeputyCommissionerOnly, ""},
		{NotePost, DeputyCommissionerOnly, ""},
		{CaretakerSeat, DeputyCommissionerOnly, ""},
		{"commish.franchise.edit", DeputyCommissionerOnly, ""},
		{FranchiseDirection, DeputyCommissionerOnly, ""},
	}
}

func commissionerOnlyPeopleRules() []DeputyCommandRule {
	return []DeputyCommandRule{
		{"commish.seat.invite", DeputyCommissionerOnly, ""},
		{"commish.seat.invite.revoke", DeputyCommissionerOnly, ""},
		{"commish.seat.remove", DeputyCommissionerOnly, ""},
		{"commish.seat.replace", DeputyCommissionerOnly, ""},
		{"commish.abilities.set", DeputyCommissionerOnly, ""},
		{"commish.visibility.set", DeputyCommissionerOnly, ""},
		{"commish.role.grant", DeputyCommissionerOnly, ""},
		{"commish.role.revoke", DeputyCommissionerOnly, ""},
		{DeputyOrder, DeputyCommissionerOnly, ""},
		{DeputyPermissionsSet, DeputyCommissionerOnly, ""},
		{"commish.succession.stepdown", DeputyCommissionerOnly, ""},
		{"commish.succession.inactivity.set", DeputyCommissionerOnly, ""},
		{"commish.succession.activity", DeputyCommissionerOnly, ""},
		{"commish.succession.voteout.open", DeputyCommissionerOnly, ""},
		{"commish.succession.election.open", DeputyCommissionerOnly, ""},
		{"commish.succession.periods.set", DeputyCommissionerOnly, ""},
		{"commish.vote.close", DeputyCommissionerOnly, ""},
		{"commish.vote.tiebreak", DeputyCommissionerOnly, ""},
	}
}

func commissionerOnlyMarketRules() []DeputyCommandRule {
	return []DeputyCommandRule{
		{WaiverProcess, DeputyCommissionerOnly, ""},
		{WaiverOrderSet, DeputyCommissionerOnly, ""},
		{WaiverOrderAuto, DeputyCommissionerOnly, ""},
		{WaiverOverride, DeputyCommissionerOnly, ""},
		{OLineAuctionClose, DeputyCommissionerOnly, ""},
		{BidsProcess, DeputyCommissionerOnly, ""},
		{BidsOverride, DeputyCommissionerOnly, ""},
	}
}

func memberActionCommandRules() []DeputyCommandRule {
	return []DeputyCommandRule{
		{TradeOffer, DeputyMemberAction, ""},
		{TradeAccept, DeputyMemberAction, ""},
		{TradeDecline, DeputyMemberAction, ""},
		{TradeWithdraw, DeputyMemberAction, ""},
		{TradeAppeal, DeputyMemberAction, ""},
		{RosterReserve, DeputyMemberAction, ""},
		{RosterCut, DeputyMemberAction, ""},
		{WaiverClaim, DeputyMemberAction, ""},
		{BidPlace, DeputyMemberAction, ""},
		{BidRaise, DeputyMemberAction, ""},
		{BidWithdraw, DeputyMemberAction, ""},
		{DraftPick, DeputyMemberAction, ""},
		{"commish.seat.invite.redeem", DeputyMemberAction, ""},
		{"commish.vote.cast", DeputyMemberAction, ""},
		{"commish.succession.accept", DeputyMemberAction, ""},
		{"commish.succession.decline", DeputyMemberAction, ""},
		{"commish.succession.takeover", DeputyMemberAction, ""},
	}
}

func DeputyCommandRuleFor(command string) (DeputyCommandRule, bool) {
	for _, rule := range DeputyCommandRules() {
		if rule.Command == command {
			return rule, true
		}
	}
	return DeputyCommandRule{}, false
}

func checkDeputyRulePermission(raw RawCommand) error {
	owner, err := ownerFor(leaguerules.FieldPath(raw.Args.Field))
	if err != nil {
		return err
	}
	control, ok := governance.CensusControl(owner.control)
	if !ok {
		return fmt.Errorf("this league rule is not available")
	}
	if control.COnly {
		return fmt.Errorf("this rule is commissioner-only")
	}
	return nil
}

func commissionerOnlyForAll(command string) bool {
	switch command {
	case "commish.role.grant", "commish.role.revoke", DeputyOrder, DeputyPermissionsSet,
		"commish.succession.inactivity.set", "commish.succession.periods.set", "commish.vote.close", "commish.vote.tiebreak":
		return true
	default:
		return false
	}
}
