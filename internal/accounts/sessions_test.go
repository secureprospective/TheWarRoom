package accounts_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func loginDevice(t *testing.T, f *totpFixture, service *accounts.SessionService, name string) accounts.DeviceSession {
	t.Helper()
	session, err := service.Login(t.Context(), f.credentials, name)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func TestDeviceSessionIdleAndAbsoluteBoundaries(t *testing.T) {
	f := newTOTPFixture(t)
	service := accounts.NewSessions(accounts.New(f.store), f.store)
	issued := f.now
	for _, delta := range []time.Duration{-time.Nanosecond, 0, time.Nanosecond} {
		f.now = issued
		session := loginDevice(t, f, service, "idle device")
		f.now = issued.Add(accounts.SessionIdleLifetime + delta)
		_, err := service.Authenticate(t.Context(), session.RawSession)
		if (err == nil) != (delta < 0) {
			t.Fatalf("idle boundary %s: %v", delta, err)
		}
	}
	f.now = issued
	active := loginDevice(t, f, service, "active device")
	for day := 1; day < 30; day++ {
		f.now = issued.Add(time.Duration(day) * 24 * time.Hour)
		if _, err := service.Authenticate(t.Context(), active.RawSession); err != nil {
			t.Fatal(err)
		}
	}
	for _, delta := range []time.Duration{-time.Nanosecond, 0, time.Nanosecond} {
		f.now = issued.Add(accounts.SessionAbsoluteLifetime + delta)
		_, err := service.Authenticate(t.Context(), active.RawSession)
		if (err == nil) != (delta < 0) {
			t.Fatalf("absolute boundary %s: %v", delta, err)
		}
	}
	if !active.ExpiresAt.Equal(issued.Add(accounts.SessionAbsoluteLifetime)) {
		t.Fatal("wrong absolute expiration")
	}
	f.now = issued.Add(-time.Nanosecond)
	if _, err := service.Authenticate(t.Context(), active.RawSession); !errors.Is(err, accounts.ErrSessionRefused) {
		t.Fatalf("backward clock: %v", err)
	}
}

func TestDeviceSessionsHashOnlyRevocationAndEpoch(t *testing.T) {
	f := newTOTPFixture(t)
	service := accounts.NewSessions(accounts.New(f.store), f.store)
	first := loginDevice(t, f, service, "desktop")
	second := loginDevice(t, f, service, "laptop")
	if first.Bearer == second.Bearer {
		t.Fatal("shared bearer")
	}
	var stored string
	if err := f.pools.Read().QueryRowContext(t.Context(), `SELECT token_hash FROM account_sessions WHERE device = ?`, first.Device).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != digestToken(first.Bearer) || strings.Contains(stored, first.Bearer) {
		t.Fatal("not hash-only storage")
	}
	body, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), first.Bearer) || strings.Contains(fmt.Sprintf("%v %+v %#v", first, first.RawSession, first), first.Bearer) {
		t.Fatal("bearer leaked")
	}
	if err := service.Revoke(t.Context(), first.RawSession, false); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Authenticate(t.Context(), first.RawSession); !errors.Is(err, accounts.ErrSessionRefused) {
		t.Fatalf("revoked device: %v", err)
	}
	if _, err := service.Authenticate(t.Context(), second.RawSession); err != nil {
		t.Fatal(err)
	}
	third := loginDevice(t, f, service, "phone")
	if err := service.Revoke(t.Context(), second.RawSession, true); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Authenticate(t.Context(), third.RawSession); !errors.Is(err, accounts.ErrSessionRefused) {
		t.Fatalf("all-device revoke: %v", err)
	}
	fresh := loginDevice(t, f, service, "new epoch device")
	// Reinitialization must not invalidate sessions; only explicit restore rotation does.
	if err := f.store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Authenticate(t.Context(), fresh.RawSession); err != nil {
		t.Fatal(err)
	}
	if err := f.store.RotateAuthEpoch(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Authenticate(t.Context(), fresh.RawSession); !errors.Is(err, accounts.ErrSessionRefused) {
		t.Fatalf("old epoch: %v", err)
	}
	latest := loginDevice(t, f, service, "after rotation")
	if _, err := service.Authenticate(t.Context(), latest.RawSession); err != nil {
		t.Fatal(err)
	}
}

