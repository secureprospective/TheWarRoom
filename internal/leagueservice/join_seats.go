package leagueservice

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// JoinSeats is the league ledger as pick-your-team sign-up uses it. Claim records a link's digest
// in the unique request key of the existing commish.seat.replace event, consuming it atomically with
// the seat. Seat and Release retain the normal commissioner event, policy and revision checks.
type JoinSeats struct {
	Store *history.Store
	// Leagues, when set, finds each league's own store (the hosted server, one file per league).
	Leagues history.LeagueResolver
}

func (j JoinSeats) store(ctx context.Context, league string) (*history.Store, error) {
	if j.Leagues == nil {
		return j.Store, nil
	}
	store, err := j.Leagues.League(ctx, league)
	if err != nil {
		return nil, accounts.ErrJoinRefused
	}
	return store, nil
}

func (j JoinSeats) policy(ctx context.Context, league string) ([]commissioner.Record, governance.State, error) {
	store, err := j.store(ctx, league)
	if err != nil {
		return nil, governance.State{}, err
	}
	records, err := Records(ctx, store, league, "join")
	if err != nil {
		return nil, governance.State{}, err
	}
	if len(records) == 0 {
		return nil, governance.State{}, accounts.ErrJoinRefused
	}
	state, err := commissioner.GovernanceState(records)
	if err != nil {
		return nil, governance.State{}, fmt.Errorf("league policy: %w", err)
	}
	return records, state, nil
}

// Franchises lists the league's franchises in order with their names and whether each has an owner.
func (j JoinSeats) Franchises(ctx context.Context, league string) ([]accounts.JoinFranchise, error) {
	records, state, err := j.policy(ctx, league)
	if err != nil {
		return nil, err
	}
	named := map[string]commissioner.NativeFranchise{}
	if assets, err := commissioner.NativeAssetsReading(league, records); err == nil && assets.Snapshot != nil {
		for _, f := range assets.Snapshot.Franchises {
			named[f.ID] = f
		}
	}
	ids := commissioner.LeagueFranchises(records)
	out := make([]accounts.JoinFranchise, 0, len(ids))
	for i, id := range ids {
		f, _ := state.Franchise(governance.FranchiseID(id))
		name, abbreviation := joinName(i, f.Name, f.Abbreviation, named[id])
		out = append(out, accounts.JoinFranchise{ID: id, Name: name, Abbreviation: abbreviation, Taken: f.Seat != "", Seat: string(f.Seat)})
	}
	return out, nil
}

// Seated counts the league's franchises that have an owner now (the invalidation stream budget scales with it).
func (j JoinSeats) Seated(ctx context.Context, league string) (int, error) {
	records, state, err := j.policy(ctx, league)
	if err != nil {
		return 0, err
	}
	seated := 0
	for _, id := range commissioner.LeagueFranchises(records) {
		if f, _ := state.Franchise(governance.FranchiseID(id)); f.Seat != "" {
			seated++
		}
	}
	return seated, nil
}

// IsCommissioner reports whether p holds the league's commissioner role now.
func (j JoinSeats) IsCommissioner(ctx context.Context, league string, p governance.Principal) (bool, error) {
	_, state, err := j.policy(ctx, league)
	if err != nil {
		return false, err
	}
	return state.HasRole(p, governance.Commissioner), nil
}

// Commissioner returns the league's one commissioner, for the server console's no-login seat door.
func (j JoinSeats) Commissioner(ctx context.Context, league string) (governance.Principal, error) {
	_, state, err := j.policy(ctx, league)
	if err != nil {
		return "", err
	}
	members := state.Members(governance.Commissioner)
	if len(members) != 1 {
		return "", fmt.Errorf("league has %d commissioners; the no-login seat needs exactly one", len(members))
	}
	return members[0], nil
}

