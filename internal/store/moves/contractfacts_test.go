package moves

import (
	"context"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/engine/contractacts"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
)

func TestContractMigrationUpRoundTripAndAtomicValidation(t *testing.T) {
	s, pools := testStore(t)
	ctx := context.Background()
	existing, bidFacts := testBid(t, "existing")
	if err := s.SaveBid(ctx, existing, bidFacts); err != nil {
		t.Fatal(err)
	}
	s, pools = legacyMovesFile(t, pools, "contract_facts")
	if err := s.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, "existing"); err != nil {
		t.Fatal("migration lost history", err)
	}
	spec := existing.Receipt().Spec
	spec.Intent = "rfa.tender"
	spec.Expected.RosterStatus = ""
	f := envelope.ContractFacts{Player: bidFacts.Player, Name: "Example", Intent: spec.Intent, Season: 2026, Baseline: []contractacts.Cell{{Season: 2026, Salary: 100_000_000}}, Terms: contractacts.Flat(2026, 1, 110_000_000, "§7"), Reasons: []string{"Rule unclear: window"}}
	e, err := envelope.DraftContract(bidAt(), spec, f, func() string { return "contract-1" })
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveContract(ctx, e, f); err != nil {
		t.Fatal(err)
	}
	got, err := s.ContractFacts(ctx, e.ID())
	if err != nil || !reflect.DeepEqual(got, f) {
		t.Fatalf("facts %+v %v", got, err)
	}
	restored, err := s.Get(ctx, e.ID())
	if err != nil || restored.State() != envelope.Blocked {
		t.Fatal("restore failed", err)
	}
	f.Intent = "contract.tag"
	if err := s.SaveContract(ctx, e, f); err == nil {
		t.Fatal("mismatched facts accepted")
	}
	if _, err := pools.Write().ExecContext(ctx, `DELETE FROM contract_facts WHERE correlation_id=?`, e.ID()); err == nil {
		t.Fatal("history facts deleted")
	}
}
