package commissioner

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func memoMax(t *testing.T, m *RulesMemo, records []Record, at time.Time) int {
	t.Helper()
	rules, err := m.InForce(records, 2026, at)
	if err != nil {
		t.Fatal(err)
	}
	return rules.Roster.Limits.Value.Max.Value
}

func TestRulesMemoReadsALedgerSeasonAndInstantOnce(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	m := NewRulesMemo()
	want, err := RulesInForce(b.records, 2026, seasonMoment(5))
	if err != nil {
		t.Fatal(err)
	}
	for range 5 {
		got, err := m.InForce(b.records, 2026, seasonMoment(5))
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("memo answered differently from the resolver: %v", err)
		}
	}
	if m.Resolved() != 1 {
		t.Fatalf("one ledger, season and instant was read %d times", m.Resolved())
	}
	// The same instant written in another zone is not a different instant.
	memoMax(t, m, b.records, seasonMoment(5).In(time.FixedZone("far", 9*3600)))
	if m.Resolved() != 1 {
		t.Fatalf("the same instant in another zone was read again: %d", m.Resolved())
	}
}

func TestRulesMemoKeepsAMidSeasonChangeAtItsOwnInstant(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	change := seasonMoment(10)
	b.set("roster.limits", `{"min":20,"max":70}`, "now", change)
	m := NewRulesMemo()
	for _, c := range []struct {
		name string
		at   time.Time
		want int
	}{
		{"just before the change starts", change.Add(-time.Nanosecond), 80},
		{"the moment it starts", change, 70},
		{"before it again, from memory", change.Add(-time.Nanosecond), 80},
		{"after it, from memory", change, 70},
		{"long after it", seasonMoment(30), 70},
	} {
		if got := memoMax(t, m, b.records, c.at); got != c.want {
			t.Errorf("%s: roster maximum %d, want %d", c.name, got, c.want)
		}
	}
	if m.Resolved() != 3 {
		t.Fatalf("three instants were read %d times", m.Resolved())
	}
}

func TestRulesMemoIsNotSharedBetweenLedgers(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	prefix := slices.Clone(b.records)
	b.set("roster.limits", `{"min":20,"max":70}`, "now", seasonMoment(3))
	full := slices.Clone(b.records)
	other := newLedger(t, PresetLegacy)
	other.set("roster.limits", `{"min":20,"max":60}`, "now", seasonMoment(3))
	m := NewRulesMemo()
	for _, c := range []struct {
		name    string
		records []Record
		want    int
	}{
		{"the ledger before the change", prefix, 80},
		{"the ledger with it", full, 70},
		{"a ledger of the same length with another change", other.records, 60},
		{"a prefix of the second sharing its storage", full[:len(full)-1], 80},
		{"the first again, from memory", prefix, 80},
		{"the second again, from memory", full, 70},
	} {
		if got := memoMax(t, m, c.records, seasonMoment(30)); got != c.want {
			t.Errorf("%s: roster maximum %d, want %d", c.name, got, c.want)
		}
	}
	if m.Resolved() != 4 {
		t.Fatalf("four ledgers were read %d times", m.Resolved())
	}
}

func TestRulesMemoRemembersAFailureAndNeverGuessesAnEmptyLedger(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	m := NewRulesMemo()
	for range 2 {
		if _, err := m.InForce(b.records, 2026+7, seasonMoment(5)); err == nil {
			t.Fatal("a season outside the league's history was resolved")
		}
	}
	if m.Resolved() != 1 {
		t.Fatalf("a failure was read %d times", m.Resolved())
	}
	if _, err := m.InForce(nil, 2026, seasonMoment(5)); err == nil || m.Resolved() != 1 {
		t.Fatalf("an empty ledger: %v after %d reads", err, m.Resolved())
	}
}

