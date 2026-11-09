package commissioner

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/deepclone"
)

// rulesCacheLedgers are ledgers whose rules move with the reading time in every way RulesInForce knows: a
// mid-season change, a change from the next rollover, a season start, and a recorded rollover with its
// explicit before and after rules.
func rulesCacheLedgers(t *testing.T) map[string][]Record {
	t.Helper()
	mid := rolloverLedger(t)
	mid.set("roster.limits", `{"min":20,"max":70}`, "now", seasonMoment(10))
	mid.set("roster.limits", `{"min":20,"max":60}`, "next_rollover", seasonMoment(11))
	rolled := rolloverLedger(t)
	rolled.set("roster.limits", `{"min":20,"max":75}`, "now", seasonMoment(6))
	rolled.rollover(2027, seasonMoment(8))
	rolled.set("roster.limits", `{"min":20,"max":65}`, "now", seasonMoment(12))
	return map[string][]Record{"assets fixture": assetsCacheLedger(t),
		"mid-season and next-rollover changes": mid.records, "recorded rollover": rolled.records}
}

// rulesSweep is every reading instant that can matter: the fixture month's sweep plus each epoch threshold,
// a nanosecond each side of it, and the same instant in another zone.
func rulesSweep(t *testing.T, records []Record) []time.Time {
	t.Helper()
	thresholds, _, ok := epochThresholds(records)
	if !ok {
		t.Fatal("fixture times must parse")
	}
	out := epochSweep()
	for _, th := range thresholds {
		out = append(out, th.Add(-time.Nanosecond), th, th.Add(time.Nanosecond), th.In(time.FixedZone("far", 9*3600)))
	}
	return out
}

func rulesAnswer(rules any, err error) any {
	return []any{rules, errText(err)}
}

// TestRulesCacheMatchesTheFold: for every season and every instant of the sweep, the held revision's cached
// rules (read cold, then warm) equal the resolver on a plain copy of the ledger, errors included. The sweep
// must see the rules change, or it proves nothing.
func TestRulesCacheMatchesTheFold(t *testing.T) {
	t.Parallel()
	for name, records := range rulesCacheLedgers(t) {
		held := WithReplayCache(records)
		distinct := map[string]bool{}
		for _, at := range rulesSweep(t, held) {
			for season := 2025; season <= 2028; season++ {
				want := rulesAnswer(rulesInForce(slices.Clone(records), season, at))
				for _, pass := range []string{"cold", "warm"} {
					got := rulesAnswer(RulesInForce(held, season, at))
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("%s: season %d at %v (%s): cached rules differ from the fold", name, season, at, pass)
					}
				}
				distinct[fmtAnswer(want)] = true
			}
		}
		if len(distinct) < 4 {
			t.Fatalf("%s: the sweep is vacuous: only %d distinct answers", name, len(distinct))
		}
	}
}

// TestRulesCacheHandsOutOwnedCopies: a caller that writes into every map, slice and string of the rules it
// was given changes nothing the next caller reads.
func TestRulesCacheHandsOutOwnedCopies(t *testing.T) {
	t.Parallel()
	for name, records := range rulesCacheLedgers(t) {
		held := WithReplayCache(deepclone.Of(records))
		for _, at := range rulesSweep(t, held) {
			// A snapshot, not the value: a recorded rollover's rules are the ledger entry's own maps.
			want := fmtAnswer(rulesAnswer(rulesInForce(deepclone.Of(records), 2026, at)))
			for range 2 {
				rules, err := RulesInForce(held, 2026, at)
				if err == nil {
					scribble(reflect.ValueOf(&rules).Elem())
				}
			}
			if got := fmtAnswer(rulesAnswer(RulesInForce(held, 2026, at))); got != want {
				t.Fatalf("%s at %v: a caller's write reached the cache", name, at)
			}
		}
	}
}

func fmtAnswer(v any) string {
	body, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(body)
}
