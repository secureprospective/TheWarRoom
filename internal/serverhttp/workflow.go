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
	if raw.Mode != "read" && h.FootballDigest == "" {
		http.Error(w, "Ask the server operator to select football evidence before making changes", http.StatusConflict)
		return
	}
	if raw.Mode != "read" && !location.Writable {
		http.Error(w, location.Label, http.StatusConflict)
		return
	}
	ctx := leagueservice.WithFootballDigest(r.Context(), raw.Football)
	data, err := runWorkflow(ctx, service, raw)
	if err != nil {
		// The command name and mode are fixed catalogue words; arguments, bearer and proof are never logged.
		log.Printf("hosted league %s %s refused: %v", raw.Command, raw.Mode, err)
		http.Error(w, "The league could not accept this action. Refresh before retrying; check your seat, league rules and authenticator.", http.StatusConflict)
		return
	}
	writeHosted(w, WorkflowResponse{Label: location.Label, Writable: location.Writable, Football: h.FootballDigest, Data: data})
}

func runWorkflow(ctx context.Context, service *leagueservice.Service, raw RawWorkflow) (json.RawMessage, error) {
	//nolint:wrapcheck // Preserve the existing workflow result and refusal text.
	return catalog.RunWorkflow(ctx, service, raw)
}
