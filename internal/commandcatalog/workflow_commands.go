package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

type WorkflowCommand struct {
	Validate func(RawWorkflow) error
	Run      func(context.Context, *leagueservice.Service, RawWorkflow) (json.RawMessage, error)
}

// Membership is executable: the same entry validates and runs a workflow. This bounded
// transport does not expose the other native commands supported by generic dispatch.
func WorkflowCommands() map[string]WorkflowCommand {
	out := workflowReads()
	for _, id := range []string{
		commissioner.LineupSet, commissioner.RosterReserve, commissioner.DraftTimer, commissioner.DraftSkip,
		commissioner.BidPlace, commissioner.BidRaise, commissioner.BidWithdraw, commissioner.BidsProcess, commissioner.BidsOverride,
		commissioner.RosterCut, commissioner.WaiverClaim, commissioner.WaiverProcess, commissioner.WaiverOrderSet, commissioner.WaiverOrderAuto, commissioner.WaiverOverride,
		commissioner.TradeOffer, commissioner.TradeAccept, commissioner.TradeDecline, commissioner.TradeWithdraw, commissioner.TradeReview, commissioner.TradeRecuse, commissioner.TradeAppeal, commissioner.TradeAppealRule, commissioner.TradeOverride,
		commissioner.SeasonRollover, commissioner.ScoreFranchise, commissioner.StandingsRecordAdjust, commissioner.StandingsOverrideClear,
	} {
		out[id] = WorkflowCommand{Validate: validateWorkflowChange, Run: runWorkflowChange}
	}
	out[commissioner.DraftPick] = WorkflowCommand{Validate: validateWorkflowPick, Run: runWorkflowPick}
	return out
}

func workflowRead(read func(*leagueservice.Service, RawWorkflowScope) (json.RawMessage, error)) WorkflowCommand {
	return WorkflowCommand{
		Validate: func(raw RawWorkflow) error {
			if raw.Mode != "read" {
				return fmt.Errorf("this is a reading only")
			}
			var scope RawWorkflowScope
			if err := decodeWorkflow(raw.Arguments, &scope); err != nil {
				return err
			}
			return scope.validateCommand(raw.Command)
		},
		Run: func(_ context.Context, service *leagueservice.Service, raw RawWorkflow) (json.RawMessage, error) {
			var scope RawWorkflowScope
			if err := decodeWorkflow(raw.Arguments, &scope); err != nil {
				return nil, err
			}
			return read(service, scope)
		},
	}
}

func workflowReads() map[string]WorkflowCommand {
	return map[string]WorkflowCommand{
		"commish.market.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetCommissionerMarket(r.League, r.Command))
		}),
		"bids.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeBids(r.League, r.Franchise))
		}),
		"waivers.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeWaivers(r.League, r.Franchise, time.Now().UTC()))
		}),
		"trades.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeTrades(r.League, r.Viewer))
		}),
		"results.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeResults(r.League, r.Season, r.Week))
		}),
		"standings.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeStandings(r.League, r.Season))
		}),
		"commish.corrections.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetCorrections(r.League))
		}),
		"commish.rules.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetCommissionerRules(r.League))
		}),
		"native.assets.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeAssets(r.League))
		}),
		"native.reserve.check": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeReserveCheck(r.League))
		}),
		"native.roster.check": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeRosterCheck(r.League))
		}),
		"lineup.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeLineup(r.League, r.Franchise, r.Week, r.At))
		}),
		"draft.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeDraft(commissioner.RawNativeDraftScope{League: r.League, Franchise: r.Franchise, Season: r.Season}))
		}),
	}
}

func validateWorkflowChange(raw RawWorkflow) error {
	if raw.Mode != "preview" && raw.Mode != "execute" {
		return fmt.Errorf("check or confirm the change")
	}
	var command commissioner.RawCommand
	if err := decodeWorkflow(raw.Arguments, &command); err != nil {
		return err
	}
	if command.Command != raw.Command {
		return fmt.Errorf("choose one league action")
	}
	return nil
}

func runWorkflowChange(ctx context.Context, service *leagueservice.Service, raw RawWorkflow) (json.RawMessage, error) {
	var command commissioner.RawCommand
	if err := decodeWorkflow(raw.Arguments, &command); err != nil {
		return nil, err
	}
	if raw.Mode == "preview" {
		//nolint:contextcheck // Preview reads through the request-scoped host.
		return workflowJSON(service.Preview(command))
	}
	if command.Command == commissioner.LineupSet {
		return workflowJSON(service.SetLineup(ctx, command))
	}
	return workflowJSON(service.Execute(ctx, command))
}

func validateWorkflowPick(raw RawWorkflow) error {
	if raw.Mode != "execute" {
		return fmt.Errorf("confirm your rookie pick")
	}
	var pick commissioner.DraftPickInput
	return decodeWorkflow(raw.Arguments, &pick)
}

func runWorkflowPick(ctx context.Context, service *leagueservice.Service, raw RawWorkflow) (json.RawMessage, error) {
	var pick commissioner.DraftPickInput
	if err := decodeWorkflow(raw.Arguments, &pick); err != nil {
		return nil, err
	}
	return workflowJSON(service.Pick(ctx, pick))
}
