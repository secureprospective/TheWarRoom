package state

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// rowQuerier is the read surface the cell-cap loader needs — satisfied by *sql.DB (the
// committed read pool used at load) and *sql.Tx (a future in-tx reader).
type rowQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// loadCellCap reads every current-season PAID ledger cell joined to its franchise, returning
// (a) each player's RAW cap-counting cents (PlayerState.CapSalary — the §8/§11 rule base) and
// (b) each franchise's cap usage = the sum of its cells SNAPPED to $10k (universal rounding).
// The ledger cell is the money source of truth (KING) as of Ship 3 — the legacy contract
// salary columns are frozen and unread. Exactly one PAID cell exists per rostered player for
// the current year (PK league_id, mfl_id, league_year), so a per-cell snap is a per-player
// snap; a player with no current PAID cell contributes 0 (an expired/UFA year).
func loadCellCap(ctx context.Context, q rowQuerier, leagueID string, season int) (perPlayer, perFranchise map[string]domain.Money, err error) {
	rows, err := q.QueryContext(ctx, `
SELECT r.franchise_id, cy.mfl_id, cy.salary_cents
FROM contract_years cy
JOIN rosters r ON r.league_id = cy.league_id AND r.mfl_id = cy.mfl_id AND r.season = ?
WHERE cy.league_id = ? AND cy.league_year = ? AND cy.year_status = ?`,
		season, leagueID, season, yearStatusPaid)
	if err != nil {
		return nil, nil, fmt.Errorf("state: cell cap read: %w", err)
	}
	defer func() { _ = rows.Close() }()
	perPlayer, perFranchise = map[string]domain.Money{}, map[string]domain.Money{}
	for rows.Next() {
		var fid, mflID string
		var cents int64
		if err := rows.Scan(&fid, &mflID, &cents); err != nil {
			return nil, nil, fmt.Errorf("state: cell cap scan: %w", err)
		}
		perPlayer[mflID] = domain.Money(cents)
		perFranchise[fid] += domain.RoundToNearest10k(domain.Money(cents))
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("state: cell cap iterate: %w", err)
	}
	return perPlayer, perFranchise, nil
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
	// Money is seeded into the exact-cents base column; the cap-counting figure lives in the
	// ledger cells (seedLedgerPlayer), never in a contracts money column. The legacy REAL
	// columns and adjusted_salary_cents were dropped in Ship 4.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO contracts (id, league_id, mfl_id, franchise_id, annual_salary_cents,
		   contract_years, expiration_year, contract_status,
		   is_restructured, is_tagged, season, last_updated)
		 VALUES (?, ?, ?, ?, ?, 0, ?, ?, 0, 0, ?, ?)`,
		"c:"+key, leagueID, mflID, franchiseID, p.Salary.Cents(), p.ContractYear,
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
			&p.Salary, &p.ContractYears, &p.ExpirationYear,
			&p.ContractStatus, &restructured, &tagged); err != nil {
			return nil, nil, fmt.Errorf("state: scan: %w", err)
		}
		p.IsRestructured, p.IsTagged = restructured != 0, tagged != 0
		f, ok := fr[p.FranchiseID]
		if !ok {
			f = &FranchiseState{FranchiseID: p.FranchiseID}
			fr[p.FranchiseID] = f
		}
		// CapSalary and CapUsed are set from the ledger cells AFTER the scan (load()); the
		// legacy salary columns are frozen. Salary here is only the BASE (annual) figure.
		f.Players = append(f.Players, p)
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
