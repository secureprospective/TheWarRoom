package mflseed

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestMirrorAssetsAnswerInTheNativeShapeWithHeldFranchiseIDs(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	source := SeedSource{
		Available: func() bool { return true },
		Roster: func() (state.MirrorSnapshot, string) {
			return state.MirrorSnapshot{Season: 2026, Players: []state.MirrorPlayer{{MFLID: "0099", FranchiseID: "0002",
				RosterStatus: domain.RosterActive, Salary: 5, ContractYear: 2028}}}, at.Format(time.RFC3339Nano)
		},
		Season: func() int { return 2026 },
		Draft: func() (season.Feed[[]draftreports.Pick], season.Feed[[]draftreports.FuturePick]) {
			return season.Feed[[]draftreports.Pick]{FetchedAt: at}, season.Feed[[]draftreports.FuturePick]{FetchedAt: at,
				Value: []draftreports.FuturePick{{Year: 2027, Round: 2, Original: "0001", Owner: "0002"}}}
		},
		Config: func() league.RawConfig {
			return league.RawConfig{Franchises: []league.Franchise{{ID: "0001", Name: "One"}, {ID: "0002", Name: "Two"}}}
		},
		Directory: func() []snapshot.Player {
			return []snapshot.Player{{ID: mustID(t, "0099"), Name: "Player, Named", Position: domain.Position("WR")}}
		},
	}
	got, err := MirrorAssets(source, "14432")
	if err != nil {
		t.Fatal(err)
	}
	s := got.Snapshot
	if got.League != "14432" || got.Revision != at.Unix() || !got.Seeded || s == nil ||
		s.Franchises[0].ID != "0001" || s.Franchises[1].ID != "0002" || s.Players[0].Franchise != "0002" ||
		s.Picks[0].Original != "0001" || s.Picks[0].Owner != "0002" {
		t.Fatalf("mirror assets keep MFL franchise IDs and the roster time as revision: %+v %+v", got, s)
	}
	if len(got.Identities) != 1 || got.Identities[0].Name != "Player, Named" || got.Identities[0].Position != "WR" ||
		got.Identities[0].Franchise != "0002" {
		t.Fatalf("mirror identities name each rostered player: %+v", got.Identities)
	}
}
