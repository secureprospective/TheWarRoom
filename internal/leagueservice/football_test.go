package leagueservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type footballTestHost struct{ fixtureChangeHost }

func (footballTestHost) CommissionerPrincipal() (string, error) { return "owner.two", nil }
func (footballTestHost) NativeFacts() nativecheck.Facts         { panic("desktop facts must not be read") }

func importTestFootball(t *testing.T, f *dueFixture, name string) string {
	t.Helper()
	raw := history.RawFootballEvidence{Version: 1, Season: 2026, Source: "keyless reference fixture", AsOf: f.clock.Now(),
		Players: map[string]history.RawFootballPlayer{"0099": {Identity: "reference.99", Facts: nativecheck.PlayerFacts{Name: name, Position: leaguerules.QB, Team: "DAL", DesignationHeld: true}}},
		Weeks:   []leagueweek.Week{{Number: 1, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: f.clock.Now().Add(time.Hour)}}}}}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(body)
	digest := hex.EncodeToString(hash[:])
	if err := f.store.ImportFootball(t.Context(), body, digest); err != nil {
		t.Fatal(err)
	}
	return digest
}

func TestHostedCommandRequiresPinnedFootballDigestWithoutDesktopKey(t *testing.T) {
	f := newDueFixture(t)
	digest := importTestFootball(t, f, "Player 99")
	host, err := NewFootballHost(t.Context(), footballTestHost{fixtureChangeHost{Store: f.store}}, f.store, digest)
	if err != nil {
		t.Fatal(err)
	}
	service := NewHosted(host, admissionFunc(func(context.Context, commissioner.Access) error { return nil }))
	raw := f.raw(t, commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0099", Franchise: "franchise.02"}})
	ledger, err := service.TargetCommissionerLedger(dueLeague)
	if err != nil {
		t.Fatal(err)
	}
	raw.Expected = ledger.Revision
	changed := importTestFootball(t, f, "Changed identity facts")
	before := len(f.events(t))
	for _, pin := range []string{"", changed, strings.Repeat("0", 64)} {
		if _, err := service.Execute(WithFootballDigest(t.Context(), pin), raw); err == nil {
			t.Fatal("missing or changed evidence accepted")
		}
		if len(f.events(t)) != before {
			t.Fatal("refused command appended")
		}
	}
	for _, pin := range []string{"", changed} {
		if _, err := host.PreviewCommissionerWithFacts(WithFootballDigest(t.Context(), pin), raw, "owner.two", nil, commissioner.PreviewCatalogue{}, nil, time.UTC); err == nil {
			t.Fatal("preview accepted changed or missing evidence")
		}
	}
	preview, err := host.PreviewCommissionerWithFacts(commissioner.WithRecipient(WithFootballDigest(t.Context(), digest), "owner.two", f.clock.Now()), raw, "owner.two", nil, commissioner.PreviewCatalogue{}, nil, time.UTC)
	if err != nil || len(preview.Blockers) != 0 || len(f.events(t)) != before {
		t.Fatalf("keyless preview: %+v %v", preview, err)
	}
	receipt, err := service.Execute(WithFootballDigest(t.Context(), digest), raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.events(t)) != before+1 {
		t.Fatal("keyless command not committed once")
	}
	retry, err := service.Execute(WithFootballDigest(t.Context(), digest), raw)
	if err != nil || retry.Entry != receipt.Entry {
		t.Fatalf("retry: %+v %v", retry, err)
	}
	if _, err := service.Execute(WithFootballDigest(t.Context(), changed), raw); err == nil {
		t.Fatal("retry bypassed evidence admission")
	}
	if len(f.events(t)) != before+1 {
		t.Fatal("retry appended")
	}
	weeks, err := host.NativeScheduleDates(t.Context(), 2026, f.clock.Now())
	if err != nil || weeks[0].Games[0].Teams[0] != "DAL" {
		t.Fatalf("server schedule: %+v %v", weeks, err)
	}
	if host.NativeFacts()["0099"].Name != "Player 99" {
		t.Fatal("later import replaced pinned facts")
	}
	if _, err := NewFootballHost(t.Context(), nil, f.store, changed+"bad"); err == nil {
		t.Fatal("invalid startup selection accepted")
	}
	lineup, err := f.store.NativeLineup(t.Context(), dueLeague, "franchise.02", "owner.two", 1, f.clock.Now())
	if err != nil {
		t.Fatalf("invalid selection damaged prior lineup environment: %+v %v", lineup, err)
	}
}

func TestFootballAdmissionRechecksInsideAppendAndPreservesCredential(t *testing.T) {
	f := newDueFixture(t)
	digest := importTestFootball(t, f, "Player 99")
	host, err := NewFootballHost(t.Context(), footballTestHost{fixtureChangeHost{Store: f.store}}, f.store, digest)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	service := NewHosted(host, admissionFunc(func(_ context.Context, access commissioner.Access) error {
		calls++
		if len(access.Records) > 0 {
			return context.Canceled
		}
		return nil
	}))
	before := len(f.events(t))
	raw := f.raw(t, commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0099", Franchise: "franchise.02"}})
	if _, err := service.Execute(WithFootballDigest(t.Context(), digest), raw); err == nil {
		t.Fatal("football guard replaced credential admission")
	}
	if calls != 2 || len(f.events(t)) != before {
		t.Fatalf("admission calls=%d, state changed=%v", calls, len(f.events(t)) != before)
	}
}
