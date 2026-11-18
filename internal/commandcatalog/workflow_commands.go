package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

type WorkflowCommand struct {
	Validate   func(RawWorkflow) error
	Run        func(context.Context, *leagueservice.Service, RawWorkflow) (json.RawMessage, error)
	RunAccount func(context.Context, accounts.HomeLayoutRepository, string, RawWorkflow) (json.RawMessage, error)
}

// Membership is executable: the same entry validates and runs a workflow. The changes are exactly
// what an agent token can carry (accounts.DelegableCommand) less the desktop-only held-data seed; the service keeps each command's gates.
// FUTURE-SEAM(commands): new actions (moments, invites, chat) arrive as commissioner definitions and
// appear here by derivation; never hand-list one. See docs/architecture/future-seams.md.
func WorkflowCommands() map[string]WorkflowCommand {
	out := workflowReads()
	out[leagueservice.BackupNowCommand] = backupNowWorkflow()
	out["draft.queue.read"] = workflowDraftQueueRead()
	out["draft.queue.set"] = workflowDraftQueueSet()
	for id, command := range accountPreferenceWorkflows() {
		out[id] = command
	}
	for id, command := range privatePersonalWorkflows() {
		out[id] = command
	}
	for _, row := range nativeMutations() {
		if accounts.DelegableCommand(row.id) && row.id != commissioner.NativeSeed {
			out[row.id] = WorkflowCommand{Validate: validateWorkflowChange, Run: runWorkflowChange}
		}
	}
	out[commissioner.DraftPick] = WorkflowCommand{Validate: validateWorkflowPick, Run: runWorkflowPick}
	return out
}

func workflowDraftQueueRead() WorkflowCommand {
	decode := func(raw RawWorkflow) (leagueservice.RawDraftQueueScope, error) {
		var input leagueservice.RawDraftQueueScope
		if raw.Mode != "read" {
			return input, fmt.Errorf("this is a reading only")
		}
		return input, decodeWorkflow(raw.Arguments, &input)
	}
	return WorkflowCommand{
		Validate: func(raw RawWorkflow) error {
			_, err := decode(raw)
			return err
		},
		Run: func(ctx context.Context, service *leagueservice.Service, raw RawWorkflow) (json.RawMessage, error) {
			input, err := decode(raw)
			if err != nil {
				return nil, err
			}
			return workflowJSON(service.TargetDraftQueue(ctx, input))
		},
	}
}

