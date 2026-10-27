package offers

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func TestCapRoomAvailabilityAndReadOrder(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		known bool
		reads int
		room  *domain.Money
	}{
		{" ", true, 0, nil}, {"bad", true, 1, nil}, {"100", false, 1, nil},
	} {
		reads := 0
		got := CapRoom(tc.raw, func() (domain.Money, bool) { reads++; return 0, tc.known })
		if got != nil || reads != tc.reads {
			t.Fatal(got, reads)
		}
	}
	limit, err := domain.ParseMoneyMillions("100")
	if err != nil {
		t.Fatal(err)
	}
	got := CapRoom(" 100 ", func() (domain.Money, bool) { return limit + 1, true })
	if got == nil || *got != -1 {
		t.Fatal(got)
	}
}
