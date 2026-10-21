package comms

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
)

func TestCommsWireTwinsMatch(t *testing.T) {
	row := report.Row{Sort: []float64{1}, Titles: []string{"title"}, Cells: []string{"cell"}, Range: &report.Range{Max: 2, Text: "range"}, Mine: true, Bars: []float64{2}, Group: "group"}
	reading := report.Report{Entities: []report.Entity{{Text: "entity", Kind: "team", ID: "0001"}}, SortAscending: "oldest", SortDescending: "newest",
		ID: "id", Title: "title", Note: "note", Status: "ok", Source: "source", AsOf: "asOf", How: "how", Columns: []string{"column"}, Rows: []report.Row{row}, Summary: []report.Row{row},
		Link: &report.Link{Label: "label", Endpoint: "endpoint"}, BarMax: 3, BarLabels: []string{"bar"}}
	for _, board := range []Board{{}, {Events: []Event{}, BoardPosts: []report.Report{}},
		{Feed: reading, Transactions: reading, Events: []Event{{Sentence: "sentence", Date: "date"}}, BoardNote: "note", LeagueTalk: "talk", BoardPosts: []report.Report{reading},
			Team: "team", Kind: "kind", Teams: []report.Choice{{Value: "team", Label: "label"}}, Kinds: []report.Choice{{Value: "kind", Label: "label"}}},
	} {
		source, err := json.Marshal(board)
		if err != nil {
			t.Fatal(err)
		}
		wire, err := json.Marshal(wireComms(board))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(source, wire) {
			t.Fatalf("wire twin differs\n%s\n%s", source, wire)
		}
	}
}
