package commissioner

import (
	"reflect"
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	domain "github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

func TestReleaseMatrixCatalogTokenIssuers(t *testing.T) {
	t.Parallel()
	f := newReleaseTokenFixture(t)
	other := f.enroll(t, "other-commissioner")
	f.createOtherLeague(t, other.Account)
	for _, role := range []string{"owner", "deputy", "second-deputy", "support", "server-operator"} {
		session := f.enroll(t, role)
		principal, err := session.Account.Principal()
		if err != nil {
			t.Fatal(err)
		}
		switch role {
		case "owner":
			f.change(t, "commish.seat.replace", domain.RawPeople{Franchise: "franchise.02", Principal: string(principal)})
		case "deputy", "second-deputy":
			f.change(t, "commish.role.grant", domain.RawPeople{Principal: string(principal), Role: governance.CoCommissioner})
		}
		before, otherBefore := f.records(t, "native.hosted"), f.records(t, "native.other")
		t.Run(role, func(t *testing.T) {
			for _, command := range NativeCommands() {
				if !slices.Contains(command.Modes, catalog.Execute) {
					continue
				}
				t.Run(command.ID, func(t *testing.T) {
					for _, league := range []string{"native.hosted", "native.other"} {
						for _, mode := range []string{"preview", "execute"} {
							_, err := f.tokens.Issue(t.Context(), session.RawSession, accounts.RawTokenScope{
								League: league, Franchise: "franchise.02", Commands: []string{command.ID}, Modes: []string{mode},
							})
							if err == nil {
								t.Fatalf("%s minted a %s token for %s without being commissioner", role, mode, league)
							}
						}
					}
				})
			}
			if !reflect.DeepEqual(before, f.records(t, "native.hosted")) || !reflect.DeepEqual(otherBefore, f.records(t, "native.other")) {
				t.Fatal("refused token issuance changed league history")
			}
		})
	}
}

func TestReleaseMatrixCaretakerTokenLosesSeatWhenOwnerArrives(t *testing.T) {
	t.Parallel()
	f := newReleaseTokenFixture(t)
	owner := f.enroll(t, "new-owner")
	f.change(t, domain.CaretakerSeat, domain.RawPeople{Franchise: "franchise.02", Principal: f.app.session})
	for _, mode := range []string{"preview", "execute"} {
		t.Run(mode, func(t *testing.T) {
			releaseCaretakerToken(t, f, owner, mode)
		})
	}
}

func releaseCaretakerToken(t *testing.T, f *releaseTokenFixture, owner accounts.DeviceSession, mode string) {
	t.Helper()
	token, err := f.tokens.Issue(t.Context(), f.creator.RawSession, accounts.RawTokenScope{
		League: "native.hosted", Franchise: "franchise.02", Commands: []string{domain.LineupSet, domain.DraftPick}, Modes: []string{mode},
	})
	if err != nil {
		t.Fatal(err)
	}
	gate := f.tokens.Admission(token.RawSession)
	lineup := domain.RawCommand{League: "native.hosted", Command: domain.LineupSet,
		Args:    domain.RawArgs{Lineup: &domain.RawLineupChange{Season: 2026, Week: 1, Franchise: "franchise.02", Starters: []string{"0099"}}},
		Request: "caretaker.lineup", Expected: int64(len(f.records(t, "native.hosted"))), Effective: "now", Visibility: "private"}
	if err := lineup.Validate(); err != nil {
		t.Fatal(err)
	}
	access := domain.Access{Principal: f.app.session, League: lineup.League, Operation: lineup.Command, Mode: mode, Raw: lineup}
	if err := gate.Admit(t.Context(), access); err != nil {
		t.Fatalf("current caretaker token refused: %v", err)
	}
	principal, err := owner.Account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	f.change(t, "commish.seat.replace", domain.RawPeople{Franchise: "franchise.02", Principal: string(principal)})
	before := f.records(t, "native.hosted")
	if err := gate.Admit(t.Context(), access); err == nil {
		t.Fatal("former caretaker token retained seat access")
	}
	service := leagueservice.NewHosted(serviceParityHost{Host: f.app.host}, gate)
	if _, err := service.SetLineup(t.Context(), lineup); err == nil {
		t.Fatal("former caretaker changed an owner's lineup through the shared service")
	}
	if !reflect.DeepEqual(before, f.records(t, "native.hosted")) {
		t.Fatal("former caretaker refusal changed league history")
	}
	f.change(t, "commish.seat.remove", domain.RawPeople{Franchise: "franchise.02"})
	before = f.records(t, "native.hosted")
	if err := gate.Admit(t.Context(), access); err == nil {
		t.Fatal("removing the owner resurrected caretaker authority")
	}
	if !reflect.DeepEqual(before, f.records(t, "native.hosted")) {
		t.Fatal("vacant-seat refusal changed league history")
	}
	f.change(t, domain.CaretakerSeat, domain.RawPeople{Franchise: "franchise.02", Principal: f.app.session})
}
