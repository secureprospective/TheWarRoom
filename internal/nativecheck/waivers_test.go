package nativecheck

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func waiverPolicy(t *testing.T, method leaguerules.WaiverMethod) WaiverPolicy {
	t.Helper()
	rules := leaguerules.LegacyNFL().Waivers
	rules.Method = leaguerules.Known(method)
	policy, err := NewWaiverPolicy(rules)
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func TestWaiverPolicyRequiresEveryCompetitiveSetting(t *testing.T) {
	legacy := leaguerules.LegacyNFL().Waivers
	for _, c := range []struct {
		name string
		edit func(*leaguerules.WaiversRules)
	}{
		{"trigger unset", func(r *leaguerules.WaiversRules) {
			r.Trigger = leaguerules.Needs[leaguerules.WaiverTrigger]("")
		}},
		{"unknown trigger", func(r *leaguerules.WaiversRules) {
			r.Trigger = leaguerules.Known(leaguerules.WaiverTrigger("invented"))
		}},
		{"duration unset", func(r *leaguerules.WaiversRules) { r.Duration = leaguerules.Needs[time.Duration]("") }},
		{"zero duration", func(r *leaguerules.WaiversRules) { r.Duration = leaguerules.Known(time.Duration(0)) }},
		{"negative duration", func(r *leaguerules.WaiversRules) { r.Duration = leaguerules.Known(-time.Hour) }},
		{"method unset", func(r *leaguerules.WaiversRules) {
			r.Method = leaguerules.Needs[leaguerules.WaiverMethod]("")
		}},
		{"unknown method", func(r *leaguerules.WaiversRules) {
			r.Method = leaguerules.Known(leaguerules.WaiverMethod("invented"))
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			rules := legacy
			c.edit(&rules)
			if _, err := NewWaiverPolicy(rules); err == nil || !strings.Contains(err.Error(), "needs a ruling") {
				t.Fatalf("invented a competitive setting: %v", err)
			}
		})
	}
}

func TestWaiverCloseUsesElapsedDurationAcrossDaylightSaving(t *testing.T) {
	zone, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	policy := waiverPolicy(t, leaguerules.ReverseStandings)
	for _, c := range []struct {
		cut  time.Time
		want string
	}{
		{time.Date(2026, 3, 7, 12, 0, 0, 0, zone), "2026-03-08T13:00:00-04:00"},
		{time.Date(2026, 10, 31, 12, 0, 0, 0, zone), "2026-11-01T11:00:00-05:00"},
	} {
		closeAt, err := policy.Close(c.cut)
		if err != nil || closeAt.Format(time.RFC3339) != c.want || closeAt.Sub(c.cut) != 24*time.Hour {
			t.Fatalf("cut %s closes %s: %v; want %s", c.cut, closeAt, err, c.want)
		}
	}
	for _, cut := range []time.Time{{}, time.Date(9999, 12, 31, 12, 0, 0, 0, time.UTC)} {
		if _, err := policy.Close(cut); err == nil {
			t.Fatalf("accepted unsupported cut %s", cut)
		}
	}
	if _, err := (WaiverPolicy{}).Close(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("zero policy dated a window")
	}
}

func TestWaiverPriorityIgnoresClaimArrivalOrder(t *testing.T) {
	priority := WaiverPriority{Standings: []string{"last", "middle", "first"},
		Rolling: []string{"first", "middle", "last"}}
	before := WaiverPriority{Standings: slices.Clone(priority.Standings), Rolling: slices.Clone(priority.Rolling)}
	for _, eligible := range [][]string{{"first", "middle"}, {"middle", "first"}} {
		claims := slices.Clone(eligible)
		got, err := waiverPolicy(t, leaguerules.ReverseStandings).Award(priority, eligible)
		if err != nil || got.Winner != "middle" || !slices.Equal(got.Order, priority.Standings) {
			t.Fatalf("reverse standings: %+v, %v", got, err)
		}
		if !slices.Equal(eligible, claims) {
			t.Fatal("changed the eligible claims")
		}
		got.Order[0] = "changed"
		if !reflect.DeepEqual(priority, before) {
			t.Fatal("award aliases its supplied order")
		}
	}
}

func TestRollingWaiverWinnerMovesToBack(t *testing.T) {
	policy := waiverPolicy(t, leaguerules.Rolling)
	order := []string{"first", "middle", "last"}
	for _, c := range []struct {
		claims []string
		winner string
		want   []string
	}{
		{[]string{"last", "first"}, "first", []string{"middle", "last", "first"}},
		{[]string{"middle", "last"}, "middle", []string{"first", "last", "middle"}},
		{[]string{"last"}, "last", []string{"first", "middle", "last"}},
		{nil, "", []string{"first", "middle", "last"}},
	} {
		got, err := policy.Award(WaiverPriority{Rolling: order}, c.claims)
		if err != nil || got.Winner != c.winner || !slices.Equal(got.Order, c.want) {
			t.Fatalf("claims %v: %+v, %v; want %s %v", c.claims, got, err, c.winner, c.want)
		}
		if !slices.Equal(order, []string{"first", "middle", "last"}) {
			t.Fatal("rolling award mutated its input")
		}
	}
}

func TestWaiverPriorityRefusesMissingOrMalformedOrders(t *testing.T) {
	for _, c := range []struct {
		method   leaguerules.WaiverMethod
		priority WaiverPriority
		claims   []string
		words    string
	}{
		{leaguerules.ReverseStandings, WaiverPriority{Rolling: []string{"a"}}, nil, "needs the standings"},
		{leaguerules.Rolling, WaiverPriority{Standings: []string{"a"}}, nil, "set the rolling waiver order"},
		{leaguerules.Rolling, WaiverPriority{Rolling: []string{"a", "a"}}, nil, "each franchise once"},
		{leaguerules.Rolling, WaiverPriority{Rolling: []string{"a", " "}}, nil, "without blanks"},
		{leaguerules.Rolling, WaiverPriority{Rolling: []string{" a"}}, nil, "without blanks"},
		{leaguerules.Rolling, WaiverPriority{Rolling: []string{"a"}}, []string{"b"}, "eligible claim"},
		{leaguerules.Rolling, WaiverPriority{Rolling: []string{"a"}}, []string{"a", "a"}, "eligible claim"},
	} {
		_, err := waiverPolicy(t, c.method).Award(c.priority, c.claims)
		if err == nil || !strings.Contains(err.Error(), c.words) {
			t.Fatalf("%+v: %v", c, err)
		}
	}
	if _, err := (WaiverPolicy{}).Award(WaiverPriority{}, nil); err == nil {
		t.Fatal("zero policy awarded a claim")
	}
}
