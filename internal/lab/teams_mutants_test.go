package lab

import (
	"encoding/json"
	"testing"
)

// TestMutantsTeams replays the teams pipeline with one seeded bug each; every
// mutant must fail parity against the frozen golden, or the golden has no teeth.
func TestMutantsTeams(t *testing.T) {
	d := loadData(t)
	b := readGoldenBytes(t, "testdata/goldens/teams.json.gz")
	var golden struct {
		TradeSides []any `json:"tradeSides"`
		Fixtures   []struct {
			Name   string                     `json:"name"`
			Scope  TeamScope                  `json:"scope"`
			Values map[string]json.RawMessage `json:"values"`
		} `json:"fixtures"`
	}
	if err := json.Unmarshal(b, &golden); err != nil {
		t.Fatal(err)
	}
	// Mutant: count a side's realised return after one played season instead of
	// two. Sides of trades with exactly one played season flip real null->value.
	maxError := 0.0
	if err := parity(golden.TradeSides, jsonValue(t, tradeSides(d, 1)), "tradeSides.realThreshold1", &maxError); err == nil {
		t.Fatal("played-seasons mutant survived")
	} else {
		t.Logf("played-seasons mutant killed: %v", err)
	}
	// The same mutant must flip a fixture cell (value<->null), not just side rows.
	measured := golden.Fixtures[1]
	if measured.Name != "measured" {
		t.Fatalf("expected the measured fixture second; got %s", measured.Name)
	}
	var want any
	if err := json.Unmarshal(measured.Values["behaviour"], &want); err != nil {
		t.Fatal(err)
	}
	sides := sidesInScope(d, measured.Scope, 1)
	if err := parity(want, jsonValue(t, Behaviour(sides, nil)), "mutant.behaviour.realThreshold1", &maxError); err == nil {
		t.Fatal("fixture cell survived the played-seasons mutant")
	} else {
		t.Logf("fixture cell killed: %v", err)
	}
}
