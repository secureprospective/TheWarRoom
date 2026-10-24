package accounts

import (
	"context"
	"errors"
	"fmt"
)

var ErrPasswordBusy = errors.New("password service busy; try again shortly")

// PasswordWork is shared by all hosted password doors, including enrollment and TOTP.
// An active Argon2 invocation cannot be canceled; its slot stays held until it returns.
type PasswordWork struct {
	active   chan struct{}
	admitted chan struct{}
}

func NewPasswordWork() *PasswordWork {
	return &PasswordWork{active: make(chan struct{}, 2), admitted: make(chan struct{}, 6)}
}

func (p *PasswordWork) acquire(ctx context.Context) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("password admission: %w", err)
	}
	select {
	case p.admitted <- struct{}{}:
	default:
		return nil, ErrPasswordBusy
	}
	select {
	case p.active <- struct{}{}:
		if err := ctx.Err(); err != nil {
			<-p.active
			<-p.admitted
			return nil, fmt.Errorf("password admission: %w", err)
		}
		return func() { <-p.active; <-p.admitted }, nil
	case <-ctx.Done():
		<-p.admitted
		return nil, fmt.Errorf("password queue: %w", ctx.Err())
	}
}

func (p *PasswordWork) Hash(ctx context.Context, password string) (string, error) {
	release, err := p.acquire(ctx)
	if err != nil {
		return "", err
	}
	defer release()
	return HashPassword(password)
}

func (p *PasswordWork) Verify(ctx context.Context, encoded, password string) (bool, error) {
	release, err := p.acquire(ctx)
	if err != nil {
		return false, err
	}
	defer release()
	return VerifyPassword(encoded, password)
}
