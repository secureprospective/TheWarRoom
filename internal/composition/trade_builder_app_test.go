package composition

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	tradesapp "github.com/secureprospective/TheWarRoom/internal/app/trades"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

func realTradeBuilderApp(t *testing.T) (*App, *seasonTransport) {
	t.Helper()
	a, transport, _ := lineupDraftApp(t)
	a.core.MyFranchise = "0001"
	pools, err := db.Open(a.core.Ctx, filepath.Join(t.TempDir(), "whatif.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	})
	a.core.WhatifPools = pools
	a.core.Whatif = state.New(pools, ingestion.LeagueID, 2026, a.core.Rulebook)
	if err := a.core.Whatif.Initialize(a.core.Ctx, a.core.League); err != nil {
		t.Fatal(err)
	}
	a.core.Coordinator, err = transactions.New(a.core.Whatif.Writer(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	a.core.ClockChanged = func(context.Context) {}
	if err := a.controlAPI().TargetSetWindow("trade_deadline", time.Now().Add(time.Hour).UTC().Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile("frontend/src/app/data/fixtures/snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	var snap snapshot.Snapshot
	if err := json.Unmarshal(body, &snap); err != nil {
		t.Fatal(err)
	}
	a.core.TargetSnapshot = snap
	a.core.HasTargetSnapshot = true
	m := state.MirrorSnapshot{Season: 2026}
	for _, roster := range snap.Rosters.Value {
		for _, p := range roster.Players {
			last := 2026
			if p.YearsRemaining != nil {
				last += *p.YearsRemaining - 1
			}
			m.Players = append(m.Players, state.MirrorPlayer{MFLID: p.ID.String(), FranchiseID: roster.FranchiseID,
				RosterStatus: p.RosterStatus, Salary: p.Salary, ContractYear: last, ContractStatus: p.ContractStatus})
		}
	}
	if _, err := a.core.League.Replace(a.core.Ctx, m); err != nil {
		t.Fatal(err)
	}
	if err := a.core.Rulebook.SetOverride(a.core.Ctx, "cap", "salaryCapAmount", "125", "fixture rulebook"); err != nil {
		t.Fatal(err)
	}
	picks, err := draftreports.ParseFuture([]byte(seasonFixture(t, "draftreports", "future-2026.json")))
	if err != nil {
		t.Fatal(err)
	}
	a.core.SeasonFuturePicks = heldSeasonFeed[[]draftreports.FuturePick]{Value: picks, FetchedAt: a.core.MovesNow()}
	a.core.OpenURL = func(context.Context, string) {}
	transport.requests = nil
	return a, transport
}

// builderReading sends r to the builder command that owns its action, as the frontend does.
func builderReading(a *App, r tradesapp.RawTradeBuilder) tradesapp.TradeBuilderReading {
	switch r.Action {
	case "read":
		return a.tradesAPI().TargetTradeBuilderRead(r.Mine)
	case "preview":
		return a.tradesAPI().TargetTradeBuilderPreview(r)
	default:
		return a.tradesAPI().TargetTradeBuilder(r)
	}
}

func builderCall(t *testing.T, a *App, r tradesapp.RawTradeBuilder) tradesapp.TradeBuilderView {
	t.Helper()
	reading := builderReading(a, r)
	if reading.Status != report.ReadingOK {
		reason := "<nil>"
		if reading.AbsenceReason != nil {
			reason = *reading.AbsenceReason
		}
		t.Fatalf("builder %s: %s", r.Action, reason)
	}
	return reading.Value
}
func realDeal() tradesapp.RawTradeBuilder {
	return tradesapp.RawTradeBuilder{Mine: "0001", Other: "0025", Action: "preview", Give: []string{"11675"}, Get: []string{"15761"}, Rationale: "Balance salary with a younger starter."}
}
func TestTradeBuilderRealRosterExactCentsBothSides(t *testing.T) {
	t.Parallel()
	a, transport := realTradeBuilderApp(t)
	m, _, _ := a.core.League.ContractsView()
	// Sub-dollar cents on two observed contracts expose ledger rounding mistakes.
	for i := range m.Players {
		switch m.Players[i].MFLID {
		case "11675":
			m.Players[i].Salary += 37
		case "15761":
			m.Players[i].Salary += 83
		}
	}
	if _, err := a.core.League.Replace(a.core.Ctx, m); err != nil {
		t.Fatal(err)
	}
	m, stamp, used := a.core.League.ContractsView()
	preview := builderCall(t, a, realDeal()).Preview
	if len(preview.Teams) != 2 {
		t.Fatal(preview)
	}
	for i, fid := range []string{"0001", "0025"} {
		before := used[fid]
		delta := domain.Money(-827_000_000 + 46)
		if i == 1 {
			delta = -delta
		}
		want := snapshot.ContractAmountOf(before + delta).Text
		if preview.Teams[i].Figures[0].After != want {
			t.Fatalf("%s after %s want %s", fid, preview.Teams[i].Figures[0].After, want)
		}
		if len(preview.Teams[i].Roster) == 0 {
			t.Fatal("roster projection empty")
		}
		for _, f := range preview.Teams[i].Figures {
			if strings.HasPrefix(f.Label, "Active roster") || strings.HasPrefix(f.Label, "Taxi squad") {
				before, errB := strconv.Atoi(f.Before)
				after, errA := strconv.Atoi(f.After)
				if errB != nil || errA != nil || (f.Change == "") != (before == after) {
					t.Fatalf("%s %s: before %q after %q change %q", fid, f.Label, f.Before, f.After, f.Change)
				}
			}
		}
	}
	after, nextStamp, _ := a.core.League.ContractsView()
	if !reflect.DeepEqual(m, after) || stamp != nextStamp || len(transport.requests) != 0 {
		t.Fatal("projection changed mirror or fetched")
	}
	if !strings.Contains(preview.Post, "a year through 2026") || strings.Contains(preview.Post, "11675") {
		t.Fatal(preview.Post)
	}
	body, err := json.Marshal(builderReading(a, realDeal()))
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("TWR_UPDATE_TRADE_FIXTURES") == "1" {
		if err := os.WriteFile("frontend/src/app/shell/trade-wire.testdata.json", body, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
func TestTradeBuilderDeadlineOwnershipPersistenceAndSafeHandoff(t *testing.T) {
	t.Parallel()
	a, transport := realTradeBuilderApp(t)
	r := realDeal()
	r.Give = []string{"14167"}
	r.Action = "save"
	saved := builderCall(t, a, r).Preview
	if saved.Receipt.State != envelope.Ready {
		t.Fatalf("not ready: %+v", saved)
	}
	id := saved.ID
	plans, err := state.NewTradePlans(a.core.WhatifPools).List(a.core.Ctx, r.Mine)
	if err != nil || len(plans) != 1 || plans[0].ID != id {
		t.Fatal(plans, err)
	}
	opens := 0
	a.core.OpenURL = func(_ context.Context, target string) {
		opens++
		if target != "https://www47.myfantasyleague.com/2026/options?L=14432&O=05" {
			t.Fatal(target)
		}
	}
	if err := a.controlAPI().TargetSetWindow("trade_deadline", time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	failed := a.tradesAPI().TargetTradeBuilder(tradesapp.RawTradeBuilder{Mine: r.Mine, Action: "handoff", ID: id})
	if failed.Status != report.ReadingFailed || opens != 0 {
		t.Fatal("deadline bypass", failed)
	}
	blocked := builderCall(t, a, r).Preview
	if blocked.Receipt.State != envelope.Blocked || !strings.Contains(strings.Join(blocked.Reasons, " "), "deadline") {
		t.Fatal(blocked)
	}
	if err := a.controlAPI().TargetSetWindow("trade_deadline", time.Now().Add(time.Hour).UTC().Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	handed := a.tradesAPI().TargetTradeBuilder(tradesapp.RawTradeBuilder{Mine: r.Mine, Action: "handoff", ID: id})
	if handed.Status != report.ReadingOK || opens != 1 {
		t.Fatal(handed, opens)
	}
	if _, err := a.platformAPI().TargetHandOff(id); err == nil {
		t.Fatal("generic handoff bypassed trade pre-flight")
	}
	wrong := realDeal()
	wrong.Give = []string{"16387"}
	preview := builderCall(t, a, wrong).Preview
	if len(preview.Reasons) == 0 {
		t.Fatal("wrong ownership passed")
	}
	picks := builderCall(t, a, tradesapp.RawTradeBuilder{Mine: r.Mine, Action: "read"}).Teams
	has := false
	for _, team := range picks {
		for _, asset := range team.Assets {
			if asset.Token == "FP_0025_2027_2" {
				has = true
			}
		}
	}
	if !has || len(transport.requests) != 0 {
		t.Fatal("held picks missing or fetched")
	}
}
func TestAcquisitionForcedCutUsesExistingCharge(t *testing.T) {
	t.Parallel()
	a, transport := realTradeBuilderApp(t)
	m, _, used := a.core.League.ContractsView()
	capAmount := domain.Money(125_0000_0000)
	m.Adjustments = append(m.Adjustments, state.Adjustment{ID: "forced-cap-fixture", FranchiseID: "0001", Amount: capAmount - used["0001"] - 100_000_000})
	if _, err := a.core.League.Replace(a.core.Ctx, m); err != nil {
		t.Fatal(err)
	}
	raw := contractsapp.RawAcquisition{Mine: "0001", Player: "15761"}
	noCut := a.contracts().TargetAcquisition(raw)
	if noCut.Status != report.ReadingOK || !strings.Contains(strings.Join(noCut.Value.Reasons, " "), "does not fit") || !strings.HasPrefix(noCut.Value.Spot, "One active roster spot") {
		t.Fatal(noCut)
	}
	raw.Cut = "14109"
	cut := a.contracts().TargetAcquisition(raw)
	if cut.Status != report.ReadingOK || cut.Value.DeadCap.Cents != 945_000_000 || len(cut.Value.Seasons) != 3 {
		t.Fatal(cut)
	}
	if cut.Value.Team[0].After != "$129,380,000" {
		t.Fatal(cut.Value.Team)
	}
	if !strings.Contains(cut.Value.CutName, " ") || strings.Contains(cut.Value.CutName, "14109") {
		t.Fatal(cut.Value.CutName)
	}
	// A lower incoming held salary fits after the same cut; no automatic cheapest-cut guess.
	raw.Player = "16387"
	fit := a.contracts().TargetAcquisition(raw)
	if fit.Status != report.ReadingOK {
		t.Fatal(fit)
	}
	if strings.Contains(strings.Join(fit.Value.Reasons, " "), "does not fit") {
		t.Fatal(fit)
	}
	if len(transport.requests) != 0 {
		t.Fatal("acquisition fetched")
	}
	after, _, _ := a.core.League.ContractsView()
	if !reflect.DeepEqual(m, after) {
		t.Fatal("acquisition changed mirror")
	}
	if os.Getenv("TWR_UPDATE_TRADE_FIXTURES") == "1" {
		body, err := json.Marshal(cut)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile("frontend/src/app/contracts/acquisition-wire.testdata.json", body, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNoMutatingTradeLinkInSources(t *testing.T) {
	t.Parallel()
	forbidden := "trade" + "_response"
	root, err := os.OpenRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	}()
	err = filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		if d.IsDir() && (d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "dist") {
			return filepath.SkipDir
		}
		if d.IsDir() || (!strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".tsx")) {
			return nil
		}
		body, err := root.ReadFile(path)
		if err != nil {
			return fmt.Errorf("source: %w", err)
		}
		if strings.Contains(string(body), forbidden) {
			t.Errorf("mutating trade URL in %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
func TestTradeExitDirectionsAndAmbiguity(t *testing.T) {
	t.Parallel()
	a, _, offer := tradeDraftApp(t)
	// Exit preparation does not require a builder ledger, only a keyed held pending offer.
	for _, tc := range []struct{ mine, action string }{{"0025", "reject"}, {"0001", "revoke"}} {
		snap, held, err := a.tradeState(tc.mine)
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := a.draftTradeExit(a.core.Ctx, tradesapp.RawTradeBuilder{Mine: tc.mine, Action: tc.action, ID: offer.ID}, held)
		_ = snap
		if err != nil || receipt.State != envelope.Ready || receipt.Spec.Gravity != envelope.G3 {
			t.Fatal(receipt, err)
		}
		obs := envelope.Observation{LeagueID: receipt.Spec.LeagueID, HandedOffAt: time.Now()}
		verdict := (envelope.TradeActPredicate{}).Evaluate(receipt.Spec, obs)
		if verdict.Event != envelope.Partial || verdict.Note != envelope.TradeGoneNote {
			t.Fatal(verdict)
		}
	}
	_, held, err := a.tradeState("0025")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.draftTradeExit(a.core.Ctx, tradesapp.RawTradeBuilder{Mine: "0025", Action: "revoke", ID: offer.ID}, held); err == nil {
		t.Fatal("other owner's revoke allowed")
	}
}

// A second withdrawal plan for the same offer supersedes the first, so one offer never carries two live plans.
func TestTradeExitPlanSupersedesTheLastForTheSameOffer(t *testing.T) {
	t.Parallel()
	a, _, offer := tradeDraftApp(t)
	_, held, err := a.tradeState("0001")
	if err != nil {
		t.Fatal(err)
	}
	raw := tradesapp.RawTradeBuilder{Mine: "0001", Action: "revoke", ID: offer.ID}
	first, err := a.draftTradeExit(a.core.Ctx, raw, held)
	if err != nil {
		t.Fatal(err)
	}
	second, err := a.draftTradeExit(a.core.Ctx, raw, held)
	if err != nil || second.State != envelope.Ready {
		t.Fatal(second, err)
	}
	receipts, err := a.core.Moves.List(a.core.Ctx, second.Spec.LeagueID, "0001")
	if err != nil {
		t.Fatal(err)
	}
	live := 0
	for _, r := range receipts {
		if r.Spec.Intent == "trade.revoke" && r.State == envelope.Ready {
			live++
		}
		if r.CorrelationID == first.CorrelationID && r.State != envelope.Stale {
			t.Fatalf("first plan is %s, want stale", r.State)
		}
	}
	if live != 1 {
		t.Fatalf("%d live withdrawal plans for one offer, want 1", live)
	}
}

func TestTradeBuilderIRTaxiAndRemovedWhatIfAssets(t *testing.T) {
	t.Parallel()
	a, _ := realTradeBuilderApp(t)
	raw := realDeal()
	raw.Give = []string{"14167"}
	raw.Get = []string{"15831"}
	ir := builderCall(t, a, raw).Preview
	found := false
	for _, p := range ir.Teams[0].Roster {
		if p.Token == "15831" && strings.Contains(p.Name, "IR") {
			found = true
		}
	}
	if !found {
		t.Fatal("IR was prohibited or activated")
	}
	raw.Get = []string{"17598"}
	taxi := builderCall(t, a, raw).Preview
	found = false
	for _, p := range taxi.Teams[0].Roster {
		if p.Token == "17598" && strings.Contains(p.Name, "TAXI_SQUAD") {
			found = true
		}
	}
	if !found {
		t.Fatal("taxi status was lost")
	}
	cut := a.contracts().TargetWhatIf(contractsapp.RawWhatIfRequest{Mine: "0001", Action: "cut", Player: "14167"})
	if cut.Status != report.ReadingOK || cut.Value.Notice != "" {
		t.Fatal(cut)
	}
	removed := builderCall(t, a, raw).Preview
	if !strings.Contains(strings.Join(removed.Reasons, " "), "removed in what-if") {
		t.Fatal(removed)
	}
}
func TestTradeBuilderNextSeasonToTheCentAndPickOnly(t *testing.T) {
	t.Parallel()
	a, _ := realTradeBuilderApp(t)
	r := realDeal()
	r.Give = []string{"14167"}
	preview := builderCall(t, a, r).Preview
	if preview.Teams[0].Figures[1].Before != "$73,940,000" || preview.Teams[0].Figures[1].After != "$77,370,000" ||
		preview.Teams[1].Figures[1].After != "$101,240,000" {
		t.Fatal(preview.Teams)
	}
	r.Give = []string{"FP_0001_2028_2"} // Owned by 0021, not its original franchise.
	wrong := builderCall(t, a, r).Preview
	if len(wrong.Reasons) == 0 {
		t.Fatal("original owner confused with current owner")
	}
	r.Mine = "0002"
	a.core.MyFranchise = "0002"
	r.Give = []string{"FP_0002_2027_1"}
	r.Get = []string{"FP_0025_2027_2"}
	picks := builderCall(t, a, r).Preview
	for _, team := range picks.Teams {
		if team.Figures[0].Before != team.Figures[0].After {
			t.Fatal("pick-only trade charged cap")
		}
	}
}

func TestProposalWatcherPersistsSameStateSubmissionEvidence(t *testing.T) {
	t.Parallel()
	a, _, offer := tradeDraftApp(t)
	a.core.SeasonPendingTrades.Value = nil
	req := (tradesHost{State: a.core, app: a}).TradeRequest("0001", offer)
	req.Trade.TradeID = "local"
	e, err := envelope.DraftTradeAct(a.core.MovesNow(), "trade.propose", req, nil, func() string { return "watch-propose" })
	if err != nil {
		t.Fatal(err)
	}
	handed := a.core.MovesNow().Add(time.Minute)
	e, err = e.HandOff(handed)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.core.Moves.Save(a.core.Ctx, e); err != nil {
		t.Fatal(err)
	}
	for step, want := range []envelope.State{envelope.NotYetDone, envelope.NotYetDone, envelope.DOTReview} {
		now := handed.Add(time.Duration(step+1) * time.Minute)
		a.core.MovesNow = func() time.Time { return now }
		a.core.MovesRefresh = func(_ context.Context, source envelope.Source) error {
			switch source {
			case envelope.PendingTrades:
				a.core.SeasonPendingTrades.FetchedAt = now
				a.core.SeasonPendingTrades.Value = nil
				if step == 1 {
					offer.Proposed = handed.Add(time.Second)
					a.core.SeasonPendingTrades.Value = append(a.core.SeasonPendingTrades.Value, offer)
				}
			case envelope.Transactions:
				a.core.SeasonTransactions.FetchedAt = now
				a.core.SeasonTransactions.Value = nil
			case envelope.Rosters, envelope.Contracts, envelope.Lineups:
				t.Fatalf("unexpected feed %s", source)
			}
			return nil
		}
		if err := a.checkMoves(a.core.Ctx); err != nil {
			t.Fatal(err)
		}
		saved, err := a.core.Moves.Get(a.core.Ctx, e.ID())
		if err != nil || saved.State() != want {
			t.Fatal(saved.Receipt(), err)
		}
		if step == 1 && saved.Receipt().Audit[len(saved.Receipt().Audit)-1].Note != "Proposed on MFL" {
			t.Fatal("same-state proposal proof was dropped")
		}
	}
}

func TestTradeBuilderLandingRichAssets(t *testing.T) {
	t.Parallel()
	a, _ := realTradeBuilderApp(t)
	view := builderCall(t, a, tradesapp.RawTradeBuilder{Mine: "0001", Action: "read"})
	players, picks := 0, 0
	for _, team := range view.Teams {
		for _, asset := range team.Assets {
			if strings.HasPrefix(asset.Token, "FP_") {
				picks++
				if asset.Dynasty == nil || asset.Dynasty.Text != "Not valued" {
					t.Fatal(asset)
				}
			} else {
				players++
				if asset.Position == "" || asset.Team == "" || !strings.Contains(asset.Contract, "years") ||
					!strings.HasPrefix(asset.Contract, "$") || asset.Now == nil || asset.Dynasty == nil {
					t.Fatal(asset)
				}
			}
		}
	}
	if players == 0 || picks == 0 {
		t.Fatal(players, picks)
	}

}

type landingPickModel struct{}

func (landingPickModel) Player(string, time.Time) dynastyValue { return dynastyValue{} }
func (landingPickModel) Pick(year, round int, orig string, at time.Time) dynastyValue {
	if year != 2027 || round != 1 || orig != "0001" || at.IsZero() {
		return dynastyValue{}
	}
	return dynastyValue{Valued: true, Discounted: 0.78, Seasons: 5}
}

func TestTradeBuilderLandingPickValue(t *testing.T) {
	t.Parallel()
	a, _ := realTradeBuilderApp(t)
	a.marketHeld.PublishModel(landingPickModel{})
	got := a.builderPickValue(2027, 1, "0001", time.Now())
	if !got.Held || got.Text != "0.78 wins" {
		t.Fatal(got)
	}
}

func TestKeptTradePlanRemovalIsLocalAndVisibleInMyMoves(t *testing.T) {
	t.Parallel()
	a, transport := realTradeBuilderApp(t)
	r := realDeal()
	r.Action = "save"
	saved := builderCall(t, a, r).Preview
	opened := 0
	a.core.OpenURL = func(context.Context, string) { opened++ }
	m, stamp, _ := a.core.League.ContractsView()
	wrong := a.tradesAPI().TargetTradeBuilder(tradesapp.RawTradeBuilder{Mine: r.Other, Action: "remove", ID: saved.ID})
	if wrong.Status == report.ReadingOK {
		t.Fatal("another owner removed plan")
	}
	view := builderCall(t, a, tradesapp.RawTradeBuilder{Mine: r.Mine, Action: "remove", ID: saved.ID})
	if len(view.Plans) != 0 || view.Preview != nil {
		t.Fatalf("removed plan still on desk: %+v", view)
	}
	receipts, err := a.platformAPI().TargetMoves(r.Mine)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, receipt := range receipts {
		if receipt.CorrelationID == saved.ID {
			found = receipt.State == envelope.Withdrawn &&
				receipt.Audit[len(receipt.Audit)-1].Note == "Removed by you before it was handed off."
		}
	}
	if !found {
		t.Fatal("My moves lost the removed receipt")
	}
	after, nextStamp, _ := a.core.League.ContractsView()
	if opened != 0 || len(transport.requests) != 0 || !reflect.DeepEqual(m, after) || stamp != nextStamp {
		t.Fatal("local removal opened MFL, fetched or changed mirror")
	}
	if len(builderCall(t, a, tradesapp.RawTradeBuilder{Mine: r.Mine, Action: "read"}).Plans) != 0 {
		t.Fatal("removed plan reappeared on read")
	}
}

func TestTradeAndContractWhatIfExplainCarriedSalaries(t *testing.T) {
	t.Parallel()
	a, _ := realTradeBuilderApp(t)
	reading := builderReading(a, realDeal())
	body, err := json.Marshal(reading)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(body)), "inferred") ||
		!strings.Contains(string(body), "this season’s cap carried forward") {
		t.Fatalf("trade wording: %s", body)
	}
	whatif := a.contracts().TargetWhatIf(contractsapp.RawWhatIfRequest{Mine: "0001", Action: "read"})
	if strings.Contains(strings.ToLower(whatif.HowCounted), "inferred") ||
		!strings.Contains(whatif.HowCounted, "carry today’s MFL annual salary") {
		t.Fatal(whatif.HowCounted)
	}
}

func TestTradeExitPlanRemovalIsLocal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ mine, action string }{{"0025", "reject"}, {"0001", "revoke"}} {
		t.Run(tc.action, func(t *testing.T) {
			t.Parallel()
			a, transport, offer := tradeDraftApp(t)
			opened := 0
			a.core.OpenURL = func(context.Context, string) { opened++ }
			_, held, err := a.tradeState(tc.mine)
			if err != nil {
				t.Fatal(err)
			}
			r, err := a.draftTradeExit(a.core.Ctx, tradesapp.RawTradeBuilder{Mine: tc.mine, Action: tc.action, ID: offer.ID}, held)
			if err != nil || r.State != envelope.Ready {
				t.Fatal(r, err)
			}
			before, stamp, _ := a.core.League.ContractsView()
			if err := a.removeTradePlan(a.core.Ctx, tradesapp.RawTradeBuilder{Mine: "9999", ID: r.CorrelationID}); err == nil {
				t.Fatal("another franchise removed plan")
			}
			if err := a.removeTradePlan(a.core.Ctx, tradesapp.RawTradeBuilder{Mine: tc.mine, ID: r.CorrelationID}); err != nil {
				t.Fatal(err)
			}
			receipts, err := a.platformAPI().TargetMoves(tc.mine)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, receipt := range receipts {
				if receipt.CorrelationID == r.CorrelationID {
					found = receipt.State == envelope.Withdrawn &&
						receipt.Audit[len(receipt.Audit)-1].Note == envelope.RemovePlanNote
				}
			}
			if !found {
				t.Fatal("removed receipt missing")
			}
			r, err = a.draftTradeExit(a.core.Ctx, tradesapp.RawTradeBuilder{Mine: tc.mine, Action: tc.action, ID: offer.ID}, held)
			if err != nil {
				t.Fatal(err)
			}
			e, err := a.core.Moves.Get(a.core.Ctx, r.CorrelationID)
			if err != nil {
				t.Fatal(err)
			}
			handed, err := e.HandOff(a.core.MovesNow())
			if err != nil {
				t.Fatal(err)
			}
			if err := a.core.Moves.Save(a.core.Ctx, handed); err != nil {
				t.Fatal(err)
			}
			if err := a.removeTradePlan(a.core.Ctx, tradesapp.RawTradeBuilder{Mine: tc.mine, ID: r.CorrelationID}); err == nil {
				t.Fatal("removed plan after hand-off")
			}
			kept, err := a.core.Moves.Get(a.core.Ctx, r.CorrelationID)
			if err != nil || kept.State() != envelope.HandedOff {
				t.Fatal(kept.Receipt(), err)
			}
			after, next, _ := a.core.League.ContractsView()
			if opened != 0 || len(transport.requests) != 0 || stamp != next || !reflect.DeepEqual(before, after) {
				t.Fatal("removal touched MFL or mirror")
			}
		})
	}
}

func TestTradeReceiptPickUsesBuilderName(t *testing.T) {
	t.Parallel()
	a, transport := realTradeBuilderApp(t)
	r := realDeal()
	r.Action = "save"
	r.Give = []string{"FP_0001_2027_1"}
	saved := builderCall(t, a, r).Preview
	receipts, err := a.platformAPI().TargetMoves(r.Mine)
	if err != nil {
		t.Fatal(err)
	}
	want := tradesapp.FuturePickName(2027, 1, "0001", a.core.TargetSnapshot)
	found := false
	for _, receipt := range receipts {
		if receipt.CorrelationID != saved.ID {
			continue
		}
		for _, asset := range receipt.TradeAssets {
			if asset.Token == r.Give[0] {
				found = asset.Name == want
			}
		}
	}
	if !found || len(transport.requests) != 0 {
		t.Fatal("pick name missing or fetched", receipts)
	}
}

func TestOwnerActionsRefuseWithoutChangingState(t *testing.T) {
	a, transport := realTradeBuilderApp(t)
	raw := realDeal()
	raw.Action = "save"
	kept := builderCall(t, a, raw).Preview.Receipt
	if kept == nil {
		t.Fatal("own team could not save")
	}
	opened := 0
	a.core.OpenURL = func(context.Context, string) { opened++ }
	for _, identity := range []string{"0025", ""} {
		a.core.MyFranchise = identity
		want := "You can only act for your own team."
		if identity == "" {
			want = "Connect your MFL account to act for your team."
		}
		calls := map[string]func() string{
			"Acquisition": func() string {
				return refusal(a.contracts().TargetAcquisition(contractsapp.RawAcquisition{Mine: "0001"}))
			},
			"ContractActs": func() string {
				return refusal(a.contracts().TargetContractActs(contractsapp.RawContractAct{Mine: "0001", Action: "read"}))
			},
			"DraftLineup": func() string { _, err := a.contracts().TargetDraftLineup("0001", nil); return errText(err) },
			"ReleasePlans": func() string {
				return refusal(a.contracts().TargetReleasePlans(contractsapp.RawReleaseRequest{Mine: "0001", Action: "read"}))
			},
			"WhatIf": func() string {
				return refusal(a.contracts().TargetWhatIf(contractsapp.RawWhatIfRequest{Mine: "0001", Action: "reset"}))
			},
			"DraftIR":          func() string { _, err := a.platformAPI().TargetDraftIR("0001", "11675"); return errText(err) },
			"DraftTaxi":        func() string { _, err := a.platformAPI().TargetDraftTaxi("0001", "11675"); return errText(err) },
			"HandOff":          func() string { _, err := a.platformAPI().TargetHandOff(kept.CorrelationID); return errText(err) },
			"WithdrawRoster":   func() string { _, err := a.platformAPI().TargetWithdrawRoster(kept.CorrelationID); return errText(err) },
			"DraftTradeAccept": func() string { _, err := a.tradesAPI().TargetDraftTradeAccept("0001", "77"); return errText(err) },
			"TradeBuilder":     func() string { return refusal(builderReading(a, realDeal())) },
		}
		for name, call := range calls {
			before := ownerActionState(t, a)
			if got := call(); got != want {
				t.Fatalf("%s identity %q: %s", name, identity, got)
			}
			if after := ownerActionState(t, a); after != before || opened != 0 || len(transport.requests) != 0 {
				t.Fatalf("%s changed state", name)
			}
		}
	}
	// Supplying one's own actor does not authorize a stored plan belonging to somebody else.
	a.core.MyFranchise = "0025"
	calls := map[string]func() string{
		"ContractActs plan": func() string {
			return refusal(a.contracts().TargetContractActs(contractsapp.RawContractAct{Mine: "0025", Action: "withdraw", ID: kept.CorrelationID}))
		},
		"ReleasePlans plan": func() string {
			return refusal(a.contracts().TargetReleasePlans(contractsapp.RawReleaseRequest{Mine: "0025", Action: "handoff", ID: kept.CorrelationID, Confirmed: true}))
		},
	}
	for _, action := range []string{"remove", "handoff"} {
		calls["TradeBuilder "+action] = func() string {
			return refusal(a.tradesAPI().TargetTradeBuilder(tradesapp.RawTradeBuilder{Mine: "0025", Action: action, ID: kept.CorrelationID}))
		}
	}
	for name, call := range calls {
		before := ownerActionState(t, a)
		if got := call(); got != "You can only act for your own team." {
			t.Fatalf("%s: %s", name, got)
		}
		if ownerActionState(t, a) != before || opened != 0 {
			t.Fatalf("%s changed state", name)
		}
	}
}

func ownerActionState(t *testing.T, a *App) string {
	t.Helper()
	var sessions []state.WhatIfSession
	var receipts []envelope.Receipt
	for _, fid := range []string{"0001", "0025"} {
		s, _, err := state.NewWhatIfSessions(a.core.WhatifPools).Load(a.core.Ctx, fid)
		if err != nil {
			t.Fatal(err)
		}
		sessions = append(sessions, s)
		plans, err := a.core.Moves.List(a.core.Ctx, ingestion.LeagueID, fid)
		if err != nil {
			t.Fatal(err)
		}
		receipts = append(receipts, plans...)
	}
	mirror, stamp, _ := a.core.League.ContractsView()
	body, err := json.Marshal(struct {
		Sessions []state.WhatIfSession
		Plans    []envelope.Receipt
		Mirror   state.MirrorSnapshot
		Stamp    string
	}{sessions, receipts, mirror, stamp})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestTradeCounterpartyProjectionIgnoresPrivateWhatIf(t *testing.T) {
	a, _ := realTradeBuilderApp(t)
	first := builderCall(t, a, realDeal()).Preview
	sessions := state.NewWhatIfSessions(a.core.WhatifPools)
	if _, found, err := sessions.Load(a.core.Ctx, "0025"); err != nil || found {
		t.Fatal("preview opened other owner's session", err)
	}
	a.core.MyFranchise = "0025"
	cut := a.contracts().TargetWhatIf(contractsapp.RawWhatIfRequest{Mine: "0025", Action: "cut", Player: "15761"})
	if cut.Status != report.ReadingOK || cut.Value.Notice != "" {
		t.Fatal(cut)
	}
	before, found, err := sessions.Load(a.core.Ctx, "0025")
	if err != nil || !found {
		t.Fatal(err)
	}
	a.core.MyFranchise = "0001"
	second := builderCall(t, a, realDeal()).Preview
	after, _, err := sessions.Load(a.core.Ctx, "0025")
	if err != nil || !reflect.DeepEqual(before, after) || !reflect.DeepEqual(first.Teams, second.Teams) || !reflect.DeepEqual(first.Reasons, second.Reasons) {
		t.Fatal("counterparty private plans entered projection or changed", err)
	}
}
