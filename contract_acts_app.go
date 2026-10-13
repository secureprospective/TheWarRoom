package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/contractacts"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type RawContractAct struct {
	Mine        string `json:"mine"`
	Player      string `json:"player"`
	Action      string `json:"action"`
	Intent      string `json:"intent"`
	ID          string `json:"id"`
	Level       int    `json:"level"`
	Years       int    `json:"years"`
	Source      int    `json:"source"`
	Destination int    `json:"destination"`
	Move        string `json:"move"`
}

func (r RawContractAct) Validate() error {
	if err := (snapshot.RawContractScope{Mine: r.Mine, Sort: "name"}).Validate(); err != nil {
		return fmt.Errorf("contract Act scope: %w", err)
	}
	if r.Mine == "" || !slices.Contains([]string{"read", "options", "preview", "draft", "handoff", "posted", "withdraw", "rebase"}, r.Action) {
		return fmt.Errorf("contract Act: franchise and known action required")
	}
	if r.Action == "options" {
		if _, err := playerid.New(r.Player); err != nil {
			return fmt.Errorf("contract Act player: %w", err)
		}
	}
	if r.Action == "preview" || r.Action == "draft" {
		return r.validateTerms()
	}
	if !slices.Contains([]string{"read", "options", "preview", "draft"}, r.Action) && r.ID == "" {
		return fmt.Errorf("contract Act: plan id required")
	}
	return nil
}

func (r RawContractAct) validateTerms() error {
	if _, err := playerid.New(r.Player); err != nil {
		return fmt.Errorf("contract Act player: %w", err)
	}
	if !envelope.IsContract(r.Intent) || r.Level < 1 || r.Level > len(contractacts.TenderLevels()) || r.Years < 1 || r.Years > contractacts.MaxResignYears {
		return fmt.Errorf("contract Act: known intent, tender level 1–6 and term 1–4 required")
	}
	if r.Move != "" {
		amount, err := domain.ParseMoneyMillions(r.Move)
		if err != nil || amount < 0 || amount > 3*contractacts.Million {
			return fmt.Errorf("contract Act: move must be $0–$3M in millions")
		}
	}
	return nil
}

type ContractActFigure struct {
	Season int    `json:"season"`
	Before string `json:"before"`
	After  string `json:"after"`
	Change string `json:"change"`
}
type ContractActChoice struct {
	Value int    `json:"value"`
	Label string `json:"label"`
}
type ContractActView struct {
	ID          string              `json:"id"`
	Intent      string              `json:"intent"`
	Act         string              `json:"act"`
	Levels      []ContractActChoice `json:"levels"`
	Years       []int               `json:"years"`
	TeamFigures []WhatIfFigure      `json:"teamFigures"`
	Player      string              `json:"player"`
	Subject     string              `json:"subject"`
	Label       string              `json:"label"`
	Price       string              `json:"price"`
	Post        string              `json:"post"`
	How         string              `json:"how"`
	Reasons     []string            `json:"reasons"`
	Figures     []ContractActFigure `json:"figures"`
	Receipt     envelope.Receipt    `json:"receipt"`
	CanHandOff  bool                `json:"canHandOff"`
	CanPost     bool                `json:"canPost"`
	CanWithdraw bool                `json:"canWithdraw"`
	CanRebase   bool                `json:"canRebase"`
}

func (a *App) TargetContractActs(raw RawContractAct) Reading[[]ContractActView] {
	if err := raw.Validate(); err != nil {
		return contractActsReading(nil, err)
	}
	if err := a.ready(); err != nil {
		return contractActsReading(nil, err)
	}
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	a.movesMu.Lock()
	defer a.movesMu.Unlock()
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()
	if err := a.revalidateContractPlans(ctx); err != nil {
		return contractActsReading(nil, err)
	}
	if raw.Action == "options" {
		return a.contractActOptions(ctx, raw)
	}
	if raw.Action == "preview" {
		f, err := a.contractActFacts(ctx, raw)
		if err != nil {
			return contractActsReading(nil, err)
		}
		return contractActsReading([]ContractActView{contractActView(envelope.Receipt{}, f)}, nil)
	}
	if err := a.contractActAction(ctx, raw); err != nil {
		return contractActsReading(nil, err)
	}
	receipts, err := a.moves.List(ctx, ingestion.LeagueID, raw.Mine)
	if err != nil {
		return contractActsReading(nil, fmt.Errorf("contract plans: %w", err))
	}
	out := []ContractActView{}
	for _, r := range receipts {
		if !envelope.IsContract(r.Spec.Intent) {
			continue
		}
		f, err := a.moves.ContractFacts(ctx, r.CorrelationID)
		if err != nil {
			return contractActsReading(nil, err)
		}
		out = append(out, contractActView(r, f))
	}
	return contractActsReading(out, nil)
}
func contractActsReading(v []ContractActView, err error) Reading[[]ContractActView] {
	return contractReading(v, "", "local contract plans; MFL mirror", "Rulebook §§7,9–11; Math Rules §§0,3,5–7. All calculations in exact cents, then nearest $10,000. Future baseline cells are flat-filled inferences, not observations. Tag, extension and restructure follow the coordinator phase policy. Tender and re-sign are offseason only. Unknown history is for commissioner verification, not a block. Plans never write MFL or post to ProBoards.", len(v), err)
}

