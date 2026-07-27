package state

import (
	"context"
	"strings"
	"testing"
)

// appendCorr runs AppendCorrection through a WriteTx (the surface the CORRECT transaction uses),
// mirroring appendEvent in calendar_test.go.
func appendCorr(t *testing.T, s *Store, e CorrectionEntry) error {
	t.Helper()
	return s.WriteTx(context.Background(), func(w TxWriter) error {
		return w.AppendCorrection(context.Background(), e)
	})
}

// TestAppendCorrection pins the write path and its shape guards: a valid CORRECTED entry lands and
// shows up in the read; a missing field or an unknown status is each rejected before any row is
// written. Mirrors TestAppendCalendarEvent.
func TestAppendCorrection(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()

	if err := appendCorr(t, s, CorrectionEntry{
		TxID: "trade_notes:tx:14432:0001:0002", Kind: "TRADE",
		Status: CorrStatusCorrected, Commissioner: "0001", Reason: "picks note typo",
		Note: "fixed the 2026 3rd",
	}); err != nil {
		t.Fatalf("valid correction: %v", err)
	}
	rows, err := s.Corrections(ctx)
	if err != nil {
		t.Fatalf("Corrections: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	r := rows[0]
	if r.TxID != "trade_notes:tx:14432:0001:0002" || r.Kind != "TRADE" || r.Status != CorrStatusCorrected {
		t.Fatalf("row = %+v, want the corrected TRADE row", r)
	}
	if r.Commissioner != "0001" || r.Reason != "picks note typo" || r.Note != "fixed the 2026 3rd" {
		t.Fatalf("row audit fields = %+v, want the supplied values", r)
	}
	if r.CreatedAt == "" {
		t.Fatal("created_at was not stamped by the store")
	}

	for _, tc := range []struct {
		name string
		e    CorrectionEntry
	}{
		{"empty tx id", CorrectionEntry{TxID: "", Kind: "TRADE", Status: CorrStatusCorrected, Commissioner: "c", Reason: "r"}},
		{"empty kind", CorrectionEntry{TxID: "t", Kind: "", Status: CorrStatusCorrected, Commissioner: "c", Reason: "r"}},
		{"bad status", CorrectionEntry{TxID: "t", Kind: "TRADE", Status: "POSTED", Commissioner: "c", Reason: "r"}},
		{"empty commissioner", CorrectionEntry{TxID: "t", Kind: "TRADE", Status: CorrStatusCorrected, Commissioner: "  ", Reason: "r"}},
		{"empty reason", CorrectionEntry{TxID: "t", Kind: "TRADE", Status: CorrStatusReversed, Commissioner: "c", Reason: ""}},
	} {
		if err := appendCorr(t, s, tc.e); err == nil {
			t.Fatalf("%s: AppendCorrection accepted a bad-shape entry, want rejection", tc.name)
		}
	}
}

// TestCorrectionAppendOnlyLatestWins proves the append-only latest-row-wins idiom (the
// calendar_events pattern this table mirrors): a CORRECTED then a REVERSED of the SAME tx_id each
// APPEND a new row, the read returns both in seq order, and the reconciled state is the LATEST row
// (the feed IPC reduces to latest-per-tx_id). The original is never mutated. Mirrors
// TestCalendarRescheduleCancelChainByEventID.
func TestCorrectionAppendOnlyLatestWins(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()
	tx := "contract_year_changes:cyc:14432:0042:2026"

	// first correction — clerical fix
	if err := appendCorr(t, s, CorrectionEntry{TxID: tx, Kind: "SIGN", Status: CorrStatusCorrected, Commissioner: "comm", Reason: "note typo", Note: "v1"}); err != nil {
		t.Fatalf("corrected: %v", err)
	}
	// second correction — reversal of the same entry
	if err := appendCorr(t, s, CorrectionEntry{TxID: tx, Kind: "SIGN", Status: CorrStatusReversed, Commissioner: "comm", Reason: "wrong player signed"}); err != nil {
		t.Fatalf("reversed: %v", err)
	}
	rows, err := s.Corrections(ctx)
	if err != nil {
		t.Fatalf("Corrections: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2 (corrected + reversed preserved)", len(rows))
	}
	if rows[0].Status != CorrStatusCorrected || rows[1].Status != CorrStatusReversed {
		t.Fatalf("order = %+v, want CORRECTED then REVERSED (seq ascending)", rows)
	}
	// reconciled state = latest by seq
	latest := rows[len(rows)-1]
	if latest.Status != CorrStatusReversed || latest.Reason != "wrong player signed" {
		t.Fatalf("latest = %+v, want the REVERSED row", latest)
	}

	// underlying history preserved: two rows share the tx id
	var n int
	if err := s.pools.Read().QueryRowContext(ctx,
		`SELECT COUNT(1) FROM transaction_corrections WHERE league_id = ? AND tx_id = ?`, s.leagueID, tx).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Fatalf("transaction_corrections holds %d rows for %q, want 2", n, tx)
	}
}

// TestCorrectionScopedByLeague proves the league_id scoping: a correction for league B does not
// appear in league A's read. The store stamps league_id from its own field, so a cross-league row
// would have to be planted by hand; this guards the WHERE clause.
func TestCorrectionScopedByLeague(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()
	if err := appendCorr(t, s, CorrectionEntry{TxID: "trade_notes:x", Kind: "TRADE", Status: CorrStatusReversed, Commissioner: "c", Reason: "r"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	// plant a row for a DIFFERENT league directly (the trigger only blocks UPDATE/DELETE, not INSERT)
	if _, err := s.pools.Write().ExecContext(ctx, `
INSERT INTO transaction_corrections (league_id, tx_id, kind, status, commissioner, reason, note, created_at)
VALUES ('9999', 'trade_notes:y', 'TRADE', ?, 'c', 'r', '', '2026-01-01T00:00:00Z')`, CorrStatusCorrected); err != nil {
		t.Fatalf("plant other-league row: %v", err)
	}
	rows, err := s.Corrections(ctx)
	if err != nil {
		t.Fatalf("Corrections: %v", err)
	}
	if len(rows) != 1 || rows[0].TxID != "trade_notes:x" {
		t.Fatalf("rows = %+v, want exactly the one league-14432 row", rows)
	}
}

// TestCorrectionAppendOnlyTriggers proves the double-immutability: a raw UPDATE or DELETE that
// bypasses the (write-only) Go API still aborts at the DB. Mirrors TestCalendarAppendOnlyTriggers.
func TestCorrectionAppendOnlyTriggers(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()
	if err := appendCorr(t, s, CorrectionEntry{TxID: "trade_notes:seed", Kind: "TRADE", Status: CorrStatusCorrected, Commissioner: "c", Reason: "r"}); err != nil {
		t.Fatalf("seed correction: %v", err)
	}

	if _, err := s.pools.Write().ExecContext(ctx,
		`UPDATE transaction_corrections SET status = ? WHERE league_id = ?`, CorrStatusReversed, s.leagueID); err == nil {
		t.Fatal("raw UPDATE on transaction_corrections succeeded — append-only trigger did not fire")
	} else if !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("UPDATE error = %v, want an append-only abort", err)
	}

	if _, err := s.pools.Write().ExecContext(ctx,
		`DELETE FROM transaction_corrections WHERE league_id = ?`, s.leagueID); err == nil {
		t.Fatal("raw DELETE on transaction_corrections succeeded — append-only trigger did not fire")
	} else if !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("DELETE error = %v, want an append-only abort", err)
	}
}
