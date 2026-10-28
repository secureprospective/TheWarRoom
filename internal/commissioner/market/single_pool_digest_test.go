package market

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// A single-pool league reads exactly as it did before per-side pools: the assets and market after every
// market command, and the commissioner's choices for each, hash to what they were at b8a933e4.
func TestSinglePoolMarketIsUnchanged(t *testing.T) {
	f := marketBodiesLedger(t)
	h := sha256.New()
	write := func(v any) {
		body, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		h.Write(body)
	}
	assets, err := commissioner.NativeAssetsReading(rolloverLeague, f.b.records, f.now)
	if err != nil {
		t.Fatal(err)
	}
	write(assets)
	for _, command := range []string{commissioner.BidsProcess, commissioner.BidsOverride, commissioner.WaiverProcess,
		commissioner.WaiverOverride, commissioner.TradeOverride} {
		choices, err := ChoicesReading(rolloverLeague, command, "creator", f.b.records, f.now, time.UTC, f.facts)
		if err != nil {
			t.Fatal(err)
		}
		write(choices)
	}
	if got := hex.EncodeToString(h.Sum(nil))[:16]; got != "ecd30d23b91f56e8" {
		t.Fatalf("single-pool market digest %s", got)
	}
}
