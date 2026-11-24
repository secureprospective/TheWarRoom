package catalog

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func TestExecutableRegistryRejectsBrokenDeclarations(t *testing.T) {
	handler := func(context.Context, json.RawMessage) (json.RawMessage, error) { return json.RawMessage(`null`), nil }
	valid := Entry{ID: "test.read", Label: "Read the test", Definition: completeCommand(), Native: map[Mode]Handler{Read: handler}}
	tests := map[string]func(*Registry){
		"unqualified MFL":               func(r *Registry) { r.entries[0].MFL = r.entries[0].Native; r.entries[0].Native = nil },
		"shared native MFL":             func(r *Registry) { r.entries[0].ID = "mfl.test"; r.entries[0].MFL = r.entries[0].Native },
		"duplicate ID":                  func(r *Registry) { r.entries = append(r.entries, r.entries[0]) },
		"missing native handler":        func(r *Registry) { r.entries[0].Native[Read] = nil },
		"unsupported mode":              func(r *Registry) { r.entries[0].Native[Execute] = handler },
		"missing desktop handler":       func(r *Registry) { r.entries[0].Desktop = []DesktopExposure{{}} },
		"missing workflow handler":      func(r *Registry) { r.entries[0].Workflow = &WorkflowCommand{} },
		"empty label":                   func(r *Registry) { r.entries[0].Label = "" },
		"unsupported CLI":               func(r *Registry) { r.entries[0].CLI = true },
		"missing browser argument type": func(r *Registry) { r.entries[0].Browser = &BrowserCommand{ID: "test.read"} },
		"no door":                       func(r *Registry) { r.entries[0].Native = nil },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			entry := valid
			entry.Native = map[Mode]Handler{Read: handler}
			registry := &Registry{entries: []Entry{entry}}
			if err := registry.Validate(); err != nil {
				t.Fatal(err)
			}
			mutate(registry)
			if err := registry.Validate(); err == nil {
				t.Fatal("broken declaration accepted")
			}
		})
	}
}

func TestLeagueActivityIsAHostedLeagueViewRead(t *testing.T) {
	var found bool
	for _, command := range NativeCommands(Schema{}) {
		if command.ID != "league.activity.read" {
			continue
		}
		found = true
		if command.Method != "TargetLeagueActivity" || command.Availability != "server" ||
			command.Permission != "league-view" || len(command.Modes) != 1 || command.Modes[0] != Read {
			t.Fatalf("activity catalog entry = %+v", command)
		}
	}
	if !found {
		t.Fatal("member-safe activity read missing from catalog")
	}
	if _, ok := WorkflowCommands()["league.activity.read"]; !ok {
		t.Fatal("activity read missing from hosted workflow")
	}
	if _, ok := WorkflowCommands()["commish.ledger.filtered.read"]; !ok {
		t.Fatal("commissioner ledger filter missing from hosted workflow")
	}
}

func TestNFLPlayerStatusIsAHostedLeagueViewRead(t *testing.T) {
	var found bool
	for _, command := range NativeCommands(Schema{}) {
		if command.ID != "nfl.player-status.read" {
			continue
		}
		found = true
		if command.Method != "TargetNFLPlayerStatus" || command.Availability != "server" ||
			command.Permission != "league-view" || len(command.Modes) != 1 || command.Modes[0] != Read {
			t.Fatalf("NFL status catalog entry = %+v", command)
		}
	}
	if !found {
		t.Fatal("member-safe NFL status read missing from catalog")
	}
	if _, ok := WorkflowCommands()["nfl.player-status.read"]; !ok {
		t.Fatal("NFL status read missing from hosted workflow")
	}
}

