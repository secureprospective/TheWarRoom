package leagueservice

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// RawPlayerReadScope selects one held player fact from a league's public readings.
type RawPlayerReadScope struct {
	League string `json:"league"`
	Player string `json:"player"`
}

func (r RawPlayerReadScope) Validate() error {
	if err := (RawLeagueScheduleScope{League: r.League}).Validate(); err != nil {
		return fmt.Errorf("player reading scope: %w", err)
	}
	id, err := playerid.New(r.Player)
	if err != nil || id.String() != r.Player {
		return fmt.Errorf("use a full league player id")
	}
	return nil
}

type playerReadHost interface {
	NativePlayerUsage(context.Context, string, string) (json.RawMessage, error)
	NativePlayerAgeArc(context.Context, string, string) (json.RawMessage, error)
}

func (a *Service) TargetNativePlayerUsage(scope RawPlayerReadScope) (json.RawMessage, error) {
	return a.readPlayer(scope, "player.usage.read", "player usage", playerReadHost.NativePlayerUsage)
}

func (a *Service) TargetNativePlayerAgeArc(scope RawPlayerReadScope) (json.RawMessage, error) {
	return a.readPlayer(scope, "player.agearc.read", "player age arc", playerReadHost.NativePlayerAgeArc)
}

func (a *Service) readPlayer(scope RawPlayerReadScope, operation, label string,
	read func(playerReadHost, context.Context, string, string) (json.RawMessage, error)) (json.RawMessage, error) {
	if err := scope.Validate(); err != nil {
		return nil, fmt.Errorf("%s input: %w", label, err)
	}
	if err := a.admitReading(commissioner.Access{League: scope.League, Operation: operation, Mode: "read", Player: scope.Player}); err != nil {
		return nil, err
	}
	if err := a.host.Ready(); err != nil {
		return nil, fmt.Errorf("%s startup: %w", label, err)
	}
	host, ok := a.host.(playerReadHost)
	if !ok {
		return json.RawMessage(`{"unavailable":true,"reason":"input-not-held"}`), nil
	}
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	reading, err := read(host, ctx, scope.League, scope.Player)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	if !json.Valid(reading) {
		return nil, fmt.Errorf("%s: invalid hosted reading", label)
	}
	return reading, nil
}
