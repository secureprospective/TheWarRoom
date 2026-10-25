package serverclient

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

// Handler is mounted only by native-only Wails startup, never on a TCP listener.
// JSON POSTs and the exact webview origin prevent cross-origin browser requests.
func (d *Desktop) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if (r.URL.Path != DesktopPath && r.URL.Path != DesktopWorkflowPath) || r.URL.RawQuery != "" {
			http.NotFound(w, r)
			return
		}
		if !desktopOrigin(r.Header.Get("Origin")) {
			http.Error(w, "desktop origin required", http.StatusForbidden)
			return
		}
		if r.URL.Path == DesktopWorkflowPath {
			d.workflowHTTP(w, r)
			return
		}
		raw := RawDesktop{Action: "status"}
		if r.Method == http.MethodPost {
			if r.Header.Get("Content-Type") != "application/json" {
				http.Error(w, "JSON required", http.StatusBadRequest)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&raw); err != nil {
				http.Error(w, "invalid desktop request", http.StatusBadRequest)
				return
			}
			var trailing json.RawMessage
			if err := decoder.Decode(&trailing); err != io.EOF {
				http.Error(w, "one desktop request required", http.StatusBadRequest)
				return
			}
		} else if r.Method != http.MethodGet {
			http.Error(w, "use GET or POST", http.StatusMethodNotAllowed)
			return
		}
		view, err := d.Act(r.Context(), raw)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		body, err := json.Marshal(view)
		if err != nil {
			http.Error(w, "desktop response unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(body); err != nil {
			return
		}
	})
}

func desktopOrigin(origin string) bool {
	// Wails uses wails://wails on WebKit and http://wails.localhost on Windows.
	return origin == "" || origin == "http://wails.localhost" || origin == "https://wails.localhost" || origin == "wails://wails"
}

func (d *Desktop) workflowHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "use a JSON league action", http.StatusBadRequest)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var raw serverhttp.RawWorkflow
	if err := decoder.Decode(&raw); err != nil {
		http.Error(w, "invalid league action", http.StatusBadRequest)
		return
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		http.Error(w, "send one league action", http.StatusBadRequest)
		return
	}
	response, err := d.Workflow(r.Context(), raw)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	body, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "league response unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(body); err != nil {
		return
	}
}
