package history

import (
	"context"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestNativeAssetsPublicIdentities(t *testing.T) {
	t.Parallel()
	s, _, _ := newStore(t)
	point := perSidePoint()
	if _, err := s.StandUpRosters(t.Context(), point); err != nil {
		t.Fatal(err)
	}
	s.SetLineupEnvironment(&lineupEnvFake{facts: nativecheck.Facts{
		"1000": {Name: "Public player", Position: leaguerules.QB},
	}})
	got, err := s.NativeAssets(t.Context(), point.League)
	if err != nil {
		t.Fatal(err)
	}
	want := []commissioner.NativeAssetIdentity{
		{Player: "1000", Franchise: "franchise.01", Side: "AFC", Name: "Public player", Position: "QB"},
		{Player: "1000", Franchise: "franchise.09", Side: "NFC", Name: "Public player", Position: "QB"},
	}
	if !reflect.DeepEqual(got.Identities, want) {
		t.Fatalf("public identities: %+v", got.Identities)
	}
	if !reflect.DeepEqual(got.Snapshot.Players, point.Snapshot.Players) {
		t.Fatal("identity enrichment changed recorded players")
	}
	s.SetLineupEnvironment(nil)
	absent, err := s.NativeAssets(t.Context(), point.League)
	if err != nil {
		t.Fatal(err)
	}
	if len(absent.Identities) != 0 {
		t.Fatalf("invented facts: %+v", absent.Identities)
	}
	// A partial catalogue must not invent identities for players it does not hold.
	enrichNativeAssets(&absent, nativecheck.Facts{"other": {Name: "Other", Position: leaguerules.QB}})
	if len(absent.Identities) != 0 {
		t.Fatalf("invented facts from partial catalogue: %+v", absent.Identities)
	}
	empty := commissioner.NativeAssets{}
	enrichNativeAssets(&empty, nativecheck.Facts{"1000": {Name: "Public player"}})
	if len(empty.Identities) != 0 {
		t.Fatalf("invented facts without roster: %+v", empty.Identities)
	}
}

func TestNativeMarketPoolScope(t *testing.T) {
	t.Parallel()
	reading := commissioner.NativeAssets{Snapshot: &commissioner.RawNativeSnapshot{
		Players: []commissioner.NativePlayer{{ID: "owned"}}, Pool: []commissioner.NativePoolEntry{{Player: "lot", Name: "Lot"}},
	}}
	enrichNativeAssets(&reading, nativecheck.Facts{
		"owned": {Name: "Rostered"}, "free": {Name: "Free", Position: leaguerules.WR, Team: "FA"},
		"injured": {Name: "Injured", DesignationHeld: true, Designation: leaguerules.InjuredReserve},
	})
	if len(reading.Pool) != 3 || reading.Pool[0].Player != "free" || reading.Pool[0].String != "FA" || reading.Pool[1].String != "IR" || reading.Pool[2].Player != "lot" {
		t.Fatalf("directory minus rosters plus lots: %+v", reading.Pool)
	}
}

type poolWeekScores map[int]leaguerules.Decimal

func (scores poolWeekScores) PlayerScore(_ context.Context, _ []commissioner.Record, _ string, week int) (commissioner.NativePlayerScore, error) {
	value, ok := scores[week]
	if !ok {
		return commissioner.NativePlayerScore{}, nil
	}
	return commissioner.NativePlayerScore{Points: &value}, nil
}
func TestNativePoolSeasonTotal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		weeks []int
		want  leaguerules.Decimal
		held  bool
	}{
		{"finished weeks only", []int{1, 2}, 175000, true},
		{"held zero counts", []int{3}, 0, true},
		{"missing is unknown", []int{4}, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, held := nativePoolTotal(t.Context(), nil, poolWeekScores{1: 125000, 2: 50000, 3: 0, 5: 999000}, "sample", tc.weeks)
			if got != tc.want || held != tc.held {
				t.Fatalf("sum = %v, %v; want %v, %v", got, held, tc.want, tc.held)
			}
		})
	}
}
