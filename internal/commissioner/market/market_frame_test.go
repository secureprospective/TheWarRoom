package market

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestMarketDueWithoutSeed(t *testing.T) {
	records := legacyLeague(t)
	if got, err := DueBids("native.test", records, noChangeTime(), nil); err != nil || len(got) != 0 {
		t.Fatalf("bids: %v %v", got, err)
	}
	if got, err := DueWaiverWindows("native.test", records, noChangeTime()); err != nil || len(got) != 0 {
		t.Fatalf("waivers: %v %v", got, err)
	}
	if got, err := NextDueDeadline("native.test", records, noChangeTime()); err != nil || !got.IsZero() {
		t.Fatalf("deadline: %v %v", got, err)
	}
}

func TestFrameDefersRulesAndPreservesEarlyReturns(t *testing.T) {
	records := legacyLeague(t)
	records = append(records, commissioner.Record{Command: commissioner.RawCommand{League: "native.test", Command: commissioner.RulesSet,
		Args: commissioner.RawArgs{Field: "cap.amount", Value: []byte(`"invalid"`)}, Effective: "now"},
		Receipt: commissioner.Receipt{Entry: "bad-rule", RecordedAt: noChangeTime()}})
	f, err := commissioner.NewFrame("native.test", records, noChangeTime(), nil, time.UTC)
	if err != nil {
		t.Fatalf("fold loaded rules: %v", err)
	}
	if err := f.ReadRules(); err == nil {
		t.Fatal("invalid rules accepted")
	}
	if got, err := dueBidsFrame(f); err != nil || len(got) != 0 {
		t.Fatalf("unseeded bids loaded rules: %v %v", got, err)
	}
	if got, err := dueWaiverWindowsFrame(f); err != nil || len(got) != 0 {
		t.Fatalf("unseeded waivers loaded rules: %v %v", got, err)
	}
	if got, err := nextDueDeadlineFrame(f); err != nil || !got.IsZero() {
		t.Fatalf("unseeded deadline loaded rules: %v %v", got, err)
	}
}
