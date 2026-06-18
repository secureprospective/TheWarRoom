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

// Player normalizes one raw rosters row into a typed domain.PlayerRecord, joining
// it against the players Lookup for name/position/team/rookie. It is the single
// place a roster row becomes a typed record: id is re-derived through playerid.New
// (RISK-003 site #2), salary string→float, contractStatus dirty→enum, contractYear
// string→int. A row referencing an unknown or aggregate player fails loud rather
// than producing a half-typed record.
func Player(raw rosters.RawRoster, lookup Lookup) (domain.PlayerRecord, error) {
	id, err := playerid.New(raw.PlayerID)
	if err != nil {
		return domain.PlayerRecord{}, fmt.Errorf("normalize: roster player id %q: %w", raw.PlayerID, err)
	}
	entry, ok := lookup.entry(id)
	if !ok {
		return domain.PlayerRecord{}, fmt.Errorf("normalize: roster player %s (franchise %s) not found in players database", id, raw.FranchiseID)
	}
	if entry.isAggregate {
		return domain.PlayerRecord{}, fmt.Errorf("normalize: roster %s/%s references a team-aggregate player — should have been filtered at ingestion", raw.FranchiseID, id)
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
		Name:           entry.name,
		Position:       entry.position,
		NFLTeam:        entry.team,
		Salary:         salary,
		ContractYear:   year,
		ContractStatus: normalizeContractStatus(raw.ContractStatus),
		ContractInfo:   raw.ContractInfo,
		RosterStatus:   status,
		IsRookie:       entry.isRookie,
		FranchiseID:    raw.FranchiseID,
	}, nil
}

// Rosters normalizes a full rosters feed into per-franchise domain.Roster records,
// grouped by franchise id and returned in deterministic franchise order. One bad
// row fails the whole batch (fail-loud, B2 #3): a partially-normalized league is
// worse than a surfaced error.
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
		// Deterministic intra-roster order: MFL array order is unstable across
		// requests, which would create false diffs in downstream WAL writes
		// (B3 review). Sort by canonical id — any stable total order suffices.
		sort.Slice(recs, func(i, j int) bool {
			return recs[i].MFLID.String() < recs[j].MFLID.String()
		})
		out = append(out, domain.Roster{FranchiseID: fid, Players: recs})
	}
	return out, nil
}

// parseSalary converts the raw salary string (millions) to a float. An empty
// salary is a legitimate $0 (e.g. an unsigned slot), not an error; a non-empty
// non-numeric value is schema drift and fails loud.
func parseSalary(raw string, id playerid.PlayerID) (float64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("normalize: player %s salary %q: %w", id, raw, err)
	}
	return f, nil
}

// parseContractYear converts the raw contract-year string to an int. Empty is 0
// (no contract year on record); a non-numeric value fails loud.
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

// normalizeContractStatus cleans MFL's dirty contractStatus onto the domain enum.
// Rule (docs/data-layer/MFL_API_Reference.md): trim, then "YFA" is a confirmed typo
// for UFA, then a prefix match onto UFA/RFA/FT1/FT2 absorbs every parenthetical and
// combined variant ("FT1 (2026)", "FT1+ EXT2 (2024)"); anything else (e.g. a lone
// "EXT2") becomes CStatusFlag for admin review. YFA is matched by PREFIX too (B3
// review), so a suffixed "YFA (2024)" — if MFL ever emits one — still maps to UFA.
// The full table is locked in the normalize tests so every known live value's
// mapping is visible.
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

// normalizeRosterStatus maps MFL's roster status to the domain enum. MFL sends
// exactly "ROSTER" or "TAXI_SQUAD"; anything else fails loud rather than defaulting.
func normalizeRosterStatus(raw string, id playerid.PlayerID) (domain.RosterStatus, error) {
	switch strings.TrimSpace(raw) {
	case "ROSTER":
		return domain.RosterActive, nil
	case "TAXI_SQUAD":
		return domain.RosterTaxi, nil
	default:
		return "", fmt.Errorf("normalize: player %s unknown roster status %q", id, raw)
	}
}
