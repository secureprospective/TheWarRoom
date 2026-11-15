package leagueservice

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/tradelens"
)

// RawTradeLensScope is a proposed trade to read through both sides' lenses: what the franchise gives
// and gets, as league player ids or pick tokens (pick:season:round:original franchise). With no
// partner it values the franchise's own shelf.
type RawTradeLensScope struct {
	League    string   `json:"league"`
	Franchise string   `json:"franchise"`
	Partner   string   `json:"partner"`
	Give      []string `json:"give"`
	Get       []string `json:"get"`
	Season    int      `json:"season"`
}

func (r RawTradeLensScope) Validate() error {
	if !strings.HasPrefix(r.League, "native.") || len(r.League) <= 7 || len(r.League) > 128 ||
		r.Season < 1920 || r.Season > 9999 {
		return fmt.Errorf("choose the league and NFL season")
	}
	if r.Franchise == "" || len(r.Franchise) > 128 || len(r.Partner) > 128 || r.Franchise == r.Partner {
		return fmt.Errorf("name your franchise, and a different one to trade with")
	}
	if len(r.Give) > maxTradePlayers || len(r.Get) > maxTradePlayers {
		return fmt.Errorf("name at most %d players or picks on each side of the trade", maxTradePlayers)
	}
	seen := map[string]bool{}
	for _, token := range append(append([]string{}, r.Give...), r.Get...) {
		if seen[token] || !lensToken(token) {
			return fmt.Errorf("%q is not a league player id or pick, or is named twice", token)
		}
		seen[token] = true
	}
	return nil
}

func lensToken(token string) bool {
	if rest, ok := strings.CutPrefix(token, "pick:"); ok {
		parts := strings.SplitN(rest, ":", 3)
		if len(parts) != 3 || parts[2] == "" || len(token) > 160 {
			return false
		}
		season, err := strconv.Atoi(parts[0])
		round, err2 := strconv.Atoi(parts[1])
		return err == nil && err2 == nil && season >= 1920 && season <= 9999 && round >= 1 && round <= 99
	}
	pid, err := playerid.New(token)
	return err == nil && pid.String() == token
}

type lensHost interface {
	NativeTradeLens(context.Context, tradelens.Scope) (tradelens.Reading, error)
}

// TargetNativeTradeLens reads a proposed trade through each side's direction. It is a what-if for
// the asking franchise, computed per request and never recorded.
func (a *Service) TargetNativeTradeLens(raw RawTradeLensScope) (tradelens.Reading, error) {
	if err := raw.Validate(); err != nil {
		return tradelens.Reading{}, fmt.Errorf("trade lens input: %w", err)
	}
	if err := a.admitRead(raw.League, "tradelens.read", raw.Franchise); err != nil {
		return tradelens.Reading{}, err
	}
	if err := a.host.Ready(); err != nil {
		return tradelens.Reading{}, fmt.Errorf("trade lens startup: %w", err)
	}
	host, ok := a.host.(lensHost)
	if !ok {
		return tradelens.Reading{}, fmt.Errorf("trade lens: league values are not available on this host")
	}
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	reading, err := host.NativeTradeLens(ctx, tradelens.Scope{League: raw.League, Season: raw.Season,
		Franchise: raw.Franchise, Partner: raw.Partner, Give: raw.Give, Get: raw.Get})
	if err != nil {
		return tradelens.Reading{}, fmt.Errorf("trade lens: %w", err)
	}
	return reading, nil
}
