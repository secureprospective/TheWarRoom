package history

import (
	"context"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// The mirror ledger answers reads for its league only, and nothing can be written to it.
func TestMirrorLedgerReadsItsLeagueAndRefusesWrites(t *testing.T) {
	s, _, _ := newStore(t)
	held := []commissioner.Record{{Principal: "mfl:held"}}
	s.SetMirrorLedger(MirrorLedger{League: "native.mfl.1", Records: func(context.Context) ([]commissioner.Record, error) {
		return held, nil
	}})
	ctx := context.Background()
	got, err := s.leagueRecords(ctx, "native.mfl.1")
	if err != nil || len(got) != 1 || got[0].Principal != "mfl:held" {
		t.Fatalf("mirror league reads the held records: %v %+v", err, got)
	}
	if got, err := s.nativeRecords(ctx, "native.mfl.1"); err != nil || len(got) != 1 {
		t.Fatalf("native reads take the same path: %v %+v", err, got)
	}
	if got, err := s.leagueRecords(ctx, "native.other"); err != nil || len(got) != 0 {
		t.Fatalf("other leagues keep their own ledger: %v %+v", err, got)
	}
	_, err = s.ExecuteCommissioner(ctx, commissioner.RawCommand{League: "native.mfl.1"}, "anyone", nil)
	if err == nil || !strings.Contains(err.Error(), "run on MFL") {
		t.Fatalf("writes to the mirror league are refused: %v", err)
	}
	if _, err := s.ExecuteCommissionerBatch(ctx, []commissioner.RawCommand{{League: "native.mfl.1"}}, "anyone", nil); err == nil ||
		!strings.Contains(err.Error(), "run on MFL") {
		t.Fatalf("batched writes too: %v", err)
	}
}
