package main

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
	pools, err := db.Open(a.ctx, filepath.Join(t.TempDir(), "whatif.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	})
	a.whatifPools = pools
	a.whatif = state.New(pools, ingestion.LeagueID, 2026, a.rulebook)
	if err := a.whatif.Initialize(a.ctx, a.league); err != nil {
		t.Fatal(err)
	}
	a.coordinator, err = transactions.New(a.whatif.Writer(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	a.clockChanged = func(context.Context) {}
	if err := a.TargetSetWindow("trade_deadline", time.Now().Add(time.Hour).UTC().Format(time.RFC3339)); err != nil {
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
	a.targetSnapshot = snap
	a.hasTargetSnapshot = true
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
	if _, err := a.league.Replace(a.ctx, m); err != nil {
		t.Fatal(err)
	}
	if err := a.rulebook.SetOverride(a.ctx, "cap", "salaryCapAmount", "125", "fixture rulebook"); err != nil {
		t.Fatal(err)
	}
	picks, err := draftreports.ParseFuture([]byte(seasonFixture(t, "draftreports", "future-2026.json")))
	if err != nil {
		t.Fatal(err)
	}
	a.seasonFuturePicks = heldSeasonFeed[[]draftreports.FuturePick]{value: picks, fetchedAt: a.movesNow()}
	a.openURL = func(context.Context, string) {}
	transport.requests = nil
	return a, transport
}
func builderCall(t *testing.T, a *App, r RawTradeBuilder) TradeBuilderView {
	t.Helper()
	reading := a.TargetTradeBuilder(r)
	if reading.Status != ReadingOK {
		t.Fatalf("builder %s: %+v", r.Action, reading)
	}
	return reading.Value
}
func realDeal() RawTradeBuilder {
	return RawTradeBuilder{Mine: "0001", Other: "0025", Action: "preview", Give: []string{"11675"}, Get: []string{"15761"}, Rationale: "Balance salary with a younger starter."}
}
func TestTradeBuilderRealRosterExactCentsBothSides(t *testing.T) {
	a, transport := realTradeBuilderApp(t)
	m, _, _ := a.league.ContractsView()
	// Sub-dollar cents on two observed contracts expose ledger rounding mistakes.
	for i := range m.Players {
		switch m.Players[i].MFLID {
		case "11675":
			m.Players[i].Salary += 37
		case "15761":
			m.Players[i].Salary += 83
		}
	}
	if _, err := a.league.Replace(a.ctx, m); err != nil {
		t.Fatal(err)
	}
	m, stamp, used := a.league.ContractsView()
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
	after, nextStamp, _ := a.league.ContractsView()
	if !reflect.DeepEqual(m, after) || stamp != nextStamp || len(transport.requests) != 0 {
		t.Fatal("projection changed mirror or fetched")
	}
	if !strings.Contains(preview.Post, "a year through 2026") || strings.Contains(preview.Post, "11675") {
		t.Fatal(preview.Post)
	}
	body, err := json.Marshal(a.TargetTradeBuilder(realDeal()))
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
	a, transport := realTradeBuilderApp(t)
	r := realDeal()
	r.Give = []string{"14167"}
	r.Action = "save"
	saved := builderCall(t, a, r).Preview
	if saved.Receipt.State != envelope.Ready {
		t.Fatalf("not ready: %+v", saved)
	}
	id := saved.ID
	plans, err := state.NewTradePlans(a.whatifPools).List(a.ctx, r.Mine)
	if err != nil || len(plans) != 1 || plans[0].ID != id {
		t.Fatal(plans, err)
	}
	opens := 0
	a.openURL = func(_ context.Context, target string) {
		opens++
		if target != "https://www47.myfantasyleague.com/2026/options?L=14432&O=05" {
			t.Fatal(target)
		}
	}
	if err := a.TargetSetWindow("trade_deadline", time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	failed := a.TargetTradeBuilder(RawTradeBuilder{Mine: r.Mine, Action: "handoff", ID: id})
	if failed.Status != ReadingFailed || opens != 0 {
		t.Fatal("deadline bypass", failed)
	}
	blocked := builderCall(t, a, r).Preview
	if blocked.Receipt.State != envelope.Blocked || !strings.Contains(strings.Join(blocked.Reasons, " "), "deadline") {
		t.Fatal(blocked)
	}
	if err := a.TargetSetWindow("trade_deadline", time.Now().Add(time.Hour).UTC().Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	handed := a.TargetTradeBuilder(RawTradeBuilder{Mine: r.Mine, Action: "handoff", ID: id})
	if handed.Status != ReadingOK || opens != 1 {
		t.Fatal(handed, opens)
	}
	if _, err := a.TargetHandOff(id); err == nil {
		t.Fatal("generic handoff bypassed trade pre-flight")
	}
	wrong := realDeal()
	wrong.Give = []string{"16387"}
	preview := builderCall(t, a, wrong).Preview
	if len(preview.Reasons) == 0 {
		t.Fatal("wrong ownership passed")
	}
	picks := builderCall(t, a, RawTradeBuilder{Mine: r.Mine, Action: "read"}).Teams
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
	a, transport := realTradeBuilderApp(t)
	m, _, used := a.league.ContractsView()
	capAmount := domain.Money(125_0000_0000)
	m.Adjustments = append(m.Adjustments, state.Adjustment{ID: "forced-cap-fixture", FranchiseID: "0001", Amount: capAmount - used["0001"] - 100_000_000})
	if _, err := a.league.Replace(a.ctx, m); err != nil {
		t.Fatal(err)
	}
	raw := RawAcquisition{Mine: "0001", Player: "15761"}
	noCut := a.TargetAcquisition(raw)
	if noCut.Status != ReadingOK || !strings.Contains(strings.Join(noCut.Value.Reasons, " "), "does not fit") || !strings.HasPrefix(noCut.Value.Spot, "One active roster spot") {
		t.Fatal(noCut)
	}
	raw.Cut = "14109"
	cut := a.TargetAcquisition(raw)
	if cut.Status != ReadingOK || cut.Value.DeadCap.Cents != 945_000_000 || len(cut.Value.Seasons) != 3 {
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
	fit := a.TargetAcquisition(raw)
	if fit.Status != ReadingOK {
		t.Fatal(fit)
	}
	if strings.Contains(strings.Join(fit.Value.Reasons, " "), "does not fit") {
		t.Fatal(fit)
	}
	if len(transport.requests) != 0 {
		t.Fatal("acquisition fetched")
	}
	after, _, _ := a.league.ContractsView()
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
func TestTradeBuilderBoundaryRejectsInvalidInputs(t *testing.T) {
	for _, r := range []RawTradeBuilder{{Mine: "x", Action: "read"}, {Mine: "0001", Action: "submit"},
		{Mine: "0001", Action: "revoke"}, {Mine: "0001", Other: "0001", Action: "save"},
		{Mine: "0001", Other: "0002", Action: "save", Give: []string{"99", "0099"}, Get: []string{"12345"}},
		{Mine: "0001", Other: "0002", Action: "save", Give: []string{"BB_2"}, Get: []string{"12345"}}} {
		if err := r.Validate(); err == nil {
			t.Fatalf("accepted %+v", r)
		}
	}
}
func TestNoMutatingTradeLinkInSources(t *testing.T) {
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
	a, _, offer := tradeDraftApp(t)
	// Exit preparation does not require a builder ledger, only a keyed held pending offer.
	for _, tc := range []struct{ mine, action string }{{"0025", "reject"}, {"0001", "revoke"}} {
		snap, held, err := a.tradeState(tc.mine)
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := a.draftTradeExit(a.ctx, RawTradeBuilder{Mine: tc.mine, Action: tc.action, ID: offer.ID}, held)
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
	if _, err := a.draftTradeExit(a.ctx, RawTradeBuilder{Mine: "0025", Action: "revoke", ID: offer.ID}, held); err == nil {
		t.Fatal("other owner's revoke allowed")
	}
}

func TestTradeBuilderIRTaxiAndRemovedWhatIfAssets(t *testing.T) {
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
	cut := a.TargetWhatIf(RawWhatIfRequest{Mine: "0001", Action: "cut", Player: "14167"})
	if cut.Status != ReadingOK || cut.Value.Notice != "" {
		t.Fatal(cut)
	}
	removed := builderCall(t, a, raw).Preview
	if !strings.Contains(strings.Join(removed.Reasons, " "), "removed in what-if") {
		t.Fatal(removed)
	}
}
func TestTradeBuilderNextSeasonToTheCentAndPickOnly(t *testing.T) {
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
	a, _, offer := tradeDraftApp(t)
	a.seasonPendingTrades.value = nil
	req := a.tradeRequest("0001", offer)
	req.Trade.TradeID = "local"
	e, err := envelope.DraftTradeAct(a.movesNow(), "trade.propose", req, nil, func() string { return "watch-propose" })
	if err != nil {
		t.Fatal(err)
	}
	handed := a.movesNow().Add(time.Minute)
	e, err = e.HandOff(handed)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.moves.Save(a.ctx, e); err != nil {
		t.Fatal(err)
	}
	for step, want := range []envelope.State{envelope.NotYetDone, envelope.NotYetDone, envelope.DOTReview} {
		now := handed.Add(time.Duration(step+1) * time.Minute)
		a.movesNow = func() time.Time { return now }
		a.movesRefresh = func(_ context.Context, source envelope.Source) error {
			switch source {
			case envelope.PendingTrades:
				a.seasonPendingTrades.fetchedAt = now
				a.seasonPendingTrades.value = nil
				if step == 1 {
					offer.Proposed = handed.Add(time.Second)
					a.seasonPendingTrades.value = append(a.seasonPendingTrades.value, offer)
				}
			case envelope.Transactions:
				a.seasonTransactions.fetchedAt = now
				a.seasonTransactions.value = nil
			case envelope.Rosters, envelope.Contracts, envelope.Lineups:
				t.Fatalf("unexpected feed %s", source)
			}
			return nil
		}
		if err := a.checkMoves(a.ctx); err != nil {
			t.Fatal(err)
		}
		saved, err := a.moves.Get(a.ctx, e.ID())
		if err != nil || saved.State() != want {
			t.Fatal(saved.Receipt(), err)
		}
		if step == 1 && saved.Receipt().Audit[len(saved.Receipt().Audit)-1].Note != "Proposed on MFL" {
			t.Fatal("same-state proposal proof was dropped")
		}
	}
}

func TestRosterFigureShowsBeforeAfterAndChange(t *testing.T) {
	if f := rosterFigure("Active roster · 35–48", 37, 36); f.Before != "37" || f.After != "36" || f.Change != "-1" {
		t.Fatalf("one out: %+v", f)
	}
	if f := rosterFigure("Taxi squad · maximum 8", 6, 6); f.Change != "" {
		t.Fatalf("no change shows none: %+v", f)
	}
}

func TestTradeBuilderLandingRichAssets(t *testing.T) {
	a, _ := realTradeBuilderApp(t)
	view := builderCall(t, a, RawTradeBuilder{Mine: "0001", Action: "read"})
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
	value := builderPlayerAsset(state.MirrorPlayer{MFLID: "13593", Salary: 123456789, ContractYear: 2028},
		snapshot.Player{Position: domain.PosWR, Team: "DAL"}, "Name", AssetValue{ID: "13593",
			Now: AssetCell{Text: "12.00 PPG", Held: true}, Dynasty: AssetCell{Text: "0.3 wins", Held: true}}, 2026)
	if value.Contract != "$1,234,567.89 · 3 years" || value.Now.Text != "12.00 PPG" || value.Dynasty.Text != "0.3 wins" {
		t.Fatal(value)
	}
}

type landingPickModel struct{}

func (landingPickModel) player(string, time.Time) dynastyValue { return dynastyValue{} }
func (landingPickModel) pick(year, round int, orig string, at time.Time) dynastyValue {
	if year != 2027 || round != 1 || orig != "0001" || at.IsZero() {
		return dynastyValue{}
	}
	return dynastyValue{Valued: true, Discounted: 0.78, Seasons: 5}
}

func TestTradeBuilderLandingPickValue(t *testing.T) {
	a, _ := realTradeBuilderApp(t)
	var model assetModel = landingPickModel{}
	a.assetModel.Store(&model)
	got := a.builderPickValue(2027, 1, "0001", time.Now())
	if !got.Held || got.Text != "0.78 wins" {
		t.Fatal(got)
	}
}

func TestKeptTradePlanRemovalIsLocalAndVisibleInMyMoves(t *testing.T) {
	a, transport := realTradeBuilderApp(t)
	r := realDeal()
	r.Action = "save"
	saved := builderCall(t, a, r).Preview
	opened := 0
	a.openURL = func(context.Context, string) { opened++ }
	m, stamp, _ := a.league.ContractsView()
	wrong := a.TargetTradeBuilder(RawTradeBuilder{Mine: r.Other, Action: "remove", ID: saved.ID})
	if wrong.Status == ReadingOK {
		t.Fatal("another owner removed plan")
	}
	view := builderCall(t, a, RawTradeBuilder{Mine: r.Mine, Action: "remove", ID: saved.ID})
	if len(view.Plans) != 0 || view.Preview != nil {
		t.Fatalf("removed plan still on desk: %+v", view)
	}
	receipts, err := a.TargetMoves(r.Mine)
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
	after, nextStamp, _ := a.league.ContractsView()
	if opened != 0 || len(transport.requests) != 0 || !reflect.DeepEqual(m, after) || stamp != nextStamp {
		t.Fatal("local removal opened MFL, fetched or changed mirror")
	}
	if len(builderCall(t, a, RawTradeBuilder{Mine: r.Mine, Action: "read"}).Plans) != 0 {
		t.Fatal("removed plan reappeared on read")
	}
}

func TestTradeAndContractWhatIfExplainCarriedSalaries(t *testing.T) {
	a, _ := realTradeBuilderApp(t)
	reading := a.TargetTradeBuilder(realDeal())
	body, err := json.Marshal(reading)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(body)), "inferred") ||
		!strings.Contains(string(body), "this season’s cap carried forward") {
		t.Fatalf("trade wording: %s", body)
	}
	whatif := a.TargetWhatIf(RawWhatIfRequest{Mine: "0001", Action: "read"})
	if strings.Contains(strings.ToLower(whatif.HowCounted), "inferred") ||
		!strings.Contains(whatif.HowCounted, "carry today’s MFL annual salary") {
		t.Fatal(whatif.HowCounted)
	}
}

func TestTradeExitPlanRemovalIsLocal(t *testing.T) {
	for _, tc := range []struct{ mine, action string }{{"0025", "reject"}, {"0001", "revoke"}} {
		t.Run(tc.action, func(t *testing.T) {
			a, transport, offer := tradeDraftApp(t)
			opened := 0
			a.openURL = func(context.Context, string) { opened++ }
			_, held, err := a.tradeState(tc.mine)
			if err != nil {
				t.Fatal(err)
			}
			r, err := a.draftTradeExit(a.ctx, RawTradeBuilder{Mine: tc.mine, Action: tc.action, ID: offer.ID}, held)
			if err != nil || r.State != envelope.Ready {
				t.Fatal(r, err)
			}
			before, stamp, _ := a.league.ContractsView()
			if err := a.removeTradePlan(a.ctx, RawTradeBuilder{Mine: "9999", ID: r.CorrelationID}); err == nil {
				t.Fatal("another franchise removed plan")
			}
			if err := a.removeTradePlan(a.ctx, RawTradeBuilder{Mine: tc.mine, ID: r.CorrelationID}); err != nil {
				t.Fatal(err)
			}
			receipts, err := a.TargetMoves(tc.mine)
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
			r, err = a.draftTradeExit(a.ctx, RawTradeBuilder{Mine: tc.mine, Action: tc.action, ID: offer.ID}, held)
			if err != nil {
				t.Fatal(err)
			}
			e, err := a.moves.Get(a.ctx, r.CorrelationID)
			if err != nil {
				t.Fatal(err)
			}
			handed, err := e.HandOff(a.movesNow())
			if err != nil {
				t.Fatal(err)
			}
			if err := a.moves.Save(a.ctx, handed); err != nil {
				t.Fatal(err)
			}
			if err := a.removeTradePlan(a.ctx, RawTradeBuilder{Mine: tc.mine, ID: r.CorrelationID}); err == nil {
				t.Fatal("removed plan after hand-off")
			}
			kept, err := a.moves.Get(a.ctx, r.CorrelationID)
			if err != nil || kept.State() != envelope.HandedOff {
				t.Fatal(kept.Receipt(), err)
			}
			after, next, _ := a.league.ContractsView()
			if opened != 0 || len(transport.requests) != 0 || stamp != next || !reflect.DeepEqual(before, after) {
				t.Fatal("removal touched MFL or mirror")
			}
		})
	}
}

func TestTradeReceiptPickUsesBuilderName(t *testing.T) {
	a, transport := realTradeBuilderApp(t)
	r := realDeal()
	r.Action = "save"
	r.Give = []string{"FP_0001_2027_1"}
	saved := builderCall(t, a, r).Preview
	receipts, err := a.TargetMoves(r.Mine)
	if err != nil {
		t.Fatal(err)
	}
	want := futureTradePickName(2027, 1, "0001", a.targetSnapshot)
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
