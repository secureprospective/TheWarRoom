package lineup

import (
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
)

func TestByeShortfalls(t *testing.T) {
	rules, err := ParseRules(league.RawConfig{Starters: league.Starters{Count: "3", Positions: []league.PositionLimit{
		{Name: "QB", Limit: "1"}, {Name: "WR", Limit: "2-3"}}}})
	if err != nil {
		t.Fatal(err)
	}
	players := []ByePlayer{
		{Position: domain.PosQB, Team: "BUF", Active: true},
		{Position: domain.PosQB, Team: "IND", Active: false},
		{Position: domain.PosWR, Team: "BUF", Active: true},
		{Position: domain.PosWR, Team: "IND", Active: true},
		{Position: domain.PosWR, Team: "FA", Active: true},
	}
	got := ByeShortfalls(rules, players, map[string]int{"BUF": 7, "IND": 13})
	want := []ByeShortfall{{Week: 7, Position: domain.PosQB, Available: 0, Required: 1},
		{Week: 7, Position: domain.PosWR, Available: 1, Required: 2},
		{Week: 13, Position: domain.PosWR, Available: 1, Required: 2}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shortfall: %+v", got)
	}
	if got := ByeShortfalls(rules, players, nil); len(got) != 0 {
		t.Fatalf("invented bye dates: %v", got)
	}
}
