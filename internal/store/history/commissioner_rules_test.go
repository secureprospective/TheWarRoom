package history

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func createWith(preset string, count int) commissioner.RawCommand {
	raw := createCommand()
	raw.Args = commissioner.RawArgs{Name: "Rules league", Preset: preset, FranchiseCount: count}
	return raw
}

func ruleCommand(field, value, effective, reason string, revision int64) commissioner.RawCommand {
	return commissioner.RawCommand{League: "native.test", Command: commissioner.RulesSet,
		Args: commissioner.RawArgs{Field: field, Value: json.RawMessage(value)}, Reason: reason, Expected: revision,
		Request: fmt.Sprintf("rule.%d", revision), Effective: effective, Visibility: "league"}
}

func ruleRow(t *testing.T, s *Store, field string) commissioner.RuleRow {
	t.Helper()
	rules, err := s.CommissionerRules(t.Context(), "native.test")
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range rules.Sections {
		for _, row := range section.Rows {
			if row.Field == field {
				return row
			}
		}
	}
	t.Fatalf("no rule row %s", field)
	return commissioner.RuleRow{}
}

func revision(t *testing.T, s *Store) int64 {
	t.Helper()
	rules, err := s.CommissionerRules(t.Context(), "native.test")
	if err != nil {
		t.Fatal(err)
	}
	return rules.Revision
}

func TestCommissionerRulesPermissions(t *testing.T) {
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createWith(commissioner.PresetLegacy, 0), "creator", nil); err != nil {
		t.Fatal(err)
	}
	limits := ruleCommand("roster.limits", `{"min":20,"max":70}`, "next_rollover", "", 1)
	if _, err := s.ExecuteCommissioner(t.Context(), limits, "owner", nil); err == nil {
		t.Fatal("owner changed a league rule")
	}
	if revision(t, s) != 1 || len(ruleRow(t, s, "roster.limits").Pending) != 0 {
		t.Fatal("refused rule change was appended")
	}
	executePeople(t, s, "creator", "commish.role.grant",
		commissioner.RawPeople{Principal: "cc", Role: governance.CoCommissioner}, 1)
	limits.Expected, limits.Request = 2, "cc.limits"
	if _, err := s.ExecuteCommissioner(t.Context(), limits, "cc", nil); err != nil {
		t.Fatalf("CC refused on CM-003, a CC row: %v", err)
	}
	for _, field := range []string{"league.teams", "league.alignment", "calendar.rollover"} {
		value := map[string]string{"league.teams": `12`, "league.alignment": `{"franchise.01":"East"}`,
			"calendar.rollover": `"2027-03-01T05:00:00Z"`}[field]
		raw := ruleCommand(field, value, "next_rollover", "League shape vote", 3)
		if _, err := s.ExecuteCommissioner(t.Context(), raw, "cc", nil); err == nil ||
			!strings.Contains(err.Error(), "permission") {
			t.Fatalf("CC changed C-only %s: %v", field, err)
		}
	}
	forged := ruleCommand("roster.limits", `{"min":19,"max":70}`, "now", "", 3)
	forged.Gravity = "G2"
	if _, err := s.ExecuteCommissioner(t.Context(), forged, "creator", nil); err == nil {
		t.Fatal("caller lowered apply-now gravity")
	}
	if revision(t, s) != 3 {
		t.Fatal("refused rule changes were appended")
	}
	ledger, err := s.CommissionerLedger(t.Context(), "native.test", "creator", nil)
	if err != nil || ledger.Entries[0].What != "Active roster size: 18 to 80 → 20 to 70, from the next rollover" ||
		ledger.Entries[0].Who != "cc" || ledger.Entries[0].Gravity != "Confirmation required" {
		t.Fatalf("ledger rule entry: %+v %v", ledger.Entries, err)
	}
}

