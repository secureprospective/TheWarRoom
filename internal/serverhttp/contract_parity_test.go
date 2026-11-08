package serverhttp_test

import (
	"crypto/x509"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverclient"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

// The client/server step 1 contract gate: a command a locked screen sends through the hosted
// workflow door and the same command sent by the CLI/MCP agent client reach the same service with
// the same outcome, the same refusals, and no state change on refusal.
// Map: docs/audit/CLIENT-SERVER-CONTRACT-2026-11-08.md.

type parityStep struct {
	name  string
	as    string // "commissioner" holds franchise.01; "owner" holds franchise.02
	edit  func(*commissioner.RawCommand)
	mode  string
	allow bool
}

type parityOutcome struct {
	allowed bool
	changed bool
	state   string
}

// parityEntry sends one command through one door and reports whether it was accepted.
// scope is the command the person granted; the door may be asked to send a different one.
type parityEntry func(t *testing.T, f *hostedFixture, as, scope string, raw commissioner.RawCommand, mode string) bool

// paritySessionCommissioner reuses the fixture's one commissioner sign-in; TOTP enrolls only once.
func paritySessionCommissioner(t *testing.T, f *hostedFixture) (bearer, proof string) {
	t.Helper()
	if f.commissioner[0] == "" {
		f.commissioner[0], f.commissioner[1] = f.directCommissioner(t)
	}
	return f.commissioner[0], f.commissioner[1]
}

func parityAssets(t *testing.T, f *hostedFixture) (commissioner.NativeAssets, string) {
	t.Helper()
	bearer, proof := paritySessionCommissioner(t, f)
	raw := workflowRequest(t, "native.assets.read", "read", `{"league":"native.hosted"}`, "")
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		t.Fatal(err)
	}
	var response serverhttp.WorkflowResponse
	if err := json.Unmarshal(f.post(t, serverhttp.WorkflowPath, string(body), bearer, proof, 200), &response); err != nil {
		t.Fatal(err)
	}
	var assets commissioner.NativeAssets
	if err := json.Unmarshal(response.Data, &assets); err != nil {
		t.Fatal(err)
	}
	return assets, response.Football
}

func paritySession(t *testing.T, f *hostedFixture, as string) (bearer, proof string) {
	t.Helper()
	if as == "commissioner" {
		return paritySessionCommissioner(t, f)
	}
	session, err := f.sessions.Login(t.Context(), accounts.RawCredentials{Username: as, Password: "private-password"}, "test device")
	if err != nil {
		t.Fatal(err)
	}
	return session.Bearer, ""
}

// screenDoor is the hosted desktop/browser route: serverhttp WorkflowPath → catalog.RunWorkflow.
func screenDoor(t *testing.T, f *hostedFixture, as, _ string, raw commissioner.RawCommand, mode string) bool {
	t.Helper()
	_, pin := parityAssets(t, f)
	bearer, proof := paritySession(t, f, as)
	args, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	request := workflowRequest(t, raw.Command, mode, string(args), pin)
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &request})
	if err != nil {
		t.Fatal(err)
	}
	status, _ := f.send(t, serverhttp.WorkflowPath, string(body), bearer, proof)
	return status == 200
}

// cliDoor is cmd/twr's route: serverclient.Agent.Dispatch → serverhttp WorkflowPath (catalog), with the
// agent token the person issued for their own franchise.
func cliDoor(t *testing.T, f *hostedFixture, as, scope string, raw commissioner.RawCommand, mode string) bool {
	t.Helper()
	return agentDoor(t, f, as, scope, raw, mode, true)
}

// cliUnconfirmedDoor is the CLI with no human confirmation code: G2/G3 executes must be refused.
func cliUnconfirmedDoor(t *testing.T, f *hostedFixture, as, scope string, raw commissioner.RawCommand, mode string) bool {
	t.Helper()
	return agentDoor(t, f, as, scope, raw, mode, false)
}

