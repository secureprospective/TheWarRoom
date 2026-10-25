package leagueservice

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func TestReleaseMatrixLeagueRolesPreserveDeniedHistory(t *testing.T) {
	t.Parallel()
	f := newDueFixture(t)
	for _, deputy := range []string{"deputy.one", "deputy.two"} {
		raw := f.raw(t, "commish.role.grant", commissioner.RawArgs{People: &commissioner.RawPeople{Principal: deputy, Role: governance.CoCommissioner}})
		raw.Visibility = "private"
		if _, err := f.store.ExecuteCommissioner(t.Context(), raw, "creator", nil); err != nil {
			t.Fatal(err)
		}
	}
	f.execute(t, "creator", commissioner.CaretakerSeat, commissioner.RawArgs{People: &commissioner.RawPeople{Principal: "creator", Franchise: "franchise.03"}})
	const other = "native.other"
	raw := commissioner.RawCommand{League: other, Command: commissioner.Create, Args: commissioner.RawArgs{Name: "Other league", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 2}, Request: "create", Reason: "Create another league", Effective: "now", Visibility: "league"}
	if _, err := f.store.ExecuteCommissioner(t.Context(), raw, "other.creator", nil); err != nil {
		t.Fatal(err)
	}
	for _, league := range []string{dueLeague, other} {
		for _, actor := range []struct {
			name, principal string
			allowed         bool
		}{
			{"commissioner", "creator", league == dueLeague},
			{"deputy", "deputy.one", league == dueLeague},
			{"second-deputy", "deputy.two", league == dueLeague},
			{"owner", "owner.one", false},
			{"support-no-league-grant", "support", false},
			{"server-operator-no-league-grant", "server.operator", false},
			{"other-commissioner", "other.creator", league == other},
		} {
			t.Run(league+"/"+actor.name, func(t *testing.T) {
				for _, adapter := range []string{"store", "service"} {
					before := releaseRoleHistory(t, f, league)
					raw := commissioner.RawCommand{League: league, Command: commissioner.NotePost, Args: commissioner.RawArgs{Note: &commissioner.RawNote{Text: "League note from " + actor.name}}, Expected: int64(len(before)), Request: fmt.Sprintf("%s.%s", actor.name, adapter), Effective: "now", Visibility: "league"}
					var err error
					if adapter == "service" {
						service := viewService(t, f, actor.principal)
						if actor.allowed {
							ledger, readErr := service.TargetCommissionerLedger(league)
							if readErr != nil {
								t.Fatal(readErr)
							}
							raw.Expected = ledger.Revision
						}
						_, err = service.Execute(t.Context(), raw)
					} else {
						_, err = f.store.ExecuteCommissionerWithFacts(t.Context(), raw, actor.principal, nil, f.facts, time.UTC)
					}
					after := releaseRoleHistory(t, f, league)
					if actor.allowed {
						if err != nil || len(after) != len(before)+1 {
							t.Fatalf("%s: granted role failed: %v", adapter, err)
						}
					} else if err == nil || !reflect.DeepEqual(before, after) {
						t.Fatalf("%s: denied role changed history: %v", adapter, err)
					}
				}
			})
		}
	}
}

func releaseRoleHistory(t *testing.T, f *dueFixture, league string) []commissioner.Record {
	t.Helper()
	records, err := (StoredHost{Store: f.store}).CommissionerRecords(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	return records
}
