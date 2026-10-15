package source_test

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func yardsWeek(yards leaguerules.Decimal, position string) source.PlayerWeek {
	return source.PlayerWeek{GSIS: "00-0023459", Position: position, Season: 2026, Week: 1, SeasonType: "REG",
		Line: fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"passing_yards": yards}}}
}

func held(sha string, minute int, week *source.PlayerWeek) source.HeldWeek {
	return source.HeldWeek{SHA256: sha, FetchedAt: time.Date(2026, 10, 13, 12, minute, 0, 0, time.UTC), Week: week}
}

func pointer(week source.PlayerWeek) *source.PlayerWeek { return &week }

func TestDistinctLinesKeepsEveryChangeAndReturnToAnOldLine(t *testing.T) {
	a, b := yardsWeek(2440000, "QB"), yardsWeek(2450000, "QB")
	// Newest first: the feed went 244, 245, a file without the player, a refetch of 245, then back to 244.
	files := []source.HeldWeek{held("f5", 5, pointer(a)), held("f4", 4, pointer(b)), held("f3", 3, nil),
		held("f2", 2, pointer(b)), held("f1", 1, pointer(a))}
	lines := source.DistinctLines(files)
	if len(lines) != 3 {
		t.Fatalf("want 244, 245, 244 again, got %d versions", len(lines))
	}
	for i, want := range []leaguerules.Decimal{2440000, 2450000, 2440000} {
		if lines[i].Seq != i || lines[i].Held.Week.Line.Quantities["passing_yards"] != want {
			t.Fatalf("version %d: %+v", i, lines[i])
		}
	}
	if lines[1].Held.SHA256 != "f2" {
		t.Fatalf("a version is held by the earliest file that carried it, got %s", lines[1].Held.SHA256)
	}
	if lines[0].Digest != lines[2].Digest || lines[0].Digest == lines[1].Digest {
		t.Fatal("equal lines must share a digest and different lines must not")
	}
}

func TestLineDigestSeesPositionAndEveryQuantity(t *testing.T) {
	base := source.LineDigest(yardsWeek(2440000, "QB"))
	if base != source.LineDigest(yardsWeek(2440000, "QB")) {
		t.Fatal("the digest is not stable")
	}
	for name, other := range map[string]source.PlayerWeek{"yards": yardsWeek(2440001, "QB"),
		"position": yardsWeek(2440000, "WR")} {
		if source.LineDigest(other) == base {
			t.Fatalf("a changed %s left the digest unchanged", name)
		}
	}
}

func TestLineBookAgreesWithTheSinglePlayerHistory(t *testing.T) {
	a, b := yardsWeek(2440000, "QB"), yardsWeek(2450000, "QB")
	other := yardsWeek(100000, "WR")
	other.GSIS, other.SeasonType = "00-0099999", "POST"
	book := source.NewLineBook(2026)
	at := func(minute int) time.Time { return time.Date(2026, 10, 13, 12, minute, 0, 0, time.UTC) }
	book.Add("f3", at(3), []source.PlayerWeek{a, other})
	book.Add("f2", at(2), []source.PlayerWeek{b, other})
	book.Add("f1", at(1), []source.PlayerWeek{a})
	versions := book.Versions()
	got := versions[source.WeekKey{GSIS: "00-0023459", Week: 1}]
	want := source.DistinctLines([]source.HeldWeek{held("f3", 3, &a), held("f2", 2, &b), held("f1", 1, &a)})
	if len(got) != 3 || len(want) != 3 {
		t.Fatalf("versions %d and %d", len(got), len(want))
	}
	for i := range got {
		if got[i].Seq != want[i].Seq || got[i].Digest != want[i].Digest || got[i].Held.SHA256 != want[i].Held.SHA256 {
			t.Fatalf("the two readings disagree at %d: %+v %+v", i, got[i], want[i])
		}
	}
	if len(versions) != 1 {
		t.Fatalf("the postseason week must not be a version: %v", versions)
	}
}

func TestEvidenceSuppliedAsOneHeldWeekHasOneVersion(t *testing.T) {
	week := yardsWeek(2440000, "QB")
	one := source.NativeEvidence{Held: &source.HeldWeek{SHA256: "x", Week: &week}}
	if got := one.Versions(); len(got) != 1 || got[0].Digest != source.LineDigest(week) {
		t.Fatalf("one held week: %+v", got)
	}
	if got := (source.NativeEvidence{Held: &source.HeldWeek{SHA256: "x"}}).Versions(); len(got) != 0 {
		t.Fatalf("an absent player is not a version: %+v", got)
	}
}
