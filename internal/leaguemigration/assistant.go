package leaguemigration

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// The move's commands in the command registry. Reads change nothing; each execute is a recorded act.
const (
	CommandPrepare     = "league.migrate.prepare"
	CommandImport      = "league.migrate.import"
	CommandReview      = "league.migrate.review"
	CommandCutover     = "league.migrate.cutover"
	CommandInvites     = "league.migrate.invites"
	CommandSendInvites = "league.migrate.invites.send"
)

// RawStep is every move command's argument: which MFL league, which app league, the review or list the
// commissioner read (its digest), and the typed reason for an act.
type RawStep struct {
	MFLLeague string `json:"mflLeague"`
	League    string `json:"league,omitempty"`
	Digest    string `json:"digest,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// Validate refuses a step without its MFL league.
func (r RawStep) Validate() error {
	if strings.TrimSpace(r.MFLLeague) == "" || strings.TrimSpace(r.MFLLeague) != r.MFLLeague {
		return fmt.Errorf("name the MFL league by its league number")
	}
	return nil
}

// Invoke runs one registry command: id, mode ("read" or "execute") and its JSON argument.
type Invoke func(ctx context.Context, id, mode string, body json.RawMessage) (json.RawMessage, error)

// Assistant walks a commissioner through the move from one plain ask. It reads freely, explains each step,
// and records nothing until the human confirms that exact act: the import, the cutover and the invitation
// send each wait for a confirm with a typed reason.
type Assistant struct {
	Invoke  Invoke
	step    RawStep
	pending *pendingAct
	seq     int
}

type pendingAct struct {
	id, command string
	step        RawStep
}

// Turn is the assistant's answer: what it says, what it shows, and the act waiting for a confirm.
type Turn struct {
	Say     []string `json:"say"`
	Show    any      `json:"show,omitempty"`
	Confirm *Confirm `json:"confirm,omitempty"`
	Done    bool     `json:"done"`
}

// Confirm is an act waiting for the human: its id, what it does, and that it needs a typed reason.
type Confirm struct {
	ID          string `json:"id"`
	Act         string `json:"act"`
	Words       string `json:"words"`
	NeedsReason bool   `json:"needsReason"`
}

var leagueNumber = regexp.MustCompile(`\b\d{3,6}\b`)

// Ask takes the plain ask ("move my league 14432 over from MFL") and prepares the first act.
func (a *Assistant) Ask(ctx context.Context, text string) (Turn, error) {
	lower := strings.ToLower(text)
	if !strings.Contains(lower, "mfl") && !strings.Contains(lower, "myfantasyleague") {
		return Turn{Say: []string{"I can move a league over from MFL. Say \"move my league over from MFL\" " +
			"with its MFL league number."}}, nil
	}
	number := leagueNumber.FindString(text)
	if number == "" {
		return Turn{Say: []string{"Which MFL league? Give me its league number: it is in the league's MFL " +
			"address, after /home/."}}, nil
	}
	a.step, a.pending = RawStep{MFLLeague: number}, nil
	var prep Preparation
	if err := a.read(ctx, CommandPrepare, a.step, &prep); err != nil {
		return Turn{}, err
	}
	say := []string{fmt.Sprintf("I read MFL league %s (%s, season %d). Nothing on MFL is changed: "+
		"the app only reads it.", number, prep.Name, prep.Season)}
	if !prep.Ready {
		say = append(say, "It cannot move over yet. Fix these on MFL, then ask me again:")
		say = append(say, prep.Problems...)
		return Turn{Say: say, Show: prep, Done: true}, nil
	}
	a.step.League = prep.League
	say = append(say, fmt.Sprintf("Step 1 of 4, import: I would land %d franchises, %d rostered players and "+
		"%d draft picks, with %d settings exactly as MFL holds them. Until you switch over, the league keeps "+
		"following MFL.", prep.Franchises, prep.Players, prep.Picks, prep.Settings))
	say = append(say, prep.Notes...)
	return Turn{Say: say, Show: prep, Confirm: a.await(CommandImport, "Import the league from MFL")}, nil
}

func (a *Assistant) await(command, words string) *Confirm {
	a.seq++
	a.pending = &pendingAct{id: fmt.Sprintf("confirm.%d", a.seq), command: command, step: a.step}
	return &Confirm{ID: a.pending.id, Act: command, Words: words + ". Type a reason and confirm.", NeedsReason: true}
}

// Confirm records the waiting act, and only it, with the human's typed reason, then prepares the next.
func (a *Assistant) Confirm(ctx context.Context, id, reason string) (Turn, error) {
	if a.pending == nil || a.pending.id != id {
		return Turn{}, fmt.Errorf("there is nothing waiting for that confirm; nothing was recorded")
	}
	if strings.TrimSpace(reason) == "" {
		return Turn{}, fmt.Errorf("type the reason before confirming; nothing was recorded")
	}
	act := *a.pending
	a.pending = nil
	act.step.Reason = reason
	body, err := json.Marshal(act.step)
	if err != nil {
		return Turn{}, fmt.Errorf("encode the step: %w", err)
	}
	if _, err := a.Invoke(ctx, act.command, "execute", body); err != nil {
		return Turn{}, err
	}
	switch act.command {
	case CommandImport:
		return a.review(ctx, "Imported. Step 2 of 4, review: here is the app's league against MFL, franchise by franchise.")
	case CommandCutover:
		return a.invites(ctx)
	default:
		return Turn{Say: []string{"The invitations are sent. Owners who accept show as claimed on the people " +
			"screen. The move is done."}, Done: true}, nil
	}
}

// Review runs the comparison again, any number of times, and offers the cutover with what it says.
func (a *Assistant) Review(ctx context.Context) (Turn, error) {
	return a.review(ctx, "Here is the review again.")
}

func (a *Assistant) review(ctx context.Context, intro string) (Turn, error) {
	var review Review
	if err := a.read(ctx, CommandReview, a.step, &review); err != nil {
		return Turn{}, err
	}
	say := []string{intro, review.Summary}
	say = append(say, review.LeagueWide...)
	for _, f := range review.Franchises {
		if !f.Matches {
			say = append(say, f.Name+": "+strings.Join(f.Lines, "; "))
		}
	}
	a.step.Digest = review.Digest
	say = append(say, "Step 3 of 4, switch over: from then on the app runs the league and MFL no longer changes "+
		"it. MFL itself is never touched, so there is no undo; if you give the move up, import again.")
	return Turn{Say: say, Show: review, Confirm: a.await(CommandCutover,
		fmt.Sprintf("Switch the league over from MFL (%s)", review.Summary))}, nil
}

func (a *Assistant) invites(ctx context.Context) (Turn, error) {
	var list Invites
	if err := a.read(ctx, CommandInvites, a.step, &list); err != nil {
		return Turn{}, err
	}
	say := []string{"Switched over: the app runs the league now.",
		fmt.Sprintf("Step 4 of 4, invitations: %d of %d owners are ready to invite.", list.Ready, len(list.List))}
	for _, i := range list.List {
		say = append(say, fmt.Sprintf("%s: %s", i.Team, i.Words))
	}
	a.step.Digest = list.Digest
	if list.Ready == 0 {
		return Turn{Say: append(say, "No invitation is ready to send."), Show: list, Done: true}, nil
	}
	return Turn{Say: say, Show: list, Confirm: a.await(CommandSendInvites,
		fmt.Sprintf("Send %d invitations", list.Ready))}, nil
}

func (a *Assistant) read(ctx context.Context, command string, step RawStep, out any) error {
	body, err := json.Marshal(step)
	if err != nil {
		return fmt.Errorf("encode the step: %w", err)
	}
	reply, err := a.Invoke(ctx, command, "read", body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(reply, out); err != nil {
		return fmt.Errorf("read the %s reply: %w", command, err)
	}
	return nil
}
