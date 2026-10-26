package offers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func TestRecordOffersAdmissionAndFailures(t *testing.T) {
	failure := errors.New("failed")
	for _, available := range []bool{false, true} {
		err := RecordOffers(context.Background(), nil, time.Time{}, failure, available,
			func() (string, error) { t.Fatal("owner read after failed fetch"); return "", nil }, nil)
		if !errors.Is(err, failure) {
			t.Fatal(err)
		}
	}
	if err := RecordOffers(context.Background(), nil, time.Time{}, nil, false, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := RecordOffers(context.Background(), nil, time.Time{}, nil, true, func() (string, error) { return "", failure }, nil); !errors.Is(err, failure) || err.Error() != "offers: archive owner: failed" {
		t.Fatal(err)
	}
	at := time.Now()
	save := func(_ context.Context, owner string, pending []leaguefeed.PendingTrade, got time.Time) error {
		if owner != "0001" || len(pending) != 0 || !got.Equal(at) {
			t.Fatal(owner, pending, got)
		}
		return failure
	}
	if err := RecordOffers(context.Background(), nil, at, nil, true, func() (string, error) { return "0001", nil }, save); !errors.Is(err, failure) || err.Error() != "offers: save live read: failed" {
		t.Fatal(err)
	}
}
