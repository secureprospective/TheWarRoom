package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

const maxEvidenceRunes = 500

// RawBidInput is what the bid form sends: the owner's figures as typed. Salary is in millions
// ("5.25"), Beat in points ("6.500"), Closes an RFC 3339 time or empty.
type RawBidInput struct {
	FranchiseID string `json:"franchiseId"`
	Player      string `json:"player"`
	Salary      string `json:"salary"`
	Years       int    `json:"years"`
	Snipe       bool   `json:"snipe"`
	Beat        string `json:"beat"`
	Evidence    string `json:"evidence"`
	Closes      string `json:"closes"`
}

type bidEntry struct {
	franchise string
	entered   domain.Money // before the $10k snap
	facts     envelope.BidFacts
}

// entry reads the typed figures into exact values; the salary snaps to $10k here (§0).
func (r RawBidInput) entry() (bidEntry, error) {
	if err := ingestion.FeedFranchise(r.FranchiseID); err != nil {
		return bidEntry{}, fmt.Errorf("franchise: %w", err)
	}
	pid, err := playerid.New(r.Player)
	if err != nil {
		return bidEntry{}, fmt.Errorf("player: %w", err)
	}
	if strings.TrimSpace(r.Salary) == "" {
		return bidEntry{}, fmt.Errorf("enter the annual salary in millions, such as 5.25")
	}
	entered, err := domain.ParseMoneyMillions(r.Salary)
	if err != nil {
		return bidEntry{}, fmt.Errorf("salary %q is not an amount in millions, such as 5.25", r.Salary)
	}
	facts := envelope.BidFacts{Player: pid, Salary: domain.RoundToNearest10k(entered), Years: r.Years,
		Snipe: r.Snipe, Evidence: strings.TrimSpace(r.Evidence)}
	facts.Points, _ = bid.Score(facts.Salary, facts.Years) // a bad length fails the length rule instead
	if strings.TrimSpace(r.Beat) != "" {
		beat, err := bid.ParsePoints(r.Beat)
		if err != nil {
			return bidEntry{}, fmt.Errorf("bid to beat: %w", err)
		}
		facts.Beat = &beat
	}
	if facts.Closes, err = closesText(r.Closes); err != nil {
		return bidEntry{}, err
	}
	if utf8.RuneCountInString(facts.Evidence) > maxEvidenceRunes {
		return bidEntry{}, fmt.Errorf("evidence is longer than %d characters", maxEvidenceRunes)
	}
	return bidEntry{franchise: r.FranchiseID, entered: entered, facts: facts}, nil
}

func closesText(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	at, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("window close time %q is not a date and time", raw)
	}
	return at.UTC().Format(time.RFC3339), nil
}

// closesNote is what the watcher does with the window close the owner typed.
func closesNote(closes string) string {
	if closes == "" {
		return "No closing time is set: the app keeps watching the rosters until the player shows up on one."
	}
	return "After this time, the next roster check that does not show him on your roster ends the plan " +
		"as awarded off-app."
}

// BidPreview is the form's live answer: the figures as Go reads them and every rule's verdict.
// Problem is set instead when the typed figures cannot be read at all.
type BidPreview struct {
	Problem string                  `json:"problem"`
	Player  string                  `json:"player"`
	Salary  snapshot.ContractAmount `json:"salary"`
	Snapped string                  `json:"snapped"`
	Points  string                  `json:"points"`
	Rules   []bid.Result            `json:"rules"`
	Blocked bool                    `json:"blocked"`
	Note    string                  `json:"note"`
	Post    string                  `json:"post"`
	// RuleSummary is Go's line for the rules that pass; ClosesNote says what the app does with
	// the closing time, set or not.
	RuleSummary string `json:"ruleSummary"`
	ClosesNote  string `json:"closesNote"`
}

func amountOf(m domain.Money) snapshot.ContractAmount {
	return snapshot.ContractAmount{Cents: m, Text: bid.MoneyText(m)}
}

func pointsText(salary domain.Money, years int) string {
	points, err := bid.Score(salary, years)
	if err != nil {
		return "Not scored"
	}
	return points.String()
}

// TargetBidPreview checks typed figures against the bid rules without saving anything.
func (a *App) TargetBidPreview(raw RawBidInput) BidPreview {
	if err := a.ready(); err != nil {
		return BidPreview{Problem: sentence(err), Rules: []bid.Result{}}
	}
	entry, err := raw.entry()
	if err != nil {
		return BidPreview{Problem: sentence(err), Rules: []bid.Result{}}
	}
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()
	in, line, err := a.bidInput(ctx, entry, "")
	if err != nil {
		return BidPreview{Problem: sentence(err), Rules: []bid.Result{}}
	}
	rules := bid.Check(in)
	p := BidPreview{Player: line, Salary: amountOf(entry.facts.Salary),
		Points: pointsText(entry.facts.Salary, entry.facts.Years), Rules: rules,
		Blocked: bid.Blocked(rules), Note: bid.Note(rules), Post: postText(line, entry.facts),
		RuleSummary: bid.Summary(rules), ClosesNote: closesNote(entry.facts.Closes)}
	if entry.entered != entry.facts.Salary {
		p.Snapped = fmt.Sprintf("Entered %s; the league snaps salaries to $10,000, so %s",
			bid.MoneyText(entry.entered), bid.MoneyText(entry.facts.Salary))
	}
	return p
}

