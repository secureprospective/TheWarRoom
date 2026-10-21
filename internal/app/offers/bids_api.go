package offers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	bidsapp "github.com/secureprospective/TheWarRoom/internal/app/offers/bids"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// RawBidInput is what the bid form sends: the owner's figures as typed. Salary is in millions
// ("5.25"), Beat in points ("6.500"), Closes an RFC 3339 time or empty.
type RawBidInput struct {
	ConditionalDrop string `json:"conditionalDrop"`
	FranchiseID     string `json:"franchiseId"`
	Player          string `json:"player"`
	Salary          string `json:"salary"`
	Years           int    `json:"years"`
	Snipe           bool   `json:"snipe"`
	Beat            string `json:"beat"`
	Evidence        string `json:"evidence"`
	Closes          string `json:"closes"`
}

type bidEntry struct {
	franchise string
	entered   domain.Money
	facts     envelope.BidFacts
}

func (r RawBidInput) entry() (bidEntry, error) {
	e, err := bidsapp.Input(r).Entry()
	if err != nil {
		return bidEntry{}, err //nolint:wrapcheck // preserves typed-input error text
	}
	return bidEntry{franchise: e.Franchise, entered: e.Entered, facts: e.Facts}, nil
}
func closesNote(closes string) string { return bidsapp.ClosesNote(closes) }

// BidPreview is the form's live answer: the figures as Go reads them and every rule's verdict.
// Problem is set instead when the typed figures cannot be read at all.
type BidPreview struct {
	ConditionalDrops []BidDropOption         `json:"conditionalDrops"`
	Problem          string                  `json:"problem"`
	Player           string                  `json:"player"`
	Salary           snapshot.ContractAmount `json:"salary"`
	Snapped          string                  `json:"snapped"`
	Points           string                  `json:"points"`
	Rules            []bid.Result            `json:"rules"`
	Blocked          bool                    `json:"blocked"`
	Note             string                  `json:"note"`
	Post             string                  `json:"post"`
	// RuleSummary is Go's line for the rules that pass; ClosesNote says what the app does with
	// the closing time, set or not.
	RuleSummary string `json:"ruleSummary"`
	ClosesNote  string `json:"closesNote"`
}

func amountOf(m domain.Money) snapshot.ContractAmount {
	return snapshot.ContractAmount{Cents: m, Text: bid.MoneyText(m)}
}