func TestDeviceSessionRecoveryAndPasswordRace(t *testing.T) {
	f, recovery, codes, _ := recoveryFixture(t)
	service := accounts.NewSessions(accounts.New(f.store), f.store)
	session := loginDevice(t, f, service, "before recovery")
	old, err := f.store.Credential(t.Context(), f.account.Username)
	if err != nil {
		t.Fatal(err)
	}
	if err := recovery.Recover(t.Context(), accounts.RawRecovery{Username: f.credentials.Username, Password: "replacement-password", Code: codes.Codes[0]}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Authenticate(t.Context(), session.RawSession); !errors.Is(err, accounts.ErrSessionRefused) {
		t.Fatalf("recovery retained session: %v", err)
	}
	if _, err := f.store.CreateSession(t.Context(), old, digestToken(strings.Repeat("x", 43)), "raced login"); !errors.Is(err, accounts.ErrSessionRefused) {
		t.Fatalf("old password issued new generation: %v", err)
	}
}

func TestDeviceSessionRevokesQueuedWriteBeforeAppend(t *testing.T) {
	f := newTOTPFixture(t)
	service := accounts.NewSessions(accounts.New(f.store), f.store)
	session := loginDevice(t, f, service, "queued device")
	principal, err := f.account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	ledger := history.New(f.pools, registry)
	if err := ledger.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	raw := commissioner.RawCommand{League: "native.sessions", Command: commissioner.Create, Args: commissioner.RawArgs{Name: "Sessions", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 2}, Reason: "Create test league", Request: "create", Effective: "now", Visibility: "league"}
	gate := service.Admission(session.RawSession)
	access := commissioner.Access{Principal: string(principal), League: raw.League, Mode: "execute", Raw: raw}
	if err := gate.Admit(t.Context(), access); err != nil {
		t.Fatal(err)
	}
	spoofed := access
	spoofed.Principal = "spoofed.owner"
	if err := gate.Admit(t.Context(), spoofed); !errors.Is(err, accounts.ErrSessionRefused) {
		t.Fatalf("spoofed identity: %v", err)
	}
	tx, err := f.pools.Write().BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(t.Context(), `UPDATE account_sessions SET revoked = 1 WHERE token_hash = ?`, digestToken(session.Bearer)); err != nil {
		t.Fatal(err)
	}
	waits := f.pools.Write().Stats().WaitCount
	result := make(chan error, 1)
	go func() {
		_, err := ledger.ExecuteCommissioner(commissioner.WithCredentialAdmission(t.Context(), gate), raw, string(principal), nil)
		result <- err
	}()
	deadline := time.Now().Add(5 * time.Second)
	for f.pools.Write().Stats().WaitCount == waits {
		if time.Now().After(deadline) {
			t.Fatal("write never queued")
		}
		time.Sleep(time.Millisecond)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, accounts.ErrSessionRefused) {
			t.Fatalf("queued revoked write: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("queued write deadlocked")
	}
	events, err := ledger.Events(t.Context(), raw.League, commissioner.Stream, 0)
	if err != nil || len(events) != 0 {
		t.Fatalf("denied write appended %d events: %v", len(events), err)
	}
	// The identical command succeeds with a fresh session; the fixture is legal.
	fresh := loginDevice(t, f, service, "replacement device")
	if _, err := ledger.ExecuteCommissioner(commissioner.WithCredentialAdmission(t.Context(), service.Admission(fresh.RawSession)), raw, string(principal), nil); err != nil {
		t.Fatal(err)
	}
}

func TestDeviceSessionValidationAndAdmissionDoesNotRefreshIdle(t *testing.T) {
	for _, bearer := range []string{"", "not-a-bearer", strings.Repeat("a", 42), strings.Repeat("!", 43), strings.Repeat("A", 42) + "B"} {
		if err := (accounts.RawSession{Bearer: bearer}).Validate(); err == nil {
			t.Fatal("malformed bearer accepted")
		}
	}
	f := newTOTPFixture(t)
	service := accounts.NewSessions(accounts.New(f.store), f.store)
	wrong := f.credentials
	wrong.Password = "incorrect"
	if _, err := service.Login(t.Context(), wrong, "device"); !errors.Is(err, accounts.ErrInvalidCredentials) {
		t.Fatalf("wrong login: %v", err)
	}
	for _, device := range []string{"", strings.Repeat("a", 65), "device\nname"} {
		if _, err := service.Login(t.Context(), f.credentials, device); !errors.Is(err, accounts.ErrSessionRefused) {
			t.Fatalf("invalid device: %v", err)
		}
	}
	session := loginDevice(t, f, service, "read-only admission")
	issued := f.now
	principal, err := f.account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	gate := service.Admission(session.RawSession)
	access := commissioner.Access{Principal: string(principal)}
	f.now = issued.Add(accounts.SessionIdleLifetime - time.Nanosecond)
	if err := gate.Admit(t.Context(), access); err != nil {
		t.Fatal(err)
	}
	f.now = issued.Add(accounts.SessionIdleLifetime)
	if err := gate.Admit(t.Context(), access); !errors.Is(err, accounts.ErrSessionRefused) {
		t.Fatalf("admission refreshed idle: %v", err)
	}
	f.now = issued
	active := loginDevice(t, f, service, "idle refreshed")
	f.now = issued.Add(6 * 24 * time.Hour)
	if _, err := service.Authenticate(t.Context(), active.RawSession); err != nil {
		t.Fatal(err)
	}
	f.now = issued.Add(12 * 24 * time.Hour)
	if _, err := service.Authenticate(t.Context(), active.RawSession); err != nil {
		t.Fatalf("activity did not refresh idle: %v", err)
	}
}

func TestSessionCheckDoesNotTouchIdleAndRejectsRevocation(t *testing.T) {
	f := newTOTPFixture(t)
	service := accounts.NewSessions(accounts.New(f.store), f.store)
	session := loginDevice(t, f, service, "idle stream")
	issued := f.now
	f.now = issued.Add(accounts.SessionIdleLifetime - time.Nanosecond)
	if _, err := service.Check(t.Context(), session.RawSession); err != nil {
		t.Fatal(err)
	}
	f.now = issued.Add(accounts.SessionIdleLifetime)
	if _, err := service.Check(t.Context(), session.RawSession); !errors.Is(err, accounts.ErrSessionRefused) {
		t.Fatalf("check extended idle lifetime: %v", err)
	}
	f.now = issued
	session = loginDevice(t, f, service, "revoked stream")
	if err := service.Revoke(t.Context(), session.RawSession, false); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Check(t.Context(), session.RawSession); !errors.Is(err, accounts.ErrSessionRefused) {
		t.Fatalf("check accepted revoked session: %v", err)
	}
}
