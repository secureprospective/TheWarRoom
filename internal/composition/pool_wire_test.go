package composition

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

const poolWireFile = "frontend/src/app/pool/wire.testdata.json"

// normalizeWire fixes what changes between runs: times, the random plan ids (numbered in order
// of appearance), ages and the build stamp, so the file the frontend tests parse can be
// compared byte for byte.
func normalizeWire(raw []byte) []byte {
	seen := map[string]string{}
	raw = regexp.MustCompile(`bid-[0-9a-f]{32}`).ReplaceAllFunc(raw, func(id []byte) []byte {
		if _, ok := seen[string(id)]; !ok {
			seen[string(id)] = fmt.Sprintf("bid-%032d", len(seen)+1)
		}
		return []byte(seen[string(id)])
	})
	for pattern, fixed := range map[string]string{
		`\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d+)?Z`: "2026-10-10T12:00:00Z",
		`"age":"\d+"`:          `"age":"27"`,
		`"buildStamp":"[^"]*"`: `"buildStamp":"dev test"`,
		`As of: [A-Za-z]{3} \d{1,2}, \d{4}, \d{1,2}:\d{2} [AP]M`: "As of: Oct 10, 2026, 12:00 PM",
	} {
		raw = regexp.MustCompile(pattern).ReplaceAll(raw, []byte(fixed))
	}
	return raw
}

// TestPoolWireFixtureMatchesGo keeps the frontend's wire fixture honest: the frontend parsers
// are tested against exactly what these bindings return. Regenerate with TWR_WRITE_WIRE=1.
func TestPoolWireFixtureMatchesGo(t *testing.T) {
	t.Parallel()
	a, _ := poolApp(t)
	if err := offersTestAPI(a).TargetPoolRefresh(true); err != nil {
		t.Fatal(err)
	}
	// Plans list newest first, so each draft gets its own second.
	clock := time.Now().Add(-time.Hour)
	a.core.MovesNow = func() time.Time { clock = clock.Add(time.Second); return clock }
	raw := rawBid("5", 3)
	raw.Snipe, raw.Beat, raw.Evidence = true, "5.500", "https://example.invalid/thread"
	if _, err := offersTestAPI(a).TargetBidDraft(raw); err != nil {
		t.Fatal(err)
	}
	a.core.Week = leagueweek.Week{Number: 13}
	blocked := rawBid("2", 1)
	blocked.Player = "13595"
	if _, err := offersTestAPI(a).TargetBidDraft(blocked); err != nil {
		t.Fatal(err)
	}
	a.core.Week = leagueweek.Week{Number: 5}
	dropped := rawBid("1", 1)
	dropped.Player = "13594"
	plan, err := offersTestAPI(a).TargetBidDraft(dropped)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := offersTestAPI(a).TargetBidHandOff(plan.Receipt.CorrelationID); err != nil {
		t.Fatal(err)
	}
	if _, err := offersTestAPI(a).TargetBidWithdraw(plan.Receipt.CorrelationID); err != nil {
		t.Fatal(err)
	}
	wire := map[string]any{
		"pool":    offersTestAPI(a).TargetPool(RawPoolScope{FranchiseID: "0001", Sort: "name"}),
		"empty":   offersTestAPI(a).TargetPool(RawPoolScope{Sort: "bogus"}),
		"board":   offersTestAPI(a).TargetBidPlans("0001"),
		"preview": offersTestAPI(a).TargetBidPreview(rawBid("5.004", 3)),
		"problem": offersTestAPI(a).TargetBidPreview(rawBid("", 3)),
	}
	got, err := json.MarshalIndent(wire, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(normalizeWire(got), '\n')
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(poolWireFile, got, 0o644); err != nil { //nolint:gosec // a test fixture
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(poolWireFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(got) {
		t.Errorf("%s is out of date; run TWR_WRITE_WIRE=1 go test -run TestPoolWireFixtureMatchesGo .", poolWireFile)
	}
}