// A reading that scores many players with one memo reads the ledger once, and a player scored under a
// ledger with a different rule is scored by that rule, not by the one the memo met first.
func TestNativePlayerScoreWithSharedMemoResolvesOnceAndFollowsTheLedger(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.seed(2026, nil, seasonMoment(2))
	rounded := slices.Clone(b.records)
	b.set("scoring.roundStatPoints", "false", "now", seasonMoment(3))
	unrounded := slices.Clone(b.records)
	evidence := completeScoreEvidence()
	evidence.Held.Week.Line = fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"tackle": 10002}}
	m := NewRulesMemo()
	score := func(records []Record, player string) NativePlayerScore {
		t.Helper()
		scope := source.RawNativeScope{Player: player, Season: 2026, Week: 1}
		out, err := NativePlayerScoreWith("native.test", scope, records, evidence, seasonMoment(5), m.InForce)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	a, c := score(rounded, "0001"), score(unrounded, "0002")
	for _, player := range []string{"0003", "0004", "0005"} {
		score(rounded, player)
		score(unrounded, player)
	}
	if m.Resolved() != 2 {
		t.Fatalf("two ledgers were read %d times", m.Resolved())
	}
	if a.Points == nil || c.Points == nil || *a.Points == *c.Points {
		t.Fatalf("the same statistics scored the same under two rules: %v %v", a.Points, c.Points)
	}
	scope := source.RawNativeScope{Player: "0001", Season: 2026, Week: 1}
	direct, err := NativePlayerScoreReading("native.test", scope, rounded, evidence, seasonMoment(5))
	if err != nil || !reflect.DeepEqual(direct, a) {
		t.Fatalf("memoised score differs from the direct reading: %+v vs %+v (%v)", a, direct, err)
	}
}

// The memo folds each ledger once for a reading, and every score it gives is the score read afresh.
func TestRulesMemoScoreFoldsOnceAndMatchesTheDirectReading(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.seed(2026, nil, seasonMoment(2))
	rounded := slices.Clone(b.records)
	b.set("scoring.roundStatPoints", "false", "now", seasonMoment(3))
	unrounded := slices.Clone(b.records)
	evidence := completeScoreEvidence()
	evidence.Held.Week.Line = fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"tackle": 10002}}
	m := NewRulesMemo()
	for _, records := range [][]Record{rounded, unrounded, rounded} {
		for _, player := range []string{"0001", "0002", "0003"} {
			scope := source.RawNativeScope{Player: player, Season: 2026, Week: 1}
			got, err := m.Score("native.test", scope, records, evidence, seasonMoment(5))
			if err != nil {
				t.Fatal(err)
			}
			want, err := NativePlayerScoreReading("native.test", scope, records, evidence, seasonMoment(5))
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("memo score differs from the direct reading: %+v vs %+v (%v)", got, want, err)
			}
		}
	}
	if len(m.folds) != 2 {
		t.Fatalf("two ledgers were folded %d times", len(m.folds))
	}
}

func TestNativePlayerScoreWithAsksForRulesOnlyWhenScoring(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	asked := 0
	count := func(records []Record, season int, at time.Time) (leaguerules.RuleSet, error) {
		asked++
		return RulesInForce(records, season, at)
	}
	scope := source.RawNativeScope{Player: "0099", Season: 2026, Week: 1}
	out, err := NativePlayerScoreWith("native.test", scope, b.records, completeScoreEvidence(), seasonMoment(5), count)
	if err != nil || out.Words != "Record the starting snapshot first" || asked != 0 {
		t.Fatalf("an unseeded league resolved rules %d times: %+v %v", asked, out, err)
	}
	b.seed(2026, nil, seasonMoment(2))
	if _, err := NativePlayerScoreWith("native.test", scope, b.records, completeScoreEvidence(), seasonMoment(5), count); err != nil ||
		asked != 1 {
		t.Fatalf("a seeded league resolved rules %d times: %v", asked, err)
	}
}

func TestRulesMemoKeepsEachSeasonsOwnRulesAtOneInstant(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.set(rolloverField, rolloverAt, "now", seasonMoment(2))
	b.set("roster.limits", `{"min":20,"max":70}`, "next_rollover", seasonMoment(3))
	after := time.Date(2027, 3, 2, 12, 0, 0, 0, time.UTC)
	m := NewRulesMemo()
	for range 2 {
		ended, err := m.InForce(b.records, 2026, after)
		if err != nil {
			t.Fatal(err)
		}
		current, err := m.InForce(b.records, 2027, after)
		if err != nil {
			t.Fatal(err)
		}
		if ended.Roster.Limits.Value.Max.Value != 80 || current.Roster.Limits.Value.Max.Value != 70 {
			t.Fatalf("one ledger at one instant must keep each season's rules: ended %d, current %d",
				ended.Roster.Limits.Value.Max.Value, current.Roster.Limits.Value.Max.Value)
		}
	}
	if m.Resolved() != 2 {
		t.Fatalf("two seasons should be read once each, read %d times", m.Resolved())
	}
}