func workflowDraftQueueSet() WorkflowCommand {
	decode := func(raw RawWorkflow) (leagueservice.RawDraftQueueSet, error) {
		var input leagueservice.RawDraftQueueSet
		if raw.Mode != "execute" {
			return input, fmt.Errorf("replace the draft queue")
		}
		return input, decodeWorkflow(raw.Arguments, &input)
	}
	return WorkflowCommand{
		Validate: func(raw RawWorkflow) error {
			_, err := decode(raw)
			return err
		},
		Run: func(ctx context.Context, service *leagueservice.Service, raw RawWorkflow) (json.RawMessage, error) {
			input, err := decode(raw)
			if err != nil {
				return nil, err
			}
			return workflowJSON(service.SetDraftQueue(ctx, input))
		},
	}
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

func workflowInputCommand[T any, P interface {
	*T
	workflowInput
}](mode, refusal string, run func(context.Context, *leagueservice.Service, T) (json.RawMessage, error)) WorkflowCommand {
	decode := func(raw RawWorkflow) (T, error) {
		var input T
		if raw.Mode != mode {
			return input, fmt.Errorf("%s", refusal)
		}
		// P is *T by its constraint; the union of vetted inputs hides that from a plain conversion.
		pointer, ok := any(&input).(P)
		if !ok {
			return input, fmt.Errorf("league action: unexpected input type")
		}
		return input, decodeWorkflow(raw.Arguments, pointer)
	}
	return WorkflowCommand{
		Validate: func(raw RawWorkflow) error {
			_, err := decode(raw)
			return err
		},
		Run: func(ctx context.Context, service *leagueservice.Service, raw RawWorkflow) (json.RawMessage, error) {
			input, err := decode(raw)
			if err != nil {
				return nil, err
			}
			return run(ctx, service, input)
		},
	}
}

func backupNowWorkflow() WorkflowCommand {
	decode := func(raw RawWorkflow) (leagueservice.RawBackupNow, error) {
		var input leagueservice.RawBackupNow
		if raw.Mode != string(Preview) && raw.Mode != string(Execute) {
			return input, fmt.Errorf("preview or confirm the league backup")
		}
		if err := decodeWorkflow(raw.Arguments, &input); err != nil {
			return input, err
		}
		return input, input.Validate()
	}
	return WorkflowCommand{
		Validate: func(raw RawWorkflow) error {
			_, err := decode(raw)
			return err
		},
		Run: func(ctx context.Context, service *leagueservice.Service, raw RawWorkflow) (json.RawMessage, error) {
			input, err := decode(raw)
			if err != nil {
				return nil, err
			}
			if raw.Mode == string(Preview) {
				return workflowJSON(service.PreviewBackupNow(ctx, input))
			}
			return workflowJSON(service.BackupNow(ctx, input))
		},
	}
}

func privatePersonalWorkflows() map[string]WorkflowCommand {
	return map[string]WorkflowCommand{
		leagueservice.WhatIfStackRead: workflowInputCommand[market.RawWhatIfStack, *market.RawWhatIfStack]("read", "this is a private owner reading only", func(ctx context.Context, s *leagueservice.Service, r market.RawWhatIfStack) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeWhatIfStackContext(ctx, r))
		}),
		leagueservice.ContractDecisionsRead: workflowInputCommand[market.RawContractDecisionsScope, *market.RawContractDecisionsScope]("read", "this is a reading only", func(ctx context.Context, s *leagueservice.Service, input market.RawContractDecisionsScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeContractDecisionsContext(ctx, input))
		}),
		leagueservice.PlayerNoteRead: workflowInputCommand[leagueservice.RawPlayerNotesScope, *leagueservice.RawPlayerNotesScope]("read", "this is a reading only", func(ctx context.Context, s *leagueservice.Service, input leagueservice.RawPlayerNotesScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetPlayerNotesContext(ctx, input))
		}),
		leagueservice.PlayerNoteSet: workflowInputCommand[leagueservice.RawPlayerNoteSet, *leagueservice.RawPlayerNoteSet]("execute", "confirm the change", func(ctx context.Context, s *leagueservice.Service, input leagueservice.RawPlayerNoteSet) (json.RawMessage, error) {
			return workflowJSON(s.SetPlayerNoteContext(ctx, input))
		}),
		leagueservice.PlayerWatchRead: workflowInputCommand[leagueservice.RawPlayerWatchReadScope, *leagueservice.RawPlayerWatchReadScope]("read", "this is a reading only", func(ctx context.Context, s *leagueservice.Service, input leagueservice.RawPlayerWatchReadScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetPlayerWatchesContext(ctx, input))
		}),
		leagueservice.PlayerWatchSet: workflowInputCommand[leagueservice.RawPlayerWatchSet, *leagueservice.RawPlayerWatchSet]("execute", "confirm the change", func(ctx context.Context, s *leagueservice.Service, input leagueservice.RawPlayerWatchSet) (json.RawMessage, error) {
			return workflowJSON(s.SetPlayerWatchContext(ctx, input))
		}),
		leagueservice.PlayerWatchOrder: workflowInputCommand[leagueservice.RawPlayerWatchOrder, *leagueservice.RawPlayerWatchOrder]("execute", "confirm the change", func(ctx context.Context, s *leagueservice.Service, input leagueservice.RawPlayerWatchOrder) (json.RawMessage, error) {
			return workflowJSON(s.SetPlayerWatchOrderContext(ctx, input))
		}),
		leagueservice.TradeDraftRead: workflowInputCommand[leagueservice.RawTradeDraftScope, *leagueservice.RawTradeDraftScope]("read", "this is a reading only", func(ctx context.Context, s *leagueservice.Service, input leagueservice.RawTradeDraftScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetTradeDrafts(ctx, input))
		}),
		leagueservice.TradeDraftSave: workflowInputCommand[leagueservice.RawTradeDraftSave, *leagueservice.RawTradeDraftSave]("execute", "save this proposal", func(ctx context.Context, s *leagueservice.Service, input leagueservice.RawTradeDraftSave) (json.RawMessage, error) {
			return workflowJSON(s.SaveTradeDraft(ctx, input))
		}),
		leagueservice.TradeDraftDelete: workflowInputCommand[leagueservice.RawTradeDraftDelete, *leagueservice.RawTradeDraftDelete]("execute", "delete this saved proposal", func(ctx context.Context, s *leagueservice.Service, input leagueservice.RawTradeDraftDelete) (json.RawMessage, error) {
			return workflowJSON(s.DeleteTradeDraft(ctx, input))
		}),
	}
}

