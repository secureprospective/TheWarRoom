package core_test

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/app/core/coretest"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func TestOffersSharedReadsDetachNestedValues(t *testing.T) {
	s := coretest.New(t).Build()
	id, err := playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	born, rookie, money := int64(123), true, domain.Money(456)
	s.SeasonFreeAgents.Value = []playerid.PlayerID{id}
	s.TargetSnapshot.Players.Value = []snapshot.Player{{ID: id, Birthdate: &born, IsRookie: &rookie}}
	s.TargetSnapshot.Franchises.Value = []snapshot.Franchise{{ID: "0001", CapUsed: &money, CapRoom: &money}}
	s.HasTargetSnapshot = true
	players, franchises, loaded := s.OffersSnapshotNames()
	if !loaded {
		t.Fatal("loaded snapshot lost")
	}
	players[0].ID = playerid.PlayerID{}
	*players[0].Birthdate, *players[0].IsRookie = 999, false
	*franchises[0].CapUsed, *franchises[0].CapRoom = 999, 999
	feed := s.OffersFreeAgents()
	feed.Value[0] = playerid.PlayerID{}
	players, franchises, _ = s.OffersSnapshotNames()
	if players[0].ID != id || *players[0].Birthdate != 123 || !*players[0].IsRookie ||
		*franchises[0].CapUsed != 456 || *franchises[0].CapRoom != 456 || s.OffersFreeAgents().Value[0] != id {
		t.Fatal("snapshot aliases shared storage")
	}
	for _, empty := range [][]playerid.PlayerID{nil, {}} {
		s.SeasonFreeAgents.Value = empty
		got := s.OffersFreeAgents().Value
		if (got == nil) != (empty == nil) {
			t.Fatal("nil/empty feed changed")
		}
	}
	for _, empty := range [][]snapshot.Player{nil, {}} {
		s.TargetSnapshot.Players.Value = empty
		got, _, _ := s.OffersSnapshotNames()
		if (got == nil) != (empty == nil) {
			t.Fatal("nil/empty names changed")
		}
	}
}

func TestOffersFeedPublicationRacesDetachedReads(t *testing.T) {
	s := coretest.New(t).Build()
	ids := make([]playerid.PlayerID, 2)
	for i, raw := range []string{"0042", "13593"} {
		id, err := playerid.New(raw)
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = id
	}
	var workers sync.WaitGroup
	workers.Go(func() {
		for range 500 {
			s.SeasonMu.Lock()
			s.SeasonFreeAgents = core.HeldFeed[[]playerid.PlayerID]{Value: slices.Clone(ids), FetchedAt: time.Now()}
			s.SeasonMu.Unlock()
		}
	})
	for range 3 {
		workers.Go(func() {
			for range 500 {
				feed := s.OffersFreeAgents()
				if len(feed.Value) > 0 {
					if !slices.Equal(feed.Value, ids) {
						t.Error("publication torn")
					}
					feed.Value[0] = playerid.PlayerID{}
				}
			}
		})
	}
	workers.Wait()
}

func TestOffersReadsCompleteDuringSerializedMove(t *testing.T) {
	s := coretest.New(t).Build()
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() { done <- s.SerializeOffersMove(func() error { close(entered); <-release; return nil }) }()
	<-entered
	defer func() {
		close(release)
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	if s.MovesMu.TryLock() {
		s.MovesMu.Unlock()
		t.Fatal("operation not serialized")
	}
	read := make(chan struct{})
	go func() { s.OffersFreeAgents(); s.OffersSnapshotNames(); s.OffersWeek(); close(read) }()
	select {
	case <-read:
	case <-time.After(time.Second):
		t.Fatal("reads wait for move I/O")
	}
}

func TestOffersSerializerRetainsNotificationOrder(t *testing.T) {
	s := coretest.New(t).Build()
	var order []string
	s.MovesChanged = func(context.Context) {
		if s.MovesMu.TryLock() {
			s.MovesMu.Unlock()
			t.Error("event escaped original serializer")
		}
		order = append(order, "event")
	}
	if err := s.SerializeOffersMove(func() error {
		order = append(order, "save", "log")
		s.OffersChanged()
		order = append(order, "wake", "read")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !s.MovesMu.TryLock() {
		t.Fatal("serializer not released")
	}
	s.MovesMu.Unlock()
	if !slices.Equal(order, []string{"save", "log", "event", "wake", "read"}) {
		t.Fatal(order)
	}
}
