package state

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// effectiveSalary is the dollar figure a player counts against the cap: the adjusted
// salary once B7 has set one, otherwise the base annual salary. Pure derivation —
// the cap USAGE, never stored as truth (AD-21).
func effectiveSalary(p PlayerState) float64 {
	if p.AdjustedSalary > 0 {
		return p.AdjustedSalary
	}
	return p.Salary
}

// seedPlayerCount totals the players across all seed rosters (the empty-seed guard).
func seedPlayerCount(rosters []domain.Roster) int {
	n := 0
	for _, r := range rosters {
		n += len(r.Players)
	}
	return n
}

// seedPlayer inserts one normalized player as a rosters row + a contracts row. Only
// normalize-provided fields are seeded; adjustment/years-remaining/tag flags stay
// zero until B7 mutates them.
func seedPlayer(ctx context.Context, tx *sql.Tx, leagueID string, season int, now, franchiseID string, p domain.PlayerRecord) error {
	mflID := p.MFLID.String()
	key := fmt.Sprintf("%s:%d:%s", leagueID, season, mflID)
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO rosters (id, league_id, mfl_id, franchise_id, roster_status, season, as_of)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"r:"+key, leagueID, mflID, franchiseID, string(p.RosterStatus), season, now); err != nil {
		return fmt.Errorf("state: seed roster %q: %w", mflID, err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO contracts (id, league_id, mfl_id, franchise_id, annual_salary,
		   adjusted_salary, contract_years, expiration_year, contract_status,
		   is_restructured, is_tagged, season, last_updated)
		 VALUES (?, ?, ?, ?, ?, 0, 0, ?, ?, 0, 0, ?, ?)`,
		"c:"+key, leagueID, mflID, franchiseID, p.Salary, p.ContractYear,
		string(p.ContractStatus), season, now); err != nil {
		return fmt.Errorf("state: seed contract %q: %w", mflID, err)
	}
	return nil
}

// scanState reads the rosters⋈contracts result set into the franchise map and the
// player→franchise index, computing each franchise's derived cap usage.
func scanState(rows *sql.Rows) (map[string]*FranchiseState, map[string]string, error) {
	fr := map[string]*FranchiseState{}
	idx := map[string]string{}
	for rows.Next() {
		var p PlayerState
		var restructured, tagged int
		if err := rows.Scan(&p.FranchiseID, &p.MFLID, &p.RosterStatus,
			&p.Salary, &p.AdjustedSalary, &p.ContractYears, &p.ExpirationYear,
			&p.ContractStatus, &restructured, &tagged); err != nil {
			return nil, nil, fmt.Errorf("state: scan: %w", err)
		}
		p.IsRestructured, p.IsTagged = restructured != 0, tagged != 0
		f, ok := fr[p.FranchiseID]
		if !ok {
			f = &FranchiseState{FranchiseID: p.FranchiseID}
			fr[p.FranchiseID] = f
		}
		f.Players = append(f.Players, p)
		f.CapUsed += effectiveSalary(p)
		idx[p.MFLID] = p.FranchiseID
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("state: iterate: %w", err)
	}
	return fr, idx, nil
}

// cloneFranchise deep-copies a franchise state so a reader never aliases store memory.
func cloneFranchise(fs *FranchiseState) FranchiseState {
	return FranchiseState{
		FranchiseID: fs.FranchiseID,
		Players:     clonePlayers(fs.Players),
		CapUsed:     fs.CapUsed,
	}
}

// clonePlayers copies a player slice. PlayerState holds no reference fields, so a
// value copy of each element is a full deep copy.
func clonePlayers(in []PlayerState) []PlayerState {
	if in == nil {
		return nil
	}
	out := make([]PlayerState, len(in))
	copy(out, in)
	return out
}

// sortedKeys returns the franchise map keys in deterministic order.
func sortedKeys(m map[string]*FranchiseState) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// validRosterStatus accepts only the normalized roster statuses.
func validRosterStatus(s domain.RosterStatus) bool {
	return s == domain.RosterActive || s == domain.RosterTaxi
}

// validContractStatus accepts the four real contract statuses (not the review FLAG).
func validContractStatus(s domain.ContractStatus) bool {
	switch s {
	case domain.CStatusUFA, domain.CStatusRFA, domain.CStatusFT1, domain.CStatusFT2:
		return true
	case domain.CStatusFlag:
		return false // the review sentinel is never a valid mutation target
	default:
		return false
	}
}

// requireOneRow asserts a mutation touched exactly its one target row. A 0-row
// update means memory and the DB disagree about the player (drift, an out-of-band
// edit, a wrong-league write) — fail loud rather than silently no-op, the same
// guarantee the empty-seed guard gives the seed path.
func requireOneRow(res sql.Result, mflID string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("state: rows affected for %q: %w", mflID, err)
	}
	if n != 1 {
		return fmt.Errorf("state: write for %q affected %d rows, want 1 (memory/DB drift)", mflID, n)
	}
	return nil
}

// boolToInt maps a bool to the 0/1 SQLite integer encoding.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
