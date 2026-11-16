package leagueservice

import (
	"context"
	"errors"
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

// ValueReader reads a league's player values: league, franchise (empty for every franchise), season.
type ValueReader interface {
	NativeValues(context.Context, string, string, int) (appvalue.Reading, error)
}

type valueHost = ValueReader

// errNoValues is the refusal of a host that holds no player values: the desktop before its history
// opens, or a server that does not collect value inputs.
var errNoValues = errors.New("player records and fitted parameters are not available on this host")

// nativeValues is a wrapping host's value read: the wrapped host's, or errNoValues.
func nativeValues(ctx context.Context, inner Host, league, franchise string, season int) (appvalue.Reading, error) {
	host, ok := inner.(valueHost)
	if !ok {
		return appvalue.Reading{}, errNoValues
	}
	return host.NativeValues(ctx, league, franchise, season) //nolint:wrapcheck // the service wraps it once
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
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	reading, err := nativeValues(ctx, a.host, raw.League, raw.Franchise, raw.Season)
	if err != nil {
		return appvalue.Reading{}, fmt.Errorf("league values: %w", err)
	}
	return reading, nil
}
