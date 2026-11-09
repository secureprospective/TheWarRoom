package commissioner

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func scoreRecord(n int, command string, change RawScoreChange, at time.Time) Record {
	return Record{Command: RawCommand{League: "native.test", Command: command, Reason: fmt.Sprintf("reason %d", n),
		Args: RawArgs{Score: &change}}, Receipt: Receipt{Entry: fmt.Sprintf("e%03d", n), RecordedAt: at}}
}

// The index must read exactly as the whole-ledger scan for every player, franchise, season and week,
// active or reversed, and the held latest instant must give ReadingInstant at every clock.
func TestScoreIndexMatchesLedgerScan(t *testing.T) {
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	delta := func(v leaguerules.Decimal) *leaguerules.Decimal { return &v }
	var records []Record
	active := map[string]bool{}
	for i := range 60 {
		player, franchise := fmt.Sprintf("%04d", 99+i%3), ""
		change := RawScoreChange{Player: player, Season: 2026, Week: 1 + i%4, Delta: delta(leaguerules.Decimal(100 * i))}
		command := ScorePlayer
		switch i % 5 {
		case 1:
			command, change = PlayerPosition, RawScoreChange{Player: player, Position: leaguerules.DT}
		case 2:
			command, change = PlayerPositionClear, RawScoreChange{Player: player}
		case 3:
			franchise = fmt.Sprintf("franchise.0%d", 1+i%2)
			command, change = ScoreFranchise, RawScoreChange{Franchise: franchise, Season: 2026, Week: 1 + i%4, Delta: delta(7)}
		}
		records = append(records, scoreRecord(i, command, change, base.Add(time.Duration(i%7)*time.Hour)))
		records = append(records, Record{Command: RawCommand{Command: StatApprove}, Receipt: Receipt{Entry: fmt.Sprintf("s%03d", i)}})
		active[fmt.Sprintf("e%03d", i)] = i%4 != 0
	}
	ix := newScoreIndex(records)
	owners := []scoreOwner{{"0099", ""}, {"0100", ""}, {"0101", ""}, {"0102", ""}, {"", "franchise.01"}, {"", "franchise.02"}}
	for _, o := range owners {
		for week := 1; week <= 5; week++ {
			want, wantErr := scoreChangesFrom(records, active, o.player, o.franchise, 2026, week)
			got, err := ix.changes(active, o.player, o.franchise, 2026, week)
			if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(err, wantErr) {
				t.Fatalf("%+v week %d: index %+v %v, scan %+v %v", o, week, got, err, want, wantErr)
			}
		}
	}
	for _, now := range []time.Time{{}, base, base.Add(3 * time.Hour), base.Add(48 * time.Hour)} {
		if got, want := instantAfter(latestRecorded(records), now), ReadingInstant(records, now); !got.Equal(want) || got.Location() != want.Location() {
			t.Fatalf("now %v: instant %v, want %v", now, got, want)
		}
	}
	if got := instantAfter(latestRecorded(nil), time.Time{}); got != ReadingInstant(nil, time.Time{}) {
		t.Fatalf("empty ledger instant %v", got)
	}
	broken := slices.Concat(records[:4], []Record{{Command: RawCommand{Command: ScorePlayer}, Receipt: Receipt{Entry: "bad"}}})
	_, wantErr := scoreChangesFrom(broken, active, "0099", "", 2026, 1)
	if _, err := newScoreIndex(broken).changes(active, "0100", "", 2026, 1); wantErr == nil || err == nil || err.Error() != wantErr.Error() {
		t.Fatalf("undecodable change: index %v, scan %v", err, wantErr)
	}
}
