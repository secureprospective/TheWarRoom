package history

import (
	"context"
	"fmt"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

func TestDurableLiveRetentionRunsOnlyAfterFinalCounts(t *testing.T) {
	for _, manual := range []bool{false, true} {
		t.Run(fmt.Sprintf("pinned=%t", manual), func(t *testing.T) {
			f := statResultsTest(t)
			player := f.starter(f.a)
			scope := source.RawNativeScope{Player: player, Season: 2026, Week: 2}
			setLiveStats(t, f.s, 2026, 2, t0(), source.ESPNLive, nil, map[string]source.PlayerWeek{player: tackleWeek(2, 30000)})
			evidence, err := f.s.nativeEvidence(t.Context(), f.s, scope)
			if err != nil {
				t.Fatal(err)
			}
			pin := *commissioner.FeedVersion(evidence.Versions())
			if manual {
				item := commissioner.StatItemID{Player: player, Season: 2026, Week: 2, Seq: pin.Seq, Digest: pin.Digest[:16]}.String()
				f.approve(t, item, "retain-live-pin")
			}
			// The intermediate capture was never exposed to a score or approval.
			setLiveStats(t, f.s, 2026, 2, t0(), source.ESPNLive, nil, map[string]source.PlayerWeek{player: tackleWeek(2, 40000)})
			setLiveStats(t, f.s, 2026, 2, t0(), source.ESPNLive, nil, map[string]source.PlayerWeek{player: tackleWeek(2, 50000)})
			finalLine := tackleWeek(2, 90000)
			finalHeld := source.HeldWeek{Authority: source.NFLVerse, SHA256: "final", Week: &finalLine}
			final := source.NativeEvidence{Held: &finalHeld, Lines: source.DistinctLines([]source.HeldWeek{finalHeld})}
			f.s.evidence = func(context.Context, source.RawNativeScope) (source.NativeEvidence, error) { return final, nil }
			f.s.statWeeks = func(context.Context, int) ([]commissioner.StatWeek, error) {
				return []commissioner.StatWeek{{Player: player, Week: 2, Evidence: final}}, nil
			}
			if err := f.s.ProcessStatFeed(t.Context(), "native.test", 2026); err != nil {
				t.Fatal(err)
			}
			wantRows, wantPoints := 2, 135000
			if manual {
				wantRows, wantPoints = 3, 45000
			}
			if n, _ := liveRowCounts(t, f.pools); n != wantRows {
				t.Fatalf("rows %d want %d", n, wantRows)
			}
			before := f.score(t, f.a, 2)
			if before.Points == nil || int(*before.Points) != wantPoints {
				t.Fatalf("wrong counted score %+v", before)
			}
			f.reopen(t)
			after := f.score(t, f.a, 2)
			if after.Points == nil || *after.Points != *before.Points || after.SHA256 != before.SHA256 {
				t.Fatal("retention changed replayed score")
			}
		})
	}
}
