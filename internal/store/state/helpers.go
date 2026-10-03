package state

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// rowQuerier is satisfied by *sql.DB and *sql.Tx.
type rowQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// CapDiscounts supplies the share of a taxi or IR player's salary that counts toward the cap
// (MFL's includeTaxiWithSalary and includeIRWithSalary, or an override). Read on every load, so an
// override takes effect immediately. nil means no discount.
type CapDiscounts interface {
	TaxiCapPercent() float64
	IRCapPercent() float64
}

// loadCellCap reads every current-season PAID cell with its franchise and returns each player's
// full cap salary (undiscounted: dead cap, buyout and tag math need the full figure) and each
// franchise's cap usage (taxi/IR-discounted, snapped to $10k). A player with no current PAID cell
// contributes 0.
func loadCellCap(ctx context.Context, q rowQuerier, leagueID string, season int, discounts CapDiscounts) (perPlayer, perFranchise map[string]domain.Money, err error) {
	rows, err := q.QueryContext(ctx, `
SELECT r.franchise_id, cy.mfl_id, r.roster_status, cy.salary_cents
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
		var fid, mflID, rosterStatus string
		var cents int64
		if err := rows.Scan(&fid, &mflID, &rosterStatus, &cents); err != nil {
			return nil, nil, fmt.Errorf("state: cell cap scan: %w", err)
		}
		perPlayer[mflID] = domain.Money(cents)
		perFranchise[fid] += domain.RoundToNearest10k(capContribution(domain.Money(cents), domain.RosterStatus(rosterStatus), discounts))
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("state: cell cap iterate: %w", err)
	}
	return perPlayer, perFranchise, nil
}

// capContribution applies the roster-status discount: active counts in full; taxi and IR count
// at the configured percentage.
func capContribution(cents domain.Money, status domain.RosterStatus, discounts CapDiscounts) domain.Money {
	var pct float64 = 100
	switch status {
	case domain.RosterActive:
		// full salary counts
	case domain.RosterTaxi:
		if discounts != nil {
			pct = discounts.TaxiCapPercent()
		}
	case domain.RosterIR:
		if discounts != nil {
			pct = discounts.IRCapPercent()
		}
	}
	if pct == 100 {
		return cents
	}
	return domain.Money(float64(cents) * pct / 100)
}

// seedPlayerCount totals the seed's players, for the empty-seed guard.
func seedPlayerCount(rosters []domain.Roster) int {
	n := 0
	for _, r := range rosters {
		n += len(r.Players)
	}
	return n
}

// seedPlayer inserts one player's roster and contract rows.
func seedPlayer(ctx context.Context, tx *sql.Tx, leagueID string, season int, now, franchiseID string, p domain.PlayerRecord) error {
	mflID := p.MFLID.String()
	key := fmt.Sprintf("%s:%d:%s", leagueID, season, mflID)
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO rosters (id, league_id, mfl_id, franchise_id, roster_status, season, as_of)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"r:"+key, leagueID, mflID, franchiseID, string(p.RosterStatus), season, now); err != nil {
		return fmt.Errorf("state: seed roster %q: %w", mflID, err)
	}
	// Only the base salary goes in contracts; the cap figure lives in the ledger cells.
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

// scanState reads the joined result set into franchise state and the player index.
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
		// CapSalary and CapUsed come from the ledger cells after the scan (see load).
		f.Players = append(f.Players, p)
		idx[p.MFLID] = p.FranchiseID
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("state: iterate: %w", err)
	}
	return fr, idx, nil
}

// cloneFranchise deep-copies, so a reader never aliases store memory.
func cloneFranchise(fs *FranchiseState) FranchiseState {
	return FranchiseState{
		FranchiseID: fs.FranchiseID,
		Players:     clonePlayers(fs.Players),
		CapUsed:     fs.CapUsed,
	}
}

// clonePlayers copies a slice; PlayerState has no reference fields, so this is a deep copy.
func clonePlayers(in []PlayerState) []PlayerState {
	if in == nil {
		return nil
	}
	out := make([]PlayerState, len(in))
	copy(out, in)
	return out
}

func sortedKeys(m map[string]*FranchiseState) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func validRosterStatus(s domain.RosterStatus) bool {
	return s == domain.RosterActive || s == domain.RosterTaxi || s == domain.RosterIR
}

// validContractStatus accepts the four real statuses, not the review flag.
func validContractStatus(s domain.ContractStatus) bool {
	switch s {
	case domain.CStatusUFA, domain.CStatusRFA, domain.CStatusFT1, domain.CStatusFT2:
		return true
	case domain.CStatusFlag:
		return false
	default:
		return false
	}
}

// requireOneRow fails unless a mutation touched exactly one row: zero rows means memory and the
// DB disagree.
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