func TestCommissionerRulesTiming(t *testing.T) {
	s, c, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createWith(commissioner.PresetLegacy, 0), "creator", nil); err != nil {
		t.Fatal(err)
	}
	c.now = c.now.Add(time.Hour)
	waiting, err := s.ExecuteCommissioner(t.Context(),
		ruleCommand("roster.limits", `{"min":20,"max":70}`, "next_rollover", "", 1), "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	row := ruleRow(t, s, "roster.limits")
	if row.Value != "18 to 80" || len(row.Pending) != 1 || row.Pending[0].Value != "20 to 70" ||
		row.Pending[0].When != "Waits for the next rollover" {
		t.Fatalf("waiting change: %+v", row)
	}
	if _, err := s.ExecuteCommissioner(t.Context(),
		ruleCommand("roster.limits", `{"min":20,"max":70}`, "now", " ", 2), "creator", nil); err == nil {
		t.Fatal("apply now accepted without a typed reason")
	}
	c.now = c.now.Add(time.Hour)
	now, err := s.ExecuteCommissioner(t.Context(), ruleCommand("roster.limits", `{"min":20,"max":70}`, "now",
		"Owners voted 24-8 to shrink rosters at once", 2), "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	row = ruleRow(t, s, "roster.limits")
	if row.Value != "20 to 70" || row.State != "Set" || !strings.HasPrefix(row.Citation, "Changed in the league") {
		t.Fatalf("apply now: %+v", row)
	}
	c.now = c.now.Add(time.Hour)
	if _, err := s.ExecuteCommissioner(t.Context(), reverseCommand(now.Entry, 3, "undo.now"), "creator", nil); err != nil {
		t.Fatal(err)
	}
	if row = ruleRow(t, s, "roster.limits"); row.Value != "18 to 80" || row.Citation != "fact R" {
		t.Fatalf("revert did not bring back the prior value: %+v", row)
	}
	if _, err := s.ExecuteCommissioner(t.Context(), reverseCommand(waiting.Entry, 4, "undo.wait"), "creator", nil); err != nil {
		t.Fatal(err)
	}
	if row = ruleRow(t, s, "roster.limits"); len(row.Pending) != 0 {
		t.Fatalf("withdrawn change still waits: %+v", row)
	}
	ledger, err := s.CommissionerLedger(t.Context(), "native.test", "owner", nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Withdrew a waiting rule change to Active roster size, from the next rollover",
		"Reversed a rule change. Active roster size: 20 to 70 → 18 to 80, in force now",
		"Active roster size: 18 to 80 → 20 to 70, in force now",
		"Active roster size: 18 to 80 → 20 to 70, from the next rollover"}
	for i, what := range want {
		if ledger.Entries[i].What != what {
			t.Fatalf("ledger entry %d: %q, want %q", i, ledger.Entries[i].What, what)
		}
	}
	if ledger.Entries[2].Gravity != "Recorded reason required" {
		t.Fatalf("apply now is not G3: %+v", ledger.Entries[2])
	}
}

func TestCommissionerRulesRefusals(t *testing.T) {
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createWith(commissioner.PresetLegacy, 0), "creator", nil); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []commissioner.RawCommand{
		ruleCommand("roster.size", `{"min":20,"max":70}`, "next_rollover", "", 1),
		ruleCommand("roster.limits", `{"min":"twenty","max":70}`, "next_rollover", "", 1),
		ruleCommand("roster.limits", `{"min":71,"max":70}`, "next_rollover", "", 1),
		ruleCommand("trades.dotSeats", `11`, "next_rollover", "", 1),
		ruleCommand("playoffs.startWeek", `18`, "now", "Push the playoffs back", 1),
		ruleCommand("roster.limits", `{"min":20,"max":70}`, "instant", "", 1),
	} {
		if _, err := s.ExecuteCommissioner(t.Context(), raw, "creator", nil); err == nil {
			t.Fatalf("refusable rule accepted: %s %s", raw.Args.Field, raw.Args.Value)
		} else if strings.Contains(err.Error(), "json:") || strings.Contains(err.Error(), "reflect") {
			t.Fatalf("refusal is not plain: %v", err)
		}
	}
	if revision(t, s) != 1 {
		t.Fatal("refused rule changes were appended")
	}
}