func TestESPNNewsReadsAreHostedLeagueViewReads(t *testing.T) {
	want := map[string]string{
		"player.news.read": "TargetPlayerNews",
		"news.feed.read":   "TargetNewsFeed",
	}
	seen := map[string]bool{}
	for _, command := range NativeCommands(Schema{}) {
		method, ok := want[command.ID]
		if !ok {
			continue
		}
		if command.Method != method || command.Availability != "server" ||
			command.Permission != "league-view" || len(command.Modes) != 1 || command.Modes[0] != Read {
			t.Fatalf("%s catalog entry = %+v", command.ID, command)
		}
		seen[command.ID] = true
	}
	for id := range want {
		if !seen[id] {
			t.Errorf("%s missing from catalog", id)
		}
		if _, ok := WorkflowCommands()[id]; !ok {
			t.Errorf("%s missing from hosted workflow", id)
		}
	}
}

func TestBidWithdrawalIsNotPublishedThroughCommandDoors(t *testing.T) {
	for _, command := range NativeCommands(Schema{}) {
		if command.ID == "bid.withdraw" {
			t.Fatal("bid withdrawal is still published in the native command catalog")
		}
	}
	if _, ok := WorkflowCommands()["bid.withdraw"]; ok {
		t.Fatal("bid withdrawal is still exposed as a workflow command")
	}
}

func TestClaimMaintenanceCommandsUseTheOwnerWorkflow(t *testing.T) {
	want := map[string]bool{"waivers.claim.order.set": false, "waivers.claim.cancel": false}
	for _, command := range NativeCommands(Schema{}) {
		if _, expected := want[command.ID]; !expected {
			continue
		}
		if command.Method != "TargetNativeWaiverAction" || command.Availability != "both" ||
			!slices.Equal(command.Modes, []Mode{Preview, Execute}) {
			t.Errorf("%s catalog entry = %+v", command.ID, command)
		}
		want[command.ID] = true
	}
	workflows := WorkflowCommands()
	for id, found := range want {
		if !found {
			t.Errorf("%s missing from the native catalog", id)
		}
		if _, found := workflows[id]; !found {
			t.Errorf("%s missing from hosted workflows", id)
		}
	}
}

func TestCommandLabelsDoNotExposeMachineSyntax(t *testing.T) {
	for id, want := range map[string]string{"mfl.bid.revise": "revise local MFL bid", "commish.rules.set": "set league rules",
		"waivers.claim.order.set": "Set your waiver claim order", "waivers.claim.cancel": "Cancel your waiver claim",
		"TargetMFLKeyStatus": "MFL Key Status", "GetParams": "Get Params"} {
		if got := commandLabel(id); got != want || strings.Contains(got, ".") {
			t.Errorf("%s label=%q, want %q", id, got, want)
		}
	}
}

func TestOperatorRegistryKeepsOfflineAuthorityConsoleOnly(t *testing.T) {
	registry, err := NewOperatorRegistry()
	if err != nil {
		t.Fatal(err)
	}
	for _, door := range []Door{NativeDoor, MFLDoor, WailsDoor, WorkflowDoor, BrowserDoor, "unknown"} {
		if _, err := registry.Invoke(t.Context(), door, "league.stand_up", Execute, json.RawMessage(`{}`)); err == nil || !strings.Contains(err.Error(), "not available") {
			t.Fatalf("offline command leaked into %s: %v", door, err)
		}
	}
	for _, mode := range []Mode{Read, Preview, "unknown"} {
		if _, err := registry.Invoke(t.Context(), ConsoleDoor, "league.stand_up", mode, json.RawMessage(`{}`)); err == nil || !strings.Contains(err.Error(), "not available") {
			t.Fatalf("offline command accepted %s: %v", mode, err)
		}
	}
	if _, err := registry.Invoke(t.Context(), ConsoleDoor, "league.stand_up", Execute, json.RawMessage(`{"unexpected":true}`)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("offline command bypassed Raw decoding: %v", err)
	}
	registry.entries[0].Console[Execute] = nil
	if err := registry.Validate(); err == nil || !strings.Contains(err.Error(), "missing execute handler") {
		t.Fatalf("offline entry accepted a missing handler: %v", err)
	}
}

