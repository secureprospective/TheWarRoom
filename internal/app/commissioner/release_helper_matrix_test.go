package commissioner

import (
	"net/http"
	"reflect"
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	domain "github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func releaseHelperMatrix(t *testing.T, f *releaseTokenFixture, h *releaseHTTP) {
	t.Helper()
	before, otherBefore := f.records(t, "native.hosted"), f.records(t, "native.other")
	defer func() {
		if !reflect.DeepEqual(before, f.records(t, "native.hosted")) || !reflect.DeepEqual(otherBefore, f.records(t, "native.other")) {
			t.Error("helper matrix changed league state")
		}
	}()
	host := leagueservice.StoredHost{Store: f.app.history, Context: t.Context, Principal: f.app.session, Registry: f.app.registry}
	revisions := map[string]int64{"native.hosted": domain.RecipientRevision(before, f.app.session, host.CommissionerNow()), "native.other": domain.RecipientRevision(otherBefore, f.app.session, host.CommissionerNow())}
	rows := releaseMutationRows(t)
	for _, command := range NativeCommands() {
		if !slices.Contains(command.Modes, catalog.Execute) {
			continue
		}
		t.Run(command.ID, func(t *testing.T) {
			for _, seat := range []string{"franchise.01", "franchise.03"} {
				for _, mode := range command.Modes {
					scope := accounts.RawTokenScope{League: "native.hosted", Franchise: seat, Commands: []string{command.ID}, Modes: []string{string(mode)}}
					token, err := f.tokens.Issue(t.Context(), f.creator.RawSession, scope)
					ownerAction := domain.OwnerBidAction(command.ID) != "" || domain.OwnerWaiverAction(command.ID) != "" || domain.OwnerTradeAction(command.ID) != ""
					ungrantable := !accounts.DelegableCommand(command.ID) || seat == "franchise.03" && ownerAction
					if ungrantable {
						if err == nil {
							t.Fatalf("%s/%s: ungrantable helper authority issued", seat, mode)
						}
						scope.Commands = []string{domain.NotePost}
						token, err = f.tokens.Issue(t.Context(), f.creator.RawSession, scope)
					}
					if err != nil {
						t.Fatal(err)
					}
					gate := f.tokens.Admission(token.RawSession)
					service := leagueservice.NewHosted(host, gate)
					opposite := "execute"
					if mode == catalog.Execute {
						opposite = "preview"
					}
					for _, league := range []string{"native.hosted", "native.other"} {
						raw := rows[command.ID].raw(t, command.ID, league, f.app.session, revisions[league])
						releaseHelperRefusal(t, h, service, token, raw, opposite, false)
						if league != "native.hosted" || ungrantable {
							releaseHelperRefusal(t, h, service, token, raw, string(mode), false)
						}
					}
					if err := f.tokens.Revoke(t.Context(), f.creator.RawSession, token.ID); err != nil {
						t.Fatal(err)
					}
					raw := rows[command.ID].raw(t, command.ID, "native.hosted", f.app.session, revisions["native.hosted"])
					releaseHelperRefusal(t, h, service, token, raw, string(mode), true)
				}
			}
		})
	}
}

func releaseHelperRefusal(t *testing.T, h *releaseHTTP, service *leagueservice.Service, token accounts.AgentToken, raw domain.RawCommand, mode string, revoked bool) {
	t.Helper()
	var err error
	if mode == "preview" {
		_, err = service.Preview(raw)
	} else {
		switch raw.Command {
		case domain.LineupSet:
			_, err = service.SetLineup(t.Context(), raw)
		case domain.DraftPick:
			_, err = service.Pick(t.Context(), domain.DraftPickInput{League: raw.League, Season: 2027, Franchise: "franchise.01", Player: "5001", Expected: raw.Expected, Request: raw.Request})
		default:
			_, err = service.Execute(t.Context(), raw)
		}
	}
	if err == nil {
		t.Fatalf("service %s revoked=%v admitted helper outside its ceiling", mode, revoked)
	}
	want := http.StatusConflict
	if mode == "preview" {
		want = http.StatusForbidden
	}
	if revoked {
		want = http.StatusUnauthorized
	}
	path := serverhttp.ExecutePath
	if mode == "preview" {
		path = serverhttp.PreviewPath
	}
	for _, route := range []string{path, serverhttp.DispatchPath} {
		request := serverhttp.RawNativeRequest{Protocol: 1, Command: &raw}
		if route == serverhttp.DispatchPath {
			request.Mode = mode
		}
		status, body := h.post(t, route, token.Bearer, "", request)
		if status != want {
			t.Fatalf("%s %s revoked=%v: status=%d want=%d body=%s", raw.Command, route, revoked, status, want, body)
		}
	}
}