func contractIntentLabel(intent string) string {
	labels := map[string]string{"rfa.tender": "Tender him (RFA)", "fa.resign": "Re-sign him", "contract.tag": "Franchise tag", "contract.extend": "Extend him", "contract.restructure": "Restructure"}
	return labels[intent]
}

// contractActChoices is what an Act's form offers: tender levels priced from the rule table, and term lengths.
func contractActChoices(intent string) ([]ContractActChoice, []int) {
	levels, years := []ContractActChoice{}, []int{}
	if intent == "rfa.tender" || intent == "fa.resign" {
		for i, l := range contractacts.TenderLevels() {
			levels = append(levels, ContractActChoice{i + 1, fmt.Sprintf("%s · $%.2fM", l.Round, l.Price.Millions())})
		}
	}
	most := map[string]int{"fa.resign": contractacts.MaxResignYears, "contract.extend": contractacts.MaxExtendYears}[intent]
	for y := 1; y <= most; y++ {
		years = append(years, y)
	}
	return levels, years
}
func mirrorContractEvidence(m state.MirrorSnapshot, mine, player string) envelope.ContractEvidence {
	p, found := releasePlayer(m, mine, player)
	if !found {
		return envelope.ContractEvidence{}
	}
	id, err := playerid.New(player)
	if err != nil {
		return envelope.ContractEvidence{}
	}
	cells := []contractacts.Cell{}
	for year := m.Season; year <= max(m.Season, p.ContractYear); year++ {
		cells = append(cells, contractacts.Cell{Season: year, Salary: p.Salary})
	}
	return envelope.ContractEvidence{Player: id, FranchiseID: mine, Cells: cells, Observed: p.ContractYear == m.Season, Status: p.ContractStatus, Info: p.ContractInfo}
}
func (a *App) contractActFacts(ctx context.Context, r RawContractAct) (envelope.ContractFacts, error) {
	m, _, _ := a.league.ContractsView()
	p, found := releasePlayer(m, r.Mine, r.Player)
	if !found {
		return envelope.ContractFacts{}, fmt.Errorf("contract Act: player is not on my roster")
	}
	phase, err := a.whatif.CurrentPhase(ctx)
	if err != nil {
		return envelope.ContractFacts{}, fmt.Errorf("contract Act phase: %w", err)
	}
	evidence := mirrorContractEvidence(m, r.Mine, r.Player)
	a.targetSnapshotMu.Lock()
	directory := a.targetSnapshot
	a.targetSnapshotMu.Unlock()
	rows := snapshot.ContractRows(m, directory, a.rulebook.FranchiseNames())
	row := snapshot.ContractRow{Name: r.Player}
	for _, candidate := range rows {
		if candidate.ID == r.Player {
			row = candidate
			break
		}
	}
	f := envelope.ContractFacts{Player: evidence.Player, Name: row.Name, Position: row.Position, Season: m.Season, Intent: r.Intent, Level: r.Level, Years: r.Years, Source: r.Source, Destination: r.Destination, Baseline: evidence.Cells, Status: p.ContractStatus, Info: p.ContractInfo, Reasons: []string{}}
	if window := envelope.ContractWindow(r.Intent, phase); window != "" {
		f.Reasons = append(f.Reasons, window)
	}
	if !contractacts.Eligible(r.Intent, f.Status, f.Baseline, f.Season, row.Position) {
		f.Reasons = append(f.Reasons, "This contract cannot take this Act.")
	}
	if r.Move != "" {
		f.Move, err = domain.ParseMoneyMillions(r.Move)
		if err != nil {
			return f, fmt.Errorf("contract Act move: %w", err)
		}
	}
	terms, priceErr := contractActTerms(f, row.Position, rows, p.Salary)
	f.Terms = terms
	if priceErr != nil {
		f.Reasons = append(f.Reasons, "Terms: "+priceErr.Error())
	}
	f.Reasons = append(f.Reasons, contractEvidenceReasons(f)...)
	if priceErr == nil {
		if err := a.contractCapFacts(ctx, r.Mine, &f); err != nil {
			return f, err
		}
	}

	return f, nil
}
func contractEvidenceReasons(f envelope.ContractFacts) []string {
	reasons := []string{}
	switch f.Intent {
	case "rfa.tender", "fa.resign", "contract.tag":
		reasons = append(reasons, fmt.Sprintf("Using his %d salary, %s, as prior-season BASE for this expiring contract.", f.Season, snapshot.ContractAmountOf(f.Baseline[0].Salary).Text))
	}
	switch f.Intent {
	case "fa.resign":
		reasons = append(reasons, "Not checked: whether another owner has made a competing offer")
	case "contract.tag":
		reasons = append(reasons, "Not checked: earlier franchise tags and this season's team tag allowance", "Not checked: exclusions for playoff signings from the position salary pool")
		if f.Status == domain.CStatusFT2 {
			reasons = append(reasons, "He has already had two consecutive franchise tags.")
		}
		if f.Status == domain.CStatusFT1 {
			reasons = append(reasons, "Not checked: first tag price; using held salary for his second tag")
		}
	case "contract.extend":
		reasons = append(reasons, "Not checked: prior extension and this season's team extension allowance")
	case "contract.restructure":
		reasons = append(reasons, "Not checked: restructure allowance already used on this contract or team, and any extension unlock")
	}
	if len(f.Baseline) > 1 {
		reasons = append(reasons, "Not checked: future salary amounts; later years assume today's salary")
	}
	return reasons
}
func contractActTerms(f envelope.ContractFacts, position string, rows []snapshot.ContractRow, salary domain.Money) (contractacts.Terms, error) {
	switch f.Intent {
	case "rfa.tender", "fa.resign":
		tender, err := contractacts.Tender(salary, f.Level)
		if err != nil {
			return contractacts.Terms{}, fmt.Errorf("RFA price: %w", err)
		}
		price, years, how := tender, 1, "§7: tender = greater of level price and 110% of prior BASE salary; exact cents then $10,000 snap. Prior-season BASE uses the expiring contract’s held salary."
		if f.Intent == "fa.resign" {
			price, err = contractacts.Resign(tender, f.Years)
			years = f.Years
			how = "§7 baseline re-sign with no competing offer: tender × (100% + 15% for each year after the first), 1–4 flat years; exact cents then $10,000 snap. Prior-season BASE uses the expiring contract’s held salary."
		}
		return contractacts.Flat(contractacts.NextSeason(f.Baseline, f.Season), years, price, how), err
	case "contract.tag":
		pool := []domain.Money{}
		for _, row := range rows {
			if row.Position == position {
				pool = append(pool, row.Salary.Cents)
			}
		}
		consecutive, first := 0, domain.Money(0)
		if f.Status == domain.CStatusFT1 {
			consecutive, first = 1, salary
		}
		if f.Status == domain.CStatusFT2 {
			consecutive = 2
		}
		price, err := contractacts.Tag(pool, salary, first, consecutive)
		return contractacts.Flat(contractacts.NextSeason(f.Baseline, f.Season), 1, price, "§9: average of the top five eligible BASE salaries at the position; if below prior salary use 120% of prior BASE, not max(average,120%). Second consecutive tag = 120% of first. Playoff signings excluded. Preview lacks history/exclusions and does not assert eligibility."), err
	case "contract.extend":
		terms, err := contractacts.Extend(f.Baseline, f.Season, f.Years, position, contractTagPrice(f, salary), false)
		if err != nil {
			return terms, fmt.Errorf("extension terms: %w", err)
		}
		return terms, nil
	case "contract.restructure":
		terms, err := contractacts.Restructure(f.Baseline, f.Source, f.Destination, f.Move, false)
		if err != nil {
			return terms, fmt.Errorf("restructure terms: %w", err)
		}
		return terms, nil
	default:
		return contractacts.Terms{}, fmt.Errorf("unsupported contract Act")
	}
}

