package composition

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/app/core/coretest"
	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func commsState(t *testing.T) (*core.State, *coretest.Transport) {
	t.Helper()
	s := coretest.New(t).Build()
	transport, client := coretest.NewTransport(t, map[string]string{
		"messageBoard": `{"messageBoard":{}}`, "polls": `{"polls":{}}`,
	})
	s.MFLClient = client.WithKeySource(s.KeyStore.Get)
	return s, transport
}

func TestCommsMissingKeyClearsMemberBoardsWithoutRequests(t *testing.T) {
	s, transport := commsState(t)
	s.CommsBoards[0] = core.HeldFeed[[]ingestion.RawLeaguePost]{Value: []ingestion.RawLeaguePost{{Title: "kept"}}, FetchedAt: time.Now()}
	if err := (commsHost{s}).RefreshComms(true); err != nil {
		t.Fatal(err)
	}
	requests, _ := transport.Requests()
	if len(requests) != 0 || s.CommsBoards[0].Value != nil || !strings.Contains(s.CommsBoards[0].RefreshError, "connect your MFL key in Control Room › App") {
		t.Fatal(requests, s.CommsBoards[0])
	}
	s.CommsBoards[0].Value = []ingestion.RawLeaguePost{{Title: "kept"}}
	s.KeyStore = nil
	(commsHost{s}).RefreshCommsBoards(context.Background())
	if len(s.CommsBoards[0].Value) != 1 || s.CommsBoards[0].RefreshError != "MFL keyring is locked or unavailable" {
		t.Fatal(s.CommsBoards[0])
	}
}

func TestCommsRefreshFreshnessOrderAndLastGood(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		force, txStale, boardStale bool
		txRuns, boardRuns          int
	}{
		{"fresh", false, false, false, 0, 0}, {"force", true, false, false, 1, 2},
		{"transactions", false, true, false, 1, 0}, {"boards", false, false, true, 0, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, transport := commsState(t)
			if err := s.KeyStore.Set(s.Ctx, "fixture-key"); err != nil {
				t.Fatal(err)
			}
			s.SeasonTransactions.FetchedAt = time.Now()
			for i := range s.CommsBoards {
				s.CommsBoards[i].FetchedAt = time.Now()
			}
			if tc.txStale {
				s.SeasonTransactions.RefreshError = "offline"
			}
			if tc.boardStale {
				s.CommsBoards[1].FetchedAt = time.Time{}
			}
			txRuns, pauses := 0, 0
			s.RefreshTransactions = func(context.Context) {
				requests, _ := transport.Requests()
				if len(requests) != 0 {
					t.Error("transactions must refresh before boards")
				}
				txRuns++
			}
			s.SeasonPause = func(ctx context.Context, d time.Duration) error {
				if d != time.Second {
					t.Error(d)
				}
				pauses++
				return ctx.Err()
			}
			if err := (commsHost{s}).RefreshComms(tc.force); err != nil {
				t.Fatal(err)
			}
			requests, keyed := transport.Requests()
			if txRuns != tc.txRuns || len(requests) != tc.boardRuns || pauses != tc.boardRuns || !reflect.DeepEqual(requests, keyed) {
				t.Fatal(txRuns, requests, keyed, pauses)
			}
			if tc.boardRuns == 0 {
				return
			}
			before := s.CommsBoards[0].FetchedAt
			transport.Fail("messageBoard")
			s.RefreshTransactions = func(context.Context) {}
			if err := (commsHost{s}).RefreshComms(true); err != nil {
				t.Fatal(err)
			}
			if s.CommsBoards[0].FetchedAt != before || s.CommsBoards[0].RefreshError == "" {
				t.Fatal("failed refresh lost last good board", s.CommsBoards[0])
			}
		})
	}
}

