package commissioner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// tackleVersion is one version of a linebacker's week: some number of tackles, as the feed reported it.
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

// noGames has no published schedule, so no game can be called either way.
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
	return out
}

// seededLedger is a Legacy league started from the held snapshot, the way a real league is seeded.
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

func TestStatQueueMakesAnItemOnlyWhereTheFeedChangedWhatTheLeagueCounted(t *testing.T) {
	b := seededLedger(t)
	first, second := tackleVersion(0, "LB", 10000, 5), tackleVersion(1, "LB", 20000, 6)
	if out := queueOf(t, b, statWeek(first)); len(out.Open) != 0 || out.Words != "No statistics are waiting for a decision" {
		t.Fatalf("an unrevised week made an item: %+v", out)
	}
	out := queueOf(t, b, statWeek(first, second))
	if len(out.Open) != 1 {
		t.Fatalf("one changed week, one item: %+v", out)
	}
	item := out.Open[0]
	if item.ID != itemOf(second) || item.Name != "Josh Allen" || item.Points == nil || *item.Points != 15000 ||
		item.Words != "Josh Allen, week 1: tackle 1 → 2, +1.5 points, we can't yet tell whether a result changes" {
		t.Fatalf("the item reads wrong: %+v", item)
	}
	if out.Words != "1 statistic change is waiting for a decision" {
		t.Fatalf("queue words: %q", out.Words)
	}
}

func TestStatQueueRejectedItemStaysOutUntilTheFeedChangesAgain(t *testing.T) {
	b := seededLedger(t)
	first, second, third := tackleVersion(0, "LB", 10000, 5), tackleVersion(1, "LB", 20000, 6), tackleVersion(2, "LB", 30000, 7)
	rejection := b.statCommand(StatReject, RawStatChange{Item: itemOf(second)}, 8)
	if out := queueOf(t, b, statWeek(first, second)); len(out.Open) != 0 || len(out.Decided) != 1 ||
		out.Decided[0].Status != "Rejected" || out.Decided[0].By != "creator" || !out.Decided[0].At.Equal(seasonMoment(8)) {
		t.Fatalf("a rejected item came back or was not recorded: %+v", out)
	}
	if out := queueOf(t, b, statWeek(first, second)); len(out.Open) != 0 {
		t.Fatal("the same revision reappeared")
	}
	if out := queueOf(t, b, statWeek(first, second, third)); len(out.Open) != 1 || out.Open[0].ID != itemOf(third) {
		t.Fatalf("a new revision must come back: %+v", out)
	}
	b.revert(rejection, seasonMoment(9))
	out := queueOf(t, b, statWeek(first, second))
	if len(out.Open) != 1 || out.Decided[0].Status != "Rejection undone" {
		t.Fatalf("an undone rejection must reopen the item: %+v", out)
	}
}

func TestStatQueueApprovalMovesTheBaselineAndUndoMovesItBack(t *testing.T) {
	b := seededLedger(t)
	first, second := tackleVersion(0, "LB", 10000, 5), tackleVersion(1, "LB", 20000, 6)
	approval := b.statCommand(StatApprove, RawStatChange{Item: itemOf(second)}, 8)
	out := queueOf(t, b, statWeek(first, second))
	if len(out.Open) != 0 || len(out.Decided) != 1 || out.Decided[0].Status != "Approved" ||
		out.Decided[0].Words != "Josh Allen, week 1: tackle 1 → 2" {
		t.Fatalf("an approved item still waits or was not recorded: %+v", out)
	}
	b.revert(approval, seasonMoment(9))
	out = queueOf(t, b, statWeek(first, second))
	if len(out.Open) != 1 || out.Decided[0].Status != "Approval undone" {
		t.Fatalf("undoing the approval must reopen the item: %+v", out)
	}
}

func TestStatQueueNewestPositionChangeIsAnItemToo(t *testing.T) {
	b := seededLedger(t)
	first, second := tackleVersion(0, "LB", 10000, 5), tackleVersion(1, "DT", 10000, 6)
	out := queueOf(t, b, statWeek(first, second))
	if len(out.Open) != 1 || !strings.Contains(out.Open[0].Words, "position LB → DT, +1 point") {
		t.Fatalf("a position change moves the score and must queue: %+v", out)
	}
}

func TestStatQueueRefusesWhatTheLeagueHoldsOnlyAfterSeeding(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	out := queueOf(t, b, statWeek(tackleVersion(0, "LB", 10000, 5), tackleVersion(1, "LB", 20000, 6)))
	if len(out.Open) != 0 || out.Words != "Record the starting snapshot first" {
		t.Fatalf("an unseeded league queued: %+v", out)
	}
}

func TestStatQueueNewestFirst(t *testing.T) {
	b := seededLedger(t)
	older := statWeek(tackleVersion(0, "LB", 10000, 5), tackleVersion(1, "LB", 20000, 6))
	newer := statWeek(tackleVersion(0, "LB", 10000, 5), tackleVersion(1, "LB", 30000, 9))
	newer.Player, older.Player = "0100", "0099"
	out := queueOf(t, b, older, newer)
	if len(out.Open) != 2 || out.Open[0].Player != "0100" || out.Open[1].Player != "0099" {
		t.Fatalf("not newest first: %+v", out.Open)
	}
}
