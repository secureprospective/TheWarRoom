package leagueservice

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// maxTradePlayers bounds one side of a proposed trade.
const maxTradePlayers = 20

// RawLineupDeltaScope is a proposed trade to judge against both parties' best lineups: what the
// franchise gives and what it gets from the partner, for a season and its coming week.
type RawLineupDeltaScope struct {
	League    string   `json:"league"`
	Franchise string   `json:"franchise"`
	Partner   string   `json:"partner"`
	Give      []string `json:"give"`
	Get       []string `json:"get"`
	Season    int      `json:"season"`
	Week      int      `json:"week"`
}

func (r RawLineupDeltaScope) Validate() error {
	if r.League == "" || r.Franchise == "" || r.Partner == "" || r.Franchise == r.Partner {
		return fmt.Errorf("name the league and the two franchises in the trade")
	}
	if len(r.Give)+len(r.Get) == 0 || len(r.Give) > maxTradePlayers || len(r.Get) > maxTradePlayers {
		return fmt.Errorf("name 1 to %d players on each side of the trade", maxTradePlayers)
	}
	seen := map[string]bool{}
	for _, id := range append(append([]string{}, r.Give...), r.Get...) {
		pid, err := playerid.New(id)
		if err != nil || pid.String() != id || seen[id] {
			return fmt.Errorf("player %q is not a league player id, or is named twice", id)
		}
		seen[id] = true
	}
	return nil
}

// TargetNativeLineupDelta reads what a proposed trade does to both parties' best lineups (D33).
// It is a what-if: the asking franchise's own view, computed per request and never recorded.
func (a *Service) TargetNativeLineupDelta(raw RawLineupDeltaScope) (history.NativeLineupDelta, error) {
	if err := a.admitRead(raw.League, "lineupdelta.read", raw.Franchise); err != nil {
		return history.NativeLineupDelta{}, err
	}
	if err := raw.Validate(); err != nil {
		return history.NativeLineupDelta{}, fmt.Errorf("lineup delta input: %w", err)
	}
	if err := a.host.Ready(); err != nil {
		return history.NativeLineupDelta{}, fmt.Errorf("lineup delta startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	reading, err := a.host.NativeLineupDelta(ctx, raw.League, raw.Season, raw.Week, history.TradeSides{
		Franchise: raw.Franchise, Partner: raw.Partner, Give: raw.Give, Get: raw.Get})
	if err != nil {
		return history.NativeLineupDelta{}, fmt.Errorf("lineup delta: %w", err)
	}
	return reading, nil
}
