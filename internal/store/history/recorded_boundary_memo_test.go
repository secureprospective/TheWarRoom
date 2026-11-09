package history

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
)

// TestRecordedBoundaryMemoMatchesTheLedger: a reading works out the recorded-results cutover once per
// ledger; every answer, first or remembered, is the ledger's own, and another ledger is never answered
// from the memo.
func TestRecordedBoundaryMemoMatchesTheLedger(t *testing.T) {
	t.Parallel()
	export := recordedImportFixture(t)
	svc, store, _ := migrationService(t, export)
	if _, err := svc.Import(t.Context(), export.LeagueID, "Import recorded results", "creator"); err != nil {
		t.Fatal(err)
	}
	league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
	records, err := store.nativeRecords(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	n := store.nativeSources(league, export.Season)
	if _, ok := n.scoreRevision(t.Context(), records); !ok {
		t.Fatal("the imported ledger must have a shared revision for the memo to apply")
	}
	for pass := range 2 {
		if got := n.recordedBoundary(records, export.Season); got != 2 {
			t.Fatalf("pass %d: cutover %d, want the ledger's 2", pass, got)
		}
		if got := n.recordedBoundary(records, export.Season+1); got != 0 {
			t.Fatalf("pass %d: a season with no recorded results has cutover %d", pass, got)
		}
	}
	other := append([]commissioner.Record(nil), records[:1]...) // a different ledger: no recorded results
	if got := n.recordedBoundary(other, export.Season); got != 0 {
		t.Fatalf("another ledger was answered from the memo: cutover %d", got)
	}
}
