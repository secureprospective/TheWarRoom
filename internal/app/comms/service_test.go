package comms_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/comms"
	"github.com/secureprospective/TheWarRoom/internal/app/comms/commstest"
	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func TestReadArchiveFailureKeepsBoardAndLogs(t *testing.T) {
	in := commstest.Real(t)
	logged := ""
	src := comms.Source{
		Archive: func(context.Context) (facts.RosterWeekColumns, facts.NflGameColumns, error) {
			return facts.RosterWeekColumns{}, facts.NflGameColumns{}, errors.New("archive unavailable")
		},
		Log: func(message string) { logged = message },
	}
	board, err := comms.Read(context.Background(), src, in, comms.RawCommsScope{})
	if err != nil {
		t.Fatal(err)
	}
	const note = " Weekly contract history unavailable; dropped salaries may not be held."
	expected, err := comms.Build(in, comms.RawCommsScope{})
	if err != nil {
		t.Fatal(err)
	}
	expected.Feed.Note += note
	expected.Transactions.Note += note
	if logged != "archive unavailable" || !reflect.DeepEqual(board, expected) {
		t.Fatal("archive failure changed the held board or lost the warning", logged)
	}
}

func TestReadArchiveColumnsReachConstruction(t *testing.T) {
	in := commstest.Real(t)
	for _, tx := range in.Transactions.Value {
		if tx.AddsDrops == nil || len(tx.AddsDrops.Out) == 0 || tx.AddsDrops.Out[0].Player == nil {
			continue
		}
		in.Transactions.Value = []leaguefeed.Transaction{tx}
		kickoff, salary, end := tx.Time.Add(-time.Hour).Unix(), 1.25, "2028"
		rosters := facts.RosterWeekColumns{Year: []int64{2026}, Week: []int64{1},
			Fid: []string{tx.Franchise}, Pid: []string{tx.AddsDrops.Out[0].Player.String()},
			Salary: []*float64{&salary}, ContractYear: []*string{&end}}
		games := facts.NflGameColumns{Year: []int64{2026}, Week: []int64{1}, Kickoff: []*int64{&kickoff}}
		src := comms.Source{Archive: func(context.Context) (facts.RosterWeekColumns, facts.NflGameColumns, error) {
			return rosters, games, nil
		}}
		got, err := comms.Read(context.Background(), src, in, comms.RawCommsScope{})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got.Events[0].Sentence, "$1,250,000, contract through 2028") {
			t.Fatal("archive columns did not reach construction", got.Events)
		}
		return
	}
	t.Fatal("fixture has no drop")
}