// sentence starts an error's words with a capital, for the owner to read.
func sentence(err error) string {
	text := err.Error()
	r, size := utf8.DecodeRuneInString(text)
	return strings.ToUpper(string(r)) + text[size:]
}

func postText(line string, f envelope.BidFacts) string {
	return bid.Post{Player: line, Salary: f.Salary, Years: f.Years, Points: f.Points, Snipe: f.Snipe}.Text()
}

// bidInput gathers what the rules read about one plan. exclude is the plan's own id, so a plan
// already saved is not counted among the owner's other active bids.
func (a *App) bidInput(ctx context.Context, e bidEntry, exclude string) (bid.Input, string, error) {
	lk, err := a.directory(ctx)
	if err != nil {
		return bid.Input{}, "", fmt.Errorf("player names unavailable: %w", err)
	}
	facts, ok := lk.Facts(e.facts.Player.String())
	if !ok {
		return bid.Input{}, "", fmt.Errorf("player %s is not in the players directory", e.facts.Player)
	}
	active, err := a.activeBids(ctx, e.franchise, exclude)
	if err != nil {
		return bid.Input{}, "", err
	}
	exp := bid.ExperienceFrom(a.league.Season(), facts.DraftYear, facts.HasDraftYear)
	week, weekKnown := a.heldWeek()
	return bid.Input{Salary: e.facts.Salary, Years: e.facts.Years, Minimum: bid.MinimumSalary(exp.Years),
		Experience: exp, Snipe: e.facts.Snipe, Beat: e.facts.Beat, Active: active, Room: a.capRoom(e.franchise),
		Week: week, WeekKnown: weekKnown}, fmt.Sprintf("%s (%s, %s)", facts.Name, facts.Position, facts.Team), nil
}

func (a *App) heldWeek() (int, bool) {
	a.weekMu.Lock()
	defer a.weekMu.Unlock()
	return a.week.Number, !a.weekFetchedAt.IsZero()
}

// capRoom is the league cap less the franchise's cap used, from the held mirror and rules.
func (a *App) capRoom(franchise string) *domain.Money {
	raw := strings.TrimSpace(a.rulebook.GetSalaryCap())
	if raw == "" {
		return nil
	}
	limit, err := domain.ParseMoneyMillions(raw)
	used, ok := a.league.CapUsed(franchise)
	if err != nil || !ok {
		return nil
	}
	room := limit - used
	return &room
}

func (a *App) activeBids(ctx context.Context, franchise, exclude string) (int, error) {
	receipts, err := a.moves.List(ctx, ingestion.LeagueID, franchise)
	if err != nil {
		return 0, fmt.Errorf("bid plans: list: %w", err)
	}
	n := 0
	for _, r := range receipts {
		if r.Spec.Intent == envelope.BidIntent && r.CorrelationID != exclude && bidActive(r.State) {
			n++
		}
	}
	return n, nil
}

// bidActive: plans that count toward the 30-bid limit are those the owner has checked or posted
// and that are not over. A Draft, Blocked or Stale plan is not on the board.
func bidActive(s envelope.State) bool {
	return slices.Contains([]envelope.State{envelope.Ready, envelope.HandedOff, envelope.NotYetDone,
		envelope.NotVerified, envelope.BidPending}, s)
}

// TargetBidDraft saves a new bid plan with its facts, checked against the bid rules at once.
func (a *App) TargetBidDraft(raw RawBidInput) (BidPlan, error) {
	if err := a.ready(); err != nil {
		return BidPlan{}, fmt.Errorf("bid draft: startup: %w", err)
	}
	entry, err := raw.entry()
	if err != nil {
		return BidPlan{}, fmt.Errorf("bid draft: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()
	a.movesMu.Lock()
	defer a.movesMu.Unlock()
	in, line, err := a.bidInput(ctx, entry, "")
	if err != nil {
		return BidPlan{}, fmt.Errorf("bid draft: %w", err)
	}
	if open, found := a.openBidPlansByPlayer(ctx, entry.franchise)[entry.facts.Player.String()]; found {
		return BidPlan{}, fmt.Errorf("bid draft: you already have an open bid plan on %s (%s); "+
			"open that plan instead of starting a second", line, strings.ToLower(open.State))
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return BidPlan{}, fmt.Errorf("bid draft: correlation ID: %w", err)
	}
	id := "bid-" + hex.EncodeToString(random[:])
	e, err := envelope.DraftBid(a.movesNow().UTC(), envelope.BidRequest{
		LeagueID: ingestion.LeagueID, FranchiseID: entry.franchise, Facts: entry.facts,
	}, envelope.BidCheck{Input: in}, func() string { return id })
	if err != nil {
		return BidPlan{}, fmt.Errorf("bid draft: %w", err)
	}
	if err := a.moves.SaveBid(ctx, e, entry.facts); err != nil {
		return BidPlan{}, fmt.Errorf("bid draft: save: %w", err)
	}
	a.logMove(e)
	a.movesChanged(a.ctx)
	return a.bidPlan(ctx, e.Receipt(), entry.facts), nil
}
