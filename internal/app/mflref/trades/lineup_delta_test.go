package trades

import (
	"context"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/lineupdelta"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type fitRoster struct{ state.Reader }

func (fitRoster) Franchises() []string { return []string{"0001", "0002"} }
func (fitRoster) Roster(fid string) ([]state.PlayerState, bool) {
	return []state.PlayerState{{MFLID: "0042", FranchiseID: fid, RosterStatus: domain.RosterActive}}, true
}
func TestReferenceLineupFitReusesTheSharedRead(t *testing.T) {
	held, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	trade := history.TradeSides{Franchise: "0001", Partner: "0002", Get: []string{"0042"}}
	got, err := ReadLineupDelta(t.Context(), held.Config, fitRoster{}, 2026, 9, trade,
		func(_ context.Context, league string, season, week int, sides history.TradeSides, base commissioner.NativeBase) (history.NativeLineupDelta, error) {
			calls++
			rules, ok := lineupdelta.Rules(base.Rules.Lineup)
			if league != "" || season != 2026 || week != 9 || sides.Franchise != trade.Franchise || !ok || rules.Total != 21 || !base.Assets.Seeded || len(base.Assets.Snapshot.Players) != 2 {
				t.Fatalf("wrong shared inputs: %s %d %d %+v %+v", league, season, week, sides, base)
			}
			return history.NativeLineupDelta{Season: season, Week: week, You: lineupdelta.Side{PerWeek: 2.4}}, nil
		})
	if err != nil || calls != 1 || got.You.PerWeek != 2.4 {
		t.Fatalf("fit read: %+v %v calls=%d", got, err, calls)
	}
}

func TestReferenceLineupFitRejectsMissingDependenciesAndFranchises(t *testing.T) {
	held, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReadLineupDelta(t.Context(), held.Config, nil, 2026, 9, history.TradeSides{}, nil); err == nil {
		t.Fatal("missing dependencies accepted")
	}
	calls := 0
	read := func(context.Context, string, int, int, history.TradeSides, commissioner.NativeBase) (history.NativeLineupDelta, error) {
		calls++
		return history.NativeLineupDelta{}, nil
	}
	if _, err := ReadLineupDelta(t.Context(), held.Config, fitRoster{}, 2026, 9, history.TradeSides{Franchise: "0001", Partner: "absent"}, read); err == nil || calls != 0 {
		t.Fatal("missing franchise reached the shared read")
	}
}
