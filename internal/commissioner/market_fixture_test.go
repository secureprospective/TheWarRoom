package commissioner

import (
	"fmt"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

type marketFixture struct {
	b     *ledgerBuilder
	facts nativecheck.Facts
	now   time.Time
}

func newMarketFixture(t *testing.T) *marketFixture {
	t.Helper()
	b := rolloverLedger(t)
	now := seasonMoment(13)
	for i := 1; i <= 32; i++ {
		b.records[1].Command.Args.Native.StandingsOrder = append(b.records[1].Command.Args.Native.StandingsOrder, fmt.Sprintf("franchise.%02d", i))
	}
	for _, rule := range [][2]string{
		{"roster.limits", `{"min":0,"max":100}`},
		{"cap.amount", `100000000000`},
		{"calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["cut","waiver_claim","ufa_bid","trade","ir"]}}`},
		{"calendar.tradeDeadline", `"2026-12-01T00:00:00Z"`},
		{"trades.dotSeats", `4`},
	} {
		b.set(rule[0], rule[1], "now", seasonMoment(5))
	}
	return &marketFixture{b: b, now: now, facts: nativecheck.Facts{"5001": {Name: "Free Agent"}}}
}
