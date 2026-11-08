package accounts

import (
	"context"
	"errors"
)

// ErrPasswordChange is a malformed change request, not a wrong current password.
var ErrPasswordChange = errors.New("choose a sign-in name and a new password that meet the account rules")

// RawPasswordChange is an owner changing their own password with the current one.
type RawPasswordChange struct {
	Username    string `json:"username"`
	OldPassword string `json:"-"`
	NewPassword string `json:"-"`
}

func (r RawPasswordChange) String() string   { return "[password change redacted]" }
func (r RawPasswordChange) GoString() string { return r.String() }

func (r RawPasswordChange) Validate() error {
	if _, err := CanonicalUsername(r.Username); err != nil {
		return ErrPasswordChange
	}
	if err := validatePassword(r.OldPassword); err != nil {
		return ErrPasswordChange
	}
	if err := validatePassword(r.NewPassword); err != nil {
		return ErrPasswordChange
	}
	return nil
}

// ChangePassword verifies the current password, then stores the new hash. The store moves the
// account's credential generation, so every device session issued before this change stops matching.
func (s *Service) ChangePassword(ctx context.Context, raw RawPasswordChange) error {
	if err := raw.Validate(); err != nil {
		return err
	}
	credential, err := s.authenticateCredential(ctx, RawCredentials{Username: raw.Username, Password: raw.OldPassword})
	if err != nil {
		return err
	}
	hash, err := s.work.Hash(ctx, raw.NewPassword)
	if err != nil {
		return err
	}
	return s.store.ChangePassword(ctx, credential.ID, hash) //nolint:wrapcheck // the store words its own failure
}
