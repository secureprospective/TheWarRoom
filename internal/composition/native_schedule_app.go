package composition

import (
	"context"
	"log"
	"time"

	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner/native"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

// This is a local reference read. It has no fetch path and never reads a fantasy league's schedule.
func (a *App) nativeScheduleDates(parent context.Context, season int, at time.Time) ([]leagueweek.Week,
	error) {
	ctx, cancel := context.WithTimeout(parent, refreshTimeout)
	defer cancel()
	return nativeapp.ScheduleDates(ctx, a.core.History.ReferenceBodies, season, at) //nolint:wrapcheck // Preserve the held NFL week dates refusal.
}

type appBidEnv struct{ app *App }

func (e appBidEnv) BidEnvironment(ctx context.Context, league string,
	at time.Time) (commissioner.BidEnvironment, error) {
	//nolint:wrapcheck // Preserve the bidding refusal.
	return nativeapp.ReadBidEnvironment(ctx, nativeapp.BidSource{Records: e.app.leagueRecords,
		Schedule: e.app.nativeScheduleDates, Bracket: e.app.core.History.NativeBracket}, league, at)
}

type nativeDueClock struct{ now func() time.Time }

func (c nativeDueClock) Now() time.Time { return c.now() }
func (nativeDueClock) Trusted() error   { return nil }

func (a *App) nativeWaiverClock(ctx context.Context) {
	worker := leagueservice.NewDueWorker(a.core.History, a.nativeFacts, nativeDueClock{now: a.timeNow})
	worker.Run(ctx, func(err error) { log.Printf("the war room: process native deadlines: %v", err) })
}
