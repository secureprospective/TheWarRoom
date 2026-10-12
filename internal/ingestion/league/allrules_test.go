package league

import (
	"os"
	"testing"
)

func TestRuleDictionaryReadsTheArchivedExport(t *testing.T) {
	body, err := os.ReadFile("testdata/allRules.json")
	if err != nil {
		t.Fatal(err)
	}
	codes, err := ParseRuleDictionary(body)
	if err != nil {
		t.Fatal(err)
	}
	names := RuleNames(codes)
	if len(codes) != 153 || names["PY"] != "Passing Yards" || names["#P"] != "Number of Passing TDs" {
		t.Fatalf("%d codes, PY %q", len(codes), names["PY"])
	}
	if _, err := ParseRuleDictionary([]byte(`{"allRules":{"rule":{"abbreviation":{"$t":" "}}}}`)); err == nil {
		t.Fatal("an entry without a code was accepted")
	}
	if _, err := ParseRuleDictionary([]byte(`{"error":{"$t":"x"}}`)); err == nil {
		t.Fatal("an MFL error body was read as the dictionary")
	}
}
