package serverhttp

import (
	"context"
	"encoding/json"
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
	league, err := raw.League()
	if err != nil {
		http.Error(w, "invalid league action", http.StatusBadRequest)
		return
	}
	//nolint:contextcheck // The service host carries this request's context and admission.
	location, err := service.Location(league)
	if err != nil {
		http.Error(w, "league access refused; sign in and check your seat", http.StatusForbidden)
		return
	}
	pin := h.pinFor(r.Context(), league)
	if raw.Mode != "read" && pin.Digest == "" {
		http.Error(w, "Ask the server operator to select football evidence before making changes", http.StatusConflict)
		return
	}
	if raw.Mode != "read" && pin.Missing {
		http.Error(w, missingFootball(pin.Digest), http.StatusConflict)
		return
	}
	if raw.Mode != "read" && !location.Writable {
		http.Error(w, location.Label, http.StatusConflict)
		return
	}
	ctx := leagueservice.WithFootballDigest(r.Context(), raw.Football)
	data, err := h.workflowData(ctx, service, raw, league)
	if err != nil {
		// The command name and mode are fixed catalogue words; arguments, bearer and proof are never logged.
		log.Printf("hosted league %s %s refused: %v", raw.Command, raw.Mode, err)
		http.Error(w, "The league could not accept this action. Refresh before retrying; check your seat, league rules and authenticator.", http.StatusConflict)
		return
	}
	response := WorkflowResponse{Label: location.Label, Writable: location.Writable && !pin.Missing, Football: pin.Digest, Data: data}
	if raw.Mode == "read" {
		writeHostedReading(w, r, response)
		return
	}
	writeHosted(w, response)
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
