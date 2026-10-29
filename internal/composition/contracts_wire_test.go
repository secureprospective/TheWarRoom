package composition

import (
	"bytes"
	"encoding/json"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/mflref/contracts"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// contractsWire encodes a binding result for checkGolden with the parts that differ on every run
// replaced: the random plan ids, the what-if move ids (a clock reading), and instants and the dates
// written from them.
func contractsWire(t *testing.T, v any) json.RawMessage {
	t.Helper()
	body, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range [][2]string{
		{`(contract|release|lineup)-[0-9a-f]{32}`, "${1}-ID"},
		{`\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(\.\d+)?(Z|[+-]\d\d:\d\d)`, "INSTANT"},
		{`"\d{18,20}"`, `"MOVE"`},
		{`(as of|time:) [^".]*?\d{4}[^".]*?(\.|")`, "$1 STAMP$2"},
	} {
		body = regexp.MustCompile(r[0]).ReplaceAll(body, []byte(r[1]))
	}
	return body
}

// contractsWireApp is the release fixture (franchise 0001 holds 0042–0046 at $10M) with a fixed plan
// clock that ticks, named players and the given league phase.
func contractsWireApp(t *testing.T, phase domain.Phase) *App {
	t.Helper()
	a := releaseTestApp(t)
	// Each reading of the plan clock is a second later, so plans list in the order they were made.
	var ticks atomic.Int64
	start := time.Date(2026, 10, 7, 15, 0, 0, 0, time.UTC)
	a.core.MovesNow = func() time.Time { return start.Add(time.Duration(ticks.Add(1)) * time.Second) }
	players := []snapshot.Player{}
	for i, id := range []string{"0042", "0043", "0044", "0045", "0046"} {
		pid, err := playerid.New(id)
		if err != nil {
			t.Fatal(err)
		}
		players = append(players, snapshot.Player{ID: pid, Position: []domain.Position{domain.PosQB, domain.PosWR, domain.PosRB, domain.PosTE, domain.PosLB}[i], Name: "Player " + id})
	}
	a.core.TargetSnapshot.Players.Value = players
	releasePhase(t, a, phase)
	return a
}

// TestContractsWireMatchesGolden pins the contracts and roster-action bindings over the release fixture:
// the contract table and one player's contract, the what-if read, cut and buyout, the release plans
// drafted from them, and a contract Act's preview, options, draft and hand-off.
func TestContractsWireMatchesGolden(t *testing.T) {
	t.Parallel()
	a := contractsWireApp(t, domain.PhaseOffseason)
	got := map[string]any{}
	got["contracts"] = a.contracts().TargetContracts(snapshot.RawContractScope{Mine: "0001", Sort: "name"})
	got["player-contract"] = a.contracts().TargetPlayerContract("0042")
	got["whatif-read"] = a.contracts().TargetWhatIf(contractsapp.RawWhatIfRequest{Mine: "0001", Action: "read"})
	cut := whatIfCall(t, a, "cut", "0046", "")
	got["whatif-cut"] = cut
	buyout := whatIfCall(t, a, "buyout", "0042", "")
	got["whatif-buyout"] = buyout
	moves := buyout.Value.Moves
	a.contracts().TargetReleasePlans(contractsapp.RawReleaseRequest{Mine: "0001", Action: "draft", Move: moves[0].ID})
	got["release-plans"] = a.contracts().TargetReleasePlans(contractsapp.RawReleaseRequest{Mine: "0001", Action: "draft", Move: moves[1].ID})
	act := contractsapp.RawContractAct{Mine: "0001", Player: "0043", Action: "preview", Intent: "contract.extend", Level: 1, Years: 2}
	got["contract-act-preview"] = a.contracts().TargetContractActs(act)
	got["contract-act-options"] = a.contracts().TargetContractActs(contractsapp.RawContractAct{Mine: "0001", Player: "0044", Action: "options"})
	act.Action = "draft"
	plans := a.contracts().TargetContractActs(act)
	got["contract-act-draft"] = plans
	if plans.Status != report.ReadingOK || len(plans.Value) != 1 {
		t.Fatalf("contract draft: %+v", plans)
	}
	got["contract-act-handoff"] = a.contracts().TargetContractActs(contractsapp.RawContractAct{Mine: "0001", Action: "handoff", ID: plans.Value[0].ID})
	for name, v := range got {
		checkGolden(t, "contracts", name, contractsWire(t, v))
	}
}

// TestLineupAndRosterWireMatchesGolden pins the held lineup, a lineup check and draft, an acquisition
// preview with a cut, and the roster reports.
func TestLineupAndRosterWireMatchesGolden(t *testing.T) {
	t.Parallel()
	a, _, ids := lineupDraftApp(t)
	got := map[string]any{}
	reading, err := a.contracts().TargetLineup("0025")
	got["lineup"], got["lineup-err"] = reading, errText(err)
	check, err := a.contracts().TargetCheckLineup("0025", changedStarters(ids))
	got["lineup-check"], got["lineup-check-err"] = check, errText(err)
	draft, err := a.contracts().TargetDraftLineup("0025", changedStarters(ids))
	got["lineup-draft"], got["lineup-draft-err"] = draft, errText(err)
	for name, v := range got {
		checkGolden(t, "contracts", name, contractsWire(t, v))
	}

	trade, _ := realTradeBuilderApp(t)
	acquisition := trade.contracts().TargetAcquisition(contractsapp.RawAcquisition{Mine: "0001", Player: "15761", Cut: "14109"})
	if len(acquisition.Value.Incoming) < 12 || len(acquisition.Value.Cuts) < 12 {
		t.Fatalf("acquisition: %+v", acquisition)
	}
	// A dozen of each list keeps the golden reviewable.
	acquisition.Value.Incoming, acquisition.Value.Cuts = acquisition.Value.Incoming[:12], acquisition.Value.Cuts[:12]
	got = map[string]any{"acquisition": acquisition}
	reports, _, _ := tradeDraftApp(t)
	id := reports.core.TargetSnapshot.Players.Value[0].ID.String()
	holdPersonalBody(t, reports, "playerScores", `{"playerScores":{"week":"YTD","playerScore":{"id":"`+id+`","score":"134.80"}}}`)
	holdPersonalBody(t, reports, "playerScores", `{"playerScores":{"week":"4","playerScore":{"id":"`+id+`","score":"0.00"}}}`)
	holdPersonalBody(t, reports, "nflByeWeeks", `{"nflByeWeeks":{"team":{"id":"BUF","bye_week":"7"}}}`)
	rosters, err := reports.contracts().TargetRosterReports("0025")
	got["roster-reports"], got["roster-reports-err"] = rosters, errText(err)
	for name, v := range got {
		checkGolden(t, "contracts", name, contractsWire(t, v))
	}
}

// errText is a binding error's words, or "" for none.
func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// refusal is what a refused caller reads: a failed reading's reason, a binding's error, a what-if
// notice, or a blocked plan's state and first audit note.
// Root and feature readings share the AbsenceReason field but not a type, so it is read by name.
func refusal(r any) string {
	reason := reflect.Indirect(reflect.ValueOf(r)).FieldByName("AbsenceReason")
	if !reason.IsValid() || reason.IsNil() {
		return "accepted"
	}
	return reason.Elem().String()
}

func blockedWords(r envelope.Receipt, err error) string {
	if err != nil {
		return err.Error()
	}
	if len(r.Audit) == 0 {
		return string(r.State)
	}
	return string(r.State) + ": " + r.Audit[len(r.Audit)-1].Note
}

// TestContractsDenialsMatchGolden pins malformed, unauthorized and illegal-state refusal text.
func TestContractsDenialsMatchGolden(t *testing.T) {
	t.Parallel()
	got := map[string]string{}
	a := contractsWireApp(t, domain.PhaseOffseason)
	for name, raw := range map[string]contractsapp.RawContractAct{
		"act-bad-franchise":    {Mine: "bad", Action: "read"},
		"act-no-franchise":     {Action: "read"},
		"act-unknown-action":   {Mine: "0001", Action: "delete"},
		"act-options-player":   {Mine: "0001", Action: "options", Player: "x"},
		"act-preview-player":   {Mine: "0001", Action: "preview", Player: "x", Intent: "rfa.tender", Level: 1, Years: 1},
		"act-unknown-intent":   {Mine: "0001", Action: "preview", Player: "0042", Intent: "contract.void", Level: 1, Years: 1},
		"act-tender-level":     {Mine: "0001", Action: "draft", Player: "0042", Intent: "rfa.tender", Level: 7, Years: 1},
		"act-extend-years":     {Mine: "0001", Action: "draft", Player: "0042", Intent: "contract.extend", Level: 1, Years: 5},
		"act-restructure-move": {Mine: "0001", Action: "draft", Player: "0042", Intent: "contract.restructure", Level: 1, Years: 1, Move: "4"},
		"act-no-plan-id":       {Mine: "0001", Action: "posted"},
		"act-not-my-player":    {Mine: "0002", Action: "preview", Player: "0042", Intent: "rfa.tender", Level: 1, Years: 1},
		"act-unknown-plan":     {Mine: "0001", Action: "withdraw", ID: "contract-none"},
	} {
		got[name] = refusal(a.contracts().TargetContractActs(raw))
	}
	resign := contractsapp.RawContractAct{Mine: "0001", Action: "draft", Player: "0046", Intent: "fa.resign", Level: 1, Years: 2}
	plans := a.contracts().TargetContractActs(resign)
	if plans.Status != report.ReadingOK || len(plans.Value) != 1 {
		t.Fatalf("re-sign draft: %+v", plans)
	}
	plan := plans.Value[0].ID
	got["act-second-open-plan"] = refusal(a.contracts().TargetContractActs(resign))
	got["act-other-franchise-plan"] = refusal(a.contracts().TargetContractActs(contractsapp.RawContractAct{Mine: "0002", Action: "withdraw", ID: plan}))
	got["act-posted-before-handoff"] = refusal(a.contracts().TargetContractActs(contractsapp.RawContractAct{Mine: "0001", Action: "posted", ID: plan}))
	got["act-rebase-fresh-plan"] = refusal(a.contracts().TargetContractActs(contractsapp.RawContractAct{Mine: "0001", Action: "rebase", ID: plan}))
	_, err := a.platformAPI().TargetHandOff(plan)
	got["act-generic-handoff"] = errText(err)

	for name, raw := range map[string]contractsapp.RawReleaseRequest{
		"release-bad-franchise":  {Mine: "bad", Action: "read"},
		"release-unknown-action": {Mine: "0001", Action: "submit"},
		"release-draft-no-move":  {Mine: "0001", Action: "draft"},
		"release-handoff-no-id":  {Mine: "0001", Action: "handoff"},
		"release-unknown-move":   {Mine: "0001", Action: "draft", Move: "1"},
		"release-unknown-plan":   {Mine: "0001", Action: "withdraw", ID: "release-none"},
	} {
		got[name] = refusal(a.contracts().TargetReleasePlans(raw))
	}
	cut := whatIfCall(t, a, "cut", "0045", "")
	move := cut.Value.Moves[len(cut.Value.Moves)-1].ID
	got["release-other-franchise-move"] = refusal(a.contracts().TargetReleasePlans(contractsapp.RawReleaseRequest{Mine: "0002", Action: "draft", Move: move}))
	released := a.contracts().TargetReleasePlans(contractsapp.RawReleaseRequest{Mine: "0001", Action: "draft", Move: move})
	if released.Status != report.ReadingOK || len(released.Value) != 1 {
		t.Fatalf("release draft: %+v", released)
	}
	release := released.Value[0].ID
	got["release-other-franchise-plan"] = refusal(a.contracts().TargetReleasePlans(contractsapp.RawReleaseRequest{Mine: "0002", Action: "handoff", ID: release, Confirmed: true}))
	got["release-unconfirmed"] = refusal(a.contracts().TargetReleasePlans(contractsapp.RawReleaseRequest{Mine: "0001", Action: "handoff", ID: release}))
	got["release-rebase-fresh-plan"] = refusal(a.contracts().TargetReleasePlans(contractsapp.RawReleaseRequest{Mine: "0001", Action: "rebase", ID: release}))

	for name, raw := range map[string]contractsapp.RawWhatIfRequest{
		"whatif-bad-franchise":     {Mine: "bad", Action: "read"},
		"whatif-no-franchise":      {Action: "read"},
		"whatif-unknown-action":    {Mine: "0001", Action: "save"},
		"whatif-cut-player":        {Mine: "0001", Action: "cut", Player: "x"},
		"whatif-undo-no-move":      {Mine: "0001", Action: "undo"},
		"whatif-unheld-franchise":  {Mine: "0002", Action: "read"},
		"whatif-not-my-player":     {Mine: "0001", Action: "cut", Player: "0099"},
		"whatif-cut-twice":         {Mine: "0001", Action: "cut", Player: "0045"},
		"whatif-undo-unknown-move": {Mine: "0001", Action: "undo", Move: "1"},
	} {
		r := a.contracts().TargetWhatIf(raw)
		got[name] = refusal(r)
		if r.AbsenceReason == nil {
			got[name] = r.Value.Notice
		}
	}

	for name, raw := range map[string]contractsapp.RawAcquisition{
		"acquisition-bad-franchise": {Mine: "bad"},
		"acquisition-two-sources":   {Mine: "0001", Player: "0042", Bid: "b"},
		"acquisition-bad-player":    {Mine: "0001", Player: "x"},
		"acquisition-bad-cut":       {Mine: "0001", Cut: "x"},
	} {
		got[name] = refusal(a.contracts().TargetAcquisition(raw))
	}
	got["contracts-bad-scope"] = refusal(a.contracts().TargetContracts(snapshot.RawContractScope{Sort: "bad"}).Table)
	got["contracts-unheld-franchise"] = refusal(a.contracts().TargetContracts(snapshot.RawContractScope{Mine: "0002", Sort: "name"}).Cap)
	got["player-contract-bad-id"] = refusal(a.contracts().TargetPlayerContract("bad"))
	got["player-contract-unrostered"] = refusal(a.contracts().TargetPlayerContract("9999"))

	season := contractsWireApp(t, domain.PhaseRegularSeason)
	tender := contractsapp.RawContractAct{Mine: "0001", Action: "draft", Player: "0046", Intent: "rfa.tender", Level: 1, Years: 1}
	blocked := season.contracts().TargetContractActs(tender)
	if blocked.Status != report.ReadingOK || len(blocked.Value) != 1 {
		t.Fatalf("in-season tender: %+v", blocked)
	}
	got["act-in-season-tender"] = blockedWords(blocked.Value[0].Receipt, nil)
	got["act-in-season-tender-handoff"] = refusal(season.contracts().TargetContractActs(contractsapp.RawContractAct{Mine: "0001", Action: "handoff", ID: blocked.Value[0].ID}))
	buyout := whatIfCall(t, season, "buyout", "0042", "")
	bought := season.contracts().TargetReleasePlans(contractsapp.RawReleaseRequest{Mine: "0001", Action: "draft", Move: buyout.Value.Moves[0].ID})
	if bought.Status != report.ReadingOK || len(bought.Value) != 1 {
		t.Fatalf("in-season buyout: %+v", bought)
	}
	got["release-in-season-buyout"] = blockedWords(bought.Value[0].Receipt, nil)
	got["release-in-season-buyout-handoff"] = refusal(season.contracts().TargetReleasePlans(contractsapp.RawReleaseRequest{Mine: "0001", Action: "handoff", ID: bought.Value[0].ID, Confirmed: true}))
	replaceWhatIfMirror(t, season, 9*100_000_000)
	got["whatif-stale"] = whatIfCall(t, season, "cut", "0043", "").Value.Notice

	lineups, _, ids := lineupDraftApp(t)
	_, err = lineups.contracts().TargetLineup("0002")
	got["lineup-unknown-franchise"] = errText(err)
	_, err = lineups.contracts().TargetCheckLineup("0002", ids)
	got["lineup-check-unknown-franchise"] = errText(err)
	_, err = lineups.contracts().TargetCheckLineup("0025", []string{"x"})
	got["lineup-check-bad-player"] = errText(err)
	_, err = lineups.contracts().TargetCheckLineup("0025", []string{ids[0], ids[0]})
	got["lineup-check-duplicate"] = errText(err)
	_, err = lineups.contracts().TargetDraftLineup("0002", ids)
	got["lineup-draft-unknown-franchise"] = errText(err)
	_, err = lineups.contracts().TargetRosterReports("0002")
	got["roster-reports-unknown-franchise"] = errText(err)
	last := leagueweek.LastLock(lineups.core.Week)
	lineups.core.MovesNow = func() time.Time { return last }
	got["lineup-draft-locked"] = blockedWords(lineups.contracts().TargetDraftLineup("0025", changedStarters(ids)))
	lineups.core.Week.Games = nil
	got["lineup-draft-lock-unknown"] = blockedWords(lineups.contracts().TargetDraftLineup("0025", changedStarters(ids)))

	checkGolden(t, "contracts", "denials", contractsWire(t, got))
}

// fillTwin sets every field of the struct v points to, but the plan receipt both twins share, as
// fillWire does.
func fillTwin(t *testing.T, v any, n *int) {
	t.Helper()
	value := reflect.ValueOf(v).Elem()
	for i := range value.NumField() {
		if value.Type().Field(i).Name != "Receipt" {
			fillWire(value.Field(i), n)
		}
	}
}

// TestContractsWireTwinsMatch proves the contracts converters lose nothing: an Act, a what-if, a
// release plan, a lineup player and each request with every field set, and empty ones, encode the
// same as their wire twins. A field added to one side, a renamed JSON tag, or a field a converter
// skips fails here.
func TestContractsWireTwinsMatch(t *testing.T) {
	t.Parallel()
	n := 0
	var act contractsapp.ActView
	fillTwin(t, &act, &n)
	var whatIf contractsapp.WhatIfView
	fillTwin(t, &whatIf, &n)
	var plan contractsapp.ReleasePlan
	fillTwin(t, &plan, &n)
	var player contractsapp.LineupPlayer
	fillTwin(t, &player, &n)
	var rawAct contractsapp.RawContractAct
	fillTwin(t, &rawAct, &n)
	var rawRelease contractsapp.RawReleaseRequest
	fillTwin(t, &rawRelease, &n)
	var rawWhatIf contractsapp.RawWhatIfRequest
	fillTwin(t, &rawWhatIf, &n)
	var rawAcquisition contractsapp.RawAcquisition
	fillTwin(t, &rawAcquisition, &n)
	for _, c := range []struct {
		name       string
		built, got any
	}{
		{"release plan", plan, contractsapp.ReleasePlanView(plan)},
		{"act request", contractsapp.Act(rawAct), rawAct},
		{"release request", contractsapp.Release(rawRelease), rawRelease},
		{"what-if request", contractsapp.WhatIf(rawWhatIf), rawWhatIf},
		{"acquisition request", contractsapp.Acquisition(rawAcquisition), rawAcquisition},
	} {
		want, err := json.Marshal(c.built)
		if err != nil {
			t.Fatal(err)
		}
		got, err := json.Marshal(c.got)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(want, got) {
			t.Errorf("%s: wire twin encodes differently\n built %s\n wire  %s", c.name, want, got)
		}
	}
}

// TestContractsSimultaneousPlansLandOnce races two drafts of one contract Act and two what-if cuts of
// one player: the plan-ledger and what-if locks still hold across the service calls, so exactly one of
// each lands and the other reads the refusal.
func TestContractsSimultaneousPlansLandOnce(t *testing.T) {
	t.Parallel()
	a := contractsWireApp(t, domain.PhaseOffseason)
	act := contractsapp.RawContractAct{Mine: "0001", Action: "draft", Player: "0043", Intent: "contract.extend", Level: 1, Years: 2}
	cut := contractsapp.RawWhatIfRequest{Mine: "0001", Action: "cut", Player: "0044"}
	acts := make([]contractsapp.ContractActsReading, 2)
	cuts := make([]contractsapp.WhatIfReading, 2)
	var wg sync.WaitGroup
	for i := range 2 {
		wg.Go(func() { acts[i] = a.contracts().TargetContractActs(act) })
		wg.Go(func() { cuts[i] = a.contracts().TargetWhatIf(cut) })
	}
	wg.Wait()
	landed, refused := 0, 0
	for _, r := range acts {
		switch {
		case r.Status == report.ReadingOK && len(r.Value) == 1:
			landed++
		case r.AbsenceReason != nil && strings.Contains(*r.AbsenceReason, "is already an open plan in My moves"):
			refused++
		}
	}
	if landed != 1 || refused != 1 {
		t.Fatalf("contract drafts: %d landed, %d refused: %+v", landed, refused, acts)
	}
	notices := []string{cuts[0].Value.Notice, cuts[1].Value.Notice}
	slices.Sort(notices)
	if notices[0] != "" || notices[1] != "what-if: player already removed; Undo first" {
		t.Fatalf("what-if cuts: %q", notices)
	}
	if final := a.contracts().TargetWhatIf(contractsapp.RawWhatIfRequest{Mine: "0001", Action: "read"}); len(final.Value.Moves) != 1 {
		t.Fatalf("what-if moves %+v", final.Value.Moves)
	}
}
