package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/rosters"
	feedtransactions "github.com/secureprospective/TheWarRoom/internal/ingestion/transactions"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func commsFixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("internal", "ingestion", "testdata", "comms", name))
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func realComms(t *testing.T) commsInput {
	t.Helper()
	race := realRace(t, "2026")
	raw, err := feedtransactions.Parse([]byte(seasonFixture(t, "transactions", "transactions.json")))
	if err != nil {
		t.Fatal(err)
	}
	txs, err := feedtransactions.ToTransactions(raw)
	if err != nil {
		t.Fatal(err)
	}
	gz, err := gzip.NewReader(bytes.NewReader(commsFixture(t, "players-2026.json.gz")))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(gz)
	if err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	rawPlayers, err := players.Parse(context.Background(), body)
	if err != nil {
		t.Fatal(err)
	}
	lookup, err := normalize.NewLookup(rawPlayers)
	if err != nil {
		t.Fatal(err)
	}
	in := commsInput{transactions: held(txs), lookup: lookup, names: map[string]string{}, zone: time.UTC, mirror: state.MirrorSnapshot{Season: 2026}}
	for _, f := range race.config.Franchises {
		in.names[f.ID] = f.Name
	}
	for i, export := range []string{"messageBoard", "polls"} {
		posts, err := ingestion.ParseLeagueBoard(commsFixture(t, export+"-2026.json"), export)
		if err != nil {
			t.Fatal(err)
		}
		in.boards[i] = held(posts)
	}
	in.mirror.Players = commsRoster(t)
	return in
}

func commsRoster(t *testing.T) []state.MirrorPlayer {
	t.Helper()
	var raw struct {
		Rosters struct {
			Franchises []struct {
				ID      string `json:"id"`
				Players []struct {
					ID     string `json:"id"`
					Salary string `json:"salary"`
					Year   string `json:"contractYear"`
					Status string `json:"status"`
				} `json:"player"`
			} `json:"franchise"`
		} `json:"rosters"`
	}
	if err := json.Unmarshal(commsFixture(t, "rosters-2026.json"), &raw); err != nil {
		t.Fatal(err)
	}
	out := []state.MirrorPlayer{}
	for _, f := range raw.Rosters.Franchises {
		for _, p := range f.Players {
			rr := rosters.RawRoster{FranchiseID: f.ID, PlayerID: p.ID, Salary: p.Salary, ContractYear: p.Year, Status: p.Status}
			if err := rr.Validate(); err != nil {
				t.Fatal(err)
			}
			rec, err := normalize.Contract(rr)
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, state.MirrorPlayer{MFLID: rec.MFLID.String(), FranchiseID: f.ID, Salary: rec.Salary, ContractYear: rec.ContractYear})
		}
	}
	return out
}

