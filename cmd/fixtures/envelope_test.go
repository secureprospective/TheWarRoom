package main

import (
	"encoding/json"
	"flag"
	"os"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func TestRosterIREndToEndFixture(t *testing.T) {
	update := flag.Lookup("update-moves")
	body, err := os.ReadFile("../../frontend/src/app/data/fixtures/snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	var snap snapshot.Snapshot
	if err := json.Unmarshal(body, &snap); err != nil {
		t.Fatal(err)
	}
	d, err := demo(snap)
	if err != nil {
		t.Fatal(err)
	}
	if d.Kind != "fixture" || d.Receipt.State != envelope.Landed {
		t.Fatal(d)
	}
	path := []envelope.State{envelope.Ready, envelope.HandedOff, envelope.NotYetDone, envelope.Landed}
	for i, want := range path {
		if d.Receipt.Audit[i].To != want {
			t.Fatal(d.Receipt.Audit)
		}
	}
	if d.Receipt.Audit[0].From != envelope.Draft {
		t.Fatal(d)
	}
	again, err := demo(snap)
	if err != nil || !reflect.DeepEqual(d, again) {
		t.Fatalf("not deterministic: %v", err)
	}
	if update != nil && update.Value.String() == "true" {
		if err := writeEnvelopeDemo("../../frontend/src/app/data/fixtures", snap); err != nil {
			t.Fatal(err)
		}
	}
	checked, err := os.ReadFile("../../frontend/src/app/data/fixtures/envelope-demo.json")
	if err != nil {
		t.Fatal(err)
	}
	var expected envelopeDemo
	if err := json.Unmarshal(checked, &expected); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d, expected) {
		t.Fatal("envelope fixture is stale")
	}
}
func init() { flag.Bool("update-moves", false, "regenerate deterministic move fixtures") }
