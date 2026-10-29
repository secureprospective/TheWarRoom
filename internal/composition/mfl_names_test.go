package composition

import (
	"encoding/json"
	"strings"
	"testing"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
)

func TestMFLRenamedPlansDispatchAndPersist(t *testing.T) {
	for _, name := range []string{"mfl.lineup.set", "mfl.trade.accept"} {
		t.Run(name, func(t *testing.T) {
			app, transport, offer := tradeDraftApp(t)
			registry, err := catalog.NewRegistry(newCommandAPIs(app).registryInputs(app))
			if err != nil {
				t.Fatal(err)
			}
			args := map[string]string{"franchiseId": "0025", "tradeId": offer.ID}
			body, err := json.Marshal(args)
			if err != nil {
				t.Fatal(err)
			}
			if name == "mfl.lineup.set" {
				ids := lineupTestSnapshot(t, app)
				body, err = json.Marshal(struct {
					Franchise string   `json:"franchiseId"`
					Starters  []string `json:"starters"`
				}{"0025", changedStarters(ids)})
				if err != nil {
					t.Fatal(err)
				}
			}
			raw, err := registry.Invoke(t.Context(), catalog.MFLDoor, name, catalog.Execute, body)
			if err != nil {
				t.Fatal(err)
			}
			var receipt envelope.Receipt
			if err := json.Unmarshal(raw, &receipt); err != nil {
				t.Fatal(err)
			}
			if receipt.Spec.Intent != name || receipt.State != envelope.Ready {
				t.Fatalf("not a ready local plan: %+v", receipt)
			}
			saved, err := app.core.Moves.Get(t.Context(), receipt.CorrelationID)
			if err != nil || saved.Receipt().Spec.Intent != name {
				t.Fatalf("saved plan: %+v %v", saved, err)
			}
			if len(transport.requests) != 0 {
				t.Fatal("local plan used MFL transport")
			}
			for _, id := range []string{strings.TrimPrefix(name, "mfl."), name} {
				door := catalog.NativeDoor
				if id != name {
					door = catalog.MFLDoor
				}
				if _, err := registry.Invoke(t.Context(), door, id, catalog.Execute, body); err == nil || !strings.Contains(err.Error(), "not available") {
					t.Fatalf("crossed boundary: %s %s: %v", door, id, err)
				}
			}
		})
	}
}