func workflowReads() map[string]WorkflowCommand {
	out := pulseWorkflowReads()
	for id, command := range otherWorkflowReads() {
		out[id] = command
	}
	return out
}

func pulseWorkflowReads() map[string]WorkflowCommand {
	return map[string]WorkflowCommand{
		"native.directory.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeDirectory(r.League))
		}),
		"nfl.game.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNFLGame(r.League, r.Game))
		}),
		"nfl.games.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNFLGames(r.League))
		}),
		"schedule.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetLeagueSchedule(leagueservice.RawLeagueScheduleScope{League: r.League}))
		}),
		"week.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			at := time.Now().UTC()
			if r.At != "" {
				var err error
				at, err = time.Parse(time.RFC3339Nano, r.At)
				if err != nil {
					return nil, fmt.Errorf("read week time: %w", err)
				}
			}
			return workflowJSON(s.TargetNativeWeek(leagueservice.RawNativeWeekScope{RawNativeScheduleScope: leagueservice.RawNativeScheduleScope{League: r.League, Franchise: r.Franchise, Season: r.Season, LocalZone: r.LocalZone}, At: at}))
		}),
		"projections.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			players, err := workflowPlayers(r.Players)
			if err != nil {
				return nil, err
			}
			return workflowJSON(s.TargetNativeProjections(r.League, players, r.Season, r.Week))
		}),
		"player.usage.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return s.TargetNativePlayerUsage(leagueservice.RawPlayerReadScope{League: r.League, Player: r.Player})
		}),
		"player.agearc.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return s.TargetNativePlayerAgeArc(leagueservice.RawPlayerReadScope{League: r.League, Player: r.Player})
		}),
		"bracket.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeBracket(r.League, r.Season))
		}),
		"market.wins.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeBidWins(r.League))
		}),
	}
}

func otherWorkflowReads() map[string]WorkflowCommand {
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
		"franchise.moves.history.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetFranchiseMovesHistory(commissioner.RawFranchiseMovesHistory{League: r.League, Franchise: r.Franchise, Season: r.Season}))
		}),
		"moves.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeMoves(r.League, r.Franchise))
		}),
		"trades.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeTrades(r.League, r.Viewer))
		}),
		"results.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeResults(r.League, r.Season, r.Week))
		}),
		"value.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeValues(leagueservice.RawValueScope{League: r.League, Franchise: r.Franchise, Season: r.Season}))
		}),
		"odds.read": workflowRead(func(s *leagueservice.Service, r RawWorkflowScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeOdds(r.League, r.Season))
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
		"native.cap.check": workflowInputCommand[commissioner.RawNativeCapScope, *commissioner.RawNativeCapScope]("read", "this is a reading only", func(ctx context.Context, s *leagueservice.Service, r commissioner.RawNativeCapScope) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeCapCheckContext(ctx, r.League, r.ViewerZone))
		}),
		"contract.quote": workflowInputCommand[market.RawContractQuote, *market.RawContractQuote]("read", "this is a reading only", func(ctx context.Context, s *leagueservice.Service, r market.RawContractQuote) (json.RawMessage, error) {
			return workflowJSON(s.TargetNativeContractQuoteContext(ctx, r))
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
