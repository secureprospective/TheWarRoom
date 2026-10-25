package commissioner

import (
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	domain "github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

type releaseActor struct {
	name      string
	principal string
	session   accounts.RawSession
}

func releaseActors(t *testing.T, f *releaseTokenFixture) []releaseActor {
	t.Helper()
	base := f.records(t, "native.hosted")
	actors := []releaseActor{{"commissioner", f.app.session, f.creator.RawSession}}
	for _, name := range []string{"deputy", "second-deputy", "DOT", "owner", "outsider"} {
		session := f.enroll(t, strings.ToLower(name))
		principal, err := session.Account.Principal()
		if err != nil {
			t.Fatal(err)
		}
		switch name {
		case "deputy", "second-deputy":
			f.change(t, "commish.role.grant", domain.RawPeople{Principal: string(principal), Role: governance.CoCommissioner})
		case "DOT":
			f.change(t, "commish.role.grant", domain.RawPeople{Principal: string(principal), Role: governance.DOT})
		case "owner":
			f.change(t, "commish.seat.replace", domain.RawPeople{Principal: string(principal), Franchise: "franchise.02"})
		}
		actors = append(actors, releaseActor{name, string(principal), session.RawSession})
	}
	other := f.enroll(t, "other-commissioner")
	principal, err := other.Account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	for i, record := range base {
		raw := record.Command
		raw.League = "native.other"
		raw.Expected = int64(i)
		raw.Args.WaiverOutcome = nil
		raw.Args.BidOutcome = nil
		raw.Args.DraftOutcome = nil
		if raw.Command == domain.DraftStart {
			draft := *raw.Args.Draft
			draft.At = time.Now().UTC().Add(time.Hour)
			raw.Args.Draft = &draft
		}
		if raw.Args.People != nil {
			people := *raw.Args.People
			people.Principal = string(principal)
			raw.Args.People = &people
		}
		if _, err := f.app.history.ExecuteCommissioner(t.Context(), raw, string(principal), nil); err != nil {
			t.Fatal(err)
		}
	}
	return actors
}

func TestReleaseMatrixCatalogRolesAndDesktopDoors(t *testing.T) {
	t.Parallel()
	rows := releaseMutationRows(t)
	f := newReleaseTokenFixture(t, true)
	actors := releaseActors(t, f)
	httpMatrix := newReleaseHTTP(t, f, actors)
	now := time.Now().UTC().Add(time.Hour)
	f.app.history.SetClock(func() time.Time { return now })
	before, otherBefore := f.records(t, "native.hosted"), f.records(t, "native.other")
	for _, actor := range actors {
		t.Run(actor.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if !reflect.DeepEqual(before, f.records(t, "native.hosted")) || !reflect.DeepEqual(otherBefore, f.records(t, "native.other")) {
					t.Error("matrix changed an existing league record")
				}
			}()
			host := leagueservice.StoredHost{Store: f.app.history, Context: t.Context, Principal: actor.principal, Registry: f.app.registry}
			revisions := map[string]int64{"native.hosted": domain.RecipientRevision(before, actor.principal, host.CommissionerNow()), "native.other": domain.RecipientRevision(otherBefore, actor.principal, host.CommissionerNow())}
			local := leagueservice.New(host)
			hosted := leagueservice.NewHosted(host, f.sessions.Admission(actor.session))
			desktop := New(fixtureHost{SessionHost: host, LeagueHost: f.app.history, ResultsHost: f.app.history, RosterHost: f.app.history, MutationHost: f.app.history})
			for command, row := range rows {
				t.Run(command, func(t *testing.T) {
					for _, league := range []string{"native.hosted", "native.other"} {
						records := before
						if league == "native.other" {
							records = otherBefore
						}
						raw := row.raw(t, command, league, actor.principal, int64(len(records)))
						_, permissionErr := domain.AuthorizeRecords(raw, actor.principal, records)
						// Fold-gated succession actions are admitted to the governance state machine,
						// which must still reject a missing election/offer or an ineligible takeover.
						fold := slices.Contains([]string{"commish.succession.accept", "commish.succession.decline", "commish.succession.takeover", "commish.vote.cast", "commish.vote.tiebreak"}, command)
						admitted := slices.Contains(row.AdmittedRoles, actor.name) && (league == "native.hosted" || fold)
						if (permissionErr == nil) != admitted {
							t.Fatalf("%s: authorization admitted=%v want=%v: %v", league, permissionErr == nil, admitted, permissionErr)
						}
						body, err := json.Marshal(raw)
						if err != nil {
							t.Fatal(err)
						}
						want, wantErr := local.Preview(raw)
						if permissionErr != nil && (len(want.Fields) != 0 || want.Target.League != "") {
							t.Fatal("denied preview disclosed league fields")
						}
						previews := map[string]func() (domain.Preview, error){
							"hosted-service": func() (domain.Preview, error) {
								scoped := raw
								scoped.Expected = revisions[league]
								got, err := hosted.Preview(scoped)
								if got.Target.Version == scoped.Expected {
									got.Target.Version = raw.Expected
								}
								return got, err
							},
						}
						if command != domain.NativeSeed {
							previews["Wails-preview"] = func() (domain.Preview, error) { return desktop.TargetCommissionerPreview(string(body)) }
						} else {
							got, err := desktop.TargetCommissionerPreview(string(body))
							if err != nil || !slices.Equal(got.Blockers, []string{"Preview starting assets through the held-data request"}) {
								t.Fatalf("generic seed preview opened: %+v %v", got, err)
							}
						}
						for name, preview := range previews {
							got, err := preview()
							if releaseError(err) != releaseError(wantErr) || !reflect.DeepEqual(got, want) {
								t.Fatalf("%s/%s: preview differs: %v\ngot=%+v\nwant=%+v", league, name, err, got, want)
							}
						}
						scoped := raw
						scoped.Expected = revisions[league]
						denied := !admitted || league != "native.hosted" || fold
						httpMatrix.humanCell(t, actor, scoped, admitted, denied)
						if !denied {
							continue
						}
						if wantErr == nil && len(want.Blockers) == 0 {
							t.Fatalf("%s: denied preview contains no blocker", league)
						}
						for name, execute := range releaseDeniedExecutors(t, f, desktop, local, hosted, raw, scoped, actor.principal) {
							err := execute()
							if err == nil {
								t.Fatalf("%s/%s: denied mutation succeeded", league, name)
							}
							if permissionErr != nil && raw.Command != domain.LineupSet && raw.Command != domain.DraftPick && raw.Command != domain.NativeSeed && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(releasePermissionReason(permissionErr))) {
								t.Fatalf("%s/%s: stopped before expected permission refusal %q: %v", league, name, permissionErr, err)
							}
						}
						for target, revision := range map[string]int64{"native.hosted": int64(len(before)), "native.other": int64(len(otherBefore))} {
							events, err := f.app.history.Events(t.Context(), target, domain.Stream, revision)
							if err != nil || len(events) != 0 {
								t.Fatalf("%s: denied mutation appended league history: %v", target, err)
							}
						}
					}
				})
			}
		})
	}
	t.Run("AI-helper", func(t *testing.T) { t.Parallel(); releaseHelperMatrix(t, f, httpMatrix) })
}

