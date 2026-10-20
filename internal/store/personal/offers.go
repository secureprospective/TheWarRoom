package personal

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

type SeenOffer struct {
	ID, Sender, Receiver, Outcome               string
	Year                                        int
	Timestamp, FirstSeen, LastSeen, FirstMissed int64
}

// ObserveOffers is called only after a successful pendingTrades read, including an empty one.
func (s *Store) ObserveOffers(ctx context.Context, mine string, year int, pending []leaguefeed.PendingTrade,
	trades []leaguefeed.Transaction, at time.Time) error {
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("offers: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, p := range pending {
		if p.Offering != mine && p.OfferedTo != mine {
			continue
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO personal_offers
 (mine,id,year,ts,sender,receiver,first_seen,last_seen,outcome) VALUES(?,?,?,?,?,?,?,?,'pending')
 ON CONFLICT(mine,year,id,ts) DO UPDATE SET last_seen=excluded.last_seen,first_missed=0,outcome='pending'`,
			mine, p.ID, year, p.Proposed.Unix(), p.Offering, p.OfferedTo, at.Unix(), at.Unix())
		if err != nil {
			return fmt.Errorf("offers: record pending: %w", err)
		}
	}
	rows, err := tx.QueryContext(ctx, seenOffersQuery, mine)
	if err != nil {
		return fmt.Errorf("offers: read for refresh: %w", err)
	}
	offers, err := scanSeenOffers(rows)
	if err != nil {
		return err
	}
	if err := recordOfferEndings(ctx, tx, mine, year, offers, pending, trades, at); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO personal_offer_checks(mine,checked) VALUES(?,?)
 ON CONFLICT(mine) DO UPDATE SET checked=excluded.checked`, mine, at.Unix()); err != nil {
		return fmt.Errorf("offers: record check: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("offers: commit: %w", err)
	}
	return nil
}

func recordOfferEndings(ctx context.Context, tx *sql.Tx, mine string, year int, offers []SeenOffer,
	pending []leaguefeed.PendingTrade, trades []leaguefeed.Transaction, at time.Time) error {

	for _, o := range offers {
		if o.Year != year || o.Outcome != "pending" || slices.ContainsFunc(pending, func(p leaguefeed.PendingTrade) bool {
			return p.ID == o.ID && p.Proposed.Unix() == o.Timestamp
		}) {
			continue
		}
		outcome := missingOfferOutcome(o, trades, at.Unix())
		if _, err := tx.ExecContext(ctx, `UPDATE personal_offers SET outcome=?,first_missed=?
 WHERE mine=? AND year=? AND id=? AND ts=?`, outcome, at.Unix(), mine, o.Year, o.ID, o.Timestamp); err != nil {
			return fmt.Errorf("offers: record ending: %w", err)
		}
	}
	return nil
}

func missingOfferOutcome(o SeenOffer, trades []leaguefeed.Transaction, missed int64) string {
	for _, t := range trades {
		if t.Kind != "TRADE" || t.Trade == nil || t.Unparsed || t.Time.Unix() < o.Timestamp ||
			t.Time.Unix() > missed {
			continue
		}
		other := t.Trade.Counterparty
		if t.Franchise == o.Sender && other == o.Receiver || t.Franchise == o.Receiver && other == o.Sender {
			return "accepted"
		}
	}
	return "unknown"
}

func (s *Store) Offers(ctx context.Context, mine string) ([]SeenOffer, int64, error) {
	rows, err := s.pools.Read().QueryContext(ctx, seenOffersQuery, mine)
	if err != nil {
		return nil, 0, fmt.Errorf("offers: read: %w", err)
	}
	offers, err := scanSeenOffers(rows)
	if err != nil {
		return nil, 0, err
	}
	var checked int64
	err = s.pools.Read().QueryRowContext(ctx, `SELECT checked FROM personal_offer_checks WHERE mine=?`, mine).Scan(&checked)
	if err != nil && err != sql.ErrNoRows {
		return nil, 0, fmt.Errorf("offers: last check: %w", err)
	}
	return offers, checked, nil
}

const seenOffersQuery = `SELECT id,year,ts,sender,receiver,first_seen,last_seen,first_missed,outcome
 FROM personal_offers WHERE mine=? ORDER BY ts,id`

func scanSeenOffers(rows *sql.Rows) ([]SeenOffer, error) {
	defer func() { _ = rows.Close() }()
	out := []SeenOffer{}
	for rows.Next() {
		var o SeenOffer
		if err := rows.Scan(&o.ID, &o.Year, &o.Timestamp, &o.Sender, &o.Receiver, &o.FirstSeen, &o.LastSeen,
			&o.FirstMissed, &o.Outcome); err != nil {
			return nil, fmt.Errorf("offers: scan: %w", err)
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("offers: rows: %w", err)
	}
	return out, nil
}
