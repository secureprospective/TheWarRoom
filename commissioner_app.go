package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

type localCommissionerSession struct{ installation string }

func (s localCommissionerSession) Principal() (string, error) {
	if s.installation == "" {
		return "", fmt.Errorf("local identity is unavailable")
	}
	identity := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d", s.installation, os.Getuid())))
	return "local." + hex.EncodeToString(identity[:]), nil
}

func (a *App) commissionerPrincipal() (string, error) {
	session := a.commissionerSession
	if session == nil {
		session = localCommissionerSession{installation: a.dataDir}
	}
	principal, err := session.Principal()
	if err != nil {
		return "", fmt.Errorf("read local identity: %w", err)
	}
	if strings.TrimSpace(principal) == "" {
		return "", fmt.Errorf("local identity is unavailable")
	}
	return principal, nil
}

func (a *App) TargetCommissionerExecute(body string) (commissioner.Receipt, error) {
	raw, err := commissioner.Decode(body)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("league change: %w", err)
	}
	if raw.Command == commissioner.NativeSeed {
		return commissioner.Receipt{}, fmt.Errorf("record the starting snapshot through the held-data action")
	}
	if err := a.ready(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("league change startup: %w", err)
	}
	principal, err := a.commissionerPrincipal()
	if err != nil {
		return commissioner.Receipt{}, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	receipt, err := a.history.ExecuteCommissioner(ctx, raw, principal, a.commissionerAuthorizer)
	var refused *commissioner.Refusal
	if errors.As(err, &refused) {
		return commissioner.Receipt{}, errors.New(refused.Reason())
	}
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("league change: %w", err)
	}
	return receipt, nil
}

func (a *App) TargetCommissionerLedger(league string) (commissioner.Ledger, error) {
	if !strings.HasPrefix(league, "native.") {
		return commissioner.Ledger{}, fmt.Errorf("choose an app league")
	}
	if err := a.ready(); err != nil {
		return commissioner.Ledger{}, fmt.Errorf("league ledger startup: %w", err)
	}
	principal, err := a.commissionerPrincipal()
	if err != nil {
		return commissioner.Ledger{}, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.CommissionerLedger(ctx, league, principal, a.commissionerAuthorizer)
	if err != nil {
		return commissioner.Ledger{}, fmt.Errorf("league ledger: %w", err)
	}
	return reading, nil
}

func (a *App) TargetCommissionerPeople(league string) (commissioner.People, error) {
	if !strings.HasPrefix(league, "native.") || league == "native." {
		return commissioner.People{}, fmt.Errorf("choose an app league")
	}
	if err := a.ready(); err != nil {
		return commissioner.People{}, fmt.Errorf("league people startup: %w", err)
	}
	principal, err := a.commissionerPrincipal()
	if err != nil {
		return commissioner.People{}, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	people, err := a.history.CommissionerPeople(ctx, league, principal)
	if err != nil {
		return commissioner.People{}, fmt.Errorf("league people: %w", err)
	}
	return people, nil
}

func (a *App) TargetCommissionerLeagues() ([]commissioner.NativeLeague, error) {
	if err := a.ready(); err != nil {
		return nil, fmt.Errorf("app leagues startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	leagues, err := a.history.NativeLeagues(ctx)
	if err != nil {
		return nil, fmt.Errorf("app leagues: %w", err)
	}
	return leagues, nil
}

func (a *App) TargetCommissionerRules(league string) (commissioner.Rules, error) {
	if !strings.HasPrefix(league, "native.") || league == "native." || strings.TrimSpace(league) != league {
		return commissioner.Rules{}, fmt.Errorf("choose an app league")
	}
	if err := a.ready(); err != nil {
		return commissioner.Rules{}, fmt.Errorf("league rules startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.CommissionerRules(ctx, league)
	if err != nil {
		return commissioner.Rules{}, fmt.Errorf("league rules: %w", err)
	}
	return reading, nil
}
