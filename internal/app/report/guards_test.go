package report

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/offline"
)

func TestOfflineSentenceMatchesFrontend(t *testing.T) {
	body, err := os.ReadFile("../../../frontend/src/app/data/plainFailure.ts")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"`+offline.Advice+`"`) || !strings.Contains(string(body), `"Can't reach the league site." + ADVICE`) ||
		Offline != "Can't reach the league site."+offline.Advice {
		t.Fatal("frontend ADVICE or OFFLINE differs from internal/offline")
	}
}

func TestIntervalWire(t *testing.T) {
	wire, err := json.Marshal(Range{Interval: Interval{Value: 0.4, Lo: 0.2, Hi: 0.8}, Text: "example"})
	if err != nil {
		t.Fatal(err)
	}
	if string(wire) != `{"value":0.4,"lo":0.2,"hi":0.8,"text":"example"}` {
		t.Fatalf("wire changed: %s", wire)
	}
}
