package control

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/pendingtrades"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
)

type CredentialStore interface {
	Get(context.Context) (mflkey.Key, error)
	Set(context.Context, mflkey.Key) error
	Delete(context.Context) error
}

// KeyWorkflow keeps verification and key-generation publication under the caller's key lock.
// Startup-owned stores and context are acquired only after Ready succeeds.
type KeyWorkflow struct {
	Ready        func() error
	Lock, Unlock func()
	Context      func() context.Context
	Store        func() CredentialStore
	Status       func(string, string) MFLKeyStatus
	Verify       func(context.Context, mflkey.Key) (string, error)
	Lookup       func(context.Context)
	Verified     func(string)
	Clear        func()
	Refresh      func()
	Now          func() time.Time
}

func (s KeyWorkflow) Read() (MFLKeyStatus, error) {
	if err := s.Ready(); err != nil {
		return MFLKeyStatus{}, err
	}
	s.Lock()
	defer s.Unlock()
	_, err := s.Store().Get(s.Context())
	if errors.Is(err, mflkey.ErrNoKey) {
		return s.Status("absent", ""), nil
	}
	if err != nil {
		return s.Status("unavailable", err.Error()), nil
	}
	return s.Status("connected", ""), nil
}

func (s KeyWorkflow) Set(key string) (MFLKeyStatus, error) {
	candidate := mflkey.Key(strings.TrimSpace(key))
	if err := s.Ready(); err != nil {
		return MFLKeyStatus{}, err
	}
	s.Lock()
	defer s.Unlock()
	// Bounds check plausibility, not MFL's key encoding.
	if len(candidate) < 16 || len(candidate) > 256 {
		return s.Status("rejected", "key must contain 16 to 256 bytes after trimming whitespace"), nil
	}
	ctx, cancel := context.WithTimeout(s.Context(), time.Minute)
	defer cancel()
	detail, err := s.Verify(ctx, candidate)
	if errors.Is(err, pendingtrades.ErrRejected) {
		return s.Status("rejected", err.Error()), nil
	}
	if err != nil {
		return s.Status("unreachable", report.FailureText(err)+" Nothing was stored."), nil
	}
	if err := s.Store().Set(ctx, candidate); err != nil {
		return s.Status("unavailable", err.Error()), nil
	}
	s.Lookup(ctx)
	s.Verified(s.Now().UTC().Format(time.RFC3339))
	s.Refresh()
	return s.Status("connected", detail), nil
}

func (s KeyWorkflow) Delete() (MFLKeyStatus, error) {
	if err := s.Ready(); err != nil {
		return MFLKeyStatus{}, err
	}
	s.Lock()
	defer s.Unlock()
	if err := s.Store().Delete(s.Context()); err != nil {
		return s.Status("unavailable", err.Error()), nil
	}
	s.Clear()
	s.Refresh()
	return s.Status("absent", ""), nil
}

// ResolveFranchise fails closed: held identity is never reused after a failed lookup.
func ResolveFranchise(ctx context.Context, store CredentialStore, lookup func(context.Context) (string, error)) string {
	if _, err := store.Get(ctx); err != nil {
		return ""
	}
	franchise, err := lookup(ctx)
	if err != nil {
		return ""
	}
	return franchise
}
