package commissioner

import (
	"context"
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

type principalsKey struct{}

// WithPrincipals supplies verified identities from a trusted adapter, not request JSON.
// Account storage and verification own the lifetime and provenance of this mapping.
func WithPrincipals(ctx context.Context, principals *governance.Principals) context.Context {
	return context.WithValue(ctx, principalsKey{}, principals)
}

// HasPrincipals reports whether a trusted adapter already supplied the identity map.
func HasPrincipals(ctx context.Context) bool {
	_, supplied := ctx.Value(principalsKey{}).(*governance.Principals)
	return supplied
}

// ResolvePrincipals binds identities to a private policy view, preserving every recorded byte.
func ResolvePrincipals(ctx context.Context, league string, records []Record) ([]Record, error) {
	principals, supplied := ctx.Value(principalsKey{}).(*governance.Principals)
	if !supplied {
		return records, nil
	}
	if principals == nil || principals.League() != league {
		return nil, Refused(fmt.Errorf("verified identities belong to another league"))
	}
	for _, r := range records {
		if r.Command.League != league {
			return nil, Refused(fmt.Errorf("record belongs to another league"))
		}
	}
	out := WithoutClosedBooks(records)
	if len(out) > 0 {
		out[0].principals = principals
	}
	return out, nil
}

func PrincipalFor(records []Record, principal string) string {
	if len(records) == 0 {
		return principal
	}
	return string(records[0].principals.Resolve(governance.Principal(principal)))
}

func SamePrincipal(records []Record, left, right string) bool {
	return left != "" && right != "" && PrincipalFor(records, left) == PrincipalFor(records, right)
}

func policyCommand(raw RawCommand, records []Record) RawCommand {
	if raw.Args.People != nil {
		people := *raw.Args.People
		people.Principal = PrincipalFor(records, people.Principal)
		people.Successor = PrincipalFor(records, people.Successor)
		people.Commissioner = PrincipalFor(records, people.Commissioner)
		if people.Choice != "" && (raw.Command == "commish.vote.tiebreak" || electionChoice(records, people.ID)) {
			people.Choice = PrincipalFor(records, people.Choice)
		}
		raw.Args.People = &people
	}
	return raw
}

// Only the governance fold uses canonicalized copies. Ledger readers retain original authors.
func policyRecords(records []Record) []Record {
	if len(records) == 0 || records[0].principals == nil {
		return records
	}
	out := slices.Clone(records)
	for i := range out {
		out[i].Principal = PrincipalFor(records, out[i].Principal)
		out[i].Command = policyCommand(out[i].Command, records)
	}
	return out
}

// MatchesRetry compares the request under the stored author's identity only when that
// author is a verified alias of the current account. Neither stored canonical is changed.
func MatchesRetry(raw RawCommand, principal string, prior Record, records []Record) (bool, error) {
	if !SamePrincipal(records, principal, prior.Principal) {
		return false, nil
	}
	if raw.Command == Create && prior.Command.Command == Create && prior.Command.Args.Source == "" &&
		raw.Args.Source == LeagueSource([]Record{prior}) {
		raw.Args.Source = ""
	}
	canonical, err := raw.Canonical(prior.Principal)
	if err != nil {
		return false, fmt.Errorf("canonical retry: %w", err)
	}
	return canonical == prior.Canonical, nil
}

func electionChoice(records []Record, id string) bool {
	for _, r := range records {
		if r.Command.Command == "commish.succession.election.open" && r.Command.Args.People != nil && r.Command.Args.People.ID == id {
			return true
		}
	}
	return false
}
