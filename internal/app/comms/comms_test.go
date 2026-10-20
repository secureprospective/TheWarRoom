package comms_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/comms"
	"github.com/secureprospective/TheWarRoom/internal/app/comms/commstest"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func TestCommsRealSentencesAndFullReport(t *testing.T) {
	in := commstest.Real(t)
	out, err := comms.Build(in, comms.RawCommsScope{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Events) != len(in.Transactions.Value) || len(out.Transactions.Rows) != len(out.Events) {
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
		scope    comms.RawCommsScope
		contains []string
	}{
		{comms.RawCommsScope{Kind: "trade"}, []string{" traded ", " to ", " for ", "pick"}},
		{comms.RawCommsScope{Kind: "ir"}, []string{") on IR", " ("}},
		{comms.RawCommsScope{Kind: "taxi"}, []string{"taxi"}},
		{comms.RawCommsScope{Kind: "add-drop"}, []string{"added", ", contract through 20"}},
	}
	for _, tc := range cases {
		view, err := comms.Build(in, tc.scope)
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
	in := commstest.Real(t)
	for _, tx := range in.Transactions.Value {
		if tx.Trade == nil {
			continue
		}
		scope := comms.RawCommsScope{Team: tx.Trade.Counterparty, Kind: "trade"}
		out, err := comms.Build(in, scope)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, e := range out.Events {
			found = found || e.Sentence == sentence(t, in, tx)
		}
		if !found {
			t.Fatal("counterparty lost trade")
		}
		break
	}
	for _, scope := range []comms.RawCommsScope{{Team: "001"}, {Team: "9999"}, {Kind: "post"}, {Kind: "TRADE"}} {
		if _, err := comms.Build(in, scope); err == nil {
			t.Fatal("accepted invalid filter", scope)
		}
	}
}

func TestCommsBoardAuthenticationAndPosts(t *testing.T) {
	if _, err := ingestion.ParseLeagueBoard(commstest.Fixture(t, "messageBoard-2013.json"), "messageBoard"); err == nil || !strings.Contains(err.Error(), "not available") {
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
	in := commstest.Real(t)
	in.Boards[0] = commstest.Held(posts)
	in.Boards[1] = commstest.Held(polls)
	out, err := comms.Build(in, comms.RawCommsScope{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out.BoardPosts[0].Rows[0].Cells, []string{"League ruling", "Buffalo Bills", "Oct 11, 2026, 2:19 PM"}) {
		t.Fatal(out.BoardPosts[0].Rows)
	}
	in.Boards[0] = report.Held[ingestion.RawLeaguePost]{Feed: report.Feed{RefreshError: "API requires logged in user"}}
	out, err = comms.Build(in, comms.RawCommsScope{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.BoardNote, "not available") || strings.Contains(out.BoardNote, "No posts") {
		t.Fatal(out.BoardNote)
	}
	if _, err := ingestion.ParseLeagueBoard([]byte(`{"messageBoard":{"posts":[{"title":"Unknown shape"}]}}`), "messageBoard"); err == nil {
		t.Fatal("unknown board read as empty")
	}
	in.Boards[0] = commstest.Held(posts)
	in.Boards[0].RefreshError = "API requires logged in user"
	out, err = comms.Build(in, comms.RawCommsScope{})
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
		got, err := comms.URL(tc.destination, tc.host, 2026)
		if err != nil || got != tc.want {
			t.Fatal(got, err)
		}
	}
	if _, err := comms.URL("polls", "", 2026); err == nil {
		t.Fatal("missing host passed")
	}
	if _, err := comms.URL("https://outside.invalid", "fixture.invalid", 2026); err == nil {
		t.Fatal("arbitrary URL passed")
	}
}

func TestCommsMissingTermsAndSources(t *testing.T) {
	in := commstest.Real(t)
	in.Mirror.Players = nil
	for _, tx := range in.Transactions.Value {
		if tx.AddsDrops == nil || len(tx.AddsDrops.In) == 0 {
			continue
		}
		if !strings.Contains(sentence(t, in, tx), "salary not held") {
			t.Fatal("invented terms")
		}
		break
	}
	in.Transactions = report.Held[leaguefeed.Transaction]{}
	out, err := comms.Build(in, comms.RawCommsScope{})
	if err != nil || out.Feed.Status != "failed" || out.Feed.Note == "" {
		t.Fatal(out, err)
	}
}

func sentence(t *testing.T, in comms.Input, tx leaguefeed.Transaction) string {
	t.Helper()
	in.Transactions.Value = []leaguefeed.Transaction{tx}
	board, err := comms.Build(in, comms.RawCommsScope{})
	if err != nil {
		t.Fatal(err)
	}
	return board.Events[0].Sentence
}
