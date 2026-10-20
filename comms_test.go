package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	commsapp "github.com/secureprospective/TheWarRoom/internal/app/comms"
	"github.com/secureprospective/TheWarRoom/internal/app/comms/commstest"
	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/zalando/go-keyring"
)

func TestCommsOpenBinding(t *testing.T) {
	a := targetTestApp(t)
	opened := ""
	a.openURL = func(_ context.Context, url string) { opened = url }
	if err := a.TargetOpenComms("league-board"); err != nil || opened != "https://legacynfl.boards.net/" {
		t.Fatal(opened, err)
	}
}

func TestCommsHeldBindingAndWire(t *testing.T) {
	in := commstest.Real(t)
	a := targetTestApp(t)
	installCommsFixture(a, in)
	if _, err := a.league.Replace(a.ctx, in.Mirror); err != nil {
		t.Fatal(err)
	}
	out, err := a.TargetComms(RawCommsScope{})
	if err != nil || len(out.Events) != len(in.Transactions.Value) {
		t.Fatal(err)
	}
	built, err := commsapp.Build(in, commsapp.RawCommsScope{})
	out = wireComms(built)
	if err != nil {
		t.Fatal(err)
	}
	events, rows := out.Events[:1:1], out.Transactions.Rows[:1:1]
	for _, phrase := range []string{" traded ", " added "} {
		for i, event := range out.Events {
			if !strings.Contains(event.Sentence, phrase) {
				continue
			}
			events = append(events, event)
			rows = append(rows, out.Transactions.Rows[i])
			break
		}
	}
	out.Events, out.Transactions.Rows = events, rows
	body, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("frontend", "src", "app", "comms", "wire.testdata.json")

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(want)) != string(body) {
		t.Fatal("Comms wire is stale")
	}
}

func TestCommsBoardWithoutKeyAsksNothingAndSaysHowToConnect(t *testing.T) {
	keyring.MockInit()
	a, transport, _, _ := seasonTestApp(t)
	a.seasonTransactions.fetchedAt = time.Now()
	a.commsBoards[0] = heldSeasonFeed[[]ingestion.RawLeaguePost]{value: []ingestion.RawLeaguePost{{Title: "kept"}},
		fetchedAt: time.Now()}
	if err := a.TargetCommsRefresh(true); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(transport.requests, "messageBoard") || slices.Contains(transport.requests, "polls") {
		t.Fatal("board requested without a key", transport.requests)
	}
	if a.commsBoards[0].value != nil {
		t.Fatal("a member-only board outlived the key")
	}
	built, err := a.TargetComms(RawCommsScope{})
	if err != nil {
		t.Fatal(err)
	}
	reports := built.BoardPosts
	if !strings.Contains(reports[0].Note, "connect your MFL key in Control Room › App") {
		t.Fatal(reports[0].Note)
	}
}

func TestCommsRefreshUsesExistingTransactionFetcher(t *testing.T) {
	keyring.MockInit()
	a, transport, pauses, _ := seasonTestApp(t)
	if err := a.keyStore.Set(context.Background(), mflkey.Key("fixture-key")); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	a.seasonTransactions.fetchedAt = now
	for i := range a.commsBoards {
		a.commsBoards[i].fetchedAt = now
	}
	if err := a.TargetCommsRefresh(false); err != nil {
		t.Fatal(err)
	}
	if len(transport.requests) != 0 {
		t.Fatal("fresh feeds fetched again")
	}
	if err := a.TargetCommsRefresh(true); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(transport.keyed, []string{"messageBoard", "polls"}) {
		t.Fatal("the member-only board must be read with the key", transport.keyed)
	}
	if !reflect.DeepEqual(transport.requests, []string{"transactions", "messageBoard", "polls"}) || len(*pauses) != 2 {
		t.Fatal(transport.requests, *pauses)
	}
	transport.failure = "messageBoard"
	if err := a.TargetCommsRefresh(true); err != nil {
		t.Fatal(err)
	}
	if a.commsBoards[0].fetchedAt.IsZero() || a.commsBoards[0].refreshError == "" {
		t.Fatal("failed refresh lost last good board")
	}
}

