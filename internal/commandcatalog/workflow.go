package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/appvalue"
	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// RawWorkflow is the catalog-generated client envelope. Publication here is deliberately bounded.
type RawWorkflow struct {
	Command   string          `json:"command"`
	Version   string          `json:"version"`
	Mode      string          `json:"mode"`
	Arguments json.RawMessage `json:"arguments"`
	Football  string          `json:"football,omitempty"`
}

type RawWorkflowScope struct {
	Command   string `json:"command,omitempty"`
	Game      string `json:"game,omitempty"`
	League    string `json:"league"`
	Franchise string `json:"franchise,omitempty"`
	Viewer    string `json:"viewer,omitempty"`
	Week      int    `json:"week,omitempty"`
	Season    int    `json:"season,omitempty"`
	At        string `json:"at,omitempty"`
	LocalZone string `json:"localZone,omitempty"`
	Players   string `json:"players,omitempty"`
	Player    string `json:"player,omitempty"`
}

// RawFilteredLedgerScope is the separately validated filter used by commissioner Control Room reads.
type RawFilteredLedgerScope struct {
	League string                       `json:"league"`
	Filter commissioner.RawLedgerFilter `json:"filter"`
}

func (r RawFilteredLedgerScope) Validate() error {
	if err := leagueservice.AppLeague(r.League); err != nil {
		return fmt.Errorf("choose an app league: %w", err)
	}
	if err := r.Filter.Validate(); err != nil {
		return fmt.Errorf("ledger filter: %w", err)
	}
	return nil
}

func (r RawWorkflowScope) Validate() error {
	if !strings.HasPrefix(r.League, "native.") || len(r.League) <= 7 || len(r.League) > 128 || r.Week < 0 || r.Week > 22 || r.Season < 0 || r.Season > 9999 {
		return fmt.Errorf("choose your league, franchise and week")
	}
	return nil
}

func (r RawWorkflowScope) validateCommand(command string) error {
	kept, complete := r.commandScope(command)
	if r != kept || !complete {
		return fmt.Errorf("choose only the details of this league reading")
	}
	if r.LocalZone != "" {
		if r.LocalZone == "Local" {
			return fmt.Errorf("choose an IANA time zone")
		}
		if _, err := time.LoadLocation(r.LocalZone); err != nil {
			return fmt.Errorf("choose an IANA time zone: %w", err)
		}
	}
	if command == "projections.read" {
		if _, err := workflowPlayers(r.Players); err != nil {
			return err
		}
	}
	if err := validateWorkflowPlayer(command, r.Player); err != nil {
		return err
	}
	if r.At != "" {
		at, err := time.Parse(time.RFC3339Nano, r.At)
		if err != nil {
			return fmt.Errorf("choose when to read the league: %w", err)
		}
		if command == "week.read" && at.After(time.Now().Add(weekReadClockSkew)) {
			return fmt.Errorf("choose a past time to read the week")
		}
	}
	return nil
}

// weekReadClockSkew lets a phone whose clock runs a little ahead of the server read the week: the
// app sends "now" from the device, and an S25 a few seconds fast got "choose a past time".
const weekReadClockSkew = 5 * time.Minute

func (r RawWorkflowScope) commandScope(command string) (RawWorkflowScope, bool) {
	if kept, complete, matched := r.franchiseScope(command); matched {
		return kept, complete
	}
	if kept, complete, matched := r.pulseScope(command); matched {
		return kept, complete
	}
	kept := RawWorkflowScope{League: r.League}
	switch command {
	case "commish.market.read":
		kept.Command = r.Command
		return kept, r.Command != ""
	case "draft.read":
		kept.Franchise, kept.Season, kept.At = r.Franchise, r.Season, r.At
		return kept, r.Franchise != ""
	case "franchise.moves.history.read":
		kept.Franchise, kept.Season = r.Franchise, r.Season
		return kept, (commissioner.RawFranchiseMovesHistory{League: r.League, Franchise: r.Franchise, Season: r.Season}).Validate() == nil
	case "bids.read", "moves.read":
		kept.Franchise = r.Franchise
		return kept, r.Franchise != ""
	case "waivers.read":
		kept.Franchise, kept.At = r.Franchise, r.At
		return kept, r.Franchise != "" && r.At != ""
	case "trades.read":
		kept.Viewer = r.Viewer
		return kept, r.Viewer != ""
	case "results.read":
		kept.Season, kept.Week = r.Season, r.Week
		return kept, r.Season >= 1920 && r.Week > 0
	case "value.read":
		kept.Franchise, kept.Season = r.Franchise, r.Season
		return kept, r.Season >= 1920
	case "standings.read", "odds.read":
		kept.Season = r.Season
		return kept, r.Season >= 1920
	default:
		return kept, true
	}
}

