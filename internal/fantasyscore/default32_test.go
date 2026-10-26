package fantasyscore

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

type L = leaguerules.Position

const (
	qb, rb, wr, te, pk = leaguerules.QB, leaguerules.RB, leaguerules.WR, leaguerules.TE, leaguerules.PK
	dt, de, lb, cb, sf = leaguerules.DT, leaguerules.DE, leaguerules.LB, leaguerules.CB, leaguerules.S
)

func yards(n ...int64) []leaguerules.Decimal {
	result := make([]leaguerules.Decimal, len(n))
	for i, v := range n {
		result[i] = leaguerules.Decimal(v * 10000)
	}
	return result
}

func quantity(id leaguerules.StatID, n leaguerules.Decimal) StatLine {
	return StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{id: n}}
}

// Every row of DEFAULT-32 spec §4.2–4.4, priced at two (or the stated) occurrences.
func TestDefault32SpecRows(t *testing.T) {
	rules := leaguerules.Default32Scoring()
	for _, tc := range []struct {
		id        leaguerules.StatID
		position  L
		qty, want leaguerules.Decimal
	}{
		{"passing_yards", qb, 3000000, 150000}, {"passing_td", qb, 20000, 100000},
		{"interception_thrown", qb, 20000, -40000}, {"passing_two_point", qb, 20000, 40000},
		{"rushing_yards", rb, 1000000, 100000}, {"rushing_attempt", rb, 200000, 30000},
		{"rushing_td", rb, 20000, 120000}, {"rushing_two_point", rb, 20000, 40000},
		{"receiving_yards", wr, 1000000, 100000}, {"reception", wr, 20000, 20000},
		{"receiving_td", wr, 20000, 120000}, {"receiving_two_point", wr, 20000, 40000},
		{"kickoff_return_yards", wr, 1000000, 50000}, {"punt_return_yards", wr, 1000000, 50000},
		{"kickoff_return_td", wr, 20000, 200000}, {"punt_return_td", wr, 20000, 120000},
		{"fumbles_total", rb, 20000, -40000}, {"own_fumble_recovery", rb, 10000, 20000},
		{"own_fumble_recovery", rb, 30000, 20000}, {"pat_made", pk, 20000, 20000},
		{"sack", de, 5000, 22500}, {"sack", de, 20000, 90000}, {"tackle_for_loss", lb, 20000, 50000},
		{"qb_hit", de, 20000, 20000}, {"forced_fumble", lb, 20000, 80000},
		{"opponent_fumble_recovery", lb, 20000, 60000}, {"opponent_fumble_recovery_yards", lb, 1000000, 100000},
		{"interception_return_yards", lb, 1000000, 100000}, {"safety", dt, 10000, 100000},
		{"blocked_fg", dt, 10000, 70000}, {"blocked_punt", dt, 10000, 70000},
		{"blocked_extra_point", dt, 10000, 70000}, {"fumble_recovery_tds", lb, 10000, 60000},
		{"fumble_recovery_tds", wr, 10000, 60000}, {"interception_return_td", cb, 10000, 60000},
		{"interception_return_td", qb, 10000, 60000},
		{"blocked_fg_td", lb, 10000, 70000}, {"blocked_punt_td", sf, 10000, 70000},
		// #BF/#BP are IDP-only.
		{"blocked_fg_td", wr, 10000, 0}, {"blocked_punt_td", rb, 10000, 0},
		// Inclusive MFL ranges: out-of-range weeks score nothing for that rule.
		{"passing_yards", qb, -500000, -25000}, {"passing_yards", qb, -510000, 0},
		{"passing_td", qb, 210000, 0}, {"passing_td", qb, 200000, 1000000}, {"sack", de, 250000, 1125000},
		{"rushing_td", rb, 110000, 0}, {"sack", de, 260000, 0},
		{"own_fumble_recovery", rb, 0, 0}, {"fumbles_total", rb, 110000, 0},
	} {
		t.Run(string(tc.id)+"/"+string(tc.position), func(t *testing.T) {
			assertScore(t, rules, tc.position, quantity(tc.id, tc.qty), tc.want)
		})
	}
}

// TE catch 1.2 against WR 1.0, and the 0.2 extra shows as its own line.
func TestDefault32CatchValueByPosition(t *testing.T) {
	rules := leaguerules.Default32Scoring()
	for position, want := range map[L]leaguerules.Decimal{te: 60000, wr: 50000, rb: 50000, qb: 50000, cb: 50000} {
		assertScore(t, rules, position, quantity("reception", 50000), want)
	}
	_, items, err := Score(rules, te, quantity("reception", 50000))
	if err != nil || len(items) != 2 || !items[1].Extra || items[1].Points.Value != 10000 {
		t.Fatalf("TE extra breakdown %+v %v", items, err)
	}
	// The TE extra is its own 0..999 rule: 100 catches fall outside the base 0..99 row, not the extra.
	assertScore(t, rules, te, quantity("reception", 1000000), 200000)
	assertScore(t, rules, wr, quantity("reception", 1000000), 0)
}

