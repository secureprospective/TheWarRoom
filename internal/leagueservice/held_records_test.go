package leagueservice

import (
	"context"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type heldOnly struct{ records []commissioner.Record }

func (heldOnly) Events(context.Context, string, string, int64) ([]history.LeagueEvent, error) {
	panic("a held ledger must not be decoded again")
}

func (h heldOnly) HeldCommissionerRecords(context.Context, string) ([]commissioner.Record, error) {
	return h.records, nil
}

// The value pass polls every second; it must read the store-held revision, not re-decode the ledger.
func TestRecordsUsesTheHeldRevisionWhenTheLedgerHasOne(t *testing.T) {
	want := []commissioner.Record{{Principal: "creator"}}
	got, err := Records(t.Context(), heldOnly{want}, "native.test", "values")
	if err != nil || len(got) != 1 || got[0].Principal != "creator" {
		t.Fatalf("records: %+v %v", got, err)
	}
}