func contractActView(r envelope.Receipt, f envelope.ContractFacts) ContractActView {
	label := contractIntentLabel(f.Intent) + " · " + f.Name
	figures := contractActFigures(f)
	price := snapshot.ContractAmountOf(f.Terms.Price).Text
	if len(f.Terms.Cells) == 0 {
		price = "Not priced"
		figures = []ContractActFigure{}
	}
	post := label + "\nOwner's contract request to the commissioner on ProBoards.\n"
	for _, c := range f.Terms.Cells {
		post += fmt.Sprintf("%d: %s\n", c.Season, snapshot.ContractAmountOf(c.Salary).Text)
	}
	post += contractPostTerms(f) + "\n"
	post += strings.Join(f.Reasons, "\n") + "\n"
	post += "Please process this request. Nothing has been posted or changed by the app."
	levels, years := contractActChoices(f.Intent)
	return ContractActView{ID: r.CorrelationID, Intent: f.Intent, Act: contractIntentLabel(f.Intent), Levels: levels, Years: years, TeamFigures: contractTeamView(f.TeamFigures), Player: f.Name, Subject: contractSubject(f), Label: label, Price: contractFigureSentence(f, price), Post: post, How: f.Terms.How, Reasons: f.Reasons, Figures: figures, Receipt: r, CanHandOff: r.State == envelope.Ready, CanPost: r.State == envelope.HandedOff, CanWithdraw: slices.Contains([]envelope.State{envelope.Draft, envelope.Blocked, envelope.Ready, envelope.HandedOff, envelope.NotYetDone, envelope.NotVerified, envelope.BidPending, envelope.Stale}, r.State), CanRebase: r.State == envelope.Stale}
}

