package leaguerules

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCatalogueDescribesEveryField(t *testing.T) {
	fields := Catalogue()
	if len(fields) != len(ruleFields()) {
		t.Fatalf("catalogue has %d of %d fields", len(fields), len(ruleFields()))
	}
	for _, f := range fields {
		if f.Label == "" || f.Section == "" || f.Schema.Kind == "" {
			t.Fatalf("incomplete catalogue entry: %+v", f)
		}
		if f.Competitive != (!strings.HasPrefix(string(f.Path), "house.") && f.Path != "calendar.timezone") {
			t.Fatalf("%s competitive = %v", f.Path, f.Competitive)
		}
	}
	byPath := map[FieldPath]CatalogueField{}
	for _, f := range fields {
		byPath[f.Path] = f
	}
	if s := byPath["trades.dotSeats"].Schema; *s.Minimum != 4 || *s.Maximum != 10 {
		t.Fatalf("DOT seat bounds: %+v", s)
	}
	if s := byPath["bids.ufa"].Schema; !reflect.DeepEqual(s.Choices, []Choice{{"blind", "Blind"}, {"open", "Open"}}) {
		t.Fatalf("bid mode choices: %+v", s)
	}
	if s := byPath["contracts.buyout"].Schema; s.Fields[4].Schema.Key.Kind != "count" {
		t.Fatalf("buyout rate years need a whole-number key: %+v", s.Fields[4])
	}
	if byPath["ir.slots"].Section != "IR" || byPath["bids.rfa"].Section != "Free-agent bids" {
		t.Fatal("section labels are not plain words")
	}
}

func TestDecodeSettingRefusals(t *testing.T) {
	for _, c := range []struct{ path, body string }{
		{"roster.unknown", `{"min":20,"max":70}`},
		{"roster.limits", `{"min":"twenty","max":70}`},
		{"roster.limits", `{"min":20,"max":70,"flex":1}`},
		{"roster.limits", `{"min":71,"max":70}`},
		{"roster.limits", `null`},
		{"roster.limits", `{"min":20,"max":70} {}`},
		{"trades.dotSeats", `11`},
		{"ir.slots", `-1`},
		{"bids.ufa", `"sealed"`},
		{"cap.irShare", `10001`},
		{"contracts.buyout", `{"rates":{"two":6000}}`},
	} {
		if _, err := DecodeSetting(FieldPath(c.path), json.RawMessage(c.body)); err == nil {
			t.Fatalf("%s %s accepted", c.path, c.body)
		} else if strings.Contains(err.Error(), "json") || strings.Contains(err.Error(), "Go value") {
			t.Fatalf("refusal is not plain: %v", err)
		}
	}
	s, err := DecodeSetting("roster.limits", json.RawMessage(`{"min":20,"max":70}`))
	if err != nil || !reflect.DeepEqual(s, Known(Bounds{Min: Known(20), Max: Known(70)})) {
		t.Fatalf("roster limits: %#v %v", s, err)
	}
	s, err = DecodeSetting("contracts.buyout", json.RawMessage(`{"enabled":true,"rates":{"2":6000}}`))
	rates := s.(Value[BuyoutRules]).Value.Rates.Value
	if err != nil || rates[2] != Known[Decimal](6000) {
		t.Fatalf("whole-number keys: %#v %v", s, err)
	}
}

func TestStoredSettingsRoundTrip(t *testing.T) {
	at := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	for _, c := range SeedChanges(LegacyNFL(), "native.test", 1, at) {
		body, err := EncodeStored(c.New)
		if err != nil {
			t.Fatal(err)
		}
		back, err := DecodeStored(c.Field, body)
		if err != nil || !reflect.DeepEqual(back, c.New) {
			t.Fatalf("%s did not survive the ledger: %v", c.Field, err)
		}
	}
	if _, err := DecodeStored("roster.limits", json.RawMessage(`{"State":1,"Value":{"Min":{"State":1,"Value":9},`+
		`"Max":{"State":1,"Value":2}},"Citation":""}`)); err == nil {
		t.Fatal("stored inverted range accepted")
	}
	if _, err := DecodeStored("roster.limits", json.RawMessage(`{"State":1,"Value":{},"Extra":1}`)); err == nil {
		t.Fatal("stored unknown detail accepted")
	}
}

func TestSettingWords(t *testing.T) {
	rules := LegacyNFL()
	words := map[FieldPath]string{}
	for i, f := range Catalogue() {
		words[f.Path] = ReadSettings(rules)[i].Words
	}
	for path, want := range map[FieldPath]string{
		"cap.amount": "$125,000,000", "roster.limits": "18 to 80", "bids.ufa": "Open", "bids.rfa": "Open",
		"waivers.duration": "1 day", "cap.irShare": "100%", "contracts.deadCap": "35%",
		"trades.review": "DOT", "draft.laterPattern": "Needs a ruling", "calendar.rolloverDate": "1 March",
		"bids.yearMultipliers": "1; 1.2; 1.4; 1.6", "lineup.groups": "Defense: Exactly 12; Offense: At most 8; " +
			"Total: Exactly 21", "standings.criteria": "Win percentage; Head to head; Division percentage; " +
			"Conference percentage; All play percentage; Points; Power rank",
	} {
		if words[path] != want {
			t.Fatalf("%s reads %q, want %q", path, words[path], want)
		}
	}
	if got := renderMoney(-12345); got != "-$123.45" {
		t.Fatalf("cents: %s", got)
	}
	if got := renderDuration(26*time.Hour + 30*time.Minute); got != "1 day 2 hours 30 minutes" {
		t.Fatalf("duration: %s", got)
	}
	if plainName("UFAEligible") != "UFA eligible" || plainName("QB") != "QB" {
		t.Fatalf("acronyms: %s %s", plainName("UFAEligible"), plainName("QB"))
	}
	for path, text := range words {
		free := path == "house.texts" || path == "calendar.timezone"
		if !free && (strings.Contains(text, "_") || strings.Contains(text, "<")) {
			t.Fatalf("%s shows an internal term: %s", path, text)
		}
	}
}
