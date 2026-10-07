package pendingtrades

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func TestSyntheticPendingTrades(t *testing.T) {
	// Synthetic only: no values are derived from an authenticated league body.
	trade := `{"trade_id":"101","offeringteam":"0001","offeredto":"0002",` +
		`"will_give_up":"12345,","will_receive":"FP_0002_2030_1,BB_10.50,",` +
		`"timestamp":"100","expires":"200","comments":"synthetic","description":"example"}`
	id, err := playerid.New("12345")
	if err != nil {
		t.Fatal(err)
	}
	cents := int64(1050)
	want := leaguefeed.PendingTrade{
		ID: "101", Offering: "0001", OfferedTo: "0002", Gives: []leaguefeed.Asset{{Player: &id}},
		Gets: []leaguefeed.Asset{
			{FuturePick: &leaguefeed.FuturePick{Franchise: "0002", Year: 2030, Round: 1}},
			{BlindBidCents: &cents},
		},
		Proposed: time.Unix(100, 0).UTC(), Expires: time.Unix(200, 0).UTC(),
		Comments: "synthetic", Description: "example",
	}
	cases := []struct {
		name, block string
		count       int
	}{
		{"absent", `{}`, 0},
		{"empty object", `{"pendingTrade":{}}`, 0},
		{"one", `{"pendingTrade":` + trade + `}`, 1},
		{"two", `{"pendingTrade":[` + trade + `,` + strings.Replace(trade, "101", "102", 1) + `]}`, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"pendingTrades":` + tc.block + `}`
			raw, err := Parse([]byte(body))
			if err != nil {
				t.Fatal(err)
			}
			got, err := ToPendingTrades(raw)
			if err != nil || len(got) != tc.count {
				t.Fatalf("got %+v, %v", got, err)
			}
			for i, row := range got {
				expected := want
				if i == 1 {
					expected.ID = "102"
				}
				if !reflect.DeepEqual(row, expected) {
					t.Fatalf("perspective: got %+v want %+v", row, expected)
				}
			}
			c, err := mfl.New("api", 10000, mfl.WithTransport(fixtureTransport{body, 200}),
				mfl.WithKeySource(func(context.Context) (mflkey.Key, error) { return "synthetic-key", nil }))
			if err != nil {
				t.Fatal(err)
			}
			fetched, err := Fetch(context.Background(), c, "2030", "league")
			if err != nil || !reflect.DeepEqual(fetched, raw) {
				t.Fatalf("fetch: %+v, %v", fetched, err)
			}
		})
	}
	for _, body := range []string{`{"error":{"$t":"denied"}}`, `{"error":{"$t":""}}`} {
		if _, err := Parse([]byte(body)); !errors.Is(err, ErrRejected) {
			t.Fatalf("error envelope: %v", err)
		}
	}
	bad := strings.Replace(trade, "12345,", "UNKNOWN,", 1)
	body := `{"pendingTrades":{"pendingTrade":[` + trade + `,` + bad + `]}}`
	if _, err := Parse([]byte(body)); err == nil || !strings.Contains(err.Error(), "index 1") {
		t.Fatalf("malformed known trade: %v", err)
	}
}
