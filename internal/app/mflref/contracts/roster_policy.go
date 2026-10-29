package contracts

import (
	"context"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

// RosterPolicy implements transactions.RosterPolicy from the rulebook's settings and the players
// directory. It lives here because the transactions package may not import either store. Every
// limit is override-aware, and 0 (or an unparseable value) means unlimited. Directory is the
// composition root's cached players lookup.
type RosterPolicy struct {
	Rulebook  *rulebook.Store
	Directory func(context.Context) (normalize.Lookup, error)
}

var _ transactions.RosterPolicy = (*RosterPolicy)(nil)

// RosterSize is the per-franchise roster cap; 0 means unlimited.
func (p *RosterPolicy) RosterSize() int { return settingInt(p.Rulebook, "rosterSize") }

// TaxiSquad is the taxi-squad slot cap; 0 means off.
func (p *RosterPolicy) TaxiSquad() int { return settingInt(p.Rulebook, "taxiSquad") }

// InjuredReserve is the IR slot cap; 0 means off.
func (p *RosterPolicy) InjuredReserve() int { return settingInt(p.Rulebook, "injuredReserve") }

// PositionLimit is the per-position roster max from the league's rosterLimits (MFL "min-max";
// "0-0" means unlimited). MFL codes map through normalize.PositionFromMFL. 0 means unlimited.
func (p *RosterPolicy) PositionLimit(pos domain.Position) int {
	cfg := p.Rulebook.ActiveConfig()
	for _, pl := range cfg.RosterLimits {
		enginePos, ok := normalize.PositionFromMFL(pl.Name)
		if !ok || enginePos != pos {
			continue
		}
		if m, ok := parseRosterLimitMax(pl.Limit); ok {
			return m
		}
	}
	return 0
}

// Position resolves a player's engine position from the shared directory. ok=false for an
// unknown player, whose position check is then skipped.
func (p *RosterPolicy) Position(ctx context.Context, mflID string) (domain.Position, bool) {
	lk, err := p.Directory(ctx)
	if err != nil {
		return domain.PosFlag, false
	}
	facts, ok := lk.Facts(mflID)
	if !ok {
		return domain.PosFlag, false
	}
	return facts.Position, true
}

// settingInt parses one rulebook setting; empty or unparseable reads as 0 (unlimited).
func settingInt(rb *rulebook.Store, key string) int {
	if rb == nil {
		return 0
	}
	v, ok := rb.GetSetting(key)
	if !ok || strings.TrimSpace(v) == "" {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0
	}
	if n < 0 {
		return 0
	}
	return n
}

// parseRosterLimitMax returns the max of MFL's "min-max" limit ("1-4", "0-0"). A bare integer
// is a max. Anything with more than one dash is unparseable: MFL never emits it.
func parseRosterLimitMax(limit string) (int, bool) {
	limit = strings.TrimSpace(limit)
	if limit == "" {
		return 0, false
	}
	if strings.Count(limit, "-") > 1 {
		return 0, false
	}
	if i := strings.Index(limit, "-"); i >= 0 {
		if i == 0 {
			return 0, false
		}
		m := strings.TrimSpace(limit[i+1:])
		n, err := strconv.Atoi(m)
		if err != nil || n < 0 {
			return 0, false
		}
		return n, true
	}
	n, err := strconv.Atoi(limit)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}
