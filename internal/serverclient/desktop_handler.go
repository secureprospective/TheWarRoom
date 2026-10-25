package serverclient

import (
	"encoding/json"
	"io"
	"net/http"
)

// Handler is mounted only by native-only Wails startup, never on a TCP listener.
// JSON POSTs and the exact webview origin prevent cross-origin browser requests.
func (d *Desktop) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.URL.Path != DesktopPath || r.URL.RawQuery != "" {
			http.NotFound(w, r)
			return
		}
		if !desktopOrigin(r.Header.Get("Origin")) {
			http.Error(w, "desktop origin required", http.StatusForbidden)
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
			if update, ok := updateRequired(err); ok {
				w.Header().Set("X-TWR-Update", string(update.Update))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUpgradeRequired)
				_ = json.NewEncoder(w).Encode(view)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		body, err := json.Marshal(view)
		if err != nil {
			http.Error(w, "desktop response unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if view.Update != "" && raw.Action != "status" {
			w.Header().Set("X-TWR-Update", view.Update)
			w.WriteHeader(http.StatusUpgradeRequired)
		}
		if _, err := w.Write(body); err != nil {
			return
		}
	})
}

func desktopOrigin(origin string) bool {
	// Wails uses wails://wails on WebKit and http://wails.localhost on Windows.
	return origin == "" || origin == "http://wails.localhost" || origin == "https://wails.localhost" || origin == "wails://wails"
}