// parityConfirm is the commissioner pasting the CLI's confirm request into The War Room.
func parityConfirm(t *testing.T, f *hostedFixture, tokenID string, raw commissioner.RawCommand) string {
	t.Helper()
	bearer, proof := paritySessionCommissioner(t, f)
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Command: &raw, TokenID: tokenID})
	if err != nil {
		t.Fatal(err)
	}
	status, result := f.send(t, serverhttp.ConfirmPath, string(body), bearer, proof)
	if status != 200 {
		return "" // an unconfirmable change (unknown command, stale revision) goes unconfirmed
	}
	var consent serverhttp.ConfirmationResponse
	if err := json.Unmarshal(result, &consent); err != nil {
		t.Fatal(err)
	}
	return consent.Confirmation
}

func agentDoor(t *testing.T, f *hostedFixture, as, scope string, raw commissioner.RawCommand, mode string, confirm bool) bool {
	t.Helper()
	bearer, _ := paritySession(t, f, as)
	franchise := "franchise.01"
	if as != "commissioner" {
		franchise = "franchise.02"
	}
	// A desktop-issued helper token carries every delegable read (platform.DelegableCommands); the
	// workflow's league location and the assets read that names the football evidence are among them.
	token, err := f.tokens.Issue(t.Context(), accounts.RawSession{Bearer: bearer}, accounts.RawTokenScope{League: "native.hosted", Franchise: franchise,
		Commands: []string{scope, "commish.ledger.read", "native.directory.read", "native.assets.read"}, Modes: []string{"read", "preview", "execute"}})
	if err != nil && as != "commissioner" {
		// Agent tokens act for the current commissioner only (TokenScopePermission): an owner's
		// CLI is refused at issuance, before any command reaches the league.
		return false
	}
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(f.server.Certificate())
	agent, err := serverclient.NewAgent(f.server.URL, "native.hosted", token.Bearer, roots)
	if err != nil {
		t.Fatal(err)
	}
	confirmation := ""
	if mode == "execute" && confirm {
		confirmation = parityConfirm(t, f, token.ID, raw)
	}
	_, err = agent.Dispatch(t.Context(), raw, mode, confirmation)
	if err != nil {
		t.Logf("cli %s %s: %v", raw.Command, mode, err)
	}
	return err == nil
}

func runParity(t *testing.T, door parityEntry, base commissioner.RawCommand, steps []parityStep) []parityOutcome {
	t.Helper()
	f := newHostedFixture(t, true)
	if strings.HasPrefix(base.Command, "draft.") {
		time.Sleep(2 * time.Second) // the practice draft starts two seconds after provisioning
	}
	var out []parityOutcome
	for _, step := range steps {
		before, _ := parityAssets(t, f)
		raw := base
		raw.Expected = before.Revision
		raw.Request = base.Request + "." + step.name
		if step.edit != nil {
			step.edit(&raw)
		}
		allowed := door(t, f, step.as, base.Command, raw, step.mode)
		after, _ := parityAssets(t, f)
		state := rosterState(after)
		if !allowed && (after.Revision != before.Revision || state != rosterState(before)) {
			t.Fatalf("%s: refusal changed league state", step.name)
		}
		if step.mode == "preview" && after.Revision != before.Revision {
			t.Fatalf("%s: preview changed league state", step.name)
		}
		out = append(out, parityOutcome{allowed: allowed, changed: after.Revision != before.Revision, state: state})
	}
	return out
}

// rosterState is who holds whom and in which bucket; revisions and ids differ between fixtures.
func rosterState(assets commissioner.NativeAssets) string {
	out := ""
	for _, p := range assets.Snapshot.Players {
		out += p.ID + "/" + p.Franchise + "/" + string(p.Status) + ";"
	}
	return out
}

func draftCommand(command string, act commissioner.RawDraftAct) commissioner.RawCommand {
	return commissioner.RawCommand{League: "native.hosted", Command: command, Request: command, Reason: "Run the draft from the tray",
		Args: commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{Season: 2027, Act: &act}}, Effective: "now", Visibility: "league"}
}

func withDraftAct(edit func(*commissioner.RawDraftAct)) func(*commissioner.RawCommand) {
	return func(r *commissioner.RawCommand) {
		d, a := *r.Args.Draft, *r.Args.Draft.Act
		edit(&a)
		d.Act = &a
		r.Args.Draft = &d
	}
}

