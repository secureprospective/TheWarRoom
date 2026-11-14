package accounts

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

// Seat claims bring a moved league's people across (Christopher, 2026-11-14). Sign-ins never leave a
// server, so a league that arrives from another server comes with one one-time claim per person it
// seats or names as commissioner. Redeeming a claim through the join page makes a new sign-in that
// takes over that person's identity in the league: the league's own file is not changed, so every
// seat, grant and past decision still names them.

// CommissionerSeat is the join page's pick for a person the league names as commissioner but who
// holds no franchise.
const CommissionerSeat = "commissioner"

// SeatClaim is one person a moved league names: the identity it seats, and the franchise (or
// CommissionerSeat) the claim opens on.
type SeatClaim struct {
	Principal governance.Principal
	Franchise string
	Name      string
}

// StoredSeatClaim is a claim as the sign-in file keeps it: the code's digest only.
type StoredSeatClaim struct {
	SeatClaim
	Digest string
}

// IssuedSeatClaim is a claim and its one-time code, shown once to the commissioner.
type IssuedSeatClaim struct {
	SeatClaim
	Code string
}

// SeatClaimRepository keeps a league's seat claims and redeems one with its new sign-in at once.
type SeatClaimRepository interface {
	SetSeatClaims(ctx context.Context, league string, claims []StoredSeatClaim) error
	SeatClaim(ctx context.Context, league, digest string) (StoredSeatClaim, error)
	RedeemSeatClaim(ctx context.Context, league, digest string, credential Credential) error
}

// IssueSeatClaims replaces league's seat claims with a fresh one-time code for each person.
func IssueSeatClaims(ctx context.Context, store SeatClaimRepository, league string, seats []SeatClaim) ([]IssuedSeatClaim, error) {
	stored := make([]StoredSeatClaim, 0, len(seats))
	issued := make([]IssuedSeatClaim, 0, len(seats))
	for _, seat := range seats {
		if _, err := accountID(seat.Principal); err != nil || seat.Franchise == "" {
			return nil, fmt.Errorf("league %s seats %q, which cannot be claimed on this server", league, seat.Principal)
		}
		var secret [32]byte
		if _, err := rand.Read(secret[:]); err != nil {
			return nil, fmt.Errorf("generate seat claim: %w", err)
		}
		code := base64.RawURLEncoding.EncodeToString(secret[:])
		stored = append(stored, StoredSeatClaim{SeatClaim: seat, Digest: enrollmentDigest(code)})
		issued = append(issued, IssuedSeatClaim{SeatClaim: seat, Code: code})
	}
	if err := store.SetSeatClaims(ctx, league, stored); err != nil {
		return nil, fmt.Errorf("save seat claims: %w", err)
	}
	return issued, nil
}

// accountID is the account id a principal names; only account principals can be claimed.
func accountID(p governance.Principal) (string, error) {
	id := strings.TrimPrefix(string(p), "acct.")
	canonical, err := governance.AccountPrincipal(id)
	if err != nil || canonical != p {
		return "", ErrJoinRefused
	}
	return id, nil
}

// seatClaim finds the live seat claim raw's code opens; ok is false when there is none.
func (s *JoinService) seatClaim(ctx context.Context, raw RawJoin) (SeatClaimRepository, StoredSeatClaim, bool, error) {
	claims, ok := s.store.(SeatClaimRepository)
	valid := raw.Validate() == nil // a malformed code opens no claim
	if !ok || !valid {
		return nil, StoredSeatClaim{}, false, nil
	}
	claim, err := claims.SeatClaim(ctx, raw.League, enrollmentDigest(raw.Code))
	switch {
	case err == nil:
		return claims, claim, true, nil
	case errors.Is(err, ErrJoinRefused):
		return nil, StoredSeatClaim{}, false, nil
	default:
		return nil, StoredSeatClaim{}, false, fmt.Errorf("read seat claim: %w", err)
	}
}

// claimFranchises is what a seat claim's holder sees: only the seat it opens, free to take.
func claimFranchises(claim StoredSeatClaim) []JoinFranchise {
	return []JoinFranchise{{ID: claim.Franchise, Name: claim.Name}}
}

// claimSeat makes the claimant's new sign-in under the identity the league already seats, and spends
// the claim, in one step. A taken username leaves the claim live to try again.
func (s *JoinService) claimSeat(ctx context.Context, claims SeatClaimRepository, claim StoredSeatClaim, raw RawJoinClaim) (Account, error) {
	if raw.Franchise != claim.Franchise {
		return Account{}, ErrJoinRefused
	}
	username, err := CanonicalUsername(raw.Username)
	if err != nil {
		return Account{}, ErrJoinRefused
	}
	id, err := accountID(claim.Principal)
	if err != nil {
		return Account{}, err
	}
	hash, err := s.work.Hash(ctx, raw.Password)
	if err != nil {
		return Account{}, err
	}
	account := Account{ID: id, Username: username}
	if err := claims.RedeemSeatClaim(ctx, raw.League, claim.Digest, Credential{Account: account, PasswordHash: hash}); err != nil {
		return Account{}, err //nolint:wrapcheck // ErrUsernameTaken and ErrJoinRefused are shown as they are
	}
	return account, nil
}
