package contracts

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func TestRulesFreshnessStoredUntilChecked(t *testing.T) {
	if got := RulesFreshness(0); got.State != domain.FreshStale || got.Note != "stored league settings; not yet checked with MFL this run" {
		t.Fatal(got)
	}
	if got := RulesFreshness(123); got.State != domain.FreshLive || got.FetchedAt != time.Unix(123, 0).UTC().Format(time.RFC3339) {
		t.Fatal(got)
	}
}
