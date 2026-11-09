package commissioner

import (
	"encoding/json"
	"slices"
	"testing"
	"time"
)

// epochSignature is every league-state reading the server's reply cache keeps, as one viewer would get it
// at an instant (serverhttp cachedReadingOperation), plus each viewer's revision token.
func epochSignature(t *testing.T, records []Record, at time.Time) string {
	t.Helper()
	rules, rulesErr := RulesReading(rolloverLeague, records, at)
	corrections, correctionsErr := CorrectionsReading(rolloverLeague, records, at)
	assets, assetsErr := NativeAssetsReading(rolloverLeague, records, ReadingInstant(records, at))
	reserve, reserveErr := NativeReserveCheckReading(rolloverLeague, records, at, nil)
	roster, rosterErr := NativeRosterCheckReading(rolloverLeague, records, at, nil)
	tokens := map[string]int64{}
	for _, r := range records {
		tokens[r.Principal] = RecipientRevision(records, r.Principal, at)
	}
	body, err := json.Marshal([]any{rules, errText(rulesErr), corrections, errText(correctionsErr), assets, errText(assetsErr),
		reserve, errText(reserveErr), roster, errText(rosterErr), tokens})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

// epochSweep is every six hours of the fixture month and the nanoseconds around each noon, where its times fall,
// built apart from epochThresholds so an instant the code forgets is still read.
func epochSweep() []time.Time {
	var out []time.Time
	for at := seasonMoment(1).AddDate(0, 0, -2); at.Before(seasonMoment(31).AddDate(0, 0, 2)); at = at.Add(6 * time.Hour) {
		out = append(out, at)
	}
	for day := 1; day <= 31; day++ {
		out = append(out, seasonMoment(day).Add(-time.Nanosecond), seasonMoment(day).Add(time.Nanosecond))
	}
	return out
}

// TestReadingEpochHoldsEveryReading: two instants in one reading epoch read the same league state, so a
// reply cached in an epoch is the reply any instant of it would get. It also proves the sweep has teeth:
// the readings do change across epochs, and keying only on the asset fold's times would be wrong here.
func TestReadingEpochHoldsEveryReading(t *testing.T) {
	records := assetsCacheLedger(t)
	byEpoch := map[int]string{}
	firstAt := map[int]time.Time{}
	byAssetsKey := map[int]string{}
	assetsThresholds, _ := clockThresholds(records)
	assetsGap := false
	for _, at := range epochSweep() {
		epoch, next, ok := ReadingEpoch(records, at, nil)
		if !ok {
			t.Fatal("fixture times must parse")
		}
		if _, public, _ := epochThresholds(records); !next.IsZero() && (!next.After(at) || !slices.ContainsFunc(public, next.Equal)) {
			t.Fatalf("at %v: epoch %d ends at %v, not after the reading instant", at, epoch, next)
		}
		got := epochSignature(t, records, at)
		if want, seen := byEpoch[epoch]; seen && got != want {
			t.Fatalf("epoch %d reads differently at %v and %v", epoch, firstAt[epoch], at)
		} else if !seen {
			byEpoch[epoch], firstAt[epoch] = got, at
		}
		reached := 0
		for _, th := range assetsThresholds {
			if !th.After(ReadingInstant(records, at)) {
				reached++
			}
		}
		if prev, seen := byAssetsKey[reached]; seen && prev != got {
			assetsGap = true
		} else if !seen {
			byAssetsKey[reached] = got
		}
	}
	distinct := map[string]bool{}
	for _, s := range byEpoch {
		distinct[s] = true
	}
	if len(distinct) < 3 {
		t.Fatalf("the sweep is vacuous: only %d distinct readings over %d epochs", len(distinct), len(byEpoch))
	}
	if !assetsGap {
		t.Fatal("expected the asset fold's times alone to miss a reading change (season start); the sweep lost its teeth")
	}
}

// TestReadingEpochUnparsedTimeNeverShares: a time that does not parse leaves the league unshared.
func TestReadingEpochUnparsedTimeNeverShares(t *testing.T) {
	records := assetsCacheLedger(t)
	broken := append([]Record(nil), records...)
	last := broken[len(broken)-1]
	c := *last.Command.Args.Correction
	c.Until = "not a time"
	last.Command.Args.Correction = &c
	broken[len(broken)-1] = last
	if _, _, ok := ReadingEpoch(broken, seasonMoment(20), nil); ok {
		t.Fatal("an unparsed expiry must not share readings")
	}
}
