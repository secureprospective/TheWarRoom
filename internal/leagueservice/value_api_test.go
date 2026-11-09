package leagueservice

import (
	"context"
	"errors"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/appvalue"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/lineupdelta"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
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

type referenceFitHost struct {
	admissionHost
	calls *int
}

func (h referenceFitHost) MirrorLineupDelta(_ context.Context, season, week int, trade history.TradeSides) (history.NativeLineupDelta, error) {
	*h.calls++
	return history.NativeLineupDelta{Season: season, Week: week, You: lineupdelta.Side{Franchise: trade.Franchise, PerWeek: 2.4}}, nil
}
func TestReferenceLineupFitIsLocalOnly(t *testing.T) {
	calls := 0
	host := referenceFitHost{calls: &calls}
	raw := RawLineupDeltaScope{Franchise: "0001", Partner: "0002", Get: []string{"0042"}, Season: 2026, Week: 9}
	if _, err := NewHosted(host, admissionFunc(func(context.Context, commissioner.Access) error { return nil })).TargetNativeLineupDelta(raw); err == nil || calls != 0 {
		t.Fatal("hosted session reached the local mirror")
	}
	got, err := New(host).TargetNativeLineupDelta(raw)
	if err != nil || calls != 1 || got.You.PerWeek != 2.4 {
		t.Fatalf("local fit: %+v %v", got, err)
	}
	raw.Get = []string{"bad"}
	if _, err := New(host).TargetNativeLineupDelta(raw); err == nil || calls != 1 {
		t.Fatal("invalid local request reached the read")
	}
}
