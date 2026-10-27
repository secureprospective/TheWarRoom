package commissioner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func LocalPrincipal(installation string, uid int) (string, error) {
	if installation == "" {
		return "", fmt.Errorf("local identity is unavailable")
	}
	identity := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d", installation, uid)))
	return "local." + hex.EncodeToString(identity[:]), nil
}

func ResolvePrincipal(session commissioner.Session, fallback func() (string, error)) (string, error) {
	read := fallback
	if session != nil {
		read = session.Principal
	}
	principal, err := read()
	if err != nil {
		return "", fmt.Errorf("read local identity: %w", err)
	}
	if strings.TrimSpace(principal) == "" {
		return "", fmt.Errorf("local identity is unavailable")
	}
	return principal, nil
}

type LocalSeat struct {
	Ready     func() error
	Principal func() (string, error)
	Context   func() context.Context
	Status    func(context.Context, string) (accounts.SeatStatus, error)
	SignIn    func(context.Context, accounts.RawCredentials, string) (accounts.Account, error)
}

func (s LocalSeat) Read() (accounts.SeatStatus, error) {
	if err := s.Ready(); err != nil {
		return accounts.SeatStatus{}, err
	}
	local, err := s.Principal()
	if err != nil {
		return accounts.SeatStatus{}, err
	}
	return s.Status(s.Context(), local)
}

func (s LocalSeat) Login(raw accounts.RawCredentials, read func() (accounts.SeatStatus, error)) (accounts.SeatStatus, error) {
	if err := s.Ready(); err != nil {
		return accounts.SeatStatus{}, err
	}
	local, err := s.Principal()
	if err != nil {
		return accounts.SeatStatus{}, err
	}
	if _, err := s.SignIn(s.Context(), raw, local); err != nil {
		return accounts.SeatStatus{}, err
	}
	return read()
}
