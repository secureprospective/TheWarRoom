package main

import (
	"strings"
	"testing"
)

func TestGeneratedVerbsPreserveDispatchAndRefuseWrongInputs(t *testing.T) {
	e, league, _, address := newTestEnv(t)
	if err := e.runTwr(t, testLink, "connect", "--address", address, "--league", "native.test"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, input, mode string
		pass              bool
	}{
		{"default preview", `{"reason":"fix"}`, "", true},
		{"confirmed execute", `{"command":"correct.roster","reason":"fix"}`, "execute", true},
		{"wrong command", `{"command":"commish.rules.set"}`, "", false},
		{"unsupported mode", `{}`, "read", false},
		{"bad JSON", `{"unknown":true}`, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := len(league.dispatched)
			args := []string{"correct.roster", "-", "--confirmation", "123456"}
			if tc.mode != "" {
				args = append(args, "--mode", tc.mode)
			}
			err := e.runTwr(t, tc.input, args...)
			if !tc.pass {
				if err == nil || len(league.dispatched) != before {
					t.Fatalf("bad input reached dispatch: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got := league.dispatched[before]
			mode := tc.mode
			if mode == "" {
				mode = "preview"
			}
			if got.Mode != mode || got.Command.Command != "correct.roster" || got.Command.League != "native.test" || got.Command.Expected != 7 || league.confirms[before] != "123456" {
				t.Fatalf("dispatch changed: %+v", got)
			}
		})
	}
	for _, verb := range generatedVerbs() {
		cmd, _, err := e.root().Find([]string{verb.ID})
		if err != nil || cmd.Name() != verb.ID || !strings.HasPrefix(cmd.Use, verb.ID+" ") {
			t.Fatalf("missing generated verb %s: %v", verb.ID, err)
		}
	}
}
