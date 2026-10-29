package contracts

import (
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/contractacts"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// ActFigure is one season of a player's contract before and after an Act; it is main.ContractActFigure.
type ActFigure struct {
	Season int    `json:"season"`
	Before string `json:"before"`
	After  string `json:"after"`
	Change string `json:"change"`
}

// ActChoice is one option of an Act's form; it is main.ContractActChoice.
type ActChoice struct {
	Value int    `json:"value"`
	Label string `json:"label"`
}

// ActView is one contract Act, previewed or planned; it is main.ContractActView.
type ActView struct {
	ID          string           `json:"id"`
	Intent      string           `json:"intent"`
	Act         string           `json:"act"`
	Levels      []ActChoice      `json:"levels"`
	Years       []int            `json:"years"`
	TeamFigures []WhatIfFigure   `json:"teamFigures"`
	Player      string           `json:"player"`
	Subject     string           `json:"subject"`
	Label       string           `json:"label"`
	Price       string           `json:"price"`
	Post        string           `json:"post"`
	How         string           `json:"how"`
	Reasons     []string         `json:"reasons"`
	Figures     []ActFigure      `json:"figures"`
	Receipt     envelope.Receipt `json:"receipt"`
	CanHandOff  bool             `json:"canHandOff"`
	CanPost     bool             `json:"canPost"`
	CanWithdraw bool             `json:"canWithdraw"`
	CanRebase   bool             `json:"canRebase"`
}

// IntentLabel names an Act for the owner.
func IntentLabel(intent string) string {
	labels := map[string]string{"rfa.tender": "Tender him (RFA)", "fa.resign": "Re-sign him", "contract.tag": "Franchise tag", "contract.extend": "Extend him", "contract.restructure": "Restructure"}
	return labels[intent]
}

// Choices is what an Act's form offers: tender levels priced from the rule table, and term lengths.
func Choices(intent string) ([]ActChoice, []int) {
	levels, years := []ActChoice{}, []int{}
	if intent == "rfa.tender" || intent == "fa.resign" {
		for i, l := range contractacts.TenderLevels() {
			levels = append(levels, ActChoice{i + 1, fmt.Sprintf("%s · $%.2fM", l.Round, l.Price.Millions())})
		}
	}
	most := map[string]int{"fa.resign": contractacts.MaxResignYears, "contract.extend": contractacts.MaxExtendYears}[intent]
	for y := 1; y <= most; y++ {
		years = append(years, y)
	}
	return levels, years
}

// DescribeAct is an Act as the owner reads it: its price, its figures, the post for the commissioner and
// what the plan's state allows next.
func DescribeAct(r envelope.Receipt, f envelope.ContractFacts) ActView {
	label := IntentLabel(f.Intent) + " · " + f.Name
	figures := Figures(f)
	price := snapshot.ContractAmountOf(f.Terms.Price).Text
	if len(f.Terms.Cells) == 0 {
		price = "Not priced"
		figures = []ActFigure{}
	}
	post := label + "\nOwner's contract request to the commissioner on ProBoards.\n"
	for _, c := range f.Terms.Cells {
		post += fmt.Sprintf("%d: %s\n", c.Season, snapshot.ContractAmountOf(c.Salary).Text)
	}
	post += postTerms(f) + "\n"
	post += strings.Join(f.Reasons, "\n") + "\n"
	post += "Please process this request. Nothing has been posted or changed by the app."
	levels, years := Choices(f.Intent)
	return ActView{ID: r.CorrelationID, Intent: f.Intent, Act: IntentLabel(f.Intent), Levels: levels, Years: years, TeamFigures: teamView(f.TeamFigures), Player: f.Name, Subject: Subject(f), Label: label, Price: figureSentence(f, price), Post: post, How: f.Terms.How, Reasons: f.Reasons, Figures: figures, Receipt: r, CanHandOff: r.State == envelope.Ready, CanPost: r.State == envelope.HandedOff, CanWithdraw: slices.Contains([]envelope.State{envelope.Draft, envelope.Blocked, envelope.Ready, envelope.HandedOff, envelope.NotYetDone, envelope.NotVerified, envelope.BidPending, envelope.Stale}, r.State), CanRebase: r.State == envelope.Stale}
}

// Subject names whose contract an Act changes, from the held contract before the Act.
func Subject(f envelope.ContractFacts) string {
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

func postTerms(f envelope.ContractFacts) string {
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
		return fmt.Sprintf("Franchise tag for %d at %s", start(f), price)
	case "rfa.tender":
		levels := contractacts.TenderLevels()
		if f.Level < 1 || f.Level > len(levels) {
			return "RFA tender: level not selected"
		}
		return fmt.Sprintf("%s-round RFA tender for %d: %s", levels[f.Level-1].Round, start(f), price)
	case "fa.resign":
		return fmt.Sprintf("Re-sign for %d %s from %d at %s a year", f.Years, unit, start(f), price)
	default:
		return ""
	}
}

// start is the first season of the new or changed contract cells.
func start(f envelope.ContractFacts) int {
	if len(f.Terms.Cells) == 0 {
		return f.Season
	}
	return f.Terms.Cells[0].Season
}

// Figures shows each season of the player's contract; held cells the Act does not touch carry over.
func Figures(f envelope.ContractFacts) []ActFigure {
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
	out := []ActFigure{}
	for _, year := range keys {
		out = append(out, ActFigure{year, snapshot.ContractAmountOf(before[year]).Text, snapshot.ContractAmountOf(after[year]).Text, snapshot.ContractAmountOf(after[year] - before[year]).Text})
	}
	return out
}

func teamView(figures []envelope.ContractFigure) []WhatIfFigure {
	out := []WhatIfFigure{}
	for _, row := range figures {
		out = append(out, WhatIfFigure(row))
	}
	return out
}

func figureSentence(f envelope.ContractFacts, price string) string {
	if f.Intent == "contract.restructure" {
		return fmt.Sprintf("Moves: %s from %d to %d", price, f.Source, f.Destination)
	}
	return "Price: " + price
}
