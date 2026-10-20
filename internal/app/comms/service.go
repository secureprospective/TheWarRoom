package comms

import (
	"context"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

// Source reads archive columns through the caller's lock and held snapshot.
// A nil Archive means archive history is not configured.
type Source struct {
	Archive func(context.Context) (facts.RosterWeekColumns, facts.NflGameColumns, error)
	Log     func(string)
}

func Read(ctx context.Context, src Source, in Input, scope RawCommsScope) (Board, error) {
	if src.Archive != nil {
		rosters, games, err := src.Archive(ctx)
		if err != nil {
			in.RosterNote = " Weekly contract history unavailable; dropped salaries may not be held."
			src.Log(err.Error())
		}
		in.Rosters, in.RosterTimes = rosters, RosterWeekTimes(games)
	}
	return Build(in, scope)
}

// RefreshSource keeps feed publication, authentication and archive fallback with their owner.
type RefreshSource struct {
	Transactions func(context.Context)
	Boards       func(context.Context)
	FreshFor     time.Duration
}

func Refresh(ctx context.Context, src RefreshSource, tx report.Held[leaguefeed.Transaction], boards [2]report.Held[ingestion.RawLeaguePost], force bool) {
	if force || tx.RefreshError != "" || time.Since(tx.FetchedAt) >= src.FreshFor {
		src.Transactions(ctx)
	}
	fresh := true
	for _, b := range boards {
		fresh = fresh && b.RefreshError == "" && time.Since(b.FetchedAt) < src.FreshFor
	}
	if force || !fresh {
		src.Boards(ctx)
	}
}
