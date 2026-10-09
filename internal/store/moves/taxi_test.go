package moves

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func TestTaxiStoreRoundTrip(t *testing.T) {
	store, _ := testStore(t)
	player, err := playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []domain.RosterStatus{domain.RosterActive, domain.RosterTaxi} {
		at := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
		snap := snapshot.Snapshot{Rosters: snapshot.Sourced[[]snapshot.Roster]{
			Value: []snapshot.Roster{{FranchiseID: "0001", Players: []snapshot.RosterPlayer{
				{ID: player, RosterStatus: status},
			}}},
		}}
		e, err := envelope.DraftTaxi(at, envelope.TaxiRequest{
			LeagueID: "1", FranchiseID: "0001", Player: player,
			Target: envelope.Target{Kind: envelope.Mapped,
				URL: "https://www47.myfantasyleague.com/2026/options?L=1&O=98"},
		}, snap, func() string { return "taxi-" + string(status) })
		if err != nil {
			t.Fatal(err)
		}
		e, err = e.HandOff(at.Add(time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		if err := store.Save(context.Background(), e); err != nil {
			t.Fatal(err)
		}
		restored, err := store.Get(context.Background(), e.ID())
		if err != nil || !reflect.DeepEqual(restored.Receipt(), e.Receipt()) {
			t.Fatalf("taxi round trip: %+v %v", restored.Receipt(), err)
		}
		before, err := json.Marshal(e.Receipt())
		if err != nil {
			t.Fatal(err)
		}
		after, err := json.Marshal(restored.Receipt())
		if err != nil || string(before) != string(after) {
			t.Fatalf("taxi JSON changed: %s != %s: %v", before, after, err)
		}
	}
}
