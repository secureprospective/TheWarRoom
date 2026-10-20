package bids

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
)

func TestFormOfWritesMillionsExactly(t *testing.T) {
	t.Parallel()
	for cents, want := range map[domain.Money]string{500_000_000: "5", 525_000_000: "5.25", 33_000_000: "0.33",
		1_001_000_000: "10.01", 100_000_000: "1"} {
		if got := FormOf(envelope.BidFacts{Salary: cents}).Salary; got != want {
			t.Errorf("FormOf(%d) = %q, want %q", cents, got, want)
		}
	}
}

func TestInputKeepsTypedFiguresAndExactDenials(t *testing.T) {
	t.Parallel()
	input := Input{FranchiseID: "0001", Player: "0042", Salary: "5.004", Years: 3, Closes: "2026-10-10T14:00:00+02:00"}
	got, err := input.Entry()
	if err != nil || got.Entered != 500400000 || got.Facts.Salary != 500000000 || got.Facts.Closes != "2026-10-10T12:00:00Z" {
		t.Fatalf("entry = %+v, %v", got, err)
	}
	input.Salary = ""
	if _, err = input.Entry(); err == nil || err.Error() != "enter the annual salary in millions, such as 5.25" {
		t.Fatal(err)
	}
}