// contractSubject names whose contract an Act changes, from the held contract before the Act.
func contractSubject(f envelope.ContractFacts) string {
	parts := []string{f.Name}
	if f.Position != "" {
		parts = append(parts, f.Position)
	}
	if len(f.Baseline) > 0 {
		last := f.Baseline[len(f.Baseline)-1].Season
		parts = append(parts, fmt.Sprintf("%s a year through %d", snapshot.ContractAmountOf(f.Baseline[0].Salary).Text, last))
	}
	if f.Status != "" {
		parts = append(parts, string(f.Status))
	}
	return strings.Join(parts, " · ")
}
func contractPostTerms(f envelope.ContractFacts) string {
	price := snapshot.ContractAmountOf(f.Terms.Price).Text
	unit := "years"
	if f.Years == 1 {
		unit = "year"
	}
	switch f.Intent {
	case "contract.extend":
		last := f.Season
		for _, c := range f.Baseline {
			last = max(last, c.Season)
		}
		parts := []string{}
		for _, c := range f.Terms.Cells {
			if c.Season > last {
				parts = append(parts, fmt.Sprintf("%d at %s", c.Season, snapshot.ContractAmountOf(c.Salary).Text))
			}
		}
		return fmt.Sprintf("Adds %d %s: %s", f.Years, unit, strings.Join(parts, "; "))
	case "contract.restructure":
		return fmt.Sprintf("Moves %s from %d to %d", price, f.Source, f.Destination)
	case "contract.tag":
		return fmt.Sprintf("Franchise tag for %d at %s", contractStart(f), price)
	case "rfa.tender":
		levels := contractacts.TenderLevels()
		if f.Level < 1 || f.Level > len(levels) {
			return "RFA tender: level not selected"
		}
		return fmt.Sprintf("%s-round RFA tender for %d: %s", levels[f.Level-1].Round, contractStart(f), price)
	case "fa.resign":
		return fmt.Sprintf("Re-sign for %d %s from %d at %s a year", f.Years, unit, contractStart(f), price)
	default:
		return ""
	}
}

// contractStart is the first season of the new or changed contract cells.
func contractStart(f envelope.ContractFacts) int {
	if len(f.Terms.Cells) == 0 {
		return f.Season
	}
	return f.Terms.Cells[0].Season
}

