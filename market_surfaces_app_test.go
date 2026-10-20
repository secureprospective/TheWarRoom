package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/lab"
)

func TestMarketSurfaceMethods(t *testing.T) {
	t.Parallel()
	a := NewApp()
	full := RawMarketScope{From: 2017, To: 2026}
	methods := []struct {
		name string
		read func(RawMarketScope) Reading[MarketSurface]
	}{
		{"lists", a.TargetMarketLists}, {"clock", a.TargetPickClock}, {"stretches", a.TargetMarketStretches},
	}
	for _, method := range methods {
		t.Run(method.name, func(t *testing.T) {
			ok := method.read(full)
			if ok.Status != ReadingOK || ok.Count == 0 || ok.HowCounted == "" || ok.Cutoff == "" || ok.InputSHA != lab.DataSHA || len(ok.Value.Sections) == 0 {
				t.Fatalf("incomplete reading: %+v", ok)
			}
			wire, err := json.Marshal(ok)
			if err != nil {
				t.Fatal(err)
			}
			for _, grade := range []string{"\"solid\"", "\"rough\"", "\"evidence\""} {
				if strings.Contains(string(wire), grade) {
					t.Fatal("grade leaked")
				}
			}
			thin := method.read(RawMarketScope{From: 2100, To: 2101})
			if thin.Status != ReadingThin || thin.AbsenceReason == nil || thin.Count != 0 {
				t.Fatal("absent scope fabricated data")
			}
			failed := method.read(RawMarketScope{From: 2026, To: 2017})
			if failed.Status != ReadingFailed || failed.AbsenceReason == nil {
				t.Fatal("invalid IPC accepted")
			}
		})
	}
}
func TestMarketInputValidationAndLoadFailure(t *testing.T) {
	t.Parallel()
	for _, raw := range []RawMarketScope{{From: 2017, To: 2026, Assets: []string{"bogus"}}, {From: 2017, To: 2026, Assets: []string{"LB", "LB"}}} {
		if raw.Validate() == nil {
			t.Fatal("invalid asset selection accepted")
		}
	}
	a := &App{labData: func() (*lab.Data, error) { return nil, errors.New("fingerprint mismatch") }}
	for _, r := range []Reading[MarketSurface]{a.TargetMarketLists(RawMarketScope{From: 2017, To: 2026}), a.TargetPickClock(RawMarketScope{From: 2017, To: 2026}), a.TargetMarketStretches(RawMarketScope{From: 2017, To: 2026}), a.TargetAssetProfile(RawMarketAsset{Scope: RawMarketScope{From: 2017, To: 2026}, Kind: "LB"})} {
		if r.Status != ReadingFailed || r.AbsenceReason == nil || !strings.Contains(*r.AbsenceReason, "fingerprint mismatch") {
			t.Fatal("load failure hidden")
		}
	}
}
