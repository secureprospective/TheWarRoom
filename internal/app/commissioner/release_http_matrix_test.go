package commissioner

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	domain "github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/tools/hostedfixture"
)

type releaseHTTP struct {
	server   *httptest.Server
	proofs   map[string]string
	football string
}

type releaseClock struct{}

func (releaseClock) Now() time.Time { return time.Now().UTC() }
func (releaseClock) Trusted() error { return nil }

func newReleaseHTTP(t *testing.T, f *releaseTokenFixture, actors []releaseActor) *releaseHTTP {
	t.Helper()
	passwords := accounts.New(f.store)
	totp, err := accounts.NewTOTP(passwords, f.store, bytes.Repeat([]byte{19}, 32))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(30 * time.Second)
	f.store.SetClock(func() time.Time { return now })
	proofs := map[string]string{}
	for _, actor := range actors {
		if actor.name != "commissioner" && actor.name != "deputy" && actor.name != "second-deputy" {
			continue
		}
		credentials := accounts.RawCredentials{Username: actor.name, Password: "release-matrix-password"}
		setup, err := totp.Begin(t.Context(), credentials)
		if err != nil {
			t.Fatal(err)
		}
		code, err := hostedfixture.Code(setup.Seed, now)
		if err != nil {
			t.Fatal(err)
		}
		if err := totp.Confirm(t.Context(), credentials, accounts.RawTOTP{Code: code}); err != nil {
			t.Fatal(err)
		}
		now = now.Add(30 * time.Second)
		code, err = hostedfixture.Code(setup.Seed, now)
		if err != nil {
			t.Fatal(err)
		}
		proof, err := totp.StepUp(t.Context(), credentials, accounts.RawTOTP{Code: code})
		if err != nil {
			t.Fatal(err)
		}
		proofs[actor.name] = proof.Token
	}
	confirmations, err := serverhttp.NewConfirmations()
	if err != nil {
		t.Fatal(err)
	}
	state, err := domain.GovernanceState(f.records(t, "native.hosted"))
	if err != nil {
		t.Fatal(err)
	}
	enrollment := accounts.NewEnrollment(f.store, releaseEnrollment{state: state, principal: governance.Principal(f.app.session)})
	handler := serverhttp.Hosted{Store: f.app.history, Sessions: f.sessions, Tokens: f.tokens, TOTP: totp, Confirmations: confirmations, Registry: f.app.registry, Enrollment: enrollment}
	if f.football != "" {
		handler.Football, err = f.app.history.LoadFootball(t.Context(), f.football)
		if err != nil {
			t.Fatal(err)
		}
		handler.FootballDigest = f.football
		f.app.history.SetPinnedFootball(handler.Football)
		handler.Worker = leagueservice.NewDueWorker(f.app.history, handler.Football.PlayerFacts, releaseClock{})
	} else {
		handler.Worker = leagueservice.NewDueWorker(f.app.history, nil, releaseClock{})
	}
	server := httptest.NewTLSServer(handler.Handler())
	t.Cleanup(server.Close)
	return &releaseHTTP{server, proofs, f.football}
}

func (h *releaseHTTP) post(t *testing.T, path, bearer, proof string, payload serverhttp.RawNativeRequest) (int, []byte) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, h.server.URL+path, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+bearer)
	request.Header.Set("X-TWR-Recent-Auth", proof)
	response, err := h.server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	result, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("cacheable authorization response")
	}
	if strings.Contains(string(result), bearer) || proof != "" && strings.Contains(string(result), proof) {
		t.Fatal("credential leaked in authorization response")
	}
	return response.StatusCode, result
}

func (h *releaseHTTP) humanCell(t *testing.T, actor releaseActor, raw domain.RawCommand, admitted, denied bool) {
	t.Helper()
	member := raw.League == "native.hosted" && actor.name != "outsider"
	request := serverhttp.RawNativeRequest{Protocol: 1, Command: &raw}
	status, body := h.post(t, serverhttp.PreviewPath, actor.session.Bearer, h.proofs[actor.name], request)
	want := http.StatusOK
	pickSeat := raw.Command == domain.DraftPick && raw.League == "native.hosted" && actor.name == "owner"
	if !admitted && !pickSeat {
		want = http.StatusForbidden
	}
	if status != want {
		t.Fatalf("HTTP preview: status=%d want=%d body=%s", status, want, body)
	}
	if status == http.StatusForbidden && string(body) != "native preview refused\n" {
		t.Fatalf("permission denial disclosed a different transcript: %s", body)
	}
	if denied && status == http.StatusOK {
		var preview domain.Preview
		if err := json.Unmarshal(body, &preview); err != nil {
			t.Fatal(err)
		}
		if len(preview.Blockers) == 0 {
			t.Fatal("HTTP preview admitted denied role")
		}
	}
	if !denied {
		return
	}
	for _, path := range []string{serverhttp.ExecutePath, serverhttp.DispatchPath} {
		request.Mode = ""
		if path == serverhttp.DispatchPath {
			request.Mode = "execute"
		}
		status, body = h.post(t, path, actor.session.Bearer, h.proofs[actor.name], request)
		if status != http.StatusConflict || string(body) != "native change refused; reread before retrying\n" {
			t.Fatalf("%s: denial transcript differs: status=%d body=%s", path, status, body)
		}
	}
	if raw.Command == domain.LineupSet {
		request.Mode = ""
		status, body = h.post(t, serverhttp.LineupPath, actor.session.Bearer, h.proofs[actor.name], request)
		if status != http.StatusConflict {
			t.Fatalf("HTTP owner lineup admitted: %d %s", status, body)
		}
	}
	args, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	workflow := serverhttp.RawWorkflow{Command: raw.Command, Version: "1", Mode: "execute", Arguments: args, Football: h.football}
	if raw.Command == domain.DraftPick {
		input := domain.DraftPickInput{League: raw.League, Season: 2027, Franchise: "franchise.02", Player: "5001", Expected: raw.Expected, Request: raw.Request}
		workflow.Arguments, err = json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		status, body = h.post(t, serverhttp.PickPath, actor.session.Bearer, h.proofs[actor.name], serverhttp.RawNativeRequest{Protocol: 1, Pick: &input})
		if status != http.StatusConflict {
			t.Fatalf("HTTP owner draft admitted: %d %s", status, body)
		}
	}
	if workflow.Validate() != nil {
		return
	}
	want = http.StatusConflict
	if !member {
		want = http.StatusForbidden
	}
	status, body = h.post(t, serverhttp.WorkflowPath, actor.session.Bearer, h.proofs[actor.name], serverhttp.RawNativeRequest{Protocol: 1, Workflow: &workflow})
	if status != want {
		t.Fatalf("HTTP workflow: status=%d want=%d body=%s", status, want, body)
	}
}
