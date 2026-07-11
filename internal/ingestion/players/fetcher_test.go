package players

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRawPlayer_Validate(t *testing.T) {
	base := RawPlayer{ID: "0531", Name: "Mahomes, Patrick", Position: "QB", Team: "KCC"}

	tests := []struct {
		name    string
		mutate  func(p *RawPlayer)
		wantErr bool
	}{
		{"valid", func(*RawPlayer) {}, false},
		{"empty id", func(p *RawPlayer) { p.ID = "" }, true},
		{"non-numeric id", func(p *RawPlayer) { p.ID = "abc" }, true},
		{"missing position", func(p *RawPlayer) { p.Position = "" }, true},
		// Name/Team are NOT validated here — an aggregate record carrying no name
		// is dropped at normalize by position, so the fetcher tolerates it.
		{"empty name tolerated", func(p *RawPlayer) { p.Name = "" }, false},
		{"empty team tolerated", func(p *RawPlayer) { p.Team = "" }, false},
		// Raw position codes pass the fetcher untouched; normalize maps/flags them.
		{"raw EDGE code tolerated", func(p *RawPlayer) { p.Position = "EDGE" }, false},
		// Birthdate (DETAILS=1): absent is legitimate (commissioner-created players);
		// a present value must be a parseable epoch-seconds integer.
		{"absent birthdate tolerated", func(p *RawPlayer) { p.Birthdate = "" }, false},
		{"valid epoch birthdate", func(p *RawPlayer) { p.Birthdate = "239432400" }, false},
		{"non-numeric birthdate", func(p *RawPlayer) { p.Birthdate = "1979-08-03" }, true},
		// DraftYear (DETAILS=1): absent is legitimate; a present value must parse as an integer
		// (the "0"/"1970" undrafted sentinels are numeric and pass — the consumer judges them).
		{"absent draft year tolerated", func(p *RawPlayer) { p.DraftYear = "" }, false},
		{"valid draft year", func(p *RawPlayer) { p.DraftYear = "2020" }, false},
		{"undrafted sentinel tolerated", func(p *RawPlayer) { p.DraftYear = "0" }, false},
		{"non-numeric draft year", func(p *RawPlayer) { p.DraftYear = "rookie" }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := base
			tt.mutate(&p)
			if err := p.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestDecode_RealShape decodes a trimmed copy of the live players payload to lock
// the field mapping, the single-element MFLList collapse on player[], and the
// raw passthrough of position codes / rookie status.
func TestDecode_RealShape(t *testing.T) {
	const body = `{"players":{"timestamp":"1750000000","player":[` +
		`{"id":"0531","name":"Mahomes, Patrick","position":"QB","team":"KCC","birthdate":"495176400","draft_year":"2017"},` +
		`{"id":"15283","name":"Moore, Rondale","position":"XX","team":"MIN"},` +
		`{"id":"14999","name":"Rookie, Some","position":"WR","team":"FA","status":"R"}` +
		`]}}`

	var env playersEnvelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	got, err := flatten(context.Background(), env)
	if err != nil {
		t.Fatalf("flatten: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d players, want 3", len(got))
	}
	if got[0].ID != "0531" || got[0].Position != "QB" || got[0].Team != "KCC" {
		t.Fatalf("player 0 mapped wrong: %+v", got[0])
	}
	if got[0].Birthdate != "495176400" {
		t.Fatalf("player 0 birthdate not mapped raw: %+v", got[0])
	}
	if got[0].DraftYear != "2017" { // DETAILS draft_year mapped raw
		t.Fatalf("player 0 draft year not mapped raw: %+v", got[0])
	}
	if got[1].Birthdate != "" { // no DETAILS field on this record → stays absent
		t.Fatalf("player 1 birthdate should be absent: %+v", got[1])
	}
	if got[1].DraftYear != "" { // no DETAILS field on this record → stays absent
		t.Fatalf("player 1 draft year should be absent: %+v", got[1])
	}
	if got[1].Position != "XX" { // raw code preserved, not flagged here
		t.Fatalf("player 1 position not raw: %+v", got[1])
	}
	if got[2].Status != "R" { // rookie flag preserved raw
		t.Fatalf("player 2 rookie status not preserved: %+v", got[2])
	}
}

// TestDecode_SingleElementCollapse covers the MFL quirk where a one-player
// response emits player as a bare object instead of a one-element array. MFLList
// must still yield a one-element slice (a crash here would be a real BLOCKER).
func TestDecode_SingleElementCollapse(t *testing.T) {
	const body = `{"players":{"player":{"id":"0531","name":"Mahomes, Patrick","position":"QB","team":"KCC"}}}`

	var env playersEnvelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	got, err := flatten(context.Background(), env)
	if err != nil {
		t.Fatalf("flatten: %v", err)
	}
	if len(got) != 1 || got[0].ID != "0531" {
		t.Fatalf("single-element collapse not handled: %+v", got)
	}
}