// Tackle lift and positional extras (§4.4).
func TestDefault32TackleExtras(t *testing.T) {
	rules := leaguerules.Default32Scoring()
	for _, tc := range []struct {
		id   leaguerules.StatID
		want map[L]leaguerules.Decimal
	}{
		{"tackle", map[L]leaguerules.Decimal{dt: 30000, de: 30000, cb: 25000, sf: 25000, lb: 20000, qb: 20000}},
		{"assist", map[L]leaguerules.Decimal{dt: 17500, de: 17500, cb: 12500, sf: 12500, lb: 12500, rb: 12500}},
		{"interception_caught", map[L]leaguerules.Decimal{cb: 60000, sf: 60000, lb: 50000, dt: 50000, de: 50000}},
		{"pass_defensed", map[L]leaguerules.Decimal{cb: 30000, sf: 30000, lb: 25000, dt: 25000, de: 25000}},
	} {
		for position, want := range tc.want {
			t.Run(string(tc.id)+"/"+string(position), func(t *testing.T) {
				assertScore(t, rules, position, quantity(tc.id, 10000), want)
			})
		}
	}
}

func TestDefault32ChunkCurve(t *testing.T) {
	rules := leaguerules.Default32Scoring()
	for _, tc := range []struct {
		plays []leaguerules.Decimal
		want  leaguerules.Decimal
	}{
		{yards(19), 0}, {yards(20), 10000}, {yards(30), 12500}, {yards(40), 15000}, {yards(60), 20000},
		{yards(98), 29500}, {yards(99), 29800}, {yards(21), 10300},
		// Summed for the week, then rounded once: 2.05, not 1.03 + 1.03.
		{yards(21, 21), 20500}, {yards(5, 19, 20, 40), 25000}, {yards(), 0},
	} {
		assertScore(t, rules, wr, StatLine{Events: map[leaguerules.StatID][]leaguerules.Decimal{"chunk_play": tc.plays}}, tc.want)
	}
	if _, _, err := Score(rules, wr, quantity("chunk_play", 10000)); err == nil {
		t.Fatal("a chunk count without play distances must not score")
	}
	changed := leaguerules.Default32Scoring()
	scale := changed.Scales.Value["chunk_play"]
	scale.Capped, scale.Cap = true, 25000
	changed.Scales.Value["chunk_play"] = scale
	assertScore(t, changed, wr, StatLine{Events: map[leaguerules.StatID][]leaguerules.Decimal{"chunk_play": yards(99)}}, 25000)
}

func TestDefault32PassOf40(t *testing.T) {
	assertScore(t, leaguerules.Default32Scoring(), qb,
		StatLine{Events: map[leaguerules.StatID][]leaguerules.Decimal{"long_pass": yards(8, 39, 40, 75)}}, 20000)
}

func TestDefault32FieldGoalsByDistance(t *testing.T) {
	rules := leaguerules.Default32Scoring()
	made := map[int64]leaguerules.Decimal{25: 30000, 30: 30000, 40: 30000, 45: 35000, 50: 40000,
		55: 45000, 57: 47000, 60: 50000, 63: 53000}
	missed := map[int64]leaguerules.Decimal{19: -41000, 20: -40000, 25: -35000, 30: -30000, 33: -27000,
		40: -20000, 42: -18000, 45: -15000, 50: -10000, 55: -5000, 58: -2000, 60: 0, 65: 0}
	for distance, want := range made {
		assertScore(t, rules, pk, StatLine{Events: map[leaguerules.StatID][]leaguerules.Decimal{"fg_made_distance": yards(distance)}}, want)
	}
	for distance, want := range missed {
		assertScore(t, rules, pk, StatLine{Events: map[leaguerules.StatID][]leaguerules.Decimal{"fg_missed_distance": yards(distance)}}, want)
	}
	week := StatLine{Events: map[leaguerules.StatID][]leaguerules.Decimal{
		"fg_made_distance": yards(25, 45, 57), "fg_missed_distance": yards(33)}}
	assertScore(t, rules, pk, week, 30000+35000+47000-27000)
	// Legacy buckets are not part of the default: a recorded bucket carries no points.
	assertScore(t, rules, pk, quantity("fg_made_40_49", 10000), 0)
}

