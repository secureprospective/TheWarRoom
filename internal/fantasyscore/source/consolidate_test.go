package source_test

import (
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestScoreSourceFinalConsolidation(t *testing.T) {
	for _, authority := range []string{source.ESPNLive, source.Sleeper, source.ESPNLive + " + " + source.Sleeper} {
		t.Run(authority, func(t *testing.T) {
			final, live := yardsWeek(0, "QB"), yardsWeek(70000, "QB")
			live.Line.Quantities["receiving_drop"] = 20000
			live.Line.Events = map[leaguerules.StatID][]leaguerules.Decimal{"long_pass": {420000}}
			live.FieldSources = map[leaguerules.StatID]string{"receiving_drop": source.Sleeper}
			versions := []source.LineRevision{
				{Seq: 0, Digest: source.LineDigest(final), Held: source.HeldWeek{Authority: source.NFLVerse, Week: &final}},
				{Seq: 1, Digest: source.LineDigest(live), Held: source.HeldWeek{Authority: authority, Week: &live}},
			}
			before := append([]source.LineRevision{}, versions...)
			got := source.ConsolidateFinal(versions)
			if len(got) != 3 || got[2].Seq != 2 || got[2].Digest == got[0].Digest || got[2].Digest == got[1].Digest {
				t.Fatalf("not a new merged revision: %+v", got)
			}
			merged := got[2].Held.Week
			if merged.Line.Quantities["passing_yards"] != 0 || merged.Line.Quantities["receiving_drop"] != 20000 || merged.Line.Events["long_pass"][0] != 420000 {
				t.Fatal(merged)
			}
			if merged.FieldSources["passing_yards"] != source.NFLVerse || merged.FieldSources["receiving_drop"] != source.Sleeper {
				t.Fatal(merged.FieldSources)
			}
			if !reflect.DeepEqual(versions, before) || source.LineDigest(final) != versions[0].Digest || source.LineDigest(live) != versions[1].Digest {
				t.Fatal("rewrote held evidence")
			}
			merged.Line.Events["long_pass"][0] = 1
			if live.Line.Events["long_pass"][0] != 420000 {
				t.Fatal("aliased held events")
			}
		})
	}
}

func TestScoreSourceMFLNeverBecomesRawEvidence(t *testing.T) {
	line, owners := source.MergeStatSources(map[string]fantasyscore.StatLine{source.MFLLive: {Quantities: map[leaguerules.StatID]leaguerules.Decimal{"tackle": 99}}})
	if len(line.Quantities) != 0 || len(owners) != 0 {
		t.Fatal(line, owners)
	}
}

func TestScoreSourceNoLivePreservesHeldReplay(t *testing.T) {
	week := yardsWeek(420000, "QB")
	original := []source.LineRevision{{Seq: 0, Digest: source.LineDigest(week), Held: source.HeldWeek{Authority: source.NFLVerse, Week: &week}}}
	if got := source.ConsolidateFinal(original); !reflect.DeepEqual(got, original) {
		t.Fatal("changed held replay")
	}
}

func TestScoreSourceNewLiveValueDoesNotBecomeFinalAuthority(t *testing.T) {
	final, live := yardsWeek(0, "QB"), yardsWeek(70000, "QB")
	live.Line.Quantities["receiving_drop"] = 10000
	versions := []source.LineRevision{
		{Seq: 0, Digest: source.LineDigest(final), Held: source.HeldWeek{Authority: source.NFLVerse, Week: &final}},
		{Seq: 1, Digest: source.LineDigest(live), Held: source.HeldWeek{Authority: source.ESPNLive, Week: &live}},
	}
	versions = source.ConsolidateFinal(versions)
	if len(source.ConsolidateFinal(versions)) != 3 {
		t.Fatal("unchanged merge duplicated")
	}
	newer := yardsWeek(90000, "QB")
	newer.Line.Quantities["receiving_drop"] = 20000
	versions = append(versions, source.LineRevision{Seq: 3, Digest: source.LineDigest(newer), Held: source.HeldWeek{Authority: source.ESPNLive, Week: &newer}})
	got := source.ConsolidateFinal(versions)
	if len(got) != 5 || got[4].Digest == got[2].Digest || got[4].Held.Week.Line.Quantities["receiving_drop"] != 20000 || got[4].Held.Week.Line.Quantities["passing_yards"] != 0 {
		t.Fatal(got)
	}
	if got[2].Held.Week.Line.Quantities["receiving_drop"] != 10000 {
		t.Fatal("rewrote merged revision")
	}
}

func TestScoreSourceHeldSleeperFillIsNotFinalAuthority(t *testing.T) {
	final, live := yardsWeek(10000, "QB"), yardsWeek(20000, "QB")
	final.Line.Quantities["receiving_drop"] = 10000
	final.FieldSources = map[leaguerules.StatID]string{"receiving_drop": source.Sleeper}
	live.Line.Quantities["receiving_drop"] = 20000
	versions := source.ConsolidateFinal([]source.LineRevision{
		{Seq: 0, Digest: source.LineDigest(final), Held: source.HeldWeek{Authority: source.NFLVerse, Week: &final}},
		{Seq: 1, Digest: source.LineDigest(live), Held: source.HeldWeek{Authority: source.ESPNLive, Week: &live}},
	})
	if len(versions) != 3 || versions[2].Held.Week.Line.Quantities["receiving_drop"] != 20000 || versions[2].Held.Week.Line.Quantities["passing_yards"] != 10000 {
		t.Fatal(versions)
	}
}

func TestScoreSourceAttributionInvalidatesLiveDigestOnly(t *testing.T) {
	week := yardsWeek(10000, "QB")
	legacy := source.LineDigest(week)
	week.FieldSources = map[leaguerules.StatID]string{"passing_yards": source.ESPNLive}
	espn := source.LiveRevisionDigest(week)
	week.FieldSources["passing_yards"] = source.Sleeper
	if espn == source.LiveRevisionDigest(week) || legacy != source.LineDigest(week) {
		t.Fatal("live attribution not keyed, or legacy digest changed")
	}
}