func TestCommsSnapshotsAreDetached(t *testing.T) {
	s, _ := commsState(t)
	id, err := playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	cents := int64(125)
	asset := leaguefeed.Asset{Player: &id, CurrentPick: &leaguefeed.CurrentPick{Round: 1}, FuturePick: &leaguefeed.FuturePick{Year: 2027}, BlindBidCents: &cents}
	change := &leaguefeed.RosterChange{In: []leaguefeed.Asset{asset}, Out: []leaguefeed.Asset{asset}}
	s.SeasonTransactions.Value = []leaguefeed.Transaction{{Trade: &leaguefeed.Trade{Gave: []leaguefeed.Asset{asset}, CounterpartyGave: []leaguefeed.Asset{asset}}, AddsDrops: change, IR: change, Taxi: change}}
	s.CommsBoards[0].Value = []ingestion.RawLeaguePost{{Title: "kept"}}
	tx, boards, zone := s.CommsFeeds()
	tx.Value[0].Trade.Gave[0].CurrentPick.Round = 9
	*tx.Value[0].Trade.CounterpartyGave[0].BlindBidCents = 999
	tx.Value[0].AddsDrops.In[0].FuturePick.Year = 2099
	tx.Value[0].IR.Out[0].CurrentPick.Round = 9
	*tx.Value[0].Taxi.In[0].Player = playerid.PlayerID{}
	boards[0].Value[0].Title = "changed"
	again, againBoards, _ := s.CommsFeeds()
	if zone != time.UTC || again.Value[0].Trade.Gave[0].CurrentPick.Round != 1 || *again.Value[0].Trade.CounterpartyGave[0].BlindBidCents != 125 || again.Value[0].AddsDrops.In[0].FuturePick.Year != 2027 || again.Value[0].IR.Out[0].CurrentPick.Round != 1 || again.Value[0].Taxi.In[0].Player.String() != "0042" || againBoards[0].Value[0].Title != "kept" {
		t.Fatal("borrowed mutable feed storage")
	}
	snap := state.MirrorSnapshot{Season: 2026, Players: []state.MirrorPlayer{{MFLID: "0042", FranchiseID: "0001"}}}
	if _, err := s.League.Replace(s.Ctx, snap); err != nil {
		t.Fatal(err)
	}
	_, names, mirror := s.CommsLeague()
	names["0001"] = "changed"
	mirror.Players[0].MFLID = "changed"
	_, names2, mirror2 := s.CommsLeague()
	if names2["0001"] != "" || mirror2.Players[0].MFLID != "0042" {
		t.Fatal("borrowed league storage")
	}
}

func TestCommsArchiveDetachesColumnsAndPointers(t *testing.T) {
	s, _ := commsState(t)
	salary, kickoff := 1.25, int64(100)
	held := facts.RosterWeekColumns{Year: []int64{2026}, Salary: []*float64{&salary}}
	games := facts.NflGameColumns{Kickoff: []*int64{&kickoff}}
	s.ReadCommsArchive = func(context.Context) (facts.RosterWeekColumns, facts.NflGameColumns, error) { return held, games, nil }
	rosters, readGames, err := s.CommsArchive(s.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	rosters.Year[0] = 1999
	*rosters.Salary[0] = 99
	*readGames.Kickoff[0] = 999
	if held.Year[0] != 2026 || salary != 1.25 || kickoff != 100 {
		t.Fatal("borrowed archive columns")
	}
	s.ReadCommsArchive = func(context.Context) (facts.RosterWeekColumns, facts.NflGameColumns, error) {
		return held, games, errors.New("offline")
	}
	if _, _, err := s.CommsArchive(s.Ctx); err == nil || err.Error() != "offline" {
		t.Fatal(err)
	}
}

func TestCommsConcurrentRefreshAndRead(t *testing.T) {
	s, _ := commsState(t)
	if err := s.KeyStore.Set(s.Ctx, "fixture-key"); err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	s.SeasonPause = func(ctx context.Context, _ time.Duration) error {
		once.Do(func() { close(entered); <-release })
		return ctx.Err()
	}
	s.RefreshTransactions = func(context.Context) {
		s.SeasonMu.Lock()
		core.HoldFeed(&s.SeasonTransactions, []leaguefeed.Transaction{{Kind: "refresh"}}, core.LiveFreshness(time.Now()), nil)
		s.SeasonMu.Unlock()
	}
	done := make(chan error, 1)
	go func() { done <- (commsHost{s}).RefreshComms(true) }()
	<-entered
	// Reading must complete while refresh holds RefreshMu and waits on I/O.
	s.CommsFeeds()
	close(release)
	var readers sync.WaitGroup
	for range 4 {
		readers.Go(func() {
			for range 200 {
				s.CommsFeeds()
				s.CommsLeague()
			}
		})
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	for range 10 {
		if err := (commsHost{s}).RefreshComms(true); err != nil {
			t.Fatal(err)
		}
	}
	readers.Wait()
	tx, boards, _ := s.CommsFeeds()
	if len(tx.Value) != 1 || boards[0].FetchedAt.IsZero() || boards[1].FetchedAt.IsZero() {
		t.Fatal("publication not observed")
	}
}
