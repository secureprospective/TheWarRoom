package leaguemigration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// Ledger is the app's league ledger as the move uses it. Batch records every change or none.
type Ledger interface {
	Records(ctx context.Context, league string) ([]commissioner.Record, error)
	Batch(ctx context.Context, raws []commissioner.RawCommand, principal string) ([]commissioner.Receipt, error)
	Execute(ctx context.Context, raw commissioner.RawCommand, principal string) (commissioner.Receipt, error)
}

// Service moves one MFL league into the app. MFL is only read; every act is a ledger change.
type Service struct {
	// Source opens a read-only reader of the named MFL league.
	Source func(mflLeague string) (Source, error)
	Ledger Ledger
	Now    func() time.Time
}

// Request names the MFL league being moved and the app league it lands in.
type Request struct {
	MFLLeague string `json:"mflLeague"`
	League    string `json:"league"`
}

// AppLeagueID is the app league an MFL league lands in: one per MFL league and season.
func AppLeagueID(mflLeague string, season int) string {
	return fmt.Sprintf("native.mfl-%s-%d", strings.TrimSpace(mflLeague), season)
}

func (s *Service) read(ctx context.Context, mflLeague string) (Export, error) {
	if strings.TrimSpace(mflLeague) == "" {
		return Export{}, fmt.Errorf("name the MFL league to move, by its MFL league number")
	}
	src, err := s.Source(mflLeague)
	if err != nil {
		return Export{}, &ReadError{"league", err}
	}
	return Read(ctx, src, s.Now())
}

// Preparation is what an import would land, in plain words; nothing is recorded by preparing.
type Preparation struct {
	League     string   `json:"league"`
	Name       string   `json:"name"`
	Season     int      `json:"season"`
	Franchises int      `json:"franchises"`
	Players    int      `json:"players"`
	Picks      int      `json:"picks"`
	Settings   int      `json:"settings"`
	Notes      []string `json:"notes"`
	Problems   []string `json:"problems"`
	Ready      bool     `json:"ready"`
}

// Prepare reads the MFL league and says what the import would land, or what stops it.
func (s *Service) Prepare(ctx context.Context, mflLeague string) (Preparation, error) {
	export, err := s.read(ctx, mflLeague)
	if err != nil {
		return Preparation{}, err
	}
	out := Preparation{League: AppLeagueID(mflLeague, export.Season), Name: export.Name, Season: export.Season,
		Notes: []string{}, Problems: []string{}}
	plan, err := Prepare(export)
	if problems, ok := AsProblems(err); ok {
		out.Problems = problems
		return out, nil
	} else if err != nil {
		return Preparation{}, err
	}
	out.Franchises, out.Players, out.Picks = len(plan.Seed.Franchises), len(plan.Seed.Players), len(plan.Seed.Picks)
	out.Settings, out.Notes, out.Ready = len(plan.Create.Seed), plan.Notes, true
	return out, nil
}

// Import reads the MFL league afresh and lands it whole in one recorded act: the league, with every
// setting MFL holds, and its starting rosters, contracts, picks, franchises and divisions. Until the
// cutover it follows MFL. A failure at any step records nothing.
func (s *Service) Import(ctx context.Context, mflLeague, reason, principal string) ([]commissioner.Receipt, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("write the reason for importing the league")
	}
	export, err := s.read(ctx, mflLeague)
	if err != nil {
		return nil, err
	}
	plan, err := Prepare(export)
	if err != nil {
		return nil, err
	}
	league := AppLeagueID(mflLeague, export.Season)
	create := commissioner.RawCommand{League: league, Command: commissioner.Create, Args: plan.Create,
		Request: "mfl-import.create", Reason: reason, Effective: "now", Visibility: "league"}
	seed := plan.Seed
	start := commissioner.RawCommand{League: league, Command: commissioner.NativeSeed,
		Args: commissioner.RawArgs{Native: &seed}, Request: "mfl-import.seed", Reason: reason, Effective: "now",
		Visibility: "league"}
	receipts, err := s.Ledger.Batch(ctx, []commissioner.RawCommand{create, start}, principal)
	if err != nil {
		return nil, fmt.Errorf("import the MFL league (nothing was recorded; start the import again): %w", err)
	}
	return receipts, nil
}

// Review compares the app league with a fresh read of MFL. It can be run any number of times.
func (s *Service) Review(ctx context.Context, req Request) (Review, error) {
	export, err := s.read(ctx, req.MFLLeague)
	if err != nil {
		return Review{}, err
	}
	plan, err := Prepare(export)
	if err != nil {
		return Review{}, err
	}
	records, err := s.Ledger.Records(ctx, req.League)
	if err != nil {
		return Review{}, fmt.Errorf("read the app league: %w", err)
	}
	if len(records) == 0 {
		return Review{}, fmt.Errorf("import the MFL league before reviewing it")
	}
	app, err := ReadApp(req.League, records, s.Now())
	if err != nil {
		return Review{}, err
	}
	return Words(req.League, plan, app, Compare(plan, app)), nil
}

