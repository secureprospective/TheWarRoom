package leagueservice

import (
	"context"
	"errors"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/appvalue"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

type valueTestHost struct {
	admissionHost
	calls *int
}

func (h valueTestHost) NativeValues(_ context.Context, league, franchise string, season int) (appvalue.Reading, error) {
	*h.calls++
	return appvalue.Reading{League: league, Franchise: franchise, Season: season}, nil
}
func TestNativeValuesValidatesAndAdmitsBeforeReading(t *testing.T) {
	calls := 0
	host := valueTestHost{calls: &calls}
	raw := RawValueScope{League: "native.test", Franchise: "franchise.01", Season: 2026}
	for _, bad := range []RawValueScope{{}, {League: "elsewhere", Season: 2026}, {League: "native.test", Season: 0}} {
		if _, err := New(host).TargetNativeValues(bad); err == nil || calls != 0 {
			t.Fatalf("invalid read reached host: %v", err)
		}
	}
	denied := errors.New("credential refused")
	if _, err := NewHosted(host, admissionFunc(func(context.Context, commissioner.Access) error { return denied })).TargetNativeValues(raw); !errors.Is(err, denied) || calls != 0 {
		t.Fatalf("denied read reached host: %v", err)
	}
	service := NewHosted(host, admissionFunc(func(_ context.Context, a commissioner.Access) error {
		if a.Operation != "value.read" || a.Franchise != raw.Franchise || a.League != raw.League {
			return errors.New("wrong scope")
		}
		return nil
	}))
	got, err := service.TargetNativeValues(raw)
	if err != nil || calls != 1 || got.League != raw.League || got.Franchise != raw.Franchise || got.Season != raw.Season {
		t.Fatalf("reading: %+v %v calls=%d", got, err, calls)
	}
	if _, err := New(admissionHost{}).TargetNativeValues(raw); err == nil {
		t.Fatal("host with no player record returned values")
	}
}
