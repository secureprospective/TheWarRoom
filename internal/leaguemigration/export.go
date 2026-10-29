// Package leaguemigration lands a read-only source plan in the native league ledger.
package leaguemigration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// epoch is the instant stored values are encoded at; the ledger records the real instant.
var epoch = time.Unix(1, 0).UTC() //nolint:gochecknoglobals // a fixed instant, read only

// Source reads a fresh candidate without changing the source league.
type Source interface {
	Read(ctx context.Context, now time.Time) (Candidate, error)
}

// Candidate separates source reading and translation from native ledger writes.
// Problems block import, but owners remain available for invitation review.
type Candidate struct {
	Name     string
	Season   int
	Owners   []Owner
	Plan     Plan
	Problems []string
}

func (c Candidate) prepared() (Plan, error) {
	if len(c.Problems) > 0 {
		return Plan{}, &Problems{List: c.Problems}
	}
	return c.Plan, nil
}

// Owner is the owner MFL lists for a franchise; either may be blank.
type Owner struct{ Franchise, Name, Email string }

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

// Plan is an MFL league ready to land: the creation with its own settings, the starting snapshot, the
// owners for the invitations, and notes on what waits for the commissioner. Go builds every part of it
// from the read; no caller supplies rosters or settings.
type Plan struct {
	Create commissioner.RawArgs
	Seed   commissioner.RawNativeSnapshot
	Rules  leaguerules.RuleSet
	Owners []Owner
	Notes  []string
	// Mapping is each MFL franchise id's app franchise.
	Mapping map[string]string
}

// CarriedFields are the settings the app counts or carries from the starting snapshot itself (the franchises,
// their divisions and conferences), so the translator leaves them to it and the review reads them from
// the franchises instead.
func CarriedFields() map[leaguerules.FieldPath]bool {
	return map[leaguerules.FieldPath]bool{"league.teams": true, "league.divisions": true, "league.alignment": true,
		"league.divisionConferences": true, "league.sides": true}
}
