package commissioner

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestAFinalLineSaysHowItMovedTheLastLiveOne(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.seed(2026, nil, seasonMoment(2))
	final := completeScoreEvidence()
	live := *final.Held
	live.Authority = source.ESPNLive
	live.Week = &source.PlayerWeek{Position: "LB", Season: 2026, Week: 1,
		Line: fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"tackle": 20000}}}
	final.Held.Authority = source.NFLVerse
	scope := source.RawNativeScope{Player: "0099", Season: 2026, Week: 1}
	read := func(lines ...source.HeldWeek) NativePlayerScore {
		t.Helper()
		final.Lines = nil
		for seq, held := range lines {
			final.Lines = append(final.Lines, source.LineRevision{Seq: seq, Digest: source.LineDigest(*held.Week), Held: held})
		}
		out, err := NativePlayerScoreReading("native.test", scope, b.records, final, seasonMoment(5))
		if err != nil || out.Status != "Complete" || *out.Points != 15000 {
			t.Fatalf("reading: %+v %v", out, err)
		}
		return out
	}
	got := read(live, *final.Held).Revision
	if got == nil || got.Was != 30000 || got.Now != 15000 || len(got.Stats) != 1 ||
		got.Stats[0] != (StatMove{Stat: "tackle", Label: "Tackle", Was: 20000, Now: 10000}) {
		t.Fatalf("revision: %+v", got)
	}
	same := live
	same.Week = final.Held.Week
	if out := read(same, *final.Held); out.Revision != nil {
		t.Fatalf("an unmoved score has a revision: %+v", out.Revision)
	}
	if out := read(*final.Held); out.Revision != nil {
		t.Fatalf("a week with no live line has a revision: %+v", out.Revision)
	}
}
