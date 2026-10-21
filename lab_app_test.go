package main

import (
	"encoding/json"
	"strings"
	"testing"

	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"

	"github.com/secureprospective/TheWarRoom/internal/app/core"

	"github.com/secureprospective/TheWarRoom/internal/lab"
)

func TestTargetMarketReading(t *testing.T) {
	t.Parallel()
	a := &App{raceHeld: &raceapp.Held{}, core: &core.State{}}
	r := a.TargetMarket(RawMarketScope{From: 2017, To: 2026})
	if r.Status != ReadingOK || r.Count < 80 || r.Threshold != 80 || r.Value.Going == nil {
		t.Fatal("missing Market reading")
	}
	if r.InputSHA != lab.DataSHA || r.BuildStamp != buildLabel()+" "+buildDate || r.Cutoff != "2026-10-06T01:42:57+00:00" {
		t.Fatal("missing provenance")
	}
	if r.Provenance.Freshness.State == "live" || r.Provenance.Freshness.FetchedAt != "2026-10-06T15:57:20+00:00" || !strings.Contains(r.Provenance.Source, r.InputSHA) {
		t.Fatal("archive labelled live")
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(b, &wire); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"value", "provenance", "status", "absenceReason", "count", "threshold", "baseline", "cutoff", "method", "howCounted", "inputSHA", "buildStamp"} {
		if _, ok := wire[key]; !ok {
			t.Errorf("missing %s", key)
		}
	}
	for _, grade := range []string{"\"evidence\"", "\"solid\"", "\"rough\"", "\"thin\""} {
		if strings.Contains(string(b), grade) {
			t.Errorf("spread grade leaked: %s", grade)
		}
	}
	if r.HowCounted == "" || r.AbsenceReason != nil {
		t.Fatal("shell disclosure")
	}
	thin := a.TargetMarket(RawMarketScope{From: 2017, To: 2026, Phases: []string{"winter"}})
	if thin.Status != ReadingThin || thin.Count == 0 || thin.Value.Rows != nil || thin.AbsenceReason == nil {
		t.Fatal("thin reading fabricated data")
	}
	for _, raw := range []RawMarketScope{{From: 2026, To: 2017}, {From: 2017, To: 2026, Phases: []string{"bogus"}}, {From: 2017, To: 2026, Phases: []string{"winter", "winter"}}} {
		failed := a.TargetMarket(raw)
		if failed.Status != ReadingFailed || failed.AbsenceReason == nil {
			t.Fatal("invalid IPC scope accepted")
		}
	}
}

func TestAppLabDecodesOnce(t *testing.T) {
	t.Parallel()
	a := NewApp()
	first, err := a.lab()
	if err != nil {
		t.Fatal(err)
	}
	second, err := a.lab()
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("the App decoded the frozen Lab facts twice")
	}
}
