package labcompile

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/lab"
)

func decodeCompiled(t *testing.T, body []byte) *lab.Data {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	data, err := lab.DecodeGzip(buf.Bytes(), hex.EncodeToString(sum[:]))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func TestReducedCompile(t *testing.T) {
	f := referenceFacts(t, false)
	in, _ := frozenEvidence(t)
	in.Pages = PageExam{}
	body, err := Compile(t.Context(), f, in)
	if err != nil {
		t.Fatal(err)
	}
	data := decodeCompiled(t, body)
	if len(data.Trades) == 0 || len(data.Players) == 0 {
		t.Fatal("empty compile")
	}
	t.Logf("reduced smoke: trades=%d legs=%d players=%d", len(data.Trades), len(data.Legs), len(data.Players))
	var doc struct {
		Market  Market
		Checks2 AnalysisChecks
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Checks2.Forecast) != 0 || doc.Checks2.Backtest != nil {
		t.Fatal("reduced zero-variance checks must be absent")
	}
	if len(doc.Market.Classes) == 0 {
		t.Fatal("missing reduced market")
	}
	for _, c := range doc.Market.Classes {
		if c.Delivered != nil || c.PerDelivered != nil {
			t.Fatalf("%s fabricated delivery evidence", c.ID)
		}
	}
	t.Log("reduced: forecast checks empty; backtest null; all delivered ratios/intervals null")
}
func TestFullAnalysisParity(t *testing.T) {
	if os.Getenv("TWR_ARCHIVE_PARITY") != "1" {
		t.Skip("full archive parity is opt-in")
	}
	if os.Getenv("TWR_FACTS_ARCHIVE") == "" {
		t.Fatal("set TWR_FACTS_ARCHIVE")
	}
	f := referenceFacts(t, true)
	in, want := frozenEvidence(t)
	before, stop, err := sampleRSS()
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	m, body, err := Build(t.Context(), f, in)
	rss := stop()
	if err != nil {
		t.Fatal(err)
	}
	if rss.Err != nil {
		t.Fatal(rss.Err)
	}
	t.Logf("full compile=%s RSS baseline=%d peak=%d rise=%d KiB", time.Since(start), before, rss.Peak, rss.Peak-before)
	data := decodeCompiled(t, body)
	t.Logf("Lab decode trades=%d legs=%d draft=%d", len(data.Trades), len(data.Legs), len(data.Draft))
	var got map[string]json.RawMessage
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range sortedRawKeys(want) {
		count, diffs := compareJSON(t, got[key], want[key], key)
		t.Logf("P4 %s values=%d differences=%d", key, count, diffs)
	}
	for key := range got {
		if _, ok := want[key]; !ok {
			t.Errorf("unexpected key %s", key)
		}
	}
	provePlayerLegs(t, m, data)
}
func provePlayerLegs(t *testing.T, m *Model, data *lab.Data) {
	t.Helper()
	count, unavailable := 0, 0
	strata := map[string]bool{}
	for _, leg := range data.Legs {
		tr := data.Trades[leg.Trade]
		if leg.Kind != "player" || leg.XClass == nil || *leg.XClass == "player:unknown" {
			continue
		}
		got := m.Player(value(leg.Pid), time.Unix(int64(tr.Ts), 0))
		if !got.Valued {
			if got.Reason == "" || got.Forecast != nil || got.Price != nil || value(leg.XValue) != 0 {
				t.Fatalf("unavailable forecast fabricated a value: %s", value(leg.Pid))
			}
			unavailable++
			continue
		}
		if count >= 256 && strata[got.Class] {
			continue
		}
		if got.Discounted != value(leg.XValue) || got.Class != value(leg.XClass) || got.Price == nil || value(got.Price) != value(leg.XPrice) {
			t.Errorf("asset API mismatch trade=%d player=%s", leg.Trade, value(leg.Pid))
		}
		if got.Label != classLabel(got.Class) || got.Range == nil || got.Range.Low > value(got.Price) || got.Range.High < value(got.Price) {
			t.Errorf("asset price range or label missing trade=%d player=%s: %+v", leg.Trade, value(leg.Pid), got.Range)
		}
		count++
		strata[got.Class] = true
	}
	if count < 256 || len(strata) != 24 {
		t.Fatalf("insufficient player API coverage: legs=%d classes=%d", count, len(strata))
	}
	t.Logf("player API: %d trade legs across %d classes equal xValue/xClass/xPrice; %d historical zero-placeholder legs correctly unvalued", count, len(strata), unavailable)
}
