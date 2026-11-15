package accounts_test

import (
	"context"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

type joinRepo struct{ digest string }

func (r *joinRepo) SetJoinLink(_ context.Context, _, digest string, _ governance.Principal) error {
	r.digest = digest
	return nil
}
func (r *joinRepo) RevokeJoinLink(context.Context, string) error { return nil }
func (r *joinRepo) JoinLink(context.Context, string) (string, governance.Principal, error) {
	return r.digest, "acct.11111111-1111-4111-8111-111111111111", nil
}
func (r *joinRepo) Insert(context.Context, accounts.Credential, bool) error { return nil }
func (r *joinRepo) SignInName(_ context.Context, id string) (string, error) {
	if id == "22222222-2222-4222-8222-222222222222" {
		return "present", nil
	}
	return "", accounts.ErrNotFound
}

type joinSeats struct{}

func (joinSeats) Franchises(context.Context, string) ([]accounts.JoinFranchise, error) {
	return []accounts.JoinFranchise{
		{ID: "franchise.01", Name: "Gone", Taken: true, Seat: "acct.33333333-3333-4333-8333-333333333333"},
		{ID: "franchise.02", Name: "Here", Taken: true, Seat: "acct.22222222-2222-4222-8222-222222222222"},
		{ID: "franchise.03", Name: "Open"},
	}, nil
}
func (joinSeats) IsCommissioner(context.Context, string, governance.Principal) (bool, error) {
	return true, nil
}
func (joinSeats) Seat(context.Context, string, string, governance.Principal, governance.Principal) error {
	return nil
}
func (joinSeats) Release(context.Context, string, string, governance.Principal) error { return nil }

// A league moved or restored onto a server can keep a seat whose account is not on it. The join
// list must still answer: that franchise stays taken with no name, and the rest list as usual.
func TestJoinFranchisesListsASeatWhoseAccountIsGone(t *testing.T) {
	service := accounts.NewJoin(&joinRepo{}, joinSeats{}, nil)
	code, err := service.Rotate(t.Context(), "native.test", "acct.11111111-1111-4111-8111-111111111111")
	if err != nil {
		t.Fatal(err)
	}
	list, err := service.Franchises(t.Context(), accounts.RawJoin{League: "native.test", Code: code.Code})
	if err != nil {
		t.Fatalf("join list with a missing owner account: %v", err)
	}
	if len(list) != 3 || !list[0].Taken || list[0].ClaimedBy != "" || list[1].ClaimedBy != "present" || list[2].Taken {
		t.Fatalf("list = %+v", list)
	}
}
