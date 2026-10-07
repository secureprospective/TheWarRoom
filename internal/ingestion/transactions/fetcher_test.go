package transactions

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func player(t *testing.T, raw string) leaguefeed.Asset {
	t.Helper()
	id, err := playerid.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	return leaguefeed.Asset{Player: &id}
}

func TestRealTransactions(t *testing.T) {
	body, err := os.ReadFile("testdata/transactions.json")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := ToTransactions(raw)
	if err != nil {
		t.Fatal(err)
	}
	counts := make(map[string]int)
	for _, row := range rows {
		counts[row.Kind]++
	}
	wantCounts := map[string]int{"FREE_AGENT": 460, "LOAD_ROSTERS": 112, "TRADE": 109, "IR": 47, "TAXI": 42}
	if len(rows) != 770 || !reflect.DeepEqual(counts, wantCounts) {
		t.Fatalf("rows %d counts %v", len(rows), counts)
	}
	empty := []leaguefeed.Asset{}
	cases := []struct {
		index int
		want  leaguefeed.Transaction
	}{
		{0, leaguefeed.Transaction{
			Kind: "IR", Time: time.Unix(1791338104, 0).UTC(), Franchise: "0024",
			IR: &leaguefeed.RosterChange{In: empty, Out: []leaguefeed.Asset{player(t, "16696"), player(t, "17166")}},
		}},
		{1, leaguefeed.Transaction{
			Kind: "FREE_AGENT", Time: time.Unix(1791336990, 0).UTC(), Franchise: "0020", ByCommish: true,
			AddsDrops: &leaguefeed.RosterChange{In: empty, Out: []leaguefeed.Asset{player(t, "16289")}},
		}},
		{9, leaguefeed.Transaction{
			Kind: "TAXI", Time: time.Unix(1790901568, 0).UTC(), Franchise: "0018",
			Taxi: &leaguefeed.RosterChange{In: []leaguefeed.Asset{player(t, "17469")},
				Out: []leaguefeed.Asset{player(t, "17762")}},
		}},
		{34, leaguefeed.Transaction{
			Kind: "LOAD_ROSTERS", Time: time.Unix(1790520354, 0).UTC(), Franchise: "0008", ByCommish: true,
			AddsDrops: &leaguefeed.RosterChange{In: empty, Out: []leaguefeed.Asset{player(t, "14904")}},
		}},
		{368, leaguefeed.Transaction{
			Kind: "TRADE", Time: time.Unix(1784747780, 0).UTC(), Franchise: "0024", ByCommish: true,
			Trade: &leaguefeed.Trade{
				Counterparty: "0021", Comments: "Included Duck for a salary match",
				Expires: time.Unix(1785348000, 0).UTC(),
				Gave: []leaguefeed.Asset{player(t, "16889"),
					{FuturePick: &leaguefeed.FuturePick{Franchise: "0024", Year: 2027, Round: 4}}},
				CounterpartyGave: []leaguefeed.Asset{
					{CurrentPick: &leaguefeed.CurrentPick{Round: 5, Pick: 10}},
					{CurrentPick: &leaguefeed.CurrentPick{Round: 5, Pick: 20}},
				},
			},
		}},
	}
	for _, tc := range cases {
		if !reflect.DeepEqual(rows[tc.index], tc.want) {
			t.Fatalf("index %d: got %+v, want %+v", tc.index, rows[tc.index], tc.want)
		}
	}
}

func TestSyntheticTransactions(t *testing.T) {
	// Synthetic: unknown kinds must not hide malformed known rows.
	unknown := `{"type":"NEW_KIND","franchise":"0001","timestamp":"100"}`
	raw, err := Parse([]byte(`{"transactions":{"transaction":` + unknown + `}}`))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := ToTransactions(raw)
	want := leaguefeed.Transaction{
		Kind: "NEW_KIND", Franchise: "0001", Time: time.Unix(100, 0).UTC(), Unparsed: true,
	}
	if err != nil || len(rows) != 1 || !reflect.DeepEqual(rows[0], want) {
		t.Fatalf("unknown: %+v, %v", rows, err)
	}
	for _, kind := range []string{"FREE_AGENT", "LOAD_ROSTERS", "TRADE", "IR", "TAXI"} {
		bad := `{"type":"` + kind + `","franchise":"0001","timestamp":"100"}`
		body := `{"transactions":{"transaction":[` + unknown + `,` + bad + `]}}`
		if _, err := Parse([]byte(body)); err == nil || !strings.Contains(err.Error(), "index 1") {
			t.Fatalf("%s: %v", kind, err)
		}
	}
	for _, body := range []string{`{}`, `{`, `{"error":{"$t":"denied"}}`,
		`{"transactions":{"transaction":{"type":"IR","franchise":"0001","timestamp":"x"}}}`} {
		if _, err := Parse([]byte(body)); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	raw.Transactions[0].Kind = "IR"
	if _, err := ToTransactions(raw); err == nil {
		t.Fatal("conversion skipped validation")
	}
}
