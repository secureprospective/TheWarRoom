package leagueservice

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func reserveRule(t *testing.T, f *dueFixture, field, value string) {
	t.Helper()
	f.execute(t, "creator", commissioner.RulesSet, commissioner.RawArgs{Field: field, Value: json.RawMessage(value)})
}

func reserveRecords(t *testing.T, f *dueFixture) []commissioner.Record {
	t.Helper()
	records := []commissioner.Record{}
	for _, e := range f.events(t) {
		records = append(records, *e.Payload.Commissioner)
	}
	return records
}

func TestWorkerReturnsReservesAndKeepsMustCutUntilHumanResolution(t *testing.T) {
	f := newDueFixture(t)
	reserveRule(t, f, "calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["ir","taxi","cut"]}}`)
	f.facts["0099"] = nativecheck.PlayerFacts{Position: leaguerules.QB, DesignationHeld: true, Designation: leaguerules.InjuredReserve}
	f.execute(t, "owner.two", commissioner.RosterReserve, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0099", Franchise: "franchise.02", Bucket: "ir"}})
	reserveRule(t, f, "roster.limits", `{"min":0,"max":0}`)
	before := len(f.events(t))
	// Legacy remains manual, even with known ineligibility.
	f.facts["0099"] = nativecheck.PlayerFacts{Position: leaguerules.QB, DesignationHeld: true}
	if _, err := f.worker.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(f.events(t)) != before {
		t.Fatal("Legacy auto-returned a player")
	}
	reserveRule(t, f, "ir.autoReturn", `true`)
	// The automatic move must not depend on owner calendar windows.
	reserveRule(t, f, "calendar.windows", `{}`)
	if _, err := f.worker.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	events := f.events(t)
	last := events[len(events)-1].Payload.Commissioner
	if last.Principal != commissioner.ReserveClockPrincipal || last.Command.Args.WaiverOutcome.Source != commissioner.ReserveClockPrincipal {
		t.Fatalf("not auto-return: %+v", last)
	}
	checkMustCut(t, f, true)
	count := len(events)
	if err := f.pools.Close(); err != nil {
		t.Fatal(err)
	}
	f.open(t)
	f.clock.instant.Add(int64(365 * 24 * time.Hour))
	if _, err := f.worker.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(f.events(t)) != count {
		t.Fatal("time cut a player or repeated the move")
	}
	checkMustCut(t, f, true)
	// Fix the roster through a human cut, not an alert acknowledgement.
	reserveRule(t, f, "calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["cut"]}}`)
	f.execute(t, "owner.two", commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0099", Franchise: "franchise.02"}})
	checkMustCut(t, f, false)
}

func checkMustCut(t *testing.T, f *dueFixture, want bool) {
	t.Helper()
	jobs, err := commissioner.JobsReading(f.league, reserveRecords(t, f), f.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, j := range jobs.Jobs {
		if strings.HasPrefix(j.ID, "must_cut.") {
			found = true
			if j.State != commissioner.JobLive || j.Closes != "" || !strings.Contains(j.Words, "Unresolved team activity") {
				t.Fatalf("bad alert: %+v", j)
			}
		}
	}
	if found != want {
		t.Fatalf("must-cut=%v want %v", found, want)
	}
}

func TestTaxiAutoPromotionWaitsForHeldNFLRosterSeasons(t *testing.T) {
	f := newDueFixture(t)
	reserveRule(t, f, "calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["taxi"]}}`)
	reserveRule(t, f, "taxi.basis", `"nfl_seasons"`)
	f.facts["0099"] = nativecheck.PlayerFacts{Position: leaguerules.QB, NFLSeasons: 3, NFLSeasonsHeld: true}
	f.execute(t, "owner.two", commissioner.RosterReserve, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0099", Franchise: "franchise.02", Bucket: "taxi"}})
	reserveRule(t, f, "taxi.autoPromote", `true`)
	f.facts["0099"] = nativecheck.PlayerFacts{Position: leaguerules.QB, DraftYear: 2000, DraftYearHeld: true}
	before := len(f.events(t))
	if _, err := f.worker.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(f.events(t)) != before {
		t.Fatal("draft year replaced missing NFL roster history")
	}
	f.facts["0099"] = nativecheck.PlayerFacts{Position: leaguerules.QB, NFLSeasons: 4, NFLSeasonsHeld: true}
	if _, err := f.worker.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	assets, err := commissioner.NativeAssetsReading(f.league, reserveRecords(t, f), f.clock.Now())
	if err != nil || assets.Snapshot.Players[0].Status != domain.RosterActive {
		t.Fatalf("promotion: %+v %v", assets, err)
	}
}

func TestReserveClockRechecksEligibilityAndCannotCut(t *testing.T) {
	f := newDueFixture(t)
	reserveRule(t, f, "calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["ir"]}}`)
	f.facts["0099"] = nativecheck.PlayerFacts{Position: leaguerules.QB, DesignationHeld: true, Designation: leaguerules.InjuredReserve}
	f.execute(t, "owner.two", commissioner.RosterReserve, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0099", Franchise: "franchise.02", Bucket: "ir"}})
	reserveRule(t, f, "ir.autoReturn", `true`)
	raw := f.raw(t, commissioner.RosterReserve, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0099", Franchise: "franchise.02", Bucket: "active"}})
	records := reserveRecords(t, f)
	if err := commissioner.PrepareDueReserve(&raw, records, f.clock.Now(), f.facts); err == nil {
		t.Fatal("still-eligible IR player returned")
	}
	f.facts["0099"] = nativecheck.PlayerFacts{Position: leaguerules.QB}
	if err := commissioner.PrepareDueReserve(&raw, records, f.clock.Now(), f.facts); err == nil {
		t.Fatal("missing designation triggered return")
	}
	f.facts["0099"] = nativecheck.PlayerFacts{Position: leaguerules.QB, DesignationHeld: true}
	if err := commissioner.PrepareDueReserve(&raw, records, f.clock.Now(), f.facts); err != nil {
		t.Fatal(err)
	}
	raw.Command = commissioner.RosterCut
	if err := commissioner.PrepareDueReserve(&raw, records, f.clock.Now(), f.facts); err == nil {
		t.Fatal("reserve clock could cut")
	}
}
