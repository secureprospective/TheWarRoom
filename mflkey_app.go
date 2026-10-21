package main

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/pendingtrades"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
)

// MFLKeyStatus exposes presence, never any portion of the credential. State is one of
// absent | connected | rejected | unreachable | unavailable. Every one of them is returned with
// a nil error: Wails rejects the JS promise on any error and drops the status with it.
type MFLKeyStatus struct {
	State      string `json:"state"`
	League     string `json:"league"`
	Season     int    `json:"season"`
	VerifiedAt string `json:"verifiedAt,omitempty"`
	Detail     string `json:"detail,omitempty"`
}

func (a *App) keyStatus(state, detail string) MFLKeyStatus {
	return MFLKeyStatus{
		State: state, League: ingestion.LeagueID, Season: a.core.Season,
		VerifiedAt: a.core.KeyVerifiedAt, Detail: detail,
	}
}

func (a *App) MFLKeyStatus() (MFLKeyStatus, error) {
	if err := a.ready(); err != nil {
		return MFLKeyStatus{}, err
	}
	a.core.KeyMu.Lock()
	defer a.core.KeyMu.Unlock()
	_, err := a.core.KeyStore.Get(a.core.Ctx)
	if errors.Is(err, mflkey.ErrNoKey) {
		return a.keyStatus("absent", ""), nil
	}
	if err != nil {
		return a.keyStatus("unavailable", err.Error()), nil
	}
	return a.keyStatus("connected", ""), nil
}

func (a *App) SetMFLKey(key string) (status MFLKeyStatus, err error) {
	// Wails recovers a binding panic and logs the raw IPC message, key included, at ERROR level
	// (on in production). Recover here first, and log only the panic's type.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("SetMFLKey: recovered panic (%T)", r)
			status, err = a.keyStatus("unavailable", "internal error while connecting; check the status"), nil
		}
	}()
	candidate := mflkey.Key(strings.TrimSpace(key))
	if err := a.ready(); err != nil {
		return MFLKeyStatus{}, err
	}
	a.core.KeyMu.Lock()
	defer a.core.KeyMu.Unlock()
	// Bounds are plausibility checks, not a claim about MFL's key encoding.
	if len(candidate) < 16 || len(candidate) > 256 {
		detail := "key must contain 16 to 256 bytes after trimming whitespace"
		return a.keyStatus("rejected", detail), nil
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, time.Minute)
	defer cancel()
	c := a.core.MFLClient.WithKeySource(func(context.Context) (mflkey.Key, error) { return candidate, nil })
	verified, err := pendingtrades.Verify(ctx, c, strconv.Itoa(a.core.Season), ingestion.LeagueID)
	if errors.Is(err, pendingtrades.ErrRejected) {
		return a.keyStatus("rejected", err.Error()), nil
	}
	if err != nil {
		return a.keyStatus("unreachable", err.Error()+"; nothing was stored"), nil
	}
	if err := a.core.KeyStore.Set(ctx, candidate); err != nil {
		return a.keyStatus("unavailable", err.Error()), nil
	}
	a.core.KeyVerifiedAt = time.Now().UTC().Format(time.RFC3339)
	a.core.KeyGeneration++
	a.refreshPendingTradesInBackground()
	return a.keyStatus("connected", verified.Detail), nil
}

func (a *App) DeleteMFLKey() (MFLKeyStatus, error) {
	if err := a.ready(); err != nil {
		return MFLKeyStatus{}, err
	}
	a.core.KeyMu.Lock()
	defer a.core.KeyMu.Unlock()
	if err := a.core.KeyStore.Delete(a.core.Ctx); err != nil {
		return a.keyStatus("unavailable", err.Error()), nil
	}
	a.core.KeyVerifiedAt = ""
	a.core.KeyGeneration++
	a.core.SeasonMu.Lock()
	a.clearPendingTrades()
	a.core.SeasonMu.Unlock()
	a.refreshPendingTradesInBackground()
	return a.keyStatus("absent", ""), nil
}