// Cutover moves the league over (G3, with a typed reason). It is offered only from a review: the review
// is run again, and its digest must be the one the commissioner read, so they saw exactly what differs.
func (s *Service) Cutover(ctx context.Context, req Request, digest, reason, principal string) (commissioner.Receipt, error) {
	if strings.TrimSpace(reason) == "" {
		return commissioner.Receipt{}, fmt.Errorf("write the reason for moving the league over")
	}
	review, err := s.Review(ctx, req)
	if err != nil {
		return commissioner.Receipt{}, err
	}
	if review.CutOver {
		return commissioner.Receipt{}, fmt.Errorf("this league has already been moved over from MFL")
	}
	if digest == "" || digest != review.Digest {
		return commissioner.Receipt{}, fmt.Errorf("the league changed since your review; review it again, " +
			"then move it over")
	}
	records, err := s.Ledger.Records(ctx, req.League)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("read the app league: %w", err)
	}
	raw := commissioner.RawCommand{League: req.League, Command: commissioner.Cutover, Expected: int64(len(records)),
		Request: "mfl-import.cutover", Reason: reason, Effective: "now", Visibility: "league"}
	receipt, err := s.Ledger.Execute(ctx, raw, principal)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("move the league over: %w", err)
	}
	return receipt, nil
}

// Invite is one prepared owner invitation. Ready is false where MFL holds no email for the owner.
type Invite struct {
	Franchise string `json:"franchise"`
	Team      string `json:"team"`
	Owner     string `json:"owner"`
	Email     string `json:"email"`
	ID        string `json:"id"`
	Ready     bool   `json:"ready"`
	Words     string `json:"words"`
	Claimed   bool   `json:"claimed"`
}

// Invites is one invitation per franchise, ready for the commissioner to review and send in one act.
type Invites struct {
	League string   `json:"league"`
	List   []Invite `json:"list"`
	Ready  int      `json:"ready"`
	Digest string   `json:"digest"`
}

// Invites prepares the invitations from the owners MFL lists, read afresh (MFL's owner emails are never
// copied into the league's shared ledger). Claimed seats and invitations already sent show as such.
func (s *Service) Invites(ctx context.Context, req Request) (Invites, error) {
	records, err := s.Ledger.Records(ctx, req.League)
	if err != nil {
		return Invites{}, fmt.Errorf("read the app league: %w", err)
	}
	if !commissioner.CutOver(records) {
		return Invites{}, fmt.Errorf("move the league over from MFL before inviting its owners")
	}
	export, err := s.read(ctx, req.MFLLeague)
	if err != nil {
		return Invites{}, err
	}
	app, err := ReadApp(req.League, records, s.Now())
	if err != nil {
		return Invites{}, err
	}
	state, err := commissioner.GovernanceState(records)
	if err != nil {
		return Invites{}, fmt.Errorf("read the league's seats: %w", err)
	}
	return prepareInvites(req.League, export, app, state), nil
}

// SendInvites issues every ready invitation in one recorded act. digest must be the list the
// commissioner reviewed; a changed list is refused so nothing unseen is sent.
func (s *Service) SendInvites(ctx context.Context, req Request, digest, reason, principal string) ([]commissioner.Receipt, error) {
	invites, err := s.Invites(ctx, req)
	if err != nil {
		return nil, err
	}
	if digest == "" || digest != invites.Digest {
		return nil, fmt.Errorf("the invitation list changed since you reviewed it; review it again, then send")
	}
	if strings.TrimSpace(reason) == "" {
		reason = "Invite the owners of the league moved over from MFL"
	}
	var raws []commissioner.RawCommand
	for _, invite := range invites.List {
		if !invite.Ready {
			continue
		}
		people := commissioner.RawPeople{ID: invite.ID, Franchise: invite.Franchise, Principal: invite.Email}
		raws = append(raws, commissioner.RawCommand{League: req.League, Command: "commish.seat.invite",
			Args: commissioner.RawArgs{People: &people}, Request: "mfl-import.invite." + invite.Franchise,
			Reason: reason, Effective: "now", Visibility: "private"})
	}
	if len(raws) == 0 {
		return nil, fmt.Errorf("no invitation is ready to send: each needs an owner email")
	}
	records, err := s.Ledger.Records(ctx, req.League)
	if err != nil {
		return nil, fmt.Errorf("read the app league: %w", err)
	}
	raws[0].Expected = int64(len(records))
	receipts, err := s.Ledger.Batch(ctx, raws, principal)
	if err != nil {
		return nil, fmt.Errorf("send the invitations (none was sent): %w", err)
	}
	return receipts, nil
}

// inviteID is stable per league, franchise and email, so a resend of the same list is a retry.
func inviteID(league, franchise, email string) string {
	sum := sha256.Sum256([]byte(league + "\x00" + franchise + "\x00" + strings.ToLower(email)))
	return "invite." + hex.EncodeToString(sum[:8])
}
