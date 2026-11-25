package serverhttp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"log"
	"net/http"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/serverversion"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

const WorkflowPath = "/v1/native/workflow"

type RawWorkflow = catalog.RawWorkflow
type RawWorkflowScope = catalog.RawWorkflowScope

// WorkflowResponse carries the authority's own wording; a cached writable bit is never a grant.
type WorkflowResponse struct {
	Label      string          `json:"label"`
	Writable   bool            `json:"writable"`
	Football   string          `json:"football"`
	AppRelease int             `json:"appRelease"`
	Data       json.RawMessage `json:"data"`
	Tags       *WorkflowTags   `json:"tags,omitempty"`
}

type WorkflowTags struct {
	ServerEpoch string            `json:"serverEpoch"`
	League      map[string]string `json:"league"`
	Football    string            `json:"football"`
	Live        string            `json:"live"`
	Values      string            `json:"values"`
}

func (r WorkflowResponse) Validate() error {
	if r.Label == "" || r.AppRelease < 1 || !json.Valid(r.Data) {
		return fmt.Errorf("unreadable league response")
	}
	return nil
}

func (h Hosted) workflow(w http.ResponseWriter, r *http.Request, service *leagueservice.Service, raw RawWorkflow, league string) {
	h.Requests.add(raw.Command, raw.Mode, 0, 0)
	//nolint:contextcheck // The service host carries this request's context and admission.
	location, err := service.Location(league)
	if err != nil {
		h.Requests.add(raw.Command, raw.Mode, 1, 0)
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			workflowRefusal(w, r, err)
		} else {
			http.Error(w, "league access refused; sign in and check your seat", http.StatusForbidden)
		}
		return
	}
	pin := h.pinFor(r.Context(), league)
	if message := workflowFootballRefusal(raw, pin); message != "" {
		h.Requests.add(raw.Command, raw.Mode, 1, 0)
		http.Error(w, message, http.StatusConflict)
		return
	}
	if !workflowLocationWritable(location.Writable, raw) {
		h.Requests.add(raw.Command, raw.Mode, 1, 0)
		http.Error(w, location.Label, http.StatusConflict)
		return
	}
	ctx := leagueservice.WithFootballDigest(
		leagueservice.WithProblemReportUserAgent(r.Context(), r.UserAgent()), raw.Football)
	data, err := h.workflowData(ctx, service, raw, league)
	if err != nil {
		logWorkflowFailure(raw, err)
		h.Requests.add(raw.Command, raw.Mode, 1, 0)
		workflowActionRefusal(w, r, raw, err)
		return
	}
	h.Requests.add(raw.Command, raw.Mode, 5, uint64(len(data)))
	writable := location.Writable && (!pin.Missing || workflowExemptFromFootballPin(raw.Command))
	response := WorkflowResponse{Label: location.Label, Writable: writable, Football: pin.Digest,
		AppRelease: serverversion.Current().Release, Data: data}
	if !h.attachPrivateWorkflowTags(r.Context(), raw.Command, league, pin.Digest, &response) {
		http.Error(w, "private reading tags unavailable", http.StatusServiceUnavailable)
		return
	}
	if raw.Mode == "read" {
		writeHostedReading(w, r, response)
		return
	}
	writeHosted(w, response)
}

func workflowLocationWritable(writable bool, raw RawWorkflow) bool {
	return raw.Mode == "read" || writable || raw.Command == leagueservice.BackupNowCommand ||
		raw.Command == commissioner.ProblemReportCreate
}

func (h Hosted) attachPrivateWorkflowTags(ctx context.Context, command, league, football string,
	response *WorkflowResponse) bool {
	if command != leagueservice.WhatIfStackRead {
		return true
	}
	response.Tags = h.workflowTags(ctx, league, football)
	return response.Tags != nil
}

func (h Hosted) workflowTags(ctx context.Context, league, football string) *WorkflowTags {
	principal, ok := ctx.Value(replyPrincipalKey{}).(string)
	if !ok || principal == "" || h.Publisher == nil {
		return nil
	}
	versions := h.Publisher.Versions(principal, league)
	leagueTags := make(map[string]string, len(versions.Topics))
	for _, topic := range versions.Topics {
		key := topic.Kind
		switch topic.Kind {
		case "team":
			key += ":" + topic.Franchise
		case "scores":
			key += fmt.Sprintf(":%d:%d", topic.Season, topic.Week)
		case "values":
			key += fmt.Sprintf(":%d:%s", topic.Season, topic.Run)
		case "league":
			if topic.Season != 0 {
				key += fmt.Sprintf(":%d", topic.Season)
			}
		}
		leagueTags[key] = topic.Version
	}
	return &WorkflowTags{ServerEpoch: versions.ServerEpoch, League: leagueTags, Football: football}
}

