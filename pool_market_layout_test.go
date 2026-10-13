package main

import (
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func TestPoolMarketLines(t *testing.T) {
	id, err := playerid.New("13589")
	if err != nil {
		t.Fatal(err)
	}
	held := map[string]heldSeasonFeed[[]marketfeeds.Feed]{}
	for _, kind := range marketKinds() {
		held[kind] = heldSeasonFeed[[]marketfeeds.Feed]{fetchedAt: time.Now(), value: []marketfeeds.Feed{{Week: 6, Players: []marketfeeds.Player{{ID: id, Value: 12.4, Low: 88, High: 140, Rank: 45, Last: 48, Change: 3}}}}}
	}
	adp := held[marketfeeds.ADP]
	adp.value[0].Players[0].Value = 112
	held[marketfeeds.ADP] = adp
	aav := held[marketfeeds.AAV]
	aav.value[0].Players[0].Value = 1.2
	held[marketfeeds.AAV] = aav
	facts := marketFacts(held)
	want := []string{"Proj 12.4 · Rank 45, Up 3", "ADP 112 (88–140) · AAV $1.2 on MFL’s $1,000 scale"}
	if got := poolMarketLines(facts["13589"]); !slices.Equal(got, want) {
		t.Fatal(got)
	}
	if got := poolMarketLines(facts[""]); !slices.Equal(got, []string{"No MFL market data"}) {
		t.Fatal(got)
	}
	if facts["13589"][2].Range == nil || facts["13589"][3].Range == nil {
		t.Fatal("Inspector ranges lost")
	}
}

func TestPoolMarketSortsMissingLast(t *testing.T) {
	for _, key := range []string{"projected", "rank", "adp", "aav"} {
		for _, desc := range []bool{false, true} {
			t.Run(key+map[bool]string{false: " ascending", true: " descending"}[desc], func(t *testing.T) {
				if err := (RawPoolScope{Sort: key, Desc: desc}).Validate(); err != nil {
					t.Fatal(err)
				}
				rows := []PoolRow{{ID: "missing", Name: "A missing"}, {ID: "high", Name: "B high"}, {ID: "zero", Name: "Z zero"}, {ID: "tie", Name: "C tie"}}
				for i := 1; i < len(rows); i++ {
					value := 12.0
					if i == 2 {
						value = 0
					}
					for range 4 {
						rows[i].Market = append(rows[i].Market, MarketFact{value: &value, Text: "Not a number"})
					}
				}
				sortPool(rows, RawPoolScope{Sort: key, Desc: desc})
				want := []string{"zero", "high", "tie", "missing"}
				if desc {
					want = []string{"tie", "high", "zero", "missing"}
				}
				got := []string{}
				for _, r := range rows {
					got = append(got, r.ID)
				}
				if !slices.Equal(got, want) {
					t.Fatalf("%v, want %v", got, want)
				}
			})
		}
	}
}

func TestMarketReportPlayerContext(t *testing.T) {
	id, err := playerid.New("13593")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := poolApp(t)
	a.seasonMarket = map[string]heldSeasonFeed[[]marketfeeds.Feed]{}
	for _, kind := range marketKinds() {
		a.seasonMarket[kind] = heldSeasonFeed[[]marketfeeds.Feed]{fetchedAt: time.Now(), value: []marketfeeds.Feed{{Week: a.marketWeek(), Players: []marketfeeds.Player{{ID: id, Value: 12, Low: 1, High: 20, Rank: 1}}}}}
	}
	board, err := a.TargetMarketReports()
	if err != nil {
		t.Fatal(err)
	}
	for _, report := range board.Reports {
		if report.Rows[0].Cells[0] != "Smith, John · WR · DAL" {
			t.Fatal(report.ID, report.Rows[0].Cells[0])
		}
	}
}
