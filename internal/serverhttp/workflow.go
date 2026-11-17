package serverhttp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"log"
	"net/http"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

const WorkflowPath = "/v1/native/workflow"

type RawWorkflow = catalog.RawWorkflow
type RawWorkflowScope = catalog.RawWorkflowScope

// WorkflowResponse carries the authority's own wording; a cached writable bit is never a grant.
type WorkflowResponse struct {
	Label    string          `json:"label"`
	Writable bool            `json:"writable"`
	Football string          `json:"football"`
	Data     json.RawMessage `json:"data"`
}

func (r WorkflowResponse) Validate() error {
	if r.Label == "" || !json.Valid(r.Data) {
		return fmt.Errorf("unreadable league response")
	}
	return nil
}

func (h Hosted) workflow(w http.ResponseWriter, r *http.Request, service *leagueservice.Service, raw RawWorkflow) {
	h.Requests.add(raw.Command, raw.Mode, 0, 0)
	league, err := raw.League()
	if err != nil {
		h.Requests.add(raw.Command, raw.Mode, 1, 0)
		http.Error(w, "invalid league action", http.StatusBadRequest)
		return
	}
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
	if raw.Mode != "read" && pin.Digest == "" {
		h.Requests.add(raw.Command, raw.Mode, 1, 0)
		http.Error(w, "Ask the server operator to select football evidence before making changes", http.StatusConflict)
		return
	}
	if raw.Mode != "read" && pin.Missing {
		h.Requests.add(raw.Command, raw.Mode, 1, 0)
		http.Error(w, missingFootball(pin.Digest), http.StatusConflict)
		return
	}
	if raw.Mode != "read" && !location.Writable {
		h.Requests.add(raw.Command, raw.Mode, 1, 0)
		http.Error(w, location.Label, http.StatusConflict)
		return
	}
	ctx := leagueservice.WithFootballDigest(r.Context(), raw.Football)
	data, err := h.workflowData(ctx, service, raw, league)
	if err != nil {
		logWorkflowFailure(raw, err)
		h.Requests.add(raw.Command, raw.Mode, 1, 0)
		workflowRefusal(w, r, err)
		return
	}
	h.Requests.add(raw.Command, raw.Mode, 5, uint64(len(data)))
	response := WorkflowResponse{Label: location.Label, Writable: location.Writable && !pin.Missing, Football: pin.Digest, Data: data}
	if raw.Mode == "read" {
		writeHostedReading(w, r, response)
		return
	}
	writeHosted(w, response)
}

func logWorkflowFailure(raw RawWorkflow, err error) {
	// Busy failures use the bounded refusal log below; other refusals retain command context.
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return
	}
	log.Printf("hosted league %s %s refused: %v", raw.Command, raw.Mode, err)
}

func workflowRefusal(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		logBusyRefusal(r.Context(), "request-deadline")
		w.Header().Set("Retry-After", busyRetryAfter)
		http.Error(w, "The league server is busy. Try again in a few seconds.", http.StatusServiceUnavailable)
		return
	}
	http.Error(w, "The league could not accept this action. Refresh before retrying; check your seat, league rules and authenticator.", http.StatusConflict)
}

// workflowData runs a workflow, answering a league-state reading from the reply cache when it holds one.
// The envelope (label, writable) is never cached: it is read for every request.
func (h Hosted) workflowData(ctx context.Context, service *leagueservice.Service, raw RawWorkflow, league string) (json.RawMessage, error) {
	if raw.Mode != "read" {
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
