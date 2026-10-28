package report

import (
	"encoding/json"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/lab"
)

func TestIntervalWireAndLabBoundary(t *testing.T) {
	if IntervalFromLab(nil) != nil {
		t.Fatal("nil Lab interval must stay nil")
	}
	got := IntervalFromLab(&lab.Interval{Value: 0.4, Lo: 0.2, Hi: 0.8})
	if *got != (Interval{Value: 0.4, Lo: 0.2, Hi: 0.8}) {
		t.Fatalf("converted interval: %+v", got)
	}
	wire, err := json.Marshal(Range{Interval: *got, Text: "example"})
	if err != nil {
		t.Fatal(err)
	}
	if string(wire) != `{"value":0.4,"lo":0.2,"hi":0.8,"text":"example"}` {
		t.Fatalf("wire changed: %s", wire)
	}
}
