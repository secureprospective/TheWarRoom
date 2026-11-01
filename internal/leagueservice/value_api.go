package leagueservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/appvalue"
)

// RawValueScope selects a league and optionally one franchise's header/roster.
type RawValueScope struct {
	League    string `json:"league"`
	Franchise string `json:"franchise"`
	Season    int    `json:"season"`
}

func (r RawValueScope) Validate() error {
	if !strings.HasPrefix(r.League, "native.") || len(r.League) <= 7 || len(r.League) > 128 || strings.TrimSpace(r.League) != r.League || len(r.Franchise) > 128 || strings.TrimSpace(r.Franchise) != r.Franchise || r.Season < 1920 || r.Season > 9999 {
		return fmt.Errorf("choose the league, franchise and NFL season")
	}
	return nil
}

type valueHost interface {
	NativeValues(context.Context, string, string, int) (appvalue.Reading, error)
}

// TargetNativeValues is a held, read-only evaluation under this league's own settings.
func (a *Service) TargetNativeValues(raw RawValueScope) (appvalue.Reading, error) {
	if err := raw.Validate(); err != nil {
		return appvalue.Reading{}, fmt.Errorf("values input: %w", err)
	}
	if err := a.admitRead(raw.League, "value.read", raw.Franchise); err != nil {
		return appvalue.Reading{}, err
	}
	if err := a.host.Ready(); err != nil {
		return appvalue.Reading{}, fmt.Errorf("values startup: %w", err)
	}
	host, ok := a.host.(valueHost)
	if !ok {
		return appvalue.Reading{}, fmt.Errorf("league values: player records and fitted parameters are not available on this host")
	}
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	reading, err := host.NativeValues(ctx, raw.League, raw.Franchise, raw.Season)
	if err != nil {
		return appvalue.Reading{}, fmt.Errorf("league values: %w", err)
	}
	return reading, nil
}
