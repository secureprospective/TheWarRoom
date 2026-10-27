package trades

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestRosterFigureShowsBeforeAfterAndChange(t *testing.T) {
	t.Parallel()
	if f := RosterFigure("Active roster · 35–48", 37, 36); f.Before != "37" || f.After != "36" || f.Change != "-1" {
		t.Fatalf("one out: %+v", f)
	}
	if f := RosterFigure("Taxi squad · maximum 8", 6, 6); f.Change != "" {
		t.Fatalf("no change shows none: %+v", f)
	}
}

func TestPlayerAssetValuesAndExactCents(t *testing.T) {
	t.Parallel()
	value := PlayerAsset(state.MirrorPlayer{MFLID: "13593", Salary: 123456789, ContractYear: 2028},
		snapshot.Player{Position: domain.PosWR, Team: "DAL"}, "Name", Value{ID: "13593",
			Now: Cell{Text: "12.00 PPG", Held: true}, Dynasty: Cell{Text: "0.3 wins", Held: true}}, 2026)
	if value.Contract != "$1,234,567.89 · 3 years" || value.Now.Text != "12.00 PPG" || value.Dynasty.Text != "0.3 wins" {
		t.Fatal(value)
	}
}

func TestTradeBuilderBoundaryRejectsInvalidInputs(t *testing.T) {
	t.Parallel()
	for _, r := range []Input{{Mine: "x", Action: "read"}, {Mine: "0001", Action: "submit"},
		{Mine: "0001", Action: "revoke"}, {Mine: "0001", Other: "0001", Action: "save"},
		{Mine: "0001", Other: "0002", Action: "save", Give: []string{"99", "0099"}, Get: []string{"12345"}},
		{Mine: "0001", Other: "0002", Action: "save", Give: []string{"BB_2"}, Get: []string{"12345"}}} {
		if err := r.Validate(); err == nil {
			t.Fatalf("accepted %+v", r)
		}
	}
}

func TestPickOwnershipReasonReadsOnlyPickTrades(t *testing.T) {
	t.Parallel()
	at := time.Unix(1800000000, 0)
	for _, tc := range []struct {
		name          string
		give, get     []string
		at            time.Time
		failure, want string
		reads         int
	}{
		{name: "players only", give: []string{"0042"}, get: []string{"0099"}},
		{name: "offering pick", give: []string{"FP_0001_2027_1"}, want: "Draft-pick ownership unavailable; refresh before handing off.", reads: 1},
		{name: "receiving pick", get: []string{"FP_0002_2027_1"}, at: at, failure: "offline", want: "Draft-pick ownership unavailable: offline", reads: 1},
		{name: "dated pick feed", get: []string{"FP_0002_2027_1"}, at: at, reads: 1},
		{name: "missing date wins", give: []string{"FP_0001_2027_1"}, failure: "offline", want: "Draft-pick ownership unavailable; refresh before handing off.", reads: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			reads := 0
			give, get := slices.Clone(tc.give), slices.Clone(tc.get)
			got := PickOwnershipReason(give, get, func() (time.Time, string) { reads++; return tc.at, tc.failure })
			if got != tc.want || reads != tc.reads || !reflect.DeepEqual(give, tc.give) || !reflect.DeepEqual(get, tc.get) {
				t.Fatalf("%q reads %d", got, reads)
			}
		})
	}
}

func TestClockReasonsPreservesFailureAndObservationOrder(t *testing.T) {
	t.Parallel()
	now := time.Unix(1800000000, 0)
	next := now.Add(time.Hour)
	calls := []string{}
	got := ClockReasons(func() (snapshot.Sourced[leagueclock.Reading], error) {
		calls = append(calls, "clock")
		return snapshot.Sourced[leagueclock.Reading]{}, errors.New("clock unavailable")
	}, func() time.Time { t.Fatal("failed clock must not sample now"); return now }, time.UTC)
	if !reflect.DeepEqual(got, []string{"Trade deadline cannot be checked: clock unavailable"}) || !reflect.DeepEqual(calls, []string{"clock"}) {
		t.Fatal(got, calls)
	}
	got = ClockReasons(func() (snapshot.Sourced[leagueclock.Reading], error) {
		return snapshot.Sourced[leagueclock.Reading]{Value: leagueclock.Reading{Windows: []leagueclock.Window{{Kind: leagueclock.TradeDeadline, Status: leagueclock.WindowDated, At: &next}}}}, nil
	}, func() time.Time { calls = append(calls, "now"); return now }, time.UTC)
	if len(got) != 0 || got == nil || !reflect.DeepEqual(calls, []string{"clock", "now"}) {
		t.Fatal(got, calls)
	}
}

func TestTradeStateStopsBeforePendingOnSelectionFailure(t *testing.T) {
	t.Parallel()
	failure := errors.New("selection failed")
	snap, held, err := ReadState("0001", func(id string) (snapshot.Snapshot, error) {
		if id != "0001" {
			t.Fatal(id)
		}
		return snapshot.Snapshot{}, failure
	}, func() snapshot.Sourced[[]leaguefeed.PendingTrade] {
		t.Fatal("failed selection read pending trades")
		return snapshot.Sourced[[]leaguefeed.PendingTrade]{}
	})
	if !errors.Is(err, failure) || len(snap.Players.Value) != 0 || held.Value != nil {
		t.Fatalf("failure returned partial state: %v", err)
	}
}

func TestBuilderTeamsDiscardsValuesOnBoardFailure(t *testing.T) {
	t.Parallel()
	snap := snapshot.Snapshot{}
	snap.Franchises.Value = []snapshot.Franchise{{ID: "0001", Name: "One"}}
	mirror := state.MirrorSnapshot{Season: 2026, Players: []state.MirrorPlayer{{MFLID: "0099", FranchiseID: "0001"}}}
	board := []Value{{ID: "0099", Now: Cell{Text: "12 PPG"}, Dynasty: Cell{Text: "1 win"}}}
	teams := BuilderTeams(snap, mirror, nil, board, errors.New("board failed"), nil)
	if len(teams) != 1 || len(teams[0].Assets) != 1 || teams[0].Assets[0].Now.Text != "No score" {
		t.Fatalf("failed board valued players: %+v", teams)
	}
	teams = BuilderTeams(snap, mirror, nil, board, nil, nil)
	if teams[0].Assets[0].Now.Text != "12 PPG" {
		t.Fatal("successful board lost value")
	}
}
