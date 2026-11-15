package commissioner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func noChangeTime() time.Time { return time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC) }

type CreatorAuthorizer struct{ Creator string }

func (a CreatorAuthorizer) Allow(_ string, principal, _ string) (string, error) {
	if a.Creator != "" && a.Creator == principal {
		return "commissioner", nil
	}
	return "", fmt.Errorf("only this league's commissioner may make this change")
}

type scoreSource struct {
	lineup WeekLineup
	score  NativePlayerScore
	err    error
}

func (s scoreSource) Lineup(context.Context, []Record, string, int) (WeekLineup, error) {
	return s.lineup, s.err
}

func (s scoreSource) PlayerScore(context.Context, []Record, string, int) (NativePlayerScore, error) {
	return s.score, s.err
}

func tackleVersion(seq int, position string, tackles leaguerules.Decimal, day int) source.LineRevision {
	week := source.PlayerWeek{Position: position, Season: 2026, Week: 1, SeasonType: "REG",
		Line: fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"tackle": tackles}}}
	return source.LineRevision{Seq: seq, Digest: source.LineDigest(week), Held: source.HeldWeek{
		SHA256: fmt.Sprint("file-", seq), FetchedAt: seasonMoment(day), Week: &week}}
}

type fixedFeed struct{ weeks []StatWeek }

func (f fixedFeed) Weeks(context.Context) ([]StatWeek, error) { return f.weeks, nil }

func (f fixedFeed) Name(player string) string {
	if player == "0099" {
		return "Josh Allen"
	}
	return ""
}

type noGames struct{}

func (noGames) Lineup(context.Context, []Record, string, int) (WeekLineup, error) {
	return WeekLineup{}, nil
}

func (noGames) PlayerScore(context.Context, []Record, string, int) (NativePlayerScore, error) {
	return NativePlayerScore{}, nil
}

func statWeek(versions ...source.LineRevision) StatWeek {
	return StatWeek{Player: "0099", Week: 1, Evidence: source.NativeEvidence{GSIS: "00-0023459",
		Held: &versions[len(versions)-1].Held, Lines: versions}}
}

func (b *ledgerBuilder) statCommand(command string, change RawStatChange, day int) string {
	b.t.Helper()
	return b.add(RawCommand{League: "native.test", Command: command, Args: RawArgs{Stat: &change},
		Request: fmt.Sprint("stat.", command, ".", len(b.records)), Effective: "now", Visibility: "league"},
		seasonMoment(day))
}

func queueOf(t *testing.T, b *ledgerBuilder, weeks ...StatWeek) StatQueue {
	t.Helper()
	out, err := StatQueueReading(t.Context(), "native.test", b.records, fixedFeed{weeks}, noGames{}, 2026, seasonMoment(20))
	if err != nil {
		t.Fatal(err)
	}
	out.Decided = slices.DeleteFunc(out.Decided, func(d StatDecision) bool { return strings.HasPrefix(d.Reason, "fixture: pin") })
	return out
}

func pinnedStatLedger(t *testing.T) *ledgerBuilder {
	b := seededLedger(t)
	first := tackleVersion(0, "LB", 10000, 5)
	for _, player := range []string{"0099", "0100"} {
		id := StatItemID{Season: 2026, Week: 1, Player: player, Seq: 0, Digest: first.Digest[:statDigestLen]}
		b.add(RawCommand{League: "native.test", Command: StatApprove, Args: RawArgs{Stat: &RawStatChange{Item: id.String()}}, Request: "pin." + player, Reason: "fixture: pin original revision", Effective: "now", Visibility: "league"}, seasonMoment(5))
	}
	return b
}

func seededLedger(t *testing.T) *ledgerBuilder {
	t.Helper()
	body, err := os.ReadFile("testdata/native-assets.wire.json")
	if err != nil {
		t.Fatal(err)
	}
	var wire NativeAssets
	if err := json.Unmarshal(body, &wire); err != nil || wire.Snapshot == nil {
		t.Fatalf("held snapshot: %v", err)
	}
	b := newLedger(t, PresetLegacy)
	b.add(RawCommand{League: "native.test", Command: NativeSeed, Args: RawArgs{Native: wire.Snapshot},
		Request: "seed", Effective: "now", Visibility: "league"}, seasonMoment(2))
	return b
}

func itemOf(version source.LineRevision) string {
	return StatItemID{Season: 2026, Week: 1, Player: "0099", Seq: version.Seq, Digest: version.Digest[:statDigestLen]}.String()
}
