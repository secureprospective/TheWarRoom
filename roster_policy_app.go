package main

import (
	"context"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

// rosterPolicyAdapter composes the rulebook's override-aware league settings + per-position
// roster limits with the players-DB Lookup into the transactions.RosterPolicy port — Session 2's
// roster/position/taxi/IR enforcement gate. It lives in the composition root (app.go's package)
// because depguard forbids the transactions package from importing either store (the three-layer
// law); the adapter is the hexagonal seam that satisfies the port without a layering violation.
//
// Each limit reads the rulebook's GetSetting (override-aware — the Session 0 "taxi/IR off" pattern
// and the commissioner's rosterSize/taxiSquad/injuredReserve overrides apply uniformly). A value
// of "0" or an unparseable string reads as 0, which the enforcement treats as "unlimited / do not
// gate on this axis" (matching Session 0's "0 = off" convention). Position resolution rides the
// cached players-DB Lookup (built lazily, shared with every other players-DB consumer).
type rosterPolicyAdapter struct {
	rb  *rulebook.Store
	app *App
}

// Compile-time assertion that rosterPolicyAdapter satisfies transactions.RosterPolicy.
var _ transactions.RosterPolicy = (*rosterPolicyAdapter)(nil)

// RosterSize returns the per-franchise total roster cap (override-aware). 0 = unlimited.
func (p *rosterPolicyAdapter) RosterSize() int { return settingInt(p.rb, "rosterSize") }

// TaxiSquad returns the per-franchise taxi-squad slot cap (override-aware). 0 = unlimited / off.
func (p *rosterPolicyAdapter) TaxiSquad() int { return settingInt(p.rb, "taxiSquad") }

// InjuredReserve returns the per-franchise IR slot cap (override-aware). 0 = unlimited / off.
func (p *rosterPolicyAdapter) InjuredReserve() int { return settingInt(p.rb, "injuredReserve") }

// PositionLimit returns the inclusive per-position roster max for one engine position, parsed from
// the league's rosterLimits (MFL "min-max" format, "0-0" = unlimited). Each rosterLimits entry's
// MFL code is translated to the engine set through normalize.PositionFromMFL (the single source of
// truth for the MFL→engine map — PK→K and EDGE→DE remaps honored). 0 = unlimited / unconfigured.
func (p *rosterPolicyAdapter) PositionLimit(pos domain.Position) int {
	cfg := p.rb.ActiveConfig()
	for _, pl := range cfg.RosterLimits {
		enginePos, ok := normalize.PositionFromMFL(pl.Name)
		if !ok || enginePos != pos {
			continue // an unrecognized MFL code must never zero-value-collide with an unresolved player position
		}
		if m, ok := parseRosterLimitMax(pl.Limit); ok {
			return m
		}
	}
	return 0 // no rosterLimits entry for this position, or unparseable — unlimited
}

// Position resolves a player id to its engine position via the cached players-DB Lookup (the same
// Lookup every other players-DB consumer shares). ok=false for an unknown player (the enforcement
// then skips the per-position check for that id — it cannot reject on a limit it cannot resolve).
func (p *rosterPolicyAdapter) Position(ctx context.Context, mflID string) (domain.Position, bool) {
	lk, err := p.app.directory(ctx)
	if err != nil {
		return domain.PosFlag, false
	}
	facts, ok := lk.Facts(mflID)
	if !ok {
		return domain.PosFlag, false
	}
	return facts.Position, true
}

// settingInt reads one override-aware scalar rulebook setting and parses it to an int. An empty,
// unset, or unparseable value returns 0 (the "unlimited / do not gate" sentinel the enforcement
// uses), matching Session 0's "0 = off" convention for taxi/IR slot counts.
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

// parseRosterLimitMax parses MFL's rosterLimits "min-max" format (e.g. "1-4", "0-0") and returns
// the max. "0-0" → (0, true) which the enforcement reads as "unlimited". A single integer ("4",
// no dash) is treated as a max-only form. ok=false for an unparseable value — INCLUDING a
// malformed multi-dash string ("1-2-3", a bare "-4"): exactly one dash is the only shape MFL
// emits, so anything else is unparseable data, not a value to guess at (a review flagged that
// LastIndex previously accepted these silently, extracting an arbitrary segment as if it were
// legitimate).
func parseRosterLimitMax(limit string) (int, bool) {
	limit = strings.TrimSpace(limit)
	if limit == "" {
		return 0, false
	}
	if strings.Count(limit, "-") > 1 {
		return 0, false // malformed — MFL's format has at most one dash
	}
	// MFL's format is "min-max"; the max is the part after the single dash.
	if i := strings.Index(limit, "-"); i >= 0 {
		if i == 0 {
			return 0, false // a bare "-N" is not "min-max" — unparseable, not a negative min
		}
		m := strings.TrimSpace(limit[i+1:])
		n, err := strconv.Atoi(m)
		if err != nil || n < 0 {
			return 0, false
		}
		return n, true
	}
	// a bare integer — treat as max-only
	n, err := strconv.Atoi(limit)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}