func (r RawWorkflowScope) franchiseScope(command string) (RawWorkflowScope, bool, bool) {
	kept := RawWorkflowScope{League: r.League}
	switch command {
	case "schedule.franchise.read":
		kept.Franchise, kept.Season, kept.LocalZone = r.Franchise, r.Season, r.LocalZone
		scope := leagueservice.RawNativeScheduleScope{League: r.League, Franchise: r.Franchise,
			Season: r.Season, LocalZone: r.LocalZone}
		return kept, scope.Validate() == nil, true
	case "lineup.read":
		kept.Franchise, kept.Week, kept.At = r.Franchise, r.Week, r.At
		return kept, r.Franchise != "" && r.Week > 0, true
	default:
		return RawWorkflowScope{}, false, false
	}
}

func (r RawWorkflowScope) pulseScope(command string) (RawWorkflowScope, bool, bool) {
	kept := RawWorkflowScope{League: r.League}
	switch command {
	case "nfl.game.read", "nfl.game.pregame.read":
		kept.Game = r.Game
		return kept, r.Game != "" && len(r.Game) <= 32 && strings.Trim(r.Game, "0123456789") == "", true
	case "week.read":
		kept.Franchise, kept.Season, kept.LocalZone, kept.At = r.Franchise, r.Season, r.LocalZone, r.At
		return kept, r.Season >= 1900 && r.LocalZone != "", true
	case "projections.read":
		kept.Players, kept.Season, kept.Week = r.Players, r.Season, r.Week
		return kept, r.Players != "" && r.Season >= 1920 && r.Week > 0, true
	case "player.usage.read", "player.agearc.read", leagueservice.PlayerNewsRead:
		kept.Player = r.Player
		return kept, r.Player != "", true
	case "bracket.read":
		kept.Season = r.Season
		return kept, r.Season >= 1920, true
	default:
		return kept, false, false
	}
}

