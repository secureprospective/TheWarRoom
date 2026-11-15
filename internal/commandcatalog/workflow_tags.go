package catalog

import (
	"fmt"
	"strings"
)

// ReadTags names the evidence used by a workflow, not its optimistic concurrency token.
type ReadTags struct {
	ServerEpoch string            `json:"serverEpoch"`
	League      map[string]string `json:"league"`
	Football    string            `json:"football"`
	Live        string            `json:"live"`
	Values      string            `json:"values"`
}

func (t ReadTags) Validate() error {
	if len(t.ServerEpoch) > 128 || len(t.Football) > 256 || len(t.Live) > 128 || len(t.Values) > 128 || t.League == nil || len(t.League) > 512 {
		return fmt.Errorf("invalid reading tags")
	}
	for key, version := range t.League {
		if len(key) == 0 || len(key) > 256 || len(version) == 0 || len(version) > 32 || strings.IndexFunc(version, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
			return fmt.Errorf("invalid topic version")
		}
	}
	return nil
}

// ReadDependency uses zero scope fields as wildcards, like the client read owner.
type ReadDependency struct {
	Kind      string
	Franchise string
	Season    int
	Week      int
}

// ReadDependencies is the single workflow invalidation policy, beside commandScope.
func ReadDependencies(command string, scope RawWorkflowScope) []ReadDependency {
	league := ReadDependency{Kind: "league"}
	scores := ReadDependency{Kind: "scores", Season: scope.Season}
	clock := ReadDependency{Kind: "clock"}
	values := ReadDependency{Kind: "values"}
	switch command {
	case "bracket.read":
		return []ReadDependency{league, scores, clock}
	case "results.read":
		scores.Week = scope.Week
		return []ReadDependency{scores, league}
	case "standings.read":
		return []ReadDependency{scores, league}
	case "odds.read":
		return []ReadDependency{scores, league, values}
	case "lineup.read":
		return []ReadDependency{{Kind: "team", Franchise: scope.Franchise}, league, clock}
	case "projections.read":
		scores.Week = scope.Week
		return []ReadDependency{values, scores}
	case "week.read":
		return []ReadDependency{clock, league}
	default:
		return []ReadDependency{league}
	}
}

// VersionsReading reconciles a caller's cache without disclosing ledger positions.
type VersionsReading struct {
	ServerEpoch  string            `json:"serverEpoch"`
	Access       string            `json:"access"`
	League       string            `json:"league"`
	Season       int               `json:"season"`
	Topics       map[string]string `json:"topics"`
	Football     string            `json:"football"`
	Live         string            `json:"live"`
	Values       string            `json:"values"`
	ServerTime   string            `json:"serverTime"`
	NextBoundary string            `json:"nextBoundary"`
}
