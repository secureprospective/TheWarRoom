package rookiedraft

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// PlayerContract is recorded native ownership and contract data, not an imported owner roster.
type PlayerContract struct {
	ID        string
	Franchise string
	Salary    domain.Money
	Status    domain.RosterStatus
}

// Candidate carries explicit rookie eligibility from the native player directory.
// An absent entry year cannot turn a veteran or unknown player into a draftable rookie.
type Candidate struct {
	ID          string
	EntrySeason int
}

type SelectionBasis struct {
	// Pool is who this draft may take; the zero value is the season's rookie class.
	Pool       Pool
	Exceptions []nativecheck.DecisionException
	Season     int
	Players    []PlayerContract
	Charges    []domain.Money
	Facts      nativecheck.Facts
	Local      *time.Location
}

type SelectionPreview struct {
	Terms
	CapWords string `json:"capWords"`
}

// PreviewSelection refuses an illegal selection without changing the draft or any caller slice.
// Charges must be this owner's season-scoped cap adjustments, supplied by the ledger adapter.
// This pure check does not authorize an owner; the native gate remains the writer's responsibility.
func PreviewSelection(r leaguerules.RuleSet, d Draft, owner string, candidate Candidate,
	basis SelectionBasis) (SelectionPreview, error) {
	if d.OnClock == nil || d.OnClock.Owner != owner {
		return SelectionPreview{}, fmt.Errorf("it is not your turn to pick")
	}
	if err := admits(basis.Pool, basis.Season, candidate); err != nil {
		return SelectionPreview{}, err
	}
	if slices.ContainsFunc(basis.Players, func(p PlayerContract) bool { return p.ID == candidate.ID }) ||
		slices.ContainsFunc(d.Picks, func(p Result) bool { return p.Player == candidate.ID }) {
		return SelectionPreview{}, fmt.Errorf("this player already belongs to a team")
	}
	terms, err := SlotTerms(r.Draft, d.OnClock.Round, d.OnClock.Slot)
	if err != nil {
		return SelectionPreview{}, err
	}
	roster, capPlayers := ownerPlayers(owner, basis.Players)
	before := nativecheck.CheckRoster(r.Roster, owner, roster, basis.Facts, basis.Exceptions...)
	roster = append(roster, nativecheck.RosterPlayer{ID: candidate.ID, Status: domain.RosterActive})
	after := nativecheck.CheckRoster(r.Roster, owner, roster, basis.Facts, basis.Exceptions...)
	if reason, over := rosterOverage(r.Roster, before, after, basis.Exceptions...); over {
		return SelectionPreview{}, fmt.Errorf("this pick would break your roster limits: %s", reason)
	}
	capPlayers = append(capPlayers, nativecheck.CapPlayer{Salary: terms.Salary, Status: domain.RosterActive})
	capState, err := nativecheck.CheckCap(r.Cap, capPlayers, basis.Charges, basis.Local, basis.Exceptions...)
	if err != nil {
		return SelectionPreview{}, fmt.Errorf("check rookie contract against the salary cap: %w", err)
	}
	allowed, words := nativecheck.CapAllows(capState, 0, basis.Local)
	if !allowed {
		return SelectionPreview{}, fmt.Errorf("this pick cannot fit under your salary cap: %s", words)
	}
	return SelectionPreview{Terms: terms, CapWords: words}, nil
}

func ownerPlayers(owner string, players []PlayerContract) ([]nativecheck.RosterPlayer, []nativecheck.CapPlayer) {
	roster := []nativecheck.RosterPlayer{}
	capPlayers := []nativecheck.CapPlayer{}
	for _, player := range players {
		if player.Franchise != owner {
			continue
		}
		roster = append(roster, nativecheck.RosterPlayer{ID: player.ID, Status: player.Status})
		capPlayers = append(capPlayers, nativecheck.CapPlayer{Salary: player.Salary, Status: player.Status})
	}
	return roster, capPlayers
}

// rosterOverage says whether adding the rookie leaves the roster above a limit, in words. A roster still
// short of its minimum is not refused: a pick can only help it.
func rosterOverage(rules leaguerules.RosterRules, before, after nativecheck.RosterVerdict, exceptions ...nativecheck.DecisionException) (string, bool) {
	_, waived := nativecheck.ExceptionReason(exceptions, "roster.limits")
	if limit := rules.Limits; !waived && limit.State == leaguerules.IsKnown && limit.Value.Max.State == leaguerules.IsKnown &&
		after.Counted > limit.Value.Max.Value {
		return after.Words, true
	}
	for _, line := range after.Problems {
		if strings.HasSuffix(line, " over") && !slices.Contains(before.Problems, line) {
			return line, true
		}
	}
	return "", false
}