// contractActFigures shows each season of the player's contract; held cells the Act does not touch carry over.
func contractActFigures(f envelope.ContractFacts) []ContractActFigure {
	years := map[int]bool{}
	before, after := map[int]domain.Money{}, map[int]domain.Money{}
	for _, c := range f.Baseline {
		before[c.Season] = c.Salary
		after[c.Season] = c.Salary
		years[c.Season] = true
	}
	for _, c := range f.Terms.Cells {
		after[c.Season] = c.Salary
		years[c.Season] = true
	}
	keys := []int{}
	for year := range years {
		keys = append(keys, year)
	}
	slices.Sort(keys)
	out := []ContractActFigure{}
	for _, year := range keys {
		out = append(out, ContractActFigure{year, snapshot.ContractAmountOf(before[year]).Text, snapshot.ContractAmountOf(after[year]).Text, snapshot.ContractAmountOf(after[year] - before[year]).Text})
	}
	return out
}

func (a *App) contractActAction(ctx context.Context, r RawContractAct) error {
	if r.Action == "read" {
		return nil
	}
	if r.Action == "draft" {
		f, err := a.contractActFacts(ctx, r)
		if err != nil {
			return err
		}
		if err := a.refuseSecondContractPlan(ctx, r.Mine, f); err != nil {
			return err
		}
		var bytes [16]byte
		if _, err := rand.Read(bytes[:]); err != nil {
			return fmt.Errorf("contract plan id: %w", err)
		}
		spec := envelope.Spec{Intent: r.Intent, LeagueID: ingestion.LeagueID, FranchiseID: r.Mine, Subject: envelope.Subject{Players: []playerid.PlayerID{f.Player}}, Expected: envelope.ExpectedChange{Player: f.Player}, Gravity: envelope.G3, Undo: envelope.Irreversible, Target: envelope.Target{Kind: envelope.Manual}}
		e, err := envelope.DraftContract(a.movesNow(), spec, f, func() string { return "contract-" + hex.EncodeToString(bytes[:]) })
		if err != nil {
			return fmt.Errorf("contract draft: %w", err)
		}
		if err := a.moves.SaveContract(ctx, e, f); err != nil {
			return fmt.Errorf("contract draft save: %w", err)
		}
		a.movesChanged(ctx)
		return nil
	}
	e, err := a.moves.Get(ctx, r.ID)
	if err != nil {
		return fmt.Errorf("contract plan: %w", err)
	}
	if !envelope.IsContract(e.Receipt().Spec.Intent) || e.Receipt().Spec.FranchiseID != r.Mine {
		return fmt.Errorf("contract plan outside scope")
	}
	f, err := a.moves.ContractFacts(ctx, r.ID)
	if err != nil {
		return fmt.Errorf("contract facts: %w", err)
	}
	next, err := a.contractActTransition(ctx, r, e, f)
	if err != nil {
		return err
	}
	if err := a.moves.Save(ctx, next); err != nil {
		return fmt.Errorf("contract action save: %w", err)
	}
	a.movesChanged(ctx)
	a.wakeMoves()
	return nil
}
func (a *App) refuseSecondContractPlan(ctx context.Context, mine string, f envelope.ContractFacts) error {
	receipts, err := a.moves.List(ctx, ingestion.LeagueID, mine)
	if err != nil {
		return fmt.Errorf("contract plans: %w", err)
	}
	for _, r := range receipts {
		if r.Spec.Intent == f.Intent && r.Spec.Expected.Player == f.Player && envelope.ReleaseOpen(r.State) {
			return fmt.Errorf("%s · %s is already an open plan in My moves; remove it there before keeping another", contractIntentLabel(f.Intent), f.Name)
		}
	}
	return nil
}
func (a *App) contractActTransition(ctx context.Context, r RawContractAct, e envelope.Envelope, f envelope.ContractFacts) (envelope.Envelope, error) {
	now := a.movesNow()
	switch r.Action {
	case "handoff":
		next, err := e.HandOffManual(now, "Contract request copied for the commissioner on ProBoards; nothing posted")
		return contractTransitionResult(next, err)
	case "posted":
		next, err := e.ContractPosted(now)
		return contractTransitionResult(next, err)
	case "withdraw":
		next, err := e.WithdrawContract(now)
		return contractTransitionResult(next, err)
	case "rebase":
		fresh, err := a.contractActFacts(ctx, RawContractAct{Mine: r.Mine, Player: f.Player.String(), Intent: f.Intent, Level: f.Level, Years: f.Years, Source: f.Source, Destination: f.Destination, Move: fmt.Sprintf("%d.%08d", f.Move/contractacts.Million, f.Move%contractacts.Million)})
		if err != nil {
			return envelope.Envelope{}, err
		}
		draft, err := e.Rebase(now, "Repriced from today's MFL mirror; rules and missing evidence checked again")
		if err != nil {
			return envelope.Envelope{}, fmt.Errorf("contract rebase: %w", err)
		}
		checked, err := draft.Check(now, snapshot.Snapshot{}, envelope.ContractCheck{Reasons: fresh.Reasons})
		if err != nil {
			return envelope.Envelope{}, fmt.Errorf("contract recheck: %w", err)
		}
		if err := a.moves.SaveContract(ctx, checked, fresh); err != nil {
			return envelope.Envelope{}, fmt.Errorf("contract rebase save: %w", err)
		}
		return checked, nil
	default:
		return envelope.Envelope{}, fmt.Errorf("unsupported contract action")
	}
}