// LinkClaimed reports whether a committed seat event has consumed this join-link digest.
func (j JoinSeats) LinkClaimed(ctx context.Context, league, digest string) (bool, error) {
	records, _, err := j.policy(ctx, league)
	if err != nil {
		return false, err
	}
	return joinLinkClaimed(records, digest), nil
}

func joinLinkClaimed(records []commissioner.Record, digest string) bool {
	request := "join.claim." + digest
	for _, record := range records {
		if record.Command.Request == request {
			return true
		}
	}
	return false
}

func (j JoinSeats) record(ctx context.Context, league, command, request, reason string, people commissioner.RawPeople, actor governance.Principal, expected int64) error {
	raw := commissioner.RawCommand{League: league, Command: command, Args: commissioner.RawArgs{People: &people}, Expected: expected,
		Request: request, Reason: reason, Effective: "now", Visibility: "league"}
	store, err := j.store(ctx, league)
	if err != nil {
		return err
	}
	if _, err := store.ExecuteCommissioner(ctx, raw, string(actor), nil); err != nil {
		return ChangeError(err)
	}
	return nil
}

// Seat records owner as franchise's owner, as the issuing commissioner. Vacancy is checked at the same
// ledger revision the change is appended at, so a seat committed in between (by any process) makes
// the append conflict instead of replacing that owner.
func (j JoinSeats) Seat(ctx context.Context, league, franchise string, owner, issuer governance.Principal) error {
	records, state, err := j.policy(ctx, league)
	if err != nil {
		return err
	}
	if f, ok := state.Franchise(governance.FranchiseID(franchise)); !ok || f.Seat != "" {
		return accounts.ErrFranchiseTaken
	}
	return j.record(ctx, league, "commish.seat.replace", "join.claim."+franchise+"."+string(owner),
		"Owner claimed this franchise through the league join link", commissioner.RawPeople{Franchise: franchise, Principal: string(owner)}, issuer,
		records[len(records)-1].Receipt.Revision)
}

// Claim spends the join link with the seat event. Its unique request key uses only the stored link digest,
// so the ledger transaction both seats the owner and permanently marks this code as used.
func (j JoinSeats) Claim(ctx context.Context, league, franchise string, owner, issuer governance.Principal, digest string) error {
	records, state, err := j.policy(ctx, league)
	if err != nil {
		return err
	}
	if joinLinkClaimed(records, digest) {
		return accounts.ErrJoinRefused
	}
	if f, ok := state.Franchise(governance.FranchiseID(franchise)); !ok || f.Seat != "" {
		return accounts.ErrFranchiseTaken
	}
	request := "join.claim." + digest
	err = j.record(ctx, league, "commish.seat.replace", request,
		"Owner claimed this franchise through the league join link", commissioner.RawPeople{Franchise: franchise, Principal: string(owner)}, issuer,
		records[len(records)-1].Receipt.Revision)
	if err != nil {
		used, checkErr := j.LinkClaimed(ctx, league, digest)
		if checkErr == nil && used {
			return accounts.ErrJoinRefused
		}
	}
	return err
}

// Release removes franchise's owner so the franchise can be claimed again.
func (j JoinSeats) Release(ctx context.Context, league, franchise string, by governance.Principal) error {
	records, _, err := j.policy(ctx, league)
	if err != nil {
		return err
	}
	head := records[len(records)-1].Receipt.Revision
	return j.record(ctx, league, "commish.seat.remove", fmt.Sprintf("join.release.%s.%d", franchise, head),
		"Commissioner released this franchise for a new owner to claim", commissioner.RawPeople{Franchise: franchise}, by, head)
}

// joinName prefers a name the league has given the franchise: a commissioner rename, else the imported
// team name, else the name it was created with.
func joinName(position int, name, abbreviation string, imported commissioner.NativeFranchise) (string, string) {
	if (name == "" || commissioner.IsDefaultFranchiseName(position, name)) && imported.Name != "" {
		return imported.Name, imported.Abbreviation
	}
	return name, abbreviation
}
