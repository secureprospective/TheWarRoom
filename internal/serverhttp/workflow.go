package serverhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

const WorkflowPath = "/v1/native/workflow"

// RawWorkflow is the catalog-generated client envelope. Publication here is deliberately bounded.
type RawWorkflow struct {
	Command   string          `json:"command"`
	Version   string          `json:"version"`
	Mode      string          `json:"mode"`
	Arguments json.RawMessage `json:"arguments"`
	Football  string          `json:"football,omitempty"`
}

type RawWorkflowScope struct {
	League    string `json:"league"`
	Franchise string `json:"franchise,omitempty"`
	Week      int    `json:"week,omitempty"`
	Season    int    `json:"season,omitempty"`
	At        string `json:"at,omitempty"`
}

func (r RawWorkflowScope) Validate() error {
	if !strings.HasPrefix(r.League, "native.") || len(r.League) <= 7 || len(r.League) > 128 || r.Week < 0 || r.Week > 22 || r.Season < 0 || r.Season > 9999 {
		return fmt.Errorf("choose your league, franchise and week")
	}
	return nil
}

func (r RawWorkflowScope) validateCommand(command string) error {
	switch command {
	case "lineup.read":
		if r.Franchise == "" || r.Week < 1 || r.Season != 0 {
			return fmt.Errorf("choose your franchise and week")
		}
	case "draft.read":
		if r.Franchise == "" || r.Week != 0 {
			return fmt.Errorf("choose your franchise and draft season")
		}
	default:
		if r.Franchise+r.At != "" || r.Week != 0 || r.Season != 0 {
			return fmt.Errorf("choose only your league")
		}
	}
	return nil
}

func (r RawWorkflow) Validate() error {
	if r.Version != "1" {
		return fmt.Errorf("update the app before using this league action")
	}
	switch r.Command {
	case "lineup.read", "draft.read", "native.assets.read", "native.reserve.check", "native.roster.check", "commish.rules.read":
		if r.Mode != "read" {
			return fmt.Errorf("this is a reading only")
		}
		var scope RawWorkflowScope
		if err := decodeWorkflow(r.Arguments, &scope); err != nil {
			return err
		}
		return scope.validateCommand(r.Command)
	case commissioner.LineupSet, commissioner.RosterReserve, commissioner.DraftTimer, commissioner.DraftSkip:
		if r.Mode != "preview" && r.Mode != "execute" {
			return fmt.Errorf("check or confirm the change")
		}
		var raw commissioner.RawCommand
		if err := decodeWorkflow(r.Arguments, &raw); err != nil {
			return err
		}
		if raw.Command != r.Command {
			return fmt.Errorf("choose one league action")
		}
		return nil
	case commissioner.DraftPick:
		if r.Mode != "execute" {
			return fmt.Errorf("confirm your rookie pick")
		}
		var pick commissioner.DraftPickInput
		return decodeWorkflow(r.Arguments, &pick)
	default:
		return fmt.Errorf("this league action is not available here")
	}
}

type workflowInput interface {
	Validate() error
	*RawWorkflowScope | *commissioner.RawCommand | *commissioner.DraftPickInput
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

// WorkflowResponse carries the authority's own wording; a cached writable bit is never a grant.
type WorkflowResponse struct {
	Label    string          `json:"label"`
	Writable bool            `json:"writable"`
	Football string          `json:"football"`
	Data     json.RawMessage `json:"data"`
}

func (r WorkflowResponse) Validate() error {
	if r.Label == "" || !json.Valid(r.Data) {
		return fmt.Errorf("unreadable league response")
	}
	return nil
}

func (h Hosted) workflow(w http.ResponseWriter, r *http.Request, service *leagueservice.Service, raw RawWorkflow) {
	league, err := raw.League()
	if err != nil {
		http.Error(w, "invalid league action", http.StatusBadRequest)
		return
	}
	//nolint:contextcheck // The service host carries this request's context and admission.
	location, err := service.Location(league)
	if err != nil {
		http.Error(w, "league access refused; sign in and check your seat", http.StatusForbidden)
		return
	}
	if raw.Mode != "read" && h.FootballDigest == "" {
		http.Error(w, "Ask the server operator to select football evidence before making changes", http.StatusConflict)
		return
	}
	if raw.Mode != "read" && !location.Writable {
		http.Error(w, location.Label, http.StatusConflict)
		return
	}
	ctx := leagueservice.WithFootballDigest(r.Context(), raw.Football)
	data, err := runWorkflow(ctx, service, raw)
	if err != nil {
		http.Error(w, "The league could not accept this action. Refresh before retrying; check your seat, league rules and authenticator.", http.StatusConflict)
		return
	}
	writeHosted(w, WorkflowResponse{Label: location.Label, Writable: location.Writable, Football: h.FootballDigest, Data: data})
}

func (r RawWorkflow) League() (string, error) {
	switch r.Command {
	case commissioner.LineupSet, commissioner.RosterReserve, commissioner.DraftTimer, commissioner.DraftSkip:
		var raw commissioner.RawCommand
		if err := decodeWorkflow(r.Arguments, &raw); err != nil {
			return "", err
		}
		return raw.League, nil
	case commissioner.DraftPick:
		var raw commissioner.DraftPickInput
		if err := decodeWorkflow(r.Arguments, &raw); err != nil {
			return "", err
		}
		return raw.League, nil
	default:
		var raw RawWorkflowScope
		if err := decodeWorkflow(r.Arguments, &raw); err != nil {
			return "", err
		}
		return raw.League, nil
	}
}

type workflowOutput interface {
	commissioner.NativeAssets | commissioner.NativeLineup | commissioner.NativeDraft | commissioner.NativeReserveCheck | commissioner.NativeRosterCheck | commissioner.Preview | commissioner.Receipt | commissioner.Rules
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

func runWorkflow(ctx context.Context, service *leagueservice.Service, raw RawWorkflow) (json.RawMessage, error) {
	if raw.Command == commissioner.DraftPick {
		var pick commissioner.DraftPickInput
		if err := decodeWorkflow(raw.Arguments, &pick); err != nil {
			return nil, err
		}
		return workflowJSON(service.Pick(ctx, pick))
	}
	if raw.Mode != "read" {
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
	var scope RawWorkflowScope
	if err := decodeWorkflow(raw.Arguments, &scope); err != nil {
		return nil, err
	}
	//nolint:contextcheck // Read APIs use the request-scoped host.
	return readWorkflow(service, raw.Command, scope)
}

func readWorkflow(service *leagueservice.Service, command string, scope RawWorkflowScope) (json.RawMessage, error) {
	switch command {
	case "commish.rules.read":
		return workflowJSON(service.TargetCommissionerRules(scope.League))
	case "native.assets.read":
		return workflowJSON(service.TargetNativeAssets(scope.League))
	case "native.reserve.check":
		return workflowJSON(service.TargetNativeReserveCheck(scope.League))
	case "native.roster.check":
		return workflowJSON(service.TargetNativeRosterCheck(scope.League))
	case "lineup.read":
		return workflowJSON(service.TargetNativeLineup(scope.League, scope.Franchise, scope.Week, scope.At))
	case "draft.read":
		return workflowJSON(service.TargetNativeDraft(commissioner.RawNativeDraftScope{League: scope.League, Franchise: scope.Franchise, Season: scope.Season}))
	default:
		return nil, fmt.Errorf("unknown league reading")
	}
}
