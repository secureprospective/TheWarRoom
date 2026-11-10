package planwatch

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
)

func TestTaxiPredicateRegistered(t *testing.T) {
	t.Parallel()
	predicate := MovePredicate("roster.taxi")
	if _, ok := predicate.(envelope.TaxiPredicate); !ok {
		t.Fatalf("taxi registry: %T", predicate)
	}
}
