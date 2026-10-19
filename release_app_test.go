package main

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func releaseTestApp(t *testing.T) *App {
	t.Helper()
	a := whatIfTestApp(t)
	a.movesChanged = func(context.Context) {}
	a.movesLog = func(string) {}
	a.openURL = func(context.Context, string) {}
	a.movesNow = func() time.Time { return time.Now().UTC() }
	m, _, _ := a.league.ContractsView()
	a.whatif = state.New(a.whatifPools, ingestion.LeagueID, 2026, a.rulebook)
	if err := a.whatif.Initialize(a.ctx, state.SnapshotSource{Snapshot: m}); err != nil {
		t.Fatal(err)
	}
	client, err := mfl.New("www47", 100, mfl.WithTransport(&directoryDown{}))
	if err != nil {
		t.Fatal(err)
	}
	a.mflClient = client
	return a
}
func releasePhase(t *testing.T, a *App, phase domain.Phase) {
	t.Helper()
	_, err := a.whatifPools.Write().ExecContext(a.ctx, `INSERT INTO season_phases (league_id,season,from_phase,to_phase,note,at) VALUES (?,2026,'',?,'fixture',?)`, ingestion.LeagueID, string(phase), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
}
func releaseCall(t *testing.T, a *App, action, move, id string, confirmed bool) []ReleasePlanView {
	t.Helper()
	r := a.TargetReleasePlans(RawReleaseRequest{Mine: "0001", Action: action, Move: move, ID: id, Confirmed: confirmed})
	if r.Status != ReadingOK {
		t.Fatalf("%s failed: %s", action, *r.AbsenceReason)
	}
	return r.Value
}
func planRelease(t *testing.T, a *App, action, player string) ReleasePlanView {
	t.Helper()
	w := whatIfCall(t, a, action, player, "")
	move := w.Value.Moves[len(w.Value.Moves)-1]
	plans := releaseCall(t, a, "draft", move.ID, "", false)
	for _, plan := range plans {
		if plan.Receipt.Spec.Expected.Player.String() == player {
			return plan
		}
	}
	t.Fatal("new plan not listed")
	return ReleasePlanView{}
}
func TestReleaseDraftFromWhatIfAndMirrorIsolation(t *testing.T) {
	t.Parallel()
	a := releaseTestApp(t)
	before, stamp, _ := a.league.ContractsView()
	plan := planRelease(t, a, "cut", "0042")
	if plan.Receipt.State != envelope.Ready || plan.Charge != "$7,000,000" || !strings.Contains(plan.Confirmation, "$7,000,000 dead cap this season") || !strings.Contains(plan.Receipt.Spec.Target.URL, "O=257") {
		t.Fatalf("plan %+v", plan)
	}
	session, _, err := state.NewWhatIfSessions(a.whatifPools).Load(a.ctx, "0001")
	if err != nil {
		t.Fatal(err)
	}
	f, err := a.moves.ReleaseFacts(a.ctx, plan.ID)
	if err != nil || f.Charge != session.Moves[0].Charge || f.RostersStamp != session.Stamps.Rosters || f.ContractStamp != session.Stamps.Contract || f.DeadCapStamp != session.Stamps.DeadCap {
		t.Fatalf("facts %+v %v", f, err)
	}
	opened := 0
	a.openURL = func(_ context.Context, url string) {
		opened++
		if !strings.Contains(url, "options?L=14432&O=257") {
			t.Error(url)
		}
	}
	rejected := a.TargetReleasePlans(RawReleaseRequest{Mine: "0001", Action: "handoff", ID: plan.ID})
	if rejected.Status != ReadingFailed || opened != 0 {
		t.Fatal("unconfirmed cut opened MFL")
	}
	if _, err := a.TargetHandOff(plan.ID); err == nil {
		t.Fatal("generic hand-off bypassed confirmation")
	}
	releaseCall(t, a, "handoff", "", plan.ID, true)
	if opened != 1 {
		t.Fatal("MFL page not opened")
	}
	duplicate := a.TargetReleasePlans(RawReleaseRequest{Mine: "0001", Action: "draft", Move: session.Moves[0].ID})
	if duplicate.Status != ReadingFailed {
		t.Fatal("second open plan allowed")
	}
	labels := whatIfCall(t, a, "read", "", "")
	if labels.Value.Moves[0].Plan != "Plan: Handed off" {
		t.Fatal(labels.Value.Moves)
	}
	after, afterStamp, _ := a.league.ContractsView()
	if !reflect.DeepEqual(before, after) || stamp != afterStamp {
		t.Fatal("release writes mirror")
	}
}
func TestReleaseBuyoutPhaseAndCountIntegration(t *testing.T) {
	t.Parallel()
	a := releaseTestApp(t)
	releasePhase(t, a, domain.PhaseRegularSeason)
	first := planRelease(t, a, "buyout", "0042")
	if first.Receipt.State != envelope.Blocked || !strings.Contains(first.Receipt.Audit[0].Note, "Offseason only") {
		t.Fatal(first)
	}
	if first.CanRebase || first.WithdrawLabel != "Remove plan" || !strings.Contains(first.Receipt.Audit[0].Note, "Offseason only: can't be done until the offseason.") {
		t.Fatal(first)
	}
	rejected := a.TargetReleasePlans(RawReleaseRequest{Mine: "0001", Action: "rebase", ID: first.ID})
	if rejected.Status != ReadingFailed {
		t.Fatal("blocked rebase allowed")
	}
	releaseCall(t, a, "withdraw", "", first.ID, false)
	releasePhase(t, a, domain.PhaseOffseason)
	whatIfCall(t, a, "reset", "", "")
	first = planRelease(t, a, "buyout", "0042")
	handed := releaseCall(t, a, "handoff", "", first.ID, false)[0]
	if handed.Receipt.State != envelope.HandedOff || handed.Receipt.Spec.Target.Kind != envelope.Manual {
		t.Fatal(handed)
	}
	second := planRelease(t, a, "buyout", "0043")
	if second.Receipt.State != envelope.Ready {
		t.Fatal(second)
	}
	whatIfCall(t, a, "reset", "", "")
	third := planRelease(t, a, "buyout", "0044")
	if third.Receipt.State != envelope.Blocked || !strings.Contains(third.Receipt.Audit[0].Note, "Two buyouts") {
		t.Fatal(third)
	}
	count, err := a.releaseBuyoutCount(a.ctx, "0001", third.ID, 2026)
	if err != nil || count != 2 {
		t.Fatal(count, err)
	}
	if count, err := a.releaseBuyoutCount(a.ctx, "0001", "", 2027); err != nil || count != 0 {
		t.Fatal(count, err)
	}
}
func TestReleaseInvalidationAndRebaseEngine(t *testing.T) {
	t.Parallel()
	for _, trigger := range []string{"contract", "adjustment", "dead-cap", "left", "window"} {
		t.Run(trigger, func(t *testing.T) {
			t.Parallel()
			a := releaseTestApp(t)
			action := "cut"
			if trigger == "window" {
				action = "buyout"
			}
			plan := planRelease(t, a, action, "0042")
			old := plan.Charge
			m, _, _ := a.league.ContractsView()
			switch trigger {
			case "contract":
				m.Players[0].Salary = 20 * 100_000_000
			case "adjustment":
				m.Adjustments = append(m.Adjustments, state.Adjustment{ID: "new", FranchiseID: "0001", Amount: 12345})
			case "dead-cap":
				if _, err := a.pools.Write().ExecContext(a.ctx, `CREATE TABLE dead_cap_ledger(franchise_id TEXT,created_at TEXT); INSERT INTO dead_cap_ledger VALUES ('0001','2099-01-01T00:00:00Z')`); err != nil {
					t.Fatal(err)
				}
			case "left":
				m.Players = m.Players[1:]
			case "window":
				releasePhase(t, a, domain.PhaseRegularSeason)
			}
			if _, err := a.league.Replace(a.ctx, m); err != nil {
				t.Fatal(err)
			}
			plans := releaseCall(t, a, "read", "", "", false)
			if plans[0].Receipt.State != envelope.Stale {
				t.Fatal("not stale", plans[0])
			}
			if trigger == "left" {
				return
			}
			plans = releaseCall(t, a, "rebase", "", plan.ID, false)
			got := plans[0]
			if trigger == "window" {
				if got.Receipt.State != envelope.Blocked {
					t.Fatal(got)
				}
			} else if got.Receipt.State != envelope.Ready {
				t.Fatal(got)
			}
			history := ""
			for _, entry := range got.Receipt.Audit {
				history += entry.Note
			}
			if !strings.Contains(history, old+" → "+got.Charge) {
				t.Fatal("rebase charge comparison absent", history)
			}
			if trigger == "contract" && got.Charge != "$14,000,000" {
				t.Fatal("engine did not reprice", got.Charge)
			}
		})
	}
}
func TestReleaseWatcherLandsAndChargeMismatchIsNote(t *testing.T) {
	t.Parallel()
	a := releaseTestApp(t)
	plan := planRelease(t, a, "cut", "0042")
	releaseCall(t, a, "handoff", "", plan.ID, true)
	a.movesRefresh = func(ctx context.Context, _ envelope.Source) error {
		m, _, _ := a.league.ContractsView()
		m.Players = m.Players[1:]
		m.Adjustments = append(m.Adjustments, state.Adjustment{ID: "charge", FranchiseID: "0001", Amount: 100_000_000, Description: "dead cap player 0042"})
		_, err := a.league.Replace(ctx, m)
		return err
	}
	if err := a.checkMoves(a.ctx); err != nil {
		t.Fatal(err)
	}
	plans := releaseCall(t, a, "read", "", "", false)
	r := plans[0].Receipt
	if r.State != envelope.Landed {
		t.Fatal(r)
	}
	last := r.Audit[len(r.Audit)-1]
	if last.Event != envelope.Invalidate || !strings.Contains(last.Note, "$7,000,000 planned vs $1,000,000 now") {
		t.Fatal(last)
	}
	before := len(r.Audit)
	if got := releaseCall(t, a, "read", "", "", false)[0]; len(got.Receipt.Audit) != before {
		t.Fatal("repeated note appended")
	}
	m, _, _ := a.league.ContractsView()
	m.Adjustments = nil
	if _, err := a.league.Replace(a.ctx, m); err != nil {
		t.Fatal(err)
	}
	if got := releaseCall(t, a, "read", "", "", false)[0]; !strings.Contains(got.Receipt.Audit[len(got.Receipt.Audit)-1].Note, "cannot see") {
		t.Fatal(got)
	}
	count, err := a.releaseBuyoutCount(a.ctx, "0001", "", 2026)
	if err != nil || count != 0 {
		t.Fatal(count, err)
	}
}

func TestReleaseBaselinePlayerNoLongerMineBlocksDraft(t *testing.T) {
	t.Parallel()
	a := releaseTestApp(t)
	w := whatIfCall(t, a, "cut", "0042", "")
	m, _, _ := a.league.ContractsView()
	m.Players = m.Players[1:]
	if _, err := a.league.Replace(a.ctx, m); err != nil {
		t.Fatal(err)
	}
	got := releaseCall(t, a, "draft", w.Value.Moves[0].ID, "", false)[0]
	if got.Receipt.State != envelope.Blocked || !strings.Contains(got.Receipt.Audit[0].Note, "not on my roster") {
		t.Fatal(got)
	}
}
func TestReleaseCountIncludesLandedBuyoutAndIgnoresOtherSeason(t *testing.T) {
	t.Parallel()
	a := releaseTestApp(t)
	plan := planRelease(t, a, "buyout", "0042")
	releaseCall(t, a, "handoff", "", plan.ID, false)
	a.movesRefresh = func(ctx context.Context, _ envelope.Source) error {
		m, _, _ := a.league.ContractsView()
		m.Players = m.Players[1:]
		_, err := a.league.Replace(ctx, m)
		return err
	}
	if err := a.checkMoves(a.ctx); err != nil {
		t.Fatal(err)
	}
	count, err := a.releaseBuyoutCount(a.ctx, "0001", "", 2026)
	if err != nil || count != 1 {
		t.Fatal("landed buyout not counted", count, err)
	}
	if count, err := a.releaseBuyoutCount(a.ctx, "0001", "", 2027); err != nil || count != 0 {
		t.Fatal("wrong season counted", count, err)
	}
}

func TestReleaseWithdrawPathsFreePlayerSlot(t *testing.T) {
	t.Parallel()
	for _, stage := range []string{"ready", "blocked", "handed_off", "not_yet_done"} {
		t.Run(stage, func(t *testing.T) {
			t.Parallel()
			a := releaseTestApp(t)
			if stage == "blocked" {
				releasePhase(t, a, domain.PhaseRegularSeason)
			}
			action := "cut"
			if stage == "blocked" {
				action = "buyout"
			}
			plan := planRelease(t, a, action, "0042")
			session, _, err := state.NewWhatIfSessions(a.whatifPools).Load(a.ctx, "0001")
			if err != nil {
				t.Fatal(err)
			}
			if stage == "handed_off" || stage == "not_yet_done" {
				releaseCall(t, a, "handoff", "", plan.ID, true)
				if stage == "not_yet_done" {
					a.movesRefresh = func(context.Context, envelope.Source) error { return nil }
					if err := a.checkMoves(a.ctx); err != nil {
						t.Fatal(err)
					}
				}
			}
			before, stamp, _ := a.league.ContractsView()
			opened := 0
			a.openURL = func(context.Context, string) { opened++ }
			current := releaseCall(t, a, "read", "", "", false)[0]
			want := "Remove plan"
			if stage == "handed_off" || stage == "not_yet_done" {
				want = "I didn't drop him"
			}
			if current.WithdrawLabel != want {
				t.Fatal(current)
			}
			gone := releaseCall(t, a, "withdraw", "", plan.ID, false)[0]
			if gone.Receipt.State != envelope.Withdrawn || gone.WithdrawLabel != "" || gone.CanHandOff || gone.CanRebase {
				t.Fatal(gone)
			}
			if opened != 0 {
				t.Fatal("withdraw opened MFL")
			}
			after, afterStamp, _ := a.league.ContractsView()
			if !reflect.DeepEqual(before, after) || stamp != afterStamp {
				t.Fatal("withdraw writes mirror")
			}
			plans := releaseCall(t, a, "draft", session.Moves[0].ID, "", false)
			if len(plans) != 2 {
				t.Fatal("withdraw did not free player slot", plans)
			}
		})
	}
}
func TestReleaseViewWordsAndPostKeepIDsOnlyInFacts(t *testing.T) {
	t.Parallel()
	for _, action := range []string{"cut", "buyout"} {
		t.Run(action, func(t *testing.T) {
			t.Parallel()
			a := releaseTestApp(t)
			plan := planRelease(t, a, action, "0042")
			if strings.Contains(strings.ReplaceAll(plan.Post, plan.Player, "Example"), "0042") || strings.Contains(plan.Post, "— player") || !strings.Contains(plan.Post, "annual salary") || !strings.Contains(plan.Post, plan.Charge) {
				t.Fatal(plan.Post)
			}
			f, err := a.moves.ReleaseFacts(a.ctx, plan.ID)
			if err != nil || f.Player.String() != "0042" {
				t.Fatal(f, err)
			}
			if plan.CanRebase || plan.RebaseLabel != "Reprice from today's MFL rosters" {
				t.Fatal(plan)
			}
			stale := plan.Receipt
			stale.State = envelope.Stale
			view := releasePlanView(stale, f)
			if !view.CanRebase {
				t.Fatal(view)
			}
		})
	}
}
