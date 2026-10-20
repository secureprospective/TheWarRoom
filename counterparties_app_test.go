package main

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	counterpartiesapp "github.com/secureprospective/TheWarRoom/internal/app/counterparties"

	"github.com/secureprospective/TheWarRoom/internal/lab"
)

func ptrFloat(v float64) *float64 { return &v }

func TestDraftToneWordsOnScreen(t *testing.T) {
	t.Parallel()
	for tone, want := range map[string]string{
		"up":   "above its slot beyond chance",
		"down": "below its slot beyond chance",
		"even": "within chance of its slot",
	} {
		line := draftCellLine(lab.DraftCell{Tone: tone, Ratio: ptrFloat(1.1), Lo: ptrFloat(1.0), Hi: ptrFloat(1.2)})
		if line.Detail != want {
			t.Fatalf("tone %s rendered %q, want %q", tone, line.Detail, want)
		}
	}
	for tone, want := range map[string]string{
		"up":   "above baseline beyond chance",
		"down": "below baseline beyond chance",
		"even": "within chance of baseline",
	} {
		h := lab.DraftHits{Rate: 0.4, Baseline: 0.2, Tone: tone}
		section := draftHitsSection(lab.DraftPosition{Hits: &h})
		if !strings.Contains(section.Rows[0].Detail, want) {
			t.Fatalf("hits tone %s missing words %q in %q", tone, want, section.Rows[0].Detail)
		}
		if strings.Contains(section.Rows[0].Detail, "· "+tone) {
			t.Fatalf("raw tone token reached the screen: %q", section.Rows[0].Detail)
		}
	}
}

func TestCountLabelPlural(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		n          int
		noun, want string
	}{
		{1, "record", "1 record"}, {2, "record", "2 records"},
		{1, "pick", "1 pick"}, {0, "trade", "0 trades"},
		{1, "trade", "1 trade"}, {1, "trade side", "1 trade side"},
	} {
		if got := countLabel(c.n, c.noun); got != c.want {
			t.Errorf("countLabel(%d, %q) = %q, want %q", c.n, c.noun, got, c.want)
		}
	}
}

func TestCounterpartiesSlotTone(t *testing.T) {
	for tone, want := range map[string]string{
		"up":   "above its slot beyond chance",
		"down": "below its slot beyond chance",
		"even": "within chance of its slot",
	} {
		if slotTone(tone) != want {
			t.Errorf("slot tone %s rendered %q", tone, slotTone(tone))
		}
	}
}

func TestCounterpartiesWireMatchesGolden(t *testing.T) {
	a := NewApp()
	for _, c := range []struct {
		name  string
		scope RawCounterpartiesScope
		fid   string
	}{
		{"league", RawCounterpartiesScope{From: 2017, To: 2026}, "0001"},
		{"filtered", RawCounterpartiesScope{From: 2017, To: 2026, Tiers: []string{"contender"}, Postures: []string{"now"}}, "0014"},
		{"thin", RawCounterpartiesScope{From: 2017, To: 2017, Windows: []string{"summer"}, Postures: []string{"later"}}, "0002"},
		{"invalid", RawCounterpartiesScope{From: 2026, To: 2017}, "0099"},
		{"unknown", RawCounterpartiesScope{From: 2017, To: 2026, Team: "0099"}, "0099"},
	} {
		checkCounterpartiesWire(t, c.name, a.TargetCounterparties(c.scope))
		checkCounterpartiesWire(t, c.name+"-team", a.TargetCounterpartTeam(RawCounterpartTeam{Scope: c.scope, Fid: c.fid}))
	}
}

func checkCounterpartiesWire[T any](t *testing.T, name string, value T) {
	t.Helper()
	got, err := json.MarshalIndent(value, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	path := "testdata/wire/counterparties/" + name + ".json"
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.MkdirAll("testdata/wire/counterparties", 0o750); err != nil {
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
	if !bytes.Equal(want, got) {
		t.Errorf("%s: Counterparties wire contract changed", path)
	}
}

func TestCounterpartiesWireTwinsMatch(t *testing.T) {
	t.Parallel()
	var board counterpartiesapp.Counterparties
	var team counterpartiesapp.TeamCounterpart
	n := 0
	fillCounterWire(reflect.ValueOf(&board).Elem(), &n)
	fillCounterWire(reflect.ValueOf(&team).Elem(), &n)
	assertCounterTwin(t, board, wireCounterparties(board))
	assertCounterTwin(t, team, wireTeamCounterpart(team))
	assertCounterTwin(t, counterpartiesapp.Counterparties{}, wireCounterparties(counterpartiesapp.Counterparties{}))
	assertCounterTwin(t, counterpartiesapp.TeamCounterpart{}, wireTeamCounterpart(counterpartiesapp.TeamCounterpart{}))
}

func assertCounterTwin[S, W any](t *testing.T, source S, wire W) {
	t.Helper()
	want, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, got) {
		t.Fatalf("wire twin differs\nservice %s\nwire %s", want, got)
	}
}

func fillCounterWire(v reflect.Value, n *int) {
	if !v.CanSet() {
		return
	}
	if v.Kind() == reflect.Struct {
		for i := range v.NumField() {
			fillCounterWire(v.Field(i), n)
		}
		return
	}
	if v.Kind() == reflect.Pointer {
		v.Set(reflect.New(v.Type().Elem()))
		fillCounterWire(v.Elem(), n)
		return
	}
	if v.Kind() == reflect.Array || v.Kind() == reflect.Slice {
		if v.Kind() == reflect.Slice {
			v.Set(reflect.MakeSlice(v.Type(), 2, 2))
		}
		for i := range v.Len() {
			fillCounterWire(v.Index(i), n)
		}
		return
	}
	if v.Kind() == reflect.Int {
		*n++
		v.SetInt(int64(*n))
		return
	}
	if v.Kind() == reflect.Map {
		v.Set(reflect.MakeMap(v.Type()))
		key := reflect.New(v.Type().Key()).Elem()
		value := reflect.New(v.Type().Elem()).Elem()
		fillCounterWire(key, n)
		fillCounterWire(value, n)
		v.SetMapIndex(key, value)
		return
	}
	fillWire(v, n)
}