func TestContractParityScreenAndCLI(t *testing.T) {
	stale := func(r *commissioner.RawCommand) { r.Expected += 5 }
	unknown := func(r *commissioner.RawCommand) { r.Command = "roster.teleport" }
	cases := []struct {
		name       string
		base       commissioner.RawCommand
		ineligible func(*commissioner.RawCommand)
	}{
		{name: commissioner.RosterReserve,
			base: commissioner.RawCommand{League: "native.hosted", Command: commissioner.RosterReserve, Request: "reserve",
				Args: commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0100", Franchise: "franchise.01", Bucket: "ir"}}, Effective: "now", Visibility: "league"},
			ineligible: func(r *commissioner.RawCommand) {
				w := *r.Args.Waiver
				w.Player = "0099"
				r.Args.Waiver = &w
			}},
		{name: commissioner.LineupSet,
			base: commissioner.RawCommand{League: "native.hosted", Command: commissioner.LineupSet, Request: "lineup",
				Args: commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{Season: 2026, Franchise: "franchise.01", Week: 1, Starters: []string{"0099"}}}, Effective: "now", Visibility: "private"},
			ineligible: func(r *commissioner.RawCommand) {
				l := *r.Args.Lineup
				l.Starters = []string{"9999"}
				r.Args.Lineup = &l
			}},
		{name: commissioner.TradeOffer,
			base: commissioner.RawCommand{League: "native.hosted", Command: commissioner.TradeOffer, Request: "offer", Reason: "Parity", Effective: "now", Visibility: "league",
				Args: commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{Franchise: "franchise.01", Parties: []string{"franchise.01", "franchise.02"},
					Assets: []commissioner.TradeAsset{{From: "franchise.01", To: "franchise.02", Player: "0101"}}}}},
			ineligible: func(r *commissioner.RawCommand) {
				tr := *r.Args.Trade
				tr.Assets = []commissioner.TradeAsset{{From: "franchise.01", To: "franchise.02", Player: "9999"}}
				r.Args.Trade = &tr
			}},
		{name: commissioner.DraftTimer, base: draftCommand(commissioner.DraftTimer, commissioner.RawDraftAct{Timer: commissioner.TimerPause}),
			ineligible: withDraftAct(func(a *commissioner.RawDraftAct) { a.Timer = commissioner.TimerResume })},
		{name: commissioner.DraftSkip, base: draftCommand(commissioner.DraftSkip, commissioner.RawDraftAct{Round: 1, Original: "franchise.01"}),
			ineligible: withDraftAct(func(a *commissioner.RawDraftAct) { a.Round = 9 })},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			steps := []parityStep{
				{name: "preview", as: "commissioner", mode: "preview", allow: true},
				{name: "stale", as: "commissioner", mode: "execute", edit: stale},
				{name: "unknown", as: "commissioner", mode: "execute", edit: unknown},
				{name: "ineligible", as: "commissioner", mode: "execute", edit: tc.ineligible},
				{name: "wrong-role", as: "owner", mode: "execute"},
				{name: "execute", as: "commissioner", mode: "execute", allow: true},
			}
			screen := runParity(t, screenDoor, tc.base, steps)
			cli := runParity(t, cliDoor, tc.base, steps)
			for i, step := range steps {
				if screen[i].allowed != step.allow || cli[i].allowed != step.allow {
					t.Errorf("%s: allowed screen=%v cli=%v want=%v", step.name, screen[i].allowed, cli[i].allowed, step.allow)
				}
				if screen[i].changed != cli[i].changed || screen[i].state != cli[i].state {
					t.Errorf("%s: outcome differs: screen changed=%v cli changed=%v", step.name, screen[i].changed, cli[i].changed)
				}
			}
			// Missing confirmation: a G2/G3 execute from the CLI without the human's code is refused and
			// leaves the league unchanged (runParity checks unchanged state on every refusal).
			definition, err := tc.base.EffectiveDefinition()
			if err != nil {
				t.Fatal(err)
			}
			unconfirmed := runParity(t, cliUnconfirmedDoor, tc.base, steps[len(steps)-1:])
			if needs := definition.Gravity == "G2" || definition.Gravity == "G3"; needs && (unconfirmed[0].allowed || unconfirmed[0].changed) {
				t.Errorf("unconfirmed CLI execute accepted: %+v", unconfirmed[0])
			}
		})
	}
}

