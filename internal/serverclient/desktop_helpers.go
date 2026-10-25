package serverclient

import (
	"context"
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

// Helper levels a commissioner can choose. Both cover every delegable command for the franchise
// they pick; "record" adds execute, and bigger changes still wait for their confirmation.
var helperModes = map[string][]string{ //nolint:gochecknoglobals // fixed, closed choice
	"suggest": {"read", "preview"},
	"record":  {"read", "preview", "execute"},
}

// SetHelperCommands is the delegable command list the composition reads from the catalog.
func (d *Desktop) SetHelperCommands(commands []string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.helperCommands = slices.Clone(commands)
}

func (r RawDesktop) validateHelper() error {
	if r.Username+r.Password != "" {
		return fmt.Errorf("unexpected credentials")
	}
	// Each helper action carries only its own fields.
	switch r.Action {
	case "helper-issue":
		if _, ok := helperModes[r.Helper]; !ok || r.Franchise == "" || len(r.Franchise) > 128 || r.Code+r.Change != "" {
			return fmt.Errorf("choose what the AI helper may do, and which franchise it acts for")
		}
	case "helper-revoke":
		if (accounts.RawSession{Bearer: r.Code}).Validate() != nil || r.Helper+r.Franchise+r.Change != "" {
			return fmt.Errorf("enter the helper's ID shown when it was connected")
		}
	default:
		if _, _, err := DecodeConfirmRequest(r.Change); err != nil || r.Helper+r.Franchise+r.Code != "" {
			return fmt.Errorf("paste the whole confirm request from your AI helper; it starts with twrc1")
		}
	}
	return nil
}

func (d *Desktop) helper(ctx context.Context, raw RawDesktop, view DesktopView) (DesktopView, error) {
	if d.bearer == "" {
		return view, fmt.Errorf("sign in first")
	}
	if d.proof == "" {
		return view, fmt.Errorf("verify a fresh authenticator code first; AI helpers need recent verification")
	}
	switch raw.Action {
	case "helper-issue":
		if !slices.ContainsFunc(d.seats, func(seat DesktopSeat) bool { return seat.Franchise == raw.Franchise }) || len(d.helperCommands) == 0 {
			return view, fmt.Errorf("open your league access first, then choose a franchise in this league")
		}
		scope := accounts.RawTokenScope{League: d.league, Franchise: raw.Franchise, Commands: d.helperCommands, Modes: helperModes[raw.Helper]}
		body, err := postDesktop(ctx, d, serverhttp.AgentTokenPath, serverhttp.RawAgentTokenRequest{Protocol: 1, Scope: &scope})
		if err != nil {
			return view, err
		}
		var token rawDesktopAgentToken
		if err := decodeDesktop(body, &token); err != nil {
			return view, err
		}
		view.HelperID, view.Connection = token.ID, HelperLink(token.ID, token.Bearer)
		view.Message = "Copy this connection line into twr connect now. It is shown once; anyone with it can act as your helper until you revoke it."
	case "helper-revoke":
		if _, err := postDesktop(ctx, d, serverhttp.AgentTokenRevokePath, serverhttp.RawAgentTokenRequest{Protocol: 1, ID: raw.Code}); err != nil {
			return view, err
		}
		view.Message = "AI helper disconnected. Its connection line no longer works."
	default:
		id, change, _ := DecodeConfirmRequest(raw.Change)
		if change.League != d.league {
			return view, fmt.Errorf("this confirm request is for a different league")
		}
		body, err := postDesktop(ctx, d, serverhttp.ConfirmPath, serverhttp.RawNativeRequest{Protocol: 1, Command: &change, TokenID: id})
		if err != nil {
			return view, err
		}
		var response rawDesktopConfirmation
		if err := decodeDesktop(body, &response); err != nil {
			return view, err
		}
		view.Confirmation = response.Confirmation
		view.Message = "Give this code to your AI helper. It works for this exact change only, for five minutes."
	}
	return view, nil
}

type rawDesktopAgentToken serverhttp.AgentTokenResponse

func (r rawDesktopAgentToken) Validate() error {
	if r.Protocol != 1 || (accounts.RawSession{Bearer: r.ID}).Validate() != nil || (accounts.RawSession{Bearer: r.Bearer}).Validate() != nil {
		return fmt.Errorf("invalid helper connection response")
	}
	return nil
}

type rawDesktopConfirmation serverhttp.ConfirmationResponse

func (r rawDesktopConfirmation) Validate() error {
	if r.Confirmation == "" || len(r.Confirmation) > 128 {
		return fmt.Errorf("invalid confirmation response")
	}
	return nil
}
