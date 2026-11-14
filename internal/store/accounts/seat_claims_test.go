package accounts

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"testing"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

// A seat claim makes its account under the claimed identity and is spent with it; a taken username
// leaves it live, a spent claim refuses, and forgetting the league drops its claims.
func TestSeatClaimRedeemsOnceUnderTheClaimedIdentity(t *testing.T) {
	ctx := t.Context()
	pools, err := db.Open(ctx, filepath.Join(t.TempDir(), "accounts.db"), ServerVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	store := NewServerFile(pools)
	if err := store.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	taken := fixtureCredential(t) // username "owner"
	taken.ID = "aaaaaaaa-1234-4234-8234-123456789abc"
	if err := store.Insert(ctx, taken, false); err != nil {
		t.Fatal(err)
	}
	const league = "native.moved"
	claimed := governance.Principal("acct.12345678-1234-4234-8234-123456789abc")
	issued, err := identity.IssueSeatClaims(ctx, store, league, []identity.SeatClaim{{Principal: claimed, Franchise: "franchise.01", Name: "Team One"}})
	if err != nil || len(issued) != 1 {
		t.Fatalf("issue: %v %v", issued, err)
	}
	digest := claimDigest(issued[0].Code)
	credential := fixtureCredential(t)
	if err := store.RedeemSeatClaim(ctx, league, digest, credential); !errors.Is(err, identity.ErrUsernameTaken) {
		t.Fatalf("a taken username: %v", err)
	}
	if left, err := store.SeatClaimsLeft(ctx, league); err != nil || left != 1 {
		t.Fatalf("after a taken username the claim is gone: %d %v", left, err)
	}
	credential.Username = "newowner"
	if err := store.RedeemSeatClaim(ctx, league, digest, credential); err != nil {
		t.Fatalf("redeem: %v", err)
	}
	if got, err := store.SignInName(ctx, credential.ID); err != nil || got != "newowner" {
		t.Fatalf("the claimed identity's sign-in: %q %v", got, err)
	}
	credential.Username = "again"
	if err := store.RedeemSeatClaim(ctx, league, digest, credential); !errors.Is(err, identity.ErrJoinRefused) {
		t.Fatalf("a spent claim: %v", err)
	}
	if _, err := identity.IssueSeatClaims(ctx, store, league, []identity.SeatClaim{{Principal: claimed, Franchise: "franchise.01"}}); err != nil {
		t.Fatal(err)
	}
	if err := store.ForgetLeague(ctx, league); err != nil {
		t.Fatal(err)
	}
	if left, err := store.SeatClaimsLeft(ctx, league); err != nil || left != 0 {
		t.Fatalf("a forgotten league kept %d claims (%v)", left, err)
	}
}

func claimDigest(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