func (a *App) revalidateContractPlans(ctx context.Context) error {
	receipts, err := a.moves.List(ctx, ingestion.LeagueID, "")
	if err != nil {
		return fmt.Errorf("contract validation list: %w", err)
	}
	m, _, _ := a.league.ContractsView()
	phase, err := a.whatif.CurrentPhase(ctx)
	if err != nil {
		return fmt.Errorf("contract validation phase: %w", err)
	}
	for _, r := range receipts {
		if !envelope.IsContract(r.Spec.Intent) || !envelope.ReleaseOpen(r.State) || r.State == envelope.Stale {
			continue
		}
		f, err := a.moves.ContractFacts(ctx, r.CorrelationID)
		if err != nil {
			return fmt.Errorf("contract validation facts: %w", err)
		}
		window := ""
		if r.State == envelope.Ready {
			window = envelope.ContractWindow(f.Intent, phase)
		}
		reason := envelope.ContractInvalidation(f, mirrorContractEvidence(m, r.Spec.FranchiseID, f.Player.String()), window)
		if !contractNeedsInvalidation(r.State, reason) {
			continue
		}
		e, err := a.moves.Get(ctx, r.CorrelationID)
		if err != nil {
			return fmt.Errorf("contract validation load: %w", err)
		}
		next, err := e.Invalidate(a.movesNow(), reason)
		if err != nil {
			return fmt.Errorf("contract invalidate: %w", err)
		}
		if err := a.moves.Save(ctx, next); err != nil {
			return fmt.Errorf("contract invalidate save: %w", err)
		}
		a.movesChanged(ctx)
	}
	return nil
}

func contractTransitionResult(next envelope.Envelope, err error) (envelope.Envelope, error) {
	if err != nil {
		return next, fmt.Errorf("contract transition: %w", err)
	}
	return next, nil
}
func contractNeedsInvalidation(state envelope.State, reason string) bool {
	if reason == "" {
		return false
	}
	return !strings.HasPrefix(reason, "Contract:") || state == envelope.Ready || state == envelope.Blocked
}

func contractTagPrice(f envelope.ContractFacts, salary domain.Money) domain.Money {
	if f.Status == domain.CStatusFT1 || f.Status == domain.CStatusFT2 {
		return salary
	}
	return 0
}

func (a *App) contractActOptions(ctx context.Context, raw RawContractAct) Reading[[]ContractActView] {
	out := []ContractActView{}
	for _, intent := range []string{"rfa.tender", "fa.resign", "contract.tag", "contract.extend", "contract.restructure"} {
		request := RawContractAct{Mine: raw.Mine, Player: raw.Player, Intent: intent, Level: 1, Years: 1, Move: "1"}
		m, _, _ := a.league.ContractsView()
		request.Source = m.Season
		if p, ok := releasePlayer(m, raw.Mine, raw.Player); ok {
			request.Destination = p.ContractYear
		}
		f, err := a.contractActFacts(ctx, request)
		if err != nil {
			return contractActsReading(nil, err)
		}
		if !slices.Contains(f.Reasons, "This contract cannot take this Act.") {
			out = append(out, contractActView(envelope.Receipt{}, f))
		}
	}
	return contractActsReading(out, nil)
}

func contractTeamView(figures []envelope.ContractFigure) []WhatIfFigure {
	out := []WhatIfFigure{}
	for _, row := range figures {
		out = append(out, WhatIfFigure(row))
	}
	return out
}

func contractFigureSentence(f envelope.ContractFacts, price string) string {
	if f.Intent == "contract.restructure" {
		return fmt.Sprintf("Moves: %s from %d to %d", price, f.Source, f.Destination)
	}
	return "Price: " + price
}
