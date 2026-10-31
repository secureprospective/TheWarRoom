package history

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestNativeLineupDelta(t *testing.T) {
	t.Parallel()
	f := lineupTest(t)
	give := f.roster[f.a][leaguerules.WR][0]
	get := f.roster[f.b][leaguerules.WR][0]
	got, err := f.s.NativeLineupDelta(t.Context(), "native.test", 2026, 1,
		TradeSides{Franchise: f.a, Partner: f.b, Give: []string{give}, Get: []string{get}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.You.Before) != 21 || len(got.You.After) != 21 || len(got.Them.After) != 21 {
		t.Fatalf("a Legacy lineup starts 21: %d %d %d", len(got.You.Before), len(got.You.After), len(got.Them.After))
	}
	if len(got.You.Unprojected) == 0 {
		t.Fatal("players without held weeks must be named, never silently zero")
	}
	for _, bad := range []TradeSides{
		{Franchise: f.a, Partner: f.b, Give: []string{get}},
		{Franchise: f.a, Partner: f.b, Get: []string{give}},
		{Franchise: f.a, Partner: f.a, Give: []string{give}},
		{Franchise: f.a, Partner: f.b},
	} {
		if _, err := f.s.NativeLineupDelta(t.Context(), "native.test", 2026, 1, bad); err == nil ||
			!strings.Contains(err.Error(), "lineup delta") {
			t.Fatalf("%+v: want a plain refusal, got %v", bad, err)
		}
	}
}
