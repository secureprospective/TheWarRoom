package offers

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/lab"
)

func TestLiveOfferCalendarMatchesArchive(t *testing.T) {
	t.Parallel()
	d, err := lab.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range d.Offers {
		got, err := LiveOffer(d, o.Year, int64(o.Ts), o.By, o.To, o.Outcome)
		if err != nil || got.Season != o.Season || got.Phase != o.Phase {
			t.Fatalf("calendar diverged at %d: %+v vs %+v, %v", o.Ts, got, o, err)
		}
	}
	for _, tc := range []struct {
		date   string
		season int
		phase  string
	}{
		{"2026-12-07T04:59:00Z", 2026, "closed"},
		{"2026-12-08T05:00:00Z", 2027, "closed"},
		{"2027-01-05T05:00:00Z", 2027, "winter"},
	} {
		at, err := time.Parse(time.RFC3339, tc.date)
		if err != nil {
			t.Fatal(err)
		}
		got, err := LiveOffer(d, 2026, at.Unix(), "0025", "0001", "pending")
		if err != nil || got.Season != tc.season || got.Phase != tc.phase {
			t.Fatal(got, err)
		}
	}
}