func workflowPlayers(raw string) (string, error) {
	parts := strings.Split(raw, ",")
	if len(parts) > 120 {
		return "", fmt.Errorf("choose 1 to 120 players")
	}
	seen := make(map[string]bool, len(parts))
	ids := make([]string, 0, len(parts))
	for _, part := range parts {
		id := strings.TrimSpace(part)
		parsed, err := playerid.New(id)
		if err != nil || parsed.String() != id {
			return "", fmt.Errorf("use full league player ids")
		}
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	sort.Strings(ids)
	return strings.Join(ids, ","), nil
}

func (r RawWorkflow) Validate() error {
	if r.Version != "1" {
		return fmt.Errorf("update the app before using this league action")
	}
	entry, ok := WorkflowCommands()[r.Command]
	if !ok {
		return fmt.Errorf("this league action is not available here")
	}
	return entry.Validate(r)
}

type workflowInput interface {
	Validate() error
	*RawWorkflowScope | *RawFilteredLedgerScope | *commissioner.RawCommand | *commissioner.DraftPickInput |
		*commissioner.RawNativeCapScope | *leagueservice.RawLeagueCalendarScope |
		*market.RawContractQuote | *market.RawWhatIfStack | *market.RawContractDecisionsScope | *leagueservice.RawDraftQueueScope |
		*leagueservice.RawDraftQueueSet | *leagueservice.RawPlayerNotesScope | *leagueservice.RawPlayerNoteSet |
		*leagueservice.RawPlayerWatchReadScope | *leagueservice.RawPlayerWatchSet | *leagueservice.RawPlayerWatchOrder | *leagueservice.RawMflMovesScope | *leagueservice.RawMflTeamSet | *leagueservice.RawMflPoolScope | *leagueservice.RawMflBidCheck | *leagueservice.RawMflLineupCheck | *leagueservice.RawMflPlanSave | *leagueservice.RawMflPlanScope |
		*leagueservice.RawTradeDraftScope | *leagueservice.RawTradeDraftSave | *leagueservice.RawTradeDraftDelete |
		*leagueservice.RawBackupNow | *accounts.RawHomeLayoutRead | *accounts.RawHomeLayoutSet |
		*commissioner.RawLeagueActivityScope | *commissioner.RawProblemReport | *commissioner.RawProblemReportsScope |
		*accounts.RawJobSnoozesRead | *accounts.RawJobSnoozeSet | *accounts.RawJobSnoozeWake
}

func decodeWorkflow[T workflowInput](body []byte, raw T) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(raw); err != nil {
		return fmt.Errorf("read league action: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("send one league action")
	}
	if err := raw.Validate(); err != nil {
		return fmt.Errorf("check league action: %w", err)
	}
	return nil
}

// League is the league a valid workflow names. Each command's own input has already been decoded
// strictly by Validate; every input carries the league under the same key.
func (r RawWorkflow) League() (string, error) {
	if err := r.Validate(); err != nil {
		return "", err
	}
	if IsAccountWorkflow(r.Command) {
		return "", nil
	}
	var scope struct {
		League string `json:"league"`
	}
	if err := json.Unmarshal(r.Arguments, &scope); err != nil {
		return "", fmt.Errorf("read league action: %w", err)
	}
	return scope.League, nil
}

type workflowOutput interface {
	commissioner.Ledger | commissioner.Jobs | commissioner.People | commissioner.ManagedSchedule |
		commissioner.LeagueNotes | commissioner.Calendar | history.NativeProjections | commissioner.NativeDirectory |
		commissioner.NativeWeek | commissioner.Schedule | commissioner.FranchiseSchedule | commissioner.NativeBracket |
		appvalue.Reading |
		market.NativeMoves | market.Market | market.NativeBids | market.NativeBidWins | market.NativeWaivers |
		market.NativeTrades | commissioner.NativeResults | commissioner.NativeStandings |
		commissioner.NativeOdds | commissioner.Corrections | commissioner.NativeAssets |
		commissioner.NativeLineup | commissioner.NativeDraft | commissioner.NativeReserveCheck |
		commissioner.NativeRosterCheck | commissioner.Preview | commissioner.Receipt | commissioner.Rules |
		leagueservice.NFLGames | leagueservice.NFLGame | leagueservice.NFLGamePregame |
		leagueservice.NFLPlayerStatus |
		leagueservice.PlayerNewsReading | leagueservice.NewsFeedReading | commissioner.NativeCapCheck |
		market.ContractQuote | market.WhatIfStack | market.ContractDecisions | leagueservice.DraftQueue |
		leagueservice.PlayerNotes | leagueservice.PlayerWatchList | leagueservice.PlayerPrivateChange | leagueservice.MflRefresh | leagueservice.MflTeams | leagueservice.MflPool | leagueservice.MflBidCheck | leagueservice.MflLineupCheck | leagueservice.MflPlan | leagueservice.MflPlans | leagueservice.MflPlanLink |
		leagueservice.TradeDraftList | leagueservice.SavedTradeDraft | leagueservice.TradeDraftDeleted |
		leagueservice.BackupResult | accounts.HomeLayoutState | commissioner.LeagueActivity |
		commissioner.ProblemReportReceipt | commissioner.ProblemReportsReading |
		accounts.JobSnoozesState
}

func workflowJSON[T workflowOutput](value T, err error) (json.RawMessage, error) {
	if err != nil {
		return nil, fmt.Errorf("league action: %w", err)
	}
	body, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode league action: %w", err)
	}
	return body, nil
}

func RunWorkflow(ctx context.Context, service *leagueservice.Service, raw RawWorkflow) (json.RawMessage, error) {
	if err := raw.Validate(); err != nil {
		return nil, err
	}
	if IsAccountWorkflow(raw.Command) {
		return nil, fmt.Errorf("this preference requires a signed-in account")
	}
	return WorkflowCommands()[raw.Command].Run(ctx, service, raw)
}

func validateWorkflowPlayer(command, player string) error {
	switch command {
	case "player.usage.read", "player.agearc.read", leagueservice.PlayerNewsRead:
		id, err := playerid.New(player)
		if err != nil || id.String() != player {
			return fmt.Errorf("use a full league player id")
		}
	}
	return nil
}