func TestDeputyPermissionPolicyCoversNativeMutationsAndHostedWorkflows(t *testing.T) {
	catalog := make(map[string]int)
	workflows := make(map[string]bool)
	for _, entry := range serverWorkflowEntries() {
		workflows[entry.ID] = true
	}
	for _, command := range nativeMutations() {
		catalog[command.id]++
	}
	policy := commissioner.DeputyCommandRules()
	seen := make(map[string]int, len(policy))
	for _, rule := range policy {
		seen[rule.Command]++
		switch rule.Class {
		case commissioner.DeputyDelegable:
			if !governance.ValidDeputyPermission(rule.Permission) {
				t.Errorf("delegated command %s has no valid permission", rule.Command)
			}
		case commissioner.DeputyCommissionerOnly, commissioner.DeputyMemberAction:
			if rule.Permission != "" {
				t.Errorf("non-delegated command %s has permission %q", rule.Command, rule.Permission)
			}
		default:
			t.Errorf("command %s has unknown deputy policy class %q", rule.Command, rule.Class)
		}
		count := catalog[rule.Command]
		if workflows[rule.Command] {
			count++
		}
		if count != 1 {
			t.Errorf("policy command %s appears %d times in native/workflow commands", rule.Command, count)
		}
	}
	for command, count := range catalog {
		if count != 1 {
			t.Errorf("catalog command %s appears %d times", command, count)
		}
		if seen[command] != 1 {
			t.Errorf("native/workflow command %s has %d permission classifications", command, seen[command])
		}
	}
}

func TestOLineCloseRegistryMetadata(t *testing.T) {
	commands := NativeCommands(Schema{})
	for _, c := range commands {
		if c.ID != "oline.auction.close" {
			continue
		}
		if c.Role != "commish" || c.Undo != "none" || c.Method != "TargetCommissionerExecute" || commandLabel(c.ID) != "Close O-Line auction" {
			t.Fatalf("close metadata: %+v", c)
		}
		for _, field := range c.Arguments.Fields {
			if field.Name == "args" && len(field.Schema.Fields) != 0 {
				t.Fatal("close advertises app-owned outcomes")
			}
		}
		return
	}
	t.Fatal("close command missing")
}

func TestPublishedEntriesCoversNativeServerCLIAndBrowserWithoutDesktop(t *testing.T) {
	entries, err := PublishedEntries()
	if err != nil {
		t.Fatal(err)
	}
	var nativeCount, serverCount, cliCount, browserCount int
	var sawSeed, sawBackup, sawQueue bool
	for _, entry := range entries {
		if strings.HasPrefix(entry.ID, "mfl.") || strings.HasPrefix(entry.ID, "desktop.") {
			t.Fatalf("desktop-only command %s leaked into catalog entries", entry.ID)
		}
		if entry.Native {
			nativeCount++
		}
		if entry.Workflow && entry.Definition.Availability == "server" {
			serverCount++
		}
		if entry.CLI {
			cliCount++
		}
		if entry.Browser != nil {
			browserCount++
		}
		switch entry.ID {
		case commissioner.NativeSeed:
			sawSeed = entry.Native && entry.Definition.Arguments.Validator == "RawNativeSeedRequest.Validate/ValidatePreview"
		case "backup.now":
			sawBackup = entry.Workflow && entry.Browser != nil && entry.Definition.Availability == "server"
		case "draft.queue.read":
			sawQueue = entry.Workflow && !entry.Native && entry.Definition.Availability == "server"
		}
	}
	if nativeCount == 0 || serverCount == 0 || cliCount == 0 || browserCount == 0 {
		t.Fatalf("empty catalog surface: native=%d server=%d cli=%d browser=%d", nativeCount, serverCount, cliCount, browserCount)
	}
	if !sawSeed || !sawBackup || !sawQueue {
		t.Fatalf("missing key entries: seed=%t backup=%t queue=%t", sawSeed, sawBackup, sawQueue)
	}
}
