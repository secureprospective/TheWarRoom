package catalog_test

import (
	"encoding/json"
	"reflect"
	"testing"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
)

func TestWorkflowReadDependencies(t *testing.T) {
	scope := catalog.RawWorkflowScope{Season: 2026, Week: 7, Franchise: "franchise.01"}
	league := catalog.ReadDependency{Kind: "league"}
	scores := catalog.ReadDependency{Kind: "scores", Season: 2026}
	weekScores := scores
	weekScores.Week = 7
	clock := catalog.ReadDependency{Kind: "clock"}
	values := catalog.ReadDependency{Kind: "values"}
	for _, tc := range []struct {
		command string
		want    []catalog.ReadDependency
	}{
		{"native.directory.read", []catalog.ReadDependency{league}},
		{"schedule.read", []catalog.ReadDependency{league}},
		{"bracket.read", []catalog.ReadDependency{league, scores, clock}},
		{"results.read", []catalog.ReadDependency{weekScores, league}},
		{"standings.read", []catalog.ReadDependency{scores, league}},
		{"odds.read", []catalog.ReadDependency{scores, league, values}},
		{"lineup.read", []catalog.ReadDependency{{Kind: "team", Franchise: "franchise.01"}, league, clock}},
		{"projections.read", []catalog.ReadDependency{values, weekScores}},
		{"week.read", []catalog.ReadDependency{clock, league}},
		{"future.read", []catalog.ReadDependency{league}},
	} {
		t.Run(tc.command, func(t *testing.T) {
			if got := catalog.ReadDependencies(tc.command, scope); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("dependencies: %+v", got)
			}
		})
	}
}

func TestWorkflowKnownTagsAreValidated(t *testing.T) {
	for _, known := range []string{`{"league":null}`, `{"league":{"league":"-1"}}`, `{"league":{"league":1}}`, `{"league":{"league":"1.5"}}`} {
		var raw catalog.RawWorkflow
		body := `{"command":"native.assets.read","version":"1","mode":"read","arguments":{"league":"native.hosted"},"known":` + known + `}`
		if err := json.Unmarshal([]byte(body), &raw); err == nil && raw.Validate() == nil {
			t.Fatalf("accepted malformed known: %s", known)
		}
	}
	raw := catalog.RawWorkflow{Command: "versions.read", Version: "1", Mode: "read", Arguments: json.RawMessage(`{"league":"native.hosted","season":2026}`)}
	if raw.Validate() == nil {
		t.Fatal("versions accepts scope beyond league")
	}
}
