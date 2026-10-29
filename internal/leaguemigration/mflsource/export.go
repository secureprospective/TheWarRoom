package mflsource

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
)

// Export is an MFL league as one read of it found it, every value as MFL states it.
type Export struct {
	LeagueID    string
	Name        string
	Season      int
	Config      league.RawConfig
	Rosters     []Rostered
	Adjustments []Adjustment
	Draft       []draftreports.Pick
	Future      []draftreports.FuturePick
	Owners      []leaguemigration.Owner
	// Players names each player MFL's player list holds, by MFL id.
	Players   map[string]string
	Champions []Champion
	ReadAt    time.Time
}

// Rostered is one rostered player's contract as MFL states it; Salary is MFL's raw millions ("1.30").
type Rostered struct {
	Franchise, Player, Status, Salary, ContractYear, ContractStatus, ContractInfo string
}

// Adjustment is one MFL salary adjustment; Amount is MFL's raw millions, negative for a credit.
type Adjustment struct{ ID, Franchise, Amount, Description string }

// Champion is the franchise MFL records as a past season's champion.
type Champion struct {
	Season    int
	Franchise string
}

// Reader reads one MFL league. Every method only reads; none may change anything on MFL.
type Reader interface {
	League(ctx context.Context) (LeagueRead, error)
	Rosters(ctx context.Context) ([]Rostered, error)
	Adjustments(ctx context.Context) ([]Adjustment, error)
	Draft(ctx context.Context) ([]draftreports.Pick, []draftreports.FuturePick, error)
	Players(ctx context.Context) (map[string]string, error)
}

// LeagueRead is MFL's league export: its settings, franchises, owners and past champions.
type LeagueRead struct {
	ID, Name  string
	Config    league.RawConfig
	Owners    []leaguemigration.Owner
	Champions []Champion
}

// Read reads the whole league from MFL in one pass. A failure at any part ends the read with a leaguemigration.ReadError,
// and the caller holds nothing partial.
func Read(ctx context.Context, src Reader, now time.Time) (Export, error) {
	var out Export
	read, err := src.League(ctx)
	if err != nil {
		return Export{}, &leaguemigration.ReadError{Part: "league settings and franchises", Err: err}
	}
	out.LeagueID, out.Name, out.Config, out.Owners, out.Champions = read.ID, read.Name, read.Config,
		read.Owners, read.Champions
	out.Season = read.Config.CurrentSeason
	steps := []struct {
		part string
		run  func() error
	}{
		{"rosters and contracts", func() (err error) { out.Rosters, err = src.Rosters(ctx); return wrapped(err) }},
		{"salary adjustments", func() (err error) { out.Adjustments, err = src.Adjustments(ctx); return wrapped(err) }},
		{"draft picks", func() (err error) { out.Draft, out.Future, err = src.Draft(ctx); return wrapped(err) }},
		{"player list", func() (err error) { out.Players, err = src.Players(ctx); return wrapped(err) }},
	}
	for _, step := range steps {
		if err := step.run(); err != nil {
			return Export{}, &leaguemigration.ReadError{Part: step.part, Err: err}
		}
		if err := ctx.Err(); err != nil {
			return Export{}, &leaguemigration.ReadError{Part: step.part, Err: err}
		}
	}
	out.ReadAt = now.UTC()
	return out, nil
}

// wrapped keeps the reader's own error under the leaguemigration.ReadError that names the part.
func wrapped(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w", err)
}
