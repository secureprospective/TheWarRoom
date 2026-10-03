package normalize

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/rosters"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// Player turns one raw roster row into a domain.PlayerRecord, joined against the players
// Lookup. A row naming an unknown or aggregate player fails rather than producing a half-typed
// record.
func Player(raw rosters.RawRoster, lookup Lookup) (domain.PlayerRecord, error) {
	rec, err := Contract(raw)
	if err != nil {
		return domain.PlayerRecord{}, err
	}
	entry, ok := lookup.entry(rec.MFLID)
	if !ok {
		return domain.PlayerRecord{}, fmt.Errorf("normalize: roster player %s (franchise %s) not found in players database", rec.MFLID, raw.FranchiseID)
	}
	if entry.isAggregate {
		return domain.PlayerRecord{}, fmt.Errorf("normalize: roster %s/%s references a team-aggregate player — should have been filtered at ingestion", raw.FranchiseID, rec.MFLID)
	}
	rec.Name, rec.Position, rec.NFLTeam, rec.IsRookie = entry.Name, entry.Position, entry.team, entry.IsRookie
	return rec, nil
}

// Contract types one raw roster row's franchise, roster status and contract, without the
// players directory: who the player is does not change what MFL says he is paid. The id is
// re-derived through playerid.New.
func Contract(raw rosters.RawRoster) (domain.PlayerRecord, error) {
	id, err := playerid.New(raw.PlayerID)
	if err != nil {
		return domain.PlayerRecord{}, fmt.Errorf("normalize: roster player id %q: %w", raw.PlayerID, err)
	}
	salary, err := parseSalary(raw.Salary, id)
	if err != nil {
		return domain.PlayerRecord{}, err
	}
	year, err := parseContractYear(raw.ContractYear, id)
	if err != nil {
		return domain.PlayerRecord{}, err
	}
	status, err := normalizeRosterStatus(raw.Status, id)
	if err != nil {
		return domain.PlayerRecord{}, err
	}
	return domain.PlayerRecord{
		MFLID:          id,
		Salary:         salary,
		ContractYear:   year,
		ContractStatus: normalizeContractStatus(raw.ContractStatus),
		ContractInfo:   raw.ContractInfo,
		RosterStatus:   status,
		FranchiseID:    raw.FranchiseID,
	}, nil
}

// Rosters normalizes a whole rosters feed into per-franchise records in franchise order.
// One bad row fails the batch: a partly normalized league is worse than an error.
func Rosters(raws []rosters.RawRoster, lookup Lookup) ([]domain.Roster, error) {
	byFranchise := make(map[string][]domain.PlayerRecord)
	order := make([]string, 0)

	for _, raw := range raws {
		pr, err := Player(raw, lookup)
		if err != nil {
			return nil, err
		}
		if _, seen := byFranchise[pr.FranchiseID]; !seen {
			order = append(order, pr.FranchiseID)
		}
		byFranchise[pr.FranchiseID] = append(byFranchise[pr.FranchiseID], pr)
	}

	sort.Strings(order)
	out := make([]domain.Roster, 0, len(order))
	for _, fid := range order {
		recs := byFranchise[fid]
		// MFL's array order varies between requests; sorting by id keeps downstream writes from
		// showing false diffs.
		sort.Slice(recs, func(i, j int) bool {
			return recs[i].MFLID.String() < recs[j].MFLID.String()
		})
		out = append(out, domain.Roster{FranchiseID: fid, Players: recs})
	}
	return out, nil
}

// parseSalary converts salary in millions straight to cents, with no float in between
// (OQ-014). Empty is a real $0; non-numeric is schema drift.
func parseSalary(raw string, id playerid.PlayerID) (domain.Money, error) {
	m, err := domain.ParseMoneyMillions(raw)
	if err != nil {
		return 0, fmt.Errorf("normalize: player %s salary %q: %w", id, raw, err)
	}
	return m, nil
}

// parseContractYear parses the contract year; empty is 0.
func parseContractYear(raw string, id playerid.PlayerID) (int, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, nil
	}
	y, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("normalize: player %s contract year %q: %w", id, raw, err)
	}
	return y, nil
}

// normalizeContractStatus maps MFL's dirty contractStatus onto the enum
// (docs/data-layer/MFL_API_Reference.md): trim, treat "YFA" as a typo for UFA, then
// prefix-match UFA/RFA/FT1/FT2 so variants like "FT1+ EXT2 (2024)" map. Anything else (a lone
// "EXT2") becomes CStatusFlag for review. The tests pin every known live value.
func normalizeContractStatus(raw string) domain.ContractStatus {
	s := strings.TrimSpace(raw)
	switch {
	case strings.HasPrefix(s, "YFA"):
		return domain.CStatusUFA
	case strings.HasPrefix(s, "UFA"):
		return domain.CStatusUFA
	case strings.HasPrefix(s, "RFA"):
		return domain.CStatusRFA
	case strings.HasPrefix(s, "FT1"):
		return domain.CStatusFT1
	case strings.HasPrefix(s, "FT2"):
		return domain.CStatusFT2
	default:
		return domain.CStatusFlag
	}
}

// normalizeRosterStatus maps "ROSTER", "TAXI_SQUAD" and "INJURED_RESERVE" (or the older "IR");
// anything else fails.
func normalizeRosterStatus(raw string, id playerid.PlayerID) (domain.RosterStatus, error) {
	switch strings.TrimSpace(raw) {
	case "ROSTER":
		return domain.RosterActive, nil
	case "TAXI_SQUAD":
		return domain.RosterTaxi, nil
	case "INJURED_RESERVE", "IR": // MFL's live export says INJURED_RESERVE (2026-10-03)
		return domain.RosterIR, nil
	default:
		return "", fmt.Errorf("normalize: player %s unknown roster status %q", id, raw)
	}
}