// TestContractParityDraftPick: a pick is the on-clock franchise's DraftPickInput. The screen records
// it; the CLI can only send a RawCommand, which the catalog's pick entry refuses, leaving the draft as it was.
func TestContractParityDraftPick(t *testing.T) {
	raw := draftCommand(commissioner.DraftPick, commissioner.RawDraftAct{Round: 1, Original: "franchise.01", Player: "5001"})
	cli := runParity(t, cliDoor, raw, []parityStep{{name: "pick", as: "commissioner", mode: "execute"}})
	if cli[0].allowed || cli[0].changed {
		t.Fatalf("CLI pick: %+v", cli[0])
	}
}

// TestContractDispatchRefusesCatalogCommands: one path per thing. A command the catalog serves is
// refused on the generic dispatch door, so it can reach the league only through the workflow.
func TestContractDispatchRefusesCatalogCommands(t *testing.T) {
	f := newHostedFixture(t, true)
	bearer, _ := paritySessionCommissioner(t, f)
	token, err := f.tokens.Issue(t.Context(), accounts.RawSession{Bearer: bearer}, accounts.RawTokenScope{League: "native.hosted", Franchise: "franchise.01",
		Commands: []string{commissioner.TradeOffer}, Modes: []string{"preview"}})
	if err != nil {
		t.Fatal(err)
	}
	before, _ := parityAssets(t, f)
	raw := commissioner.RawCommand{League: "native.hosted", Command: commissioner.TradeOffer, Request: "dispatch", Reason: "Parity", Effective: "now", Visibility: "league", Expected: before.Revision,
		Args: commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{Franchise: "franchise.01", Parties: []string{"franchise.01", "franchise.02"},
			Assets: []commissioner.TradeAsset{{From: "franchise.01", To: "franchise.02", Player: "0101"}}}}}
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Command: &raw, Mode: "preview"})
	if err != nil {
		t.Fatal(err)
	}
	// An older twr still sends catalog commands here; the refusal tells it to update (ruling 4, 11-08).
	if status, reply := f.send(t, serverhttp.DispatchPath, string(body), token.Bearer, ""); status != 400 || !strings.Contains(string(reply), "update twr") {
		t.Fatalf("catalog command on dispatch: status %d %s", status, reply)
	}
}

// TestContractCLIRefusalWording: a token missing a catalog read names that read (ruling 2, 11-08),
// and a refused preview reads as a preview refusal, not an unrecorded change (ruling 3).
func TestContractCLIRefusalWording(t *testing.T) {
	f := newHostedFixture(t, true)
	bearer, _ := paritySessionCommissioner(t, f)
	before, _ := parityAssets(t, f)
	raw := commissioner.RawCommand{League: "native.hosted", Command: commissioner.TradeOffer, Request: "offer", Reason: "Parity", Effective: "now", Visibility: "league", Expected: before.Revision,
		Args: commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{Franchise: "franchise.03", Parties: []string{"franchise.03", "franchise.02"},
			Assets: []commissioner.TradeAsset{{From: "franchise.03", To: "franchise.02", Player: "0101"}}}}} // outside the token's franchise
	roots := x509.NewCertPool()
	roots.AddCert(f.server.Certificate())
	for _, tc := range []struct {
		reads []string
		want  error
		text  string
	}{
		{reads: []string{"native.assets.read"}, text: "does not cover the league read native.directory.read"},
		{reads: []string{"native.directory.read"}, text: "does not cover the league read native.assets.read"},
		{reads: []string{"native.directory.read", "native.assets.read"}, want: serverclient.ErrPreviewRefused},
	} {
		token, err := f.tokens.Issue(t.Context(), accounts.RawSession{Bearer: bearer}, accounts.RawTokenScope{League: "native.hosted", Franchise: "franchise.01",
			Commands: append([]string{commissioner.TradeOffer}, tc.reads...), Modes: []string{"read", "preview"}})
		if err != nil {
			t.Fatal(err)
		}
		agent, err := serverclient.NewAgent(f.server.URL, "native.hosted", token.Bearer, roots)
		if err != nil {
			t.Fatal(err)
		}
		_, err = agent.Dispatch(t.Context(), raw, "preview", "")
		if err == nil || (tc.want != nil && !errors.Is(err, tc.want)) || !strings.Contains(err.Error(), tc.text) {
			t.Errorf("reads %v: %v", tc.reads, err)
		}
	}
}
