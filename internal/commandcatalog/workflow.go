package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
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
	League    string `json:"league"`
	Franchise string `json:"franchise,omitempty"`
	Viewer    string `json:"viewer,omitempty"`
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
	kept, complete := r.commandScope(command)
	if r != kept || !complete {
		return fmt.Errorf("choose only the details of this league reading")
	}
	if r.At != "" {
		if _, err := time.Parse(time.RFC3339Nano, r.At); err != nil {
			return fmt.Errorf("choose when to read the league: %w", err)
		}
	}
	return nil
}

func (r RawWorkflowScope) commandScope(command string) (RawWorkflowScope, bool) {
	kept := RawWorkflowScope{League: r.League}
	switch command {
	case "commish.market.read":
		kept.Command = r.Command
		return kept, r.Command != ""
	case "lineup.read":
		kept.Franchise, kept.Week, kept.At = r.Franchise, r.Week, r.At
		return kept, r.Franchise != "" && r.Week > 0
	case "draft.read":
		kept.Franchise, kept.Season, kept.At = r.Franchise, r.Season, r.At
		return kept, r.Franchise != ""
	case "bids.read":
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
	case "standings.read":
		kept.Season = r.Season
		return kept, r.Season >= 1920
	default:
		return kept, true
	}
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

func (r RawWorkflow) League() (string, error) {
	if r.Command == commissioner.DraftPick {
		var raw commissioner.DraftPickInput
		if err := decodeWorkflow(r.Arguments, &raw); err != nil {
			return "", err
		}
		return raw.League, nil
	}
	if r.Mode != "read" {
		var raw commissioner.RawCommand
		if err := decodeWorkflow(r.Arguments, &raw); err != nil {
			return "", err
		}
		return raw.League, nil
	}
	var raw RawWorkflowScope
	if err := decodeWorkflow(r.Arguments, &raw); err != nil {
		return "", err
	}
	return raw.League, nil
}

type workflowOutput interface {
	market.Market | market.NativeBids | market.NativeWaivers | market.NativeTrades | commissioner.NativeResults | commissioner.NativeStandings | commissioner.Corrections | commissioner.NativeAssets | commissioner.NativeLineup | commissioner.NativeDraft | commissioner.NativeReserveCheck | commissioner.NativeRosterCheck | commissioner.Preview | commissioner.Receipt | commissioner.Rules
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
	return WorkflowCommands()[raw.Command].Run(ctx, service, raw)
}
