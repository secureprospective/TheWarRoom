package trades

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func TestParallelReadKeepsHeldInputsDetached(t *testing.T) {
	t.Parallel()
	id, err := playerid.New("0099")
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	snap := snapshot.Snapshot{}
	snap.Players.Value = []snapshot.Player{{ID: id, Name: "Player", Position: domain.PosWR}}
	snap.Franchises.Value = []snapshot.Franchise{{ID: "0001", Name: "One"}, {ID: "0002", Name: "Two"}}
	held := snapshot.Sourced[[]leaguefeed.PendingTrade]{Value: []leaguefeed.PendingTrade{
		{ID: "later", Offering: "0001", OfferedTo: "0002", Gives: []leaguefeed.Asset{{Player: &id}}, Expires: at.Add(time.Hour)},
		{ID: "first", Offering: "0002", OfferedTo: "0001", Gets: []leaguefeed.Asset{{Player: &id}}, Expires: at},
	}, Provenance: snapshot.Provenance{Freshness: domain.Freshness{State: domain.FreshLive}}}
	before, err := json.Marshal(held)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 12; i++ {
		t.Run("shared snapshot", func(t *testing.T) {
			t.Parallel()
			reading := Read("0001", snap, held)
			if len(reading.Offers) != 2 || reading.Offers[0].TradeID != "first" || reading.Offers[0].Direction != "to_you" ||
				reading.Offers[1].Direction != "by_you" || reading.Offers[1].Give[0].Name != "Player" {
				t.Fatal(reading)
			}
			reading.Offers[1].Give[0].Name = "mutated"
			after, err := json.Marshal(held)
			if err != nil || !reflect.DeepEqual(before, after) || Read("0001", snap, held).Offers[1].Give[0].Name != "Player" {
				t.Fatal("mutated held inputs", err)
			}
		})
	}
}
