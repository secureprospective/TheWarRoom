package sitenews

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// leftTag is a tag left in the text; a bare ">" is MFL's menu path ("League > Setup").
var leftTag = regexp.MustCompile(`</?[a-zA-Z]`)

func TestSampleReadsNewestFirstInPlainWords(t *testing.T) {
	body, err := os.ReadFile("testdata/siteNews.json")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	items, err := Read(raw)
	if err != nil || len(items) != 23 {
		t.Fatalf("items %d, %v", len(items), err)
	}
	for i := 1; i < len(items); i++ {
		if items[i].Published.After(items[i-1].Published) {
			t.Fatal("not newest first")
		}
	}
	first := items[0]
	if first.Headline != "Week 4 Official Statistics Changes - Stat Changes for Week #4" || len(first.Lines) < 3 {
		t.Fatalf("first = %+v", first)
	}
	for _, it := range items {
		for _, l := range it.Lines {
			if leftTag.MatchString(l) || strings.Contains(l, "&amp;") {
				t.Fatalf("markup left in %q", l)
			}
		}
	}
}

func TestPlainLines(t *testing.T) {
	got := PlainLines(`<p>Stat changes &amp; more: <a href="x?L=<league_id>&C=LINEUP">Setup</a></p><p><b>Jets at Bears</b><ul><li>A  B: from 1 to 0.</li><li>C D</li></ul></p>`)
	want := []string{"Stat changes & more: Setup", "Jets at Bears", "A B: from 1 to 0.", "C D"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("%q", got)
	}
}

func TestErrorBodyAndMissingDate(t *testing.T) {
	if _, err := Parse([]byte(`{"error":{"$t":"Invalid request"}}`)); err == nil {
		t.Fatal("an MFL error body was read as news")
	}
	if _, err := Read([]RawItem{{Title: text{T: "x"}, PubDate: text{T: "yesterday"}}}); err == nil {
		t.Fatal("an unreadable date was accepted")
	}
}
