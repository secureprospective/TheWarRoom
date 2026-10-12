package lineup

import (
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func TestByeShortfalls(t *testing.T) {
	rules := Rules{Positions: []PositionRule{{Position: domain.PosQB, Min: 1}, {Position: domain.PosWR, Min: 2}}}
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