func TestCommsBindingGoldens(t *testing.T) {
	zone := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = zone })
	in := commstest.Real(t)
	a := targetTestApp(t)
	installCommsFixture(a, in)
	if _, err := a.league.Replace(a.ctx, in.Mirror); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		scope RawCommsScope
	}{
		{"all", RawCommsScope{}}, {"trade", RawCommsScope{Kind: "trade", Team: "0001"}},
		{"invalid", RawCommsScope{Kind: "post"}}, {"unknown", RawCommsScope{Team: "9999"}},
	} {
		board, err := a.TargetComms(tc.scope)
		checkCommsGolden(t, tc.name, struct {
			Board CommsBoard
			Error string
		}{board, commsError(err)})
	}
	opened := ""
	a.openURL = func(_ context.Context, url string) { opened = url }
	for _, destination := range []string{"league-board", "polls", "invalid"} {
		opened = ""
		err := a.TargetOpenComms(destination)
		checkCommsGolden(t, "open-"+destination, struct{ URL, Error string }{opened, commsError(err)})
	}
	keyring.MockInit()
	fresh, transport, pauses, _ := seasonTestApp(t)
	if err := fresh.keyStore.Set(context.Background(), mflkey.Key("fixture-key")); err != nil {
		t.Fatal(err)
	}
	err := fresh.TargetCommsRefresh(true)
	checkCommsGolden(t, "refresh", struct {
		Error           string
		Requests, Keyed []string
		Pauses          int
	}{
		commsError(err), transport.requests, transport.keyed, len(*pauses)})
}

func installCommsFixture(a *App, in commsapp.Input) {
	a.seasonTransactions = heldSeasonFeed[[]leaguefeed.Transaction]{value: in.Transactions.Value, fetchedAt: in.Transactions.FetchedAt}
	for i, feed := range in.Boards {
		a.commsBoards[i] = heldSeasonFeed[[]ingestion.RawLeaguePost]{value: feed.Value, fetchedAt: feed.FetchedAt}
	}
	a.lookup = in.Lookup
}

func TestCommsWireTwinsMatch(t *testing.T) {
	var board commsapp.Board
	n := 0
	fillCounterWire(reflect.ValueOf(&board).Elem(), &n)
	assertCounterTwin(t, board, wireComms(board))
	assertCounterTwin(t, commsapp.Board{}, wireComms(commsapp.Board{}))
}

func commsError(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

func checkCommsGolden[T any](t *testing.T, name string, value T) {
	t.Helper()
	got, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	dir := "testdata/wire/comms"
	path := filepath.Join(dir, name+".json")
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s: Comms binding contract changed", path)
	}
}

// The Now landing scenario retains its fixture facade without owning Comms construction.
type commsLandingInput struct {
	commsapp.Input
	mirror      state.MirrorSnapshot
	rosters     facts.RosterWeekColumns
	rosterTimes map[[2]int64]int64
}

func realComms(t *testing.T) commsLandingInput {
	t.Helper()
	in := commstest.Real(t)
	return commsLandingInput{Input: in, mirror: in.Mirror, rosters: in.Rosters, rosterTimes: in.RosterTimes}
}

func rosterWeekTimes(games facts.NflGameColumns) map[[2]int64]int64 {
	return commsapp.RosterWeekTimes(games)
}

func (in commsLandingInput) transactionSentence(tx leaguefeed.Transaction) string {
	in.Mirror, in.Rosters, in.RosterTimes = in.mirror, in.rosters, in.rosterTimes
	in.Transactions.Value = []leaguefeed.Transaction{tx}
	board, err := commsapp.Build(in.Input, commsapp.RawCommsScope{})
	if err != nil {
		return err.Error()
	}
	return board.Events[0].Sentence
}
