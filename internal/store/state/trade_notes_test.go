package state

import (
	"context"
	"testing"
)

// TestLogTradeNote_RoundTrips proves a trade_notes row survives a WriteTx commit with the
// exact picksNote/rationale/involved-franchises values it was written with.
func TestLogTradeNote_RoundTrips(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})

	if err := s.WriteTx(context.Background(), func(w TxWriter) error {
		return w.LogTradeNote(context.Background(), "2027 1st to 0002", "rebuilding the roster", []string{"0001", "0002"})
	}); err != nil {
		t.Fatalf("LogTradeNote: %v", err)
	}

	var picksNote, rationale, involved string
	if err := s.pools.Read().QueryRowContext(context.Background(),
		`SELECT picks_note, rationale, involved_franchises FROM trade_notes`).
		Scan(&picksNote, &rationale, &involved); err != nil {
		t.Fatalf("read trade_notes row: %v", err)
	}
	if picksNote != "2027 1st to 0002" || rationale != "rebuilding the roster" || involved != "0001,0002" {
		t.Fatalf("trade_notes row = (%q, %q, %q), want (%q, %q, %q)",
			picksNote, rationale, involved, "2027 1st to 0002", "rebuilding the roster", "0001,0002")
	}
}

// TestLogTradeNote_PicksNoteUnvalidated proves an EMPTY picksNote is accepted (Alpha scope:
// picks are free-text and optional, unlike rationale which Trade.validate() requires upstream).
func TestLogTradeNote_PicksNoteUnvalidated(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})

	if err := s.WriteTx(context.Background(), func(w TxWriter) error {
		return w.LogTradeNote(context.Background(), "", "player-only swap", []string{"0001"})
	}); err != nil {
		t.Fatalf("LogTradeNote with empty picksNote: %v", err)
	}
}
