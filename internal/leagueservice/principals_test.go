package leagueservice

import (
	"context"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type principalLedger struct{ records []commissioner.Record }

func (l principalLedger) Events(context.Context, string, string, int64) ([]history.LeagueEvent, error) {
	events := make([]history.LeagueEvent, len(l.records))
	for i := range l.records {
		events[i].Payload.Commissioner = &l.records[i]
	}
	return events, nil
}

func TestServiceRecordReadsResolveVerifiedAliases(t *testing.T) {
	t.Parallel()
	const league = "native.https-fixture"
	const account = "acct.12345678-1234-4234-8234-123456789abc"
	fixture, err := FixtureRecords()
	if err != nil {
		t.Fatal(err)
	}
	aliases, err := governance.NewPrincipals(league, []governance.VerifiedAlias{{League: league, Alias: governance.Principal(fixture[0].Principal), Account: account}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := commissioner.WithPrincipals(t.Context(), aliases)
	records, err := Records(ctx, principalLedger{records: fixture}, league, "people")
	if err != nil {
		t.Fatal(err)
	}
	people, err := commissioner.PeopleReading(league, account, records)
	if err != nil || people.YourRole != "Commissioner" {
		t.Fatalf("service lost verified grants: %+v %v", people, err)
	}
	if records[0].Principal != fixture[0].Principal || records[0].Canonical != fixture[0].Canonical {
		t.Fatal("service rewrote original principal or retry identity")
	}
	if _, err := Records(ctx, principalLedger{records: fixture}, "native.other", "people"); err == nil {
		t.Fatal("service accepted cross-league aliases")
	}
}
