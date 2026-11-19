package serverhttp

import (
	"log"
	"net/http"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
)

func (h Hosted) accountWorkflow(w http.ResponseWriter, r *http.Request, accountID string, raw RawWorkflow) {
	h.Requests.add(raw.Command, raw.Mode, 0, 0)
	if h.AccountPreferences == nil {
		h.Requests.add(raw.Command, raw.Mode, 1, 0)
		writeUnavailable(w, "Account preferences are unavailable.")
		return
	}
	data, err := catalog.RunAccountWorkflow(r.Context(), h.AccountPreferences, accountID, raw)
	if err != nil {
		log.Printf("hosted account preference %s %s refused (%T)", raw.Command, raw.Mode, err)
		h.Requests.add(raw.Command, raw.Mode, 1, 0)
		workflowRefusal(w, r, err)
		return
	}
	h.Requests.add(raw.Command, raw.Mode, 2, 0)
	h.Requests.add(raw.Command, raw.Mode, 5, uint64(len(data)))
	label := "Account preference"
	if raw.Command == "home.layout.read" || raw.Command == "home.layout.set" {
		label = "Account home layout"
	}
	response := WorkflowResponse{Label: label, Writable: true, Data: data}
	if raw.Mode == "read" {
		writeHostedReading(w, r, response)
		return
	}
	writeHosted(w, response)
}
