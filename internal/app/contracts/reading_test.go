package contracts

import (
	"fmt"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
)

// A failed read leaves every contracts reading failed; none becomes figures.
func TestFailedContractsKeepsEveryReadingFailed(t *testing.T) {
	t.Parallel()
	a, _ := planHost(t)
	failed := a.failedContracts(fmt.Errorf("mirror unavailable"))
	if failed.Table.Status != report.ReadingFailed || failed.Cap.Status != report.ReadingFailed ||
		failed.Adjustments.Status != report.ReadingFailed {
		t.Fatal("failure became figures")
	}
}