func TestCommsRealSentencesAndFullReport(t *testing.T) {
	in := realComms(t)
	out, err := buildComms(in, RawCommsScope{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Events) != len(in.transactions.value) || len(out.Transactions.Rows) != len(out.Events) {
		t.Fatal("lost transaction")
	}
	expected := []string{
		"Minnesota Vikings placed Edgerrin Cooper (LB, GBP) on IR; placed Caleb Ransaw (S, JAC) on IR.",
		"Denver Broncos added Akayleb Evans (CB, CAR) ($580,000, contract through 2026).",
		"New Orleans Saints traded Ben Sauls (K, FA), Azeez Ojulari (DE, LVR), Bub Means (WR, CLE), New Orleans Saints' 2027 round 1 pick, New Orleans Saints' 2027 round 2 pick, New Orleans Saints' 2028 round 1 pick to Carolina Panthers for Dalton Kincaid (TE, BUF), Eric Wilson (LB, MIN).",
	}
	for _, sentence := range expected {
		found := false
		for _, event := range out.Events {
			found = found || event.Sentence == sentence
		}
		if !found {
			t.Fatal("missing real sentence", sentence)
		}
	}
	if out.BoardNote != "No posts on MFL's message board or polls this season." {
		t.Fatal(out.BoardNote)
	}
	var last float64
	for i, row := range out.Transactions.Rows {
		if row.Cells[1] != out.Events[i].Sentence || row.Cells[0] != out.Events[i].Date {
			t.Fatal("second wording")
		}
		if i > 0 && row.Sort[0] > last {
			t.Fatal("not newest first")
		}
		last = row.Sort[0]
		if strings.Contains(row.Cells[1], "name not held") {
			t.Fatal(row)
		}
	}
	cases := []struct {
		scope    RawCommsScope
		contains []string
	}{
		{RawCommsScope{Kind: "trade"}, []string{" traded ", " to ", " for ", "pick"}},
		{RawCommsScope{Kind: "ir"}, []string{") on IR", " ("}},
		{RawCommsScope{Kind: "taxi"}, []string{"taxi"}},
		{RawCommsScope{Kind: "add-drop"}, []string{"added", ", contract through 20"}},
	}
	for _, tc := range cases {
		view, err := buildComms(in, tc.scope)
		if err != nil {
			t.Fatal(err)
		}
		all := ""
		for _, e := range view.Events {
			all += e.Sentence + "\n"
		}
		for _, want := range tc.contains {
			if !strings.Contains(all, want) {
				t.Fatalf("%s missing %s", tc.scope.Kind, want)
			}
		}
	}
	if !strings.Contains(out.Events[0].Date, "2026,") || !strings.Contains(out.Events[0].Date, "AM") {
		t.Fatal(out.Events[0].Date)
	}
}

func TestCommsFiltersIncludeBothTradeSides(t *testing.T) {
	in := realComms(t)
	for _, tx := range in.transactions.value {
		if tx.Trade == nil {
			continue
		}
		scope := RawCommsScope{Team: tx.Trade.Counterparty, Kind: "trade"}
		out, err := buildComms(in, scope)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, e := range out.Events {
			found = found || e.Sentence == in.transactionSentence(tx)
		}
		if !found {
			t.Fatal("counterparty lost trade")
		}
		break
	}
	for _, scope := range []RawCommsScope{{Team: "001"}, {Team: "9999"}, {Kind: "post"}, {Kind: "TRADE"}} {
		if _, err := buildComms(in, scope); err == nil {
			t.Fatal("accepted invalid filter", scope)
		}
	}
}

func TestCommsBoardAuthenticationAndPosts(t *testing.T) {
	if _, err := ingestion.ParseLeagueBoard(commsFixture(t, "messageBoard-2013.json"), "messageBoard"); err == nil || !strings.Contains(err.Error(), "not available") {
		t.Fatal(err)
	}
	polls, err := ingestion.ParseLeagueBoard([]byte(`{"polls":{"poll":{"title":"Playoff vote","franchise":"0001","timestamp":"1791728340"}}}`), "polls")
	if err != nil || len(polls) != 1 {
		t.Fatal(polls, err)
	}
	posts, err := ingestion.ParseLeagueBoard([]byte(`{"messageBoard":{"thread":{"title":"League ruling","franchise":"0001","timestamp":"1791728340"}}}`), "messageBoard")
	if err != nil {
		t.Fatal(err)
	}
	in := realComms(t)
	in.boards[0] = held(posts)
	in.boards[1] = held(polls)
	out, err := buildComms(in, RawCommsScope{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out.BoardPosts[0].Rows[0].Cells, []string{"League ruling", "Buffalo Bills", "Oct 11, 2026, 2:19 PM"}) {
		t.Fatal(out.BoardPosts[0].Rows)
	}
	in.boards[0] = heldSeasonFeed[[]ingestion.RawLeaguePost]{refreshError: "API requires logged in user"}
	out, err = buildComms(in, RawCommsScope{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.BoardNote, "not available") || strings.Contains(out.BoardNote, "No posts") {
		t.Fatal(out.BoardNote)
	}
	if _, err := ingestion.ParseLeagueBoard([]byte(`{"messageBoard":{"posts":[{"title":"Unknown shape"}]}}`), "messageBoard"); err == nil {
		t.Fatal("unknown board read as empty")
	}
	in.boards[0] = held(posts)
	in.boards[0].refreshError = "API requires logged in user"
	out, err = buildComms(in, RawCommsScope{})
	if err != nil || out.BoardPosts[0].Status != "stale" || len(out.BoardPosts[0].Rows) != 1 || strings.Contains(out.BoardNote, "No posts") {
		t.Fatal(out, err)
	}
	if _, err := ingestion.ParseLeagueBoard([]byte(`{"polls":{"poll":{"title":"Vote","franchise":"bad","timestamp":"x"}}}`), "polls"); err == nil {
		t.Fatal("invalid post passed")
	}
}

func TestCommsHandOffURLs(t *testing.T) {
	for _, tc := range []struct{ destination, host, want string }{
		{"league-board", "", "https://legacynfl.boards.net/"},
		{"message-board", "fixture.mfl.invalid", "https://fixture.mfl.invalid/2026/options?L=14432&O=28"},
		{"polls", "fixture.mfl.invalid", "https://fixture.mfl.invalid/2026/options?L=14432&O=69"},
	} {
		got, err := commsURL(tc.destination, tc.host, 2026)
		if err != nil || got != tc.want {
			t.Fatal(got, err)
		}
	}
	if _, err := commsURL("polls", "", 2026); err == nil {
		t.Fatal("missing host passed")
	}
	if _, err := commsURL("https://outside.invalid", "fixture.invalid", 2026); err == nil {
		t.Fatal("arbitrary URL passed")
	}
	a := targetTestApp(t)
	opened := ""
	a.openURL = func(_ context.Context, url string) { opened = url }
	if err := a.TargetOpenComms("league-board"); err != nil || opened != "https://legacynfl.boards.net/" {
		t.Fatal(opened, err)
	}
}

func TestCommsHeldBindingAndWire(t *testing.T) {
	in := realComms(t)
	a := targetTestApp(t)
	a.seasonTransactions, a.commsBoards, a.lookup = in.transactions, in.boards, in.lookup
	if _, err := a.league.Replace(a.ctx, in.mirror); err != nil {
		t.Fatal(err)
	}
	out, err := a.TargetComms(RawCommsScope{})
	if err != nil || len(out.Events) != len(in.transactions.value) {
		t.Fatal(err)
	}
	out, err = buildComms(in, RawCommsScope{})
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
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, append(body, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
	}
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
	reports, _ := commsPosts(commsInput{boards: a.commsBoards, zone: time.UTC})
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

func TestCommsMissingTermsAndSources(t *testing.T) {
	in := realComms(t)
	in.mirror.Players = nil
	for _, tx := range in.transactions.value {
		if tx.AddsDrops == nil || len(tx.AddsDrops.In) == 0 {
			continue
		}
		if !strings.Contains(in.transactionSentence(tx), "salary not held") {
			t.Fatal("invented terms")
		}
		break
	}
	in.transactions = heldSeasonFeed[[]leaguefeed.Transaction]{}
	out, err := buildComms(in, RawCommsScope{})
	if err != nil || out.Feed.Status != "failed" || out.Feed.Note == "" {
		t.Fatal(out, err)
	}
}
