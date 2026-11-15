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

const workflowRefusalMessage = "The league could not accept this action. Refresh before retrying; check your seat, league rules and authenticator."

type RawWorkflow = catalog.RawWorkflow
type RawWorkflowScope = catalog.RawWorkflowScope

type UnchangedResponse struct {
	Unchanged bool      `json:"unchanged"`
	Tags      *ReadTags `json:"tags"`
}

// WorkflowResponse carries the authority's own wording; a cached writable bit is never a grant.
type WorkflowResponse struct {
	Label     string          `json:"label"`
	Writable  bool            `json:"writable"`
	Football  string          `json:"football"`
	Data      json.RawMessage `json:"data,omitempty"`
	Tags      *ReadTags       `json:"tags,omitempty"`
	Unchanged bool            `json:"unchanged,omitempty"`
}

func (r WorkflowResponse) MarshalJSON() ([]byte, error) {
	if r.Unchanged {
		body, err := json.Marshal(UnchangedResponse{true, r.Tags})
		if err != nil {
			return nil, fmt.Errorf("encode unchanged workflow: %w", err)
		}
		return body, nil
	}
	type wire WorkflowResponse
	body, err := json.Marshal(wire(r))
	if err != nil {
		return nil, fmt.Errorf("encode workflow: %w", err)
	}
	return body, nil
}

func (r WorkflowResponse) Validate() error {
	if r.Tags != nil {
		if err := r.Tags.Validate(); err != nil {
			return fmt.Errorf("read response tags: %w", err)
		}
	}
	if r.Unchanged {
		if r.Tags == nil || len(r.Data) != 0 {
			return fmt.Errorf("unreadable unchanged response")
		}
		return nil
	}
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
	if refusal := workflowWriteRefusal(raw.Mode, pin, location.Label, location.Writable); refusal != "" {
		http.Error(w, refusal, http.StatusConflict)
		return
	}
	ctx := leagueservice.WithFootballDigest(r.Context(), raw.Football)
	if raw.Mode == "read" {
		//nolint:contextcheck // The service host owns this request context.
		if err := admitWorkflowRead(service, raw, league); err != nil {
			http.Error(w, workflowRefusalMessage, http.StatusConflict)
			return
		}
	}
	tags, err := h.workflowTags(ctx, service, raw, league, pin.Digest)
	if err != nil {
		http.Error(w, "league versions unavailable", http.StatusServiceUnavailable)
		return
	}
	if raw.Mode == "read" && raw.Command != "versions.read" && sameReadTags(tags, raw.Known) {
		//nolint:contextcheck // The service host owns this request context.
		writeUnchangedWorkflow(w, r, service, raw, league, tags)
		return
	}
	data, err := h.workflowReplyData(r, service, raw, league, tags)
	if err != nil {
		// The command name and mode are fixed catalogue words; arguments, bearer and proof are never logged.
		log.Printf("hosted league %s %s refused: %v", raw.Command, raw.Mode, err)
		http.Error(w, workflowRefusalMessage, http.StatusConflict)
		return
	}
	response := WorkflowResponse{Label: location.Label, Writable: location.Writable && !pin.Missing, Football: pin.Digest, Data: data, Tags: tags}
	if raw.Mode == "read" {
		//nolint:contextcheck // The service host owns this request context.
		if err := admitWorkflowRead(service, raw, league); err != nil {
			http.Error(w, "league reading refused", http.StatusForbidden)
			return
		}
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

func workflowWriteRefusal(mode string, pin LeaguePin, label string, writable bool) string {
	if mode == "read" {
		return ""
	}
	if pin.Digest == "" {
		return "Ask the server operator to select football evidence before making changes"
	}
	if pin.Missing {
		return missingFootball(pin.Digest)
	}
	if !writable {
		return label
	}
	return ""
}

func writeUnchangedWorkflow(w http.ResponseWriter, r *http.Request, service *leagueservice.Service, raw RawWorkflow, league string, tags *ReadTags) {

	if err := admitWorkflowRead(service, raw, league); err != nil {
		http.Error(w, "league reading refused", http.StatusForbidden)
		return
	}
	// TODO(request metrics): count unchanged when the Requests observer lands.
	writeHostedReading(w, r, UnchangedResponse{true, tags})
}