func releasePermissionReason(err error) string {
	var refusal *domain.Refusal
	if errors.As(domain.Refused(err), &refusal) {
		return refusal.Reason()
	}
	return ""
}

func releaseError(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

func releaseDeniedExecutors(t *testing.T, f *releaseTokenFixture, desktop *API, local, hosted *leagueservice.Service, raw, scoped domain.RawCommand, principal string) map[string]func() error {
	t.Helper()
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]func() error{
		"store": func() error {
			_, err := f.app.history.ExecuteCommissioner(t.Context(), raw, principal, nil)
			return err
		},
		"offline-service":      func() error { _, err := local.Execute(t.Context(), raw); return err },
		"hosted-service":       func() error { _, err := hosted.Execute(t.Context(), scoped); return err },
		"Wails-legacy-execute": func() error { _, err := desktop.TargetCommissionerExecute(string(body)); return err },
	}
	switch {
	case domain.OwnerBidAction(raw.Command) != "":
		out["Wails-owner-bid"] = func() error { _, err := desktop.TargetNativeBidAction(raw); return err }
	case domain.OwnerWaiverAction(raw.Command) != "":
		out["Wails-owner-waiver"] = func() error { _, err := desktop.TargetNativeWaiverAction(raw); return err }
	case domain.OwnerTradeAction(raw.Command) != "":
		out["Wails-owner-trade"] = func() error { _, err := desktop.TargetNativeTradeAction(raw); return err }
	}
	return out
}
