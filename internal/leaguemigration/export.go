// Package leaguemigration moves a league from MFL into the app: it reads the MFL league (read only), lands
// it whole as a league that follows MFL, compares the two before the switch, and prepares the owner
// invitations. MFL is never written to; a move given up is started again from a fresh read.
package leaguemigration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
)

// epoch is the instant stored values are encoded at; the ledger records the real instant.
var epoch = time.Unix(1, 0).UTC() //nolint:gochecknoglobals // a fixed instant, read only

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
	Owners      []Owner
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

// Owner is the owner MFL lists for a franchise; either may be blank.
type Owner struct{ Franchise, Name, Email string }

// Champion is the franchise MFL records as a past season's champion.
type Champion struct {
	Season    int
	Franchise string
}

// Source reads one MFL league. Every method only reads; none may change anything on MFL.
type Source interface {
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
	Owners    []Owner
	Champions []Champion
}

// ReadError says which part of the MFL league could not be read. Nothing is imported when a read fails.
type ReadError struct {
	Part string
	Err  error
}

func (e *ReadError) Error() string {
	return fmt.Sprintf("MFL could not be read while getting the %s (%v). Nothing was imported; "+
		"start the import again when MFL answers", e.Part, e.Err)
}

func (e *ReadError) Unwrap() error { return e.Err }

// Read reads the whole league from MFL in one pass. A failure at any part ends the read with a ReadError,
// and the caller holds nothing partial.
func Read(ctx context.Context, src Source, now time.Time) (Export, error) {
	var out Export
	read, err := src.League(ctx)
	if err != nil {
		return Export{}, &ReadError{"league settings and franchises", err}
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
			return Export{}, &ReadError{step.part, err}
		}
		if err := ctx.Err(); err != nil {
			return Export{}, &ReadError{step.part, err}
		}
	}
	out.ReadAt = now.UTC()
	return out, nil
}

// Problems is everything that stops an import, in plain words. A league with problems is not imported.
type Problems struct{ List []string }

func (p *Problems) Error() string {
	if len(p.List) == 1 {
		return "The MFL league cannot be imported yet: " + p.List[0]
	}
	return fmt.Sprintf("The MFL league cannot be imported yet; %d things need fixing on MFL first", len(p.List))
}

// AsProblems returns the problem list an error carries, if it carries one.
func AsProblems(err error) ([]string, bool) {
	var p *Problems
	if errors.As(err, &p) {
		return p.List, true
	}
	return nil, false
}

// wrapped keeps the reader's own error under the ReadError that names the part.
func wrapped(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w", err)
}
