package report

import (
	"encoding/json"
	"testing"
)

func TestIntervalWire(t *testing.T) {
	wire, err := json.Marshal(Range{Interval: Interval{Value: 0.4, Lo: 0.2, Hi: 0.8}, Text: "example"})
	if err != nil {
		t.Fatal(err)
	}
	if string(wire) != `{"value":0.4,"lo":0.2,"hi":0.8,"text":"example"}` {
		t.Fatalf("wire changed: %s", wire)
	}
}