func TestCommissionerPresets(t *testing.T) {
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createWith(commissioner.PresetEmpty, 0), "creator", nil); err == nil {
		t.Fatal("empty league created without a size")
	}
	if _, err := s.ExecuteCommissioner(t.Context(), createWith(commissioner.PresetEmpty, 10), "creator", nil); err != nil {
		t.Fatal(err)
	}
	rules, err := s.CommissionerRules(t.Context(), "native.test")
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range rules.Sections {
		for _, row := range section.Rows {
			if row.State != "Needs a ruling" || row.Value != "Needs a ruling" ||
				row.Citation != "Not set when the league was created" {
				t.Fatalf("empty league invented %s: %+v", row.Field, row)
			}
		}
	}
	people, err := s.CommissionerPeople(t.Context(), "native.test", "creator")
	if err != nil || len(people.Seats) != 10 || people.Seats[9].Name != "Franchise 10" {
		t.Fatalf("empty league franchises: %+v %v", people.Seats, err)
	}
	s, _, _ = newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createWith(commissioner.PresetLegacy, 0), "creator", nil); err != nil {
		t.Fatal(err)
	}
	for field, want := range map[string]string{"cap.amount": "$125,000,000", "roster.limits": "18 to 80",
		"bids.ufa": "Open", "bids.rfa": "Open", "draft.laterPattern": "Needs a ruling", "league.teams": "32"} {
		if row := ruleRow(t, s, field); row.Value != want || row.Citation == "" {
			t.Fatalf("legacy %s: %+v", field, row)
		}
	}
	people, err = s.CommissionerPeople(t.Context(), "native.test", "creator")
	if err != nil || len(people.Seats) != 32 || people.Seats[0].Franchise != "franchise.01" ||
		people.Seats[0].Name != "Franchise 1" || people.Seats[31].Abbreviation != "F32" {
		t.Fatalf("legacy franchises: %+v %v", people.Seats, err)
	}
}

func TestCommissionerRulesReplayAfterReopen(t *testing.T) {
	s, c, pools := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createWith(commissioner.PresetLegacy, 0), "creator", nil); err != nil {
		t.Fatal(err)
	}
	for i, raw := range []commissioner.RawCommand{
		ruleCommand("cap.amount", `13000000000`, "now", "Cap rises with the NFL", 1),
		ruleCommand("waivers.duration", `172800000000000`, "next_rollover", "", 2),
		ruleCommand("draft.laterPattern", `"snake"`, "now", "Ruled at the draft meeting", 3),
	} {
		c.now = c.now.Add(time.Duration(i+1) * time.Minute)
		if _, err := s.ExecuteCommissioner(t.Context(), raw, "creator", nil); err != nil {
			t.Fatal(err)
		}
	}
	executePeople(t, s, "creator", "commish.franchise.edit",
		commissioner.RawPeople{Franchise: "franchise.02", Name: "Hawks", Abbreviation: "HAW"}, 4)
	read := func(s *Store) (commissioner.Rules, commissioner.People, commissioner.Ledger) {
		rules, err := s.CommissionerRules(t.Context(), "native.test")
		if err != nil {
			t.Fatal(err)
		}
		people, err := s.CommissionerPeople(t.Context(), "native.test", "creator")
		if err != nil {
			t.Fatal(err)
		}
		ledger, err := s.CommissionerLedger(t.Context(), "native.test", "creator", nil)
		if err != nil {
			t.Fatal(err)
		}
		return rules, people, ledger
	}
	rules, people, ledger := read(s)
	if ruleRow(t, s, "cap.amount").Value != "$130,000,000" || ruleRow(t, s, "draft.laterPattern").Value != "Snake" ||
		ruleRow(t, s, "waivers.duration").Pending[0].Value != "2 days" {
		t.Fatal("rule changes missing before reopening")
	}
	var path string
	if err := pools.Write().QueryRowContext(t.Context(),
		`SELECT file FROM pragma_database_list WHERE name='main'`).Scan(&path); err != nil {
		t.Fatal(err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := db.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	fresh, _, _ := reopen(t, reopened, registry(t, ""))
	afterRules, afterPeople, afterLedger := read(fresh)
	if !reflect.DeepEqual(rules, afterRules) || !reflect.DeepEqual(people, afterPeople) ||
		!reflect.DeepEqual(ledger, afterLedger) {
		t.Fatal("readings differ after reopening the database")
	}
}
