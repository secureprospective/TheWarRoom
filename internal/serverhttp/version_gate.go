package serverhttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/serverversion"
)

// Every client request declares the protocol ("major.minor") and durable command version it was
// built for. A request without them is from a client older than these headers: protocol 1.0.
const (
	ProtocolHeader = "X-TWR-Protocol"
	CommandsHeader = "X-TWR-Commands"
	// UpdateHeader on a 426 reply names who must update: "app" or "server".
	UpdateHeader = "X-TWR-Update"
)

// RawUpdateRequiredResponse is the 426 reply body: who must update, and the sentence to show.
type RawUpdateRequiredResponse struct {
	Protocol int    `json:"protocol"`
	Update   string `json:"update"`
	Message  string `json:"message"`
}

func (r RawUpdateRequiredResponse) Validate() error {
	if r.Protocol != serverversion.ProtocolMajor || (r.Update != string(serverversion.App) && r.Update != string(serverversion.Server)) ||
		r.Message == "" || len(r.Message) > 512 {
		return fmt.Errorf("invalid update-required reply")
	}
	return nil
}

// versionGate refuses an unsupported client on every route before any session, store or change
// stream is reached, so a client that never opens the stream still learns it must update.
func versionGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		protocol, okProtocol := oneHeader(r, ProtocolHeader)
		commands, okCommands := oneHeader(r, CommandsHeader)
		err := serverversion.Admit(protocol, commands)
		var update *serverversion.UpdateRequired
		switch {
		case !okProtocol || !okCommands || errors.Is(err, serverversion.ErrMalformed):
			http.Error(w, "invalid version header", http.StatusBadRequest)
		case errors.As(err, &update):
			body, err := json.Marshal(RawUpdateRequiredResponse{Protocol: serverversion.ProtocolMajor, Update: string(update.Update), Message: update.Message})
			if err != nil {
				http.Error(w, "response unavailable", http.StatusInternalServerError)
				return
			}
			w.Header().Set(UpdateHeader, string(update.Update))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUpgradeRequired)
			_, _ = w.Write(body)
		case err != nil:
			http.Error(w, "invalid version header", http.StatusBadRequest)
		default:
			next.ServeHTTP(w, r)
		}
	})
}

// oneHeader returns a header sent at most once; present means nonempty.
func oneHeader(r *http.Request, name string) (string, bool) {
	values := r.Header.Values(name)
	switch {
	case len(values) == 0:
		return "", true
	case len(values) == 1 && values[0] != "":
		return values[0], true
	}
	return "", false
}
