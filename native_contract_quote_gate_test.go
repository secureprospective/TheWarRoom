package main

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// switchContractRulesOn gives an empty test league the Legacy NFL contract rules, so the registered
// contract actions can start once the calendar opens them. It returns the revision the next command expects.
func switchContractRulesOn(t *testing.T, app *App, raw commissioner.RawCommand, expected int64) int64 {
	t.Helper()
	fields, values := leaguerules.Catalogue(), leaguerules.Current(leaguerules.LegacyNFL())
	for i, field := range fields {
		if !strings.HasPrefix(string(field.Path), "contracts.") {
			continue
		}
		raw.Command, raw.Request, raw.Expected = commissioner.RulesSet, "rules."+string(field.Path), expected
		raw.Args = commissioner.RawArgs{Field: string(field.Path), Value: values[i]}
		executeCommissionerTest(t, app, raw)
		expected++
	}
	return expected
}