// TargetBidPreview checks typed figures against the bid rules without saving anything.
func (a *API) TargetBidPreview(raw RawBidInput) BidPreview {
	if err := a.host.Ready(); err != nil {
		return BidPreview{Problem: sentence(err), Rules: []bid.Result{}}
	}
	entry, err := raw.entry()
	if err != nil {
		return BidPreview{Problem: sentence(err), Rules: []bid.Result{}}
	}
	ctx, cancel := context.WithTimeout(a.host.OffersContext(), readTimeout)
	defer cancel()
	in, line, err := a.bidInput(ctx, entry, "")
	if err != nil {
		return BidPreview{Problem: sentence(err), Rules: []bid.Result{}}
	}
	rules := bid.Check(in)
	options := a.bidDropOptions(entry.franchise, entry.facts.Salary)
	entry.facts = withConditionalName(entry.facts, options)
	p := BidPreview{ConditionalDrops: options, Player: line, Salary: amountOf(entry.facts.Salary),
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

// bidInput gathers what the rules read about one plan. exclude is the plan's own id, so a plan
// already saved is not counted among the owner's other active bids.
func (a *API) bidInput(ctx context.Context, e bidEntry, exclude string) (bid.Input, string, error) {
	lk, err := a.host.OffersDirectory(ctx)
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
	conditional, err := a.bidConditional(e.franchise, e.facts)
	if err != nil {
		return bid.Input{}, "", err
	}
	exp := bid.ExperienceFrom(a.host.OffersSeason(), facts.DraftYear, facts.HasDraftYear)
	week, weekKnown := a.host.OffersWeek()
	return bid.Input{Salary: e.facts.Salary, Years: e.facts.Years, Minimum: bid.MinimumSalary(exp.Years),
		Conditional: conditional, Experience: exp, Snipe: e.facts.Snipe, Beat: e.facts.Beat, Active: active, Room: a.host.OffersCapRoom(e.franchise),
		Week: week, WeekKnown: weekKnown}, fmt.Sprintf("%s (%s, %s)", facts.Name, facts.Position, facts.Team), nil
}

func (a *API) activeBids(ctx context.Context, franchise, exclude string) (int, error) {
	receipts, err := a.host.OffersMoves().List(ctx, ingestion.LeagueID, franchise)
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

// TargetBidDraft saves a new bid plan with its facts, checked against the bid rules at once.
func (a *API) TargetBidDraft(raw RawBidInput) (BidPlan, error) {
	if err := a.host.Ready(); err != nil {
		return BidPlan{}, fmt.Errorf("bid draft: startup: %w", err)
	}
	entry, err := raw.entry()
	if err != nil {
		return BidPlan{}, fmt.Errorf("bid draft: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.OffersContext(), readTimeout)
	defer cancel()
	return a.serializedMove(func() (BidPlan, error) {
		in, line, err := a.bidInput(ctx, entry, "")
		if err != nil {
			return BidPlan{}, fmt.Errorf("bid draft: %w", err)
		}
		if open, found := a.openBidPlansByPlayer(ctx, entry.franchise)[entry.facts.Player.String()]; found {
			return BidPlan{}, fmt.Errorf("bid draft: you already have an open bid plan on %s (%s); "+
				"open that plan instead of starting a second", line, strings.ToLower(open.State))
		}
		entry.facts = withConditionalName(entry.facts, a.bidDropOptions(entry.franchise, entry.facts.Salary))
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			return BidPlan{}, fmt.Errorf("bid draft: correlation ID: %w", err)
		}
		id := "bid-" + hex.EncodeToString(random[:])
		e, err := envelope.DraftBid(a.host.OffersNow().UTC(), envelope.BidRequest{
			LeagueID: ingestion.LeagueID, FranchiseID: entry.franchise, Facts: entry.facts,
		}, envelope.BidCheck{Input: in}, func() string { return id })
		if err != nil {
			return BidPlan{}, fmt.Errorf("bid draft: %w", err)
		}
		if err := a.host.OffersMoves().SaveBid(ctx, e, entry.facts); err != nil {
			return BidPlan{}, fmt.Errorf("bid draft: save: %w", err)
		}
		a.host.OffersLog(e)
		a.host.OffersChanged()
		return a.bidPlan(ctx, e.Receipt(), entry.facts), nil
	})
}

func pointsText(salary domain.Money, years int) string { return bidsapp.PointsText(salary, years) }
func postText(line string, f envelope.BidFacts) string { return bidsapp.PostText(line, f) }
func bidActive(s envelope.State) bool                  { return bidsapp.Active(s) }

type BidDropOption struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Label     string       `json:"label"`
	Salary    domain.Money `json:"salary"`
	DeadCap   domain.Money `json:"deadCap"`
	Freed     domain.Money `json:"freed"`
	Covers    bool         `json:"covers"`
	Available bool         `json:"available"`
}

// Held future salaries are flat; MFL does not provide restructure history here.

func (a *API) bidDropOptions(mine string, salary domain.Money) []BidDropOption {
	lk := a.host.OffersLookup()
	m := a.host.OffersRoster()
	room := a.host.OffersCapRoom(mine)
	out := []BidDropOption{}
	for _, p := range m.Players {
		if p.FranchiseID != mine {
			continue
		}
		name := p.MFLID
		if f, ok := lk.Facts(p.MFLID); ok {
			name = fmt.Sprintf("%s (%s, %s)", f.Name, f.Position, f.Team)
		}
		charge, available := bidsapp.HeldCutCharge(p, m.Season)
		freed := p.Salary - charge
		label := fmt.Sprintf("%s · %s salary · %s · %s dead cap if dropped",
			name, bid.MoneyText(p.Salary), bid.YearsText(p.ContractYear-m.Season+1), bid.MoneyText(charge))
		if !available {
			label = name + " · contract length not held; cannot price a drop"
		}
		out = append(out, BidDropOption{ID: p.MFLID, Name: name, Label: label, Salary: p.Salary,
			DeadCap: charge, Freed: freed, Covers: available && room != nil && *room+freed >= salary,
			Available: available})
	}
	slices.SortFunc(out, func(a, b BidDropOption) int {
		if a.Covers != b.Covers {
			if a.Covers {
				return -1
			}
			return 1
		}
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return out
}

func (a *API) bidConditional(mine string, f envelope.BidFacts) (*bid.ConditionalDrop, error) {
	if f.ConditionalDrop.String() == "" {
		return nil, nil
	}
	for _, option := range a.bidDropOptions(mine, f.Salary) {
		if option.ID != f.ConditionalDrop.String() {
			continue
		}
		if !option.Available {
			return nil, fmt.Errorf("conditional drop: contract length is not held")
		}
		return &bid.ConditionalDrop{Name: option.Name, Salary: option.Salary, DeadCap: option.DeadCap}, nil
	}
	return nil, fmt.Errorf("conditional drop must be on your own roster; choose another player or clear it")
}

func withConditionalName(f envelope.BidFacts, options []BidDropOption) envelope.BidFacts {
	f.ConditionalName = ""
	for _, option := range options {
		if option.ID == f.ConditionalDrop.String() {
			f.ConditionalName = option.Name
		}
	}
	return f
}