func logWorkflowFailure(raw RawWorkflow, err error) {
	// Busy failures use the bounded refusal log below; other refusals retain command context.
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return
	}
	log.Printf("hosted league %s %s refused: %v", raw.Command, raw.Mode, err)
}

func workflowActionRefusal(w http.ResponseWriter, r *http.Request, raw RawWorkflow, err error) {
	if raw.Command == commissioner.ProblemReportCreate && errors.Is(err, history.ErrProblemReportRateLimit) {
		http.Error(w, "You have sent 10 reports in the past hour. Try again later.", http.StatusTooManyRequests)
		return
	}
	if raw.Command == leagueservice.BackupNowCommand && raw.Mode == "execute" &&
		errors.Is(err, leagueservice.ErrBackupInProgress) {
		http.Error(w, leagueservice.BackupInProgressMessage, http.StatusConflict)
		return
	}
	if privateMflMovesOperation(raw.Command) && !errors.Is(err, context.DeadlineExceeded) &&
		!errors.Is(err, context.Canceled) {
		var refusal *commissioner.Refusal
		if errors.As(err, &refusal) && !refusal.PermissionDenied() && refusal.Reason() != "" {
			http.Error(w, refusal.Reason(), http.StatusConflict)
			return
		}
	}
	workflowRefusal(w, r, err)
}

func workflowRefusal(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		logBusyRefusal(r.Context(), "request-deadline")
		writeUnavailable(w, "The league server is busy. Try again in a few seconds.")
		return
	}
	http.Error(w, "The league could not accept this action. Refresh before retrying; check your seat, league rules and authenticator.", http.StatusConflict)
}

func workflowExemptFromFootballPin(command string) bool {
	return command == "draft.queue.set" || privateTradeDraftOperation(command) ||
		privateMflMovesOperation(command)
}

func workflowFootballRefusal(raw RawWorkflow, pin LeaguePin) string {
	if raw.Mode == "read" || raw.Command == leagueservice.BackupNowCommand ||
		raw.Command == commissioner.ProblemReportCreate || workflowExemptFromFootballPin(raw.Command) {
		return ""
	}
	if pin.Digest == "" {
		return "Ask the server operator to select football evidence before making changes"
	}
	if pin.Missing {
		return missingFootball(pin.Digest)
	}
	return ""
}

// workflowData runs a workflow, answering a league-state reading from the reply cache when it holds one.
// The envelope (label, writable) is never cached: it is read for every request.
func (h Hosted) workflowData(ctx context.Context, service *leagueservice.Service, raw RawWorkflow, league string) (json.RawMessage, error) {
	if raw.Mode != "read" || raw.Command == "draft.queue.read" || raw.Command == leagueservice.TradeDraftRead ||
		raw.Command == commissioner.ProblemReportsRead || privateMflMovesOperation(raw.Command) {
		return runWorkflow(ctx, service, raw)
	}
	scope, err := h.replyScope(ctx, service, raw.Command, league, raw.Arguments, h.pinFor(ctx, league).cacheKey()+"|"+raw.Football)
	if err != nil {
		return nil, err
	}
	scope.tally, scope.command, scope.mode = h.Requests, raw.Command, raw.Mode
	return scope.reply(func() ([]byte, error) { return runWorkflow(ctx, service, raw) })
}

func runWorkflow(ctx context.Context, service *leagueservice.Service, raw RawWorkflow) (json.RawMessage, error) {
	//nolint:wrapcheck // Preserve the existing workflow result and refusal text.
	return catalog.RunWorkflow(ctx, service, raw)
}

// missingFootball is what an owner or commissioner is told when their league's football data is not
// on this server: what still works, and that changes come back by themselves.
func missingFootball(digest string) string {
	return "This league's football data (version " + digest + ") is not on this server right now, " +
		"so changes wait until it is back. Rosters, contracts and the ledger can still be viewed; " +
		"changes open again by themselves once the football data returns."
}
