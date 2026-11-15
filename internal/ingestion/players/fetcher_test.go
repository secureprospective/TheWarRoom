package players

import (
	"context"
	"errors"
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
		{"non-numeric draft round", func(p *RawPlayer) { p.DraftRound = "first" }, true},
		{"negative draft pick", func(p *RawPlayer) { p.DraftPick = "-1" }, true},
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

func TestParseReadsAnArchivedBody(t *testing.T) {
	got, err := Parse(context.Background(), []byte(`{"players":{"player":[`+
		`{"id":"15350","name":"Hutchinson, Aidan","position":"DE","team":"DET"},`+
		`{"id":"0531","name":"Mahomes, Patrick","position":"QB","team":"KCC"}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "15350" || got[0].Position != "DE" {
		t.Fatalf("Parse = %+v", got)
	}
	if _, err := Parse(context.Background(), []byte(`{"players":{"player":[]}}`)); !errors.Is(err, errEmptyPlayers) {
		t.Fatalf("empty body: err = %v, want errEmptyPlayers", err)
	}
	if _, err := Parse(context.Background(), []byte(`{"players":{"player":[{"id":"15350","position":""}]}}`)); err == nil {
		t.Fatal("a record without a position parsed")
	}
}
