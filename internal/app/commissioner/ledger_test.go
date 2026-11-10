package commissioner

import (
	"context"
	"errors"
	"testing"

	commish "github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type fakeLedger struct {
	events []history.LeagueEvent
	err    error
	asked  []string
}

func (f *fakeLedger) Events(_ context.Context, league, stream string, after int64) ([]history.LeagueEvent, error) {
	if after != 0 {
		return nil, errors.New("replay starts after revision 0")
	}
	f.asked = append(f.asked, league+" "+stream)
	return f.events, f.err
}

func TestRecordsWordsEachFailureForItsReading(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		ledger *fakeLedger
		want   string
	}{
		{"store", &fakeLedger{err: errors.New("disk")}, "read league schedule ledger: disk"},
		{"no record", &fakeLedger{events: []history.LeagueEvent{{}}}, "the league schedule has an unreadable change"},
		{"bad record", &fakeLedger{events: []history.LeagueEvent{{Payload: history.EventPayload{
			Commissioner: &commish.Record{}}}}}, "read league schedule change: "},
	} {
		_, err := Records(t.Context(), tc.ledger, "native.x", "schedule")
		if err == nil || len(err.Error()) < len(tc.want) || err.Error()[:len(tc.want)] != tc.want {
			t.Fatalf("%s: %v, want %q", tc.name, err, tc.want)
		}
	}
	ledger := &fakeLedger{}
	records, err := Records(t.Context(), ledger, "native.x", "calendar")
	if err != nil || len(records) != 0 || records == nil || ledger.asked[0] != "native.x "+commish.Stream {
		t.Fatalf("an empty ledger replays to no records: %v %v %v", records, err, ledger.asked)
	}
}
