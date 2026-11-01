package history

import (
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
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
}