func TestDefault32TouchBonus(t *testing.T) {
	rules := leaguerules.Default32Scoring()
	line := func(carries, catches leaguerules.Decimal) StatLine {
		return StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"rushing_attempt": carries * 10000, "reception": catches * 10000}}
	}
	for _, tc := range []struct {
		carries, catches leaguerules.Decimal
		position         L
		want             leaguerules.Decimal
	}{
		{15, 4, rb, 15*1500 + 40000}, {16, 4, rb, 16*1500 + 40000 + 30000}, {30, 9, rb, 30*1500 + 90000 + 30000},
		{20, 0, qb, 20*1500 + 30000}, {0, 20, te, 20*12000 + 30000}, {60, 40, rb, 60*1500 + 400000},
	} {
		assertScore(t, rules, tc.position, line(tc.carries, tc.catches), tc.want)
	}
	if _, _, err := Score(rules, rb, quantity("touches", 200000)); err == nil {
		t.Fatal("touches are counted from carries and catches, never recorded")
	}
	// Without both parts recorded the bonus is absent, not zero-guessed.
	assertScore(t, rules, rb, quantity("rushing_attempt", 250000), 25*1500)
}

func TestDefault32LineUnit(t *testing.T) {
	rules := leaguerules.Default32Scoring()
	game := func(ybc, r10, rtd, short, tfl, sacks, hits leaguerules.Decimal) UnitLine {
		return UnitLine{Played: true, Quantities: map[leaguerules.StatID]leaguerules.Decimal{
			"line_rb_yards_before_contact": ybc * 10000, "line_rushes_10_plus": r10 * 10000,
			"line_rushing_tds": rtd * 10000, "line_short_conversions": short * 10000,
			"line_tackled_for_loss": tfl * 10000, "line_sacks": sacks * 10000, "line_qb_hits": hits * 10000}}
	}
	for _, tc := range []struct {
		name string
		line UnitLine
		want leaguerules.Decimal
	}{
		{"typical", game(75, 3, 1, 2, 4, 2, 7), 30000 + 30000 + 10000 + 20000 - 20000 - 20000 - 17500},
		{"no sacks earns 2", game(50, 1, 0, 0, 2, 0, 3), 20000 + 10000 - 10000 - 7500 + 20000},
		{"exactly +15 stands", game(200, 4, 2, 2, 0, 1, 0), 150000},
		{"clamped at +15", game(300, 6, 3, 2, 0, 0, 1), 150000},
		{"clamped at -15", game(0, 0, 0, 0, 6, 9, 14), -150000},
		{"bye scores 0", UnitLine{}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			points, items, err := ScoreUnit(rules, tc.line)
			if err != nil || points.State != leaguerules.IsKnown || points.Value != tc.want {
				t.Fatalf("points %+v error %v, want %d", points, err, tc.want)
			}
			sum := leaguerules.Decimal(0)
			for _, item := range items {
				sum += item.Points.Value
			}
			if sum != points.Value {
				t.Fatalf("breakdown %d does not add up to %d", sum, points.Value)
			}
		})
	}
	partial := game(0, 1, 0, 0, 0, 1, 0)
	delete(partial.Quantities, "line_rb_yards_before_contact")
	if points, _, err := ScoreUnit(rules, partial); err != nil || points.State != leaguerules.NeedsRuling {
		t.Fatalf("missing yards before contact guessed: %+v %v", points, err)
	}
	if _, _, err := ScoreUnit(leaguerules.LegacyNFL().Scoring, game(0, 0, 0, 0, 0, 0, 0)); err == nil {
		t.Fatal("Legacy has no line unit")
	}
}

func TestDefault32IsCommissionerSettings(t *testing.T) {
	rules := leaguerules.LegacyNFL()
	rules.Scoring = leaguerules.Default32Scoring()
	for _, change := range leaguerules.SeedChanges(rules, "default-32", 2026, time.Unix(1, 0).UTC()) {
		if err := change.Validate(); err != nil {
			t.Fatalf("%s: %v", change.Field, err)
		}
		body, err := leaguerules.EncodeStored(change.New)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := leaguerules.DecodeStored(change.Field, body); err != nil {
			t.Fatalf("%s round trip: %v", change.Field, err)
		}
	}
	unset := leaguerules.Default32Scoring()
	unset.Scales = leaguerules.Needs[map[leaguerules.StatID]leaguerules.DistanceScale]("not chosen")
	if points, _, err := Score(unset, wr, quantity("reception", 10000)); err != nil || points.State != leaguerules.NeedsRuling {
		t.Fatalf("unset distance scales guessed: %+v %v", points, err)
	}
}

// EM is one rule over missed and blocked extra points, ranged 0..10 on their sum (§4.3).
func TestDefault32MissedExtraPointsAreOneRule(t *testing.T) {
	rules := leaguerules.Default32Scoring()
	line := func(missed, blocked leaguerules.Decimal) StatLine {
		return StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"pat_missed": missed * 10000, "pat_blocked": blocked * 10000}}
	}
	for _, tc := range []struct{ missed, blocked, want leaguerules.Decimal }{
		{0, 0, 0}, {2, 0, -20000}, {1, 1, -20000}, {6, 4, -100000}, {6, 5, 0},
	} {
		assertScore(t, rules, pk, line(tc.missed, tc.blocked), tc.want)
	}
}
