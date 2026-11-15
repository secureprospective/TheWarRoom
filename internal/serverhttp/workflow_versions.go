package serverhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"sync"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

type ReadTags = catalog.ReadTags

// WorkflowTopic is an already recipient-authorized publication. Key retains its exact wire scope.
type WorkflowTopic struct {
	Key       string
	Kind      string
	Franchise string
	Season    int
	Week      int
	Version   string
}

type WorkflowVersionSnapshot struct {
	ServerEpoch string
	Topics      []WorkflowTopic
	Live        string
	Values      string
}

// VersionSource must copy snapshots and serialize observations with stream publications.
// Observe must not let an older recipient revision roll back a newer observation.
type VersionSource interface {
	Observe(account, league string, revision int64)
	Snapshot(account, league string) WorkflowVersionSnapshot
}

type observedView struct {
	account, league string
}

// WorkflowVersions shares read-time observation across requests, including disconnected callers.
type WorkflowVersions struct {
	mu     sync.Mutex
	source VersionSource
	heads  map[observedView]int64
}

func NewWorkflowVersions(source VersionSource) *WorkflowVersions {
	return &WorkflowVersions{source: source, heads: make(map[observedView]int64)}
}

func (v *WorkflowVersions) snapshot(ctx context.Context, h Hosted, service *leagueservice.Service, account, league string) (WorkflowVersionSnapshot, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	head, err := h.leagueHead(ctx, league)
	if err != nil {
		return WorkflowVersionSnapshot{}, fmt.Errorf("observe workflow head: %w", err)
	}
	key := observedView{account, league}
	if previous, ok := v.heads[key]; !ok || previous != head {
		//nolint:contextcheck // The service host owns this request's context and admission.
		reading, err := service.TargetCommissionerLedger(league)
		if err != nil {
			return WorkflowVersionSnapshot{}, fmt.Errorf("observe recipient revision: %w", err)
		}
		v.source.Observe(account, league, reading.Revision)
		if len(v.heads) >= 1024 {
			clear(v.heads)
		}
		v.heads[key] = head
	}
	return v.source.Snapshot(account, league), nil
}

func (h Hosted) workflowTags(ctx context.Context, service *leagueservice.Service, raw RawWorkflow, league, football string) (*ReadTags, error) {
	account, _ := ctx.Value(replyPrincipalKey{}).(string)
	if h.Versions == nil || h.Versions.source == nil || account == "" {
		return nil, nil
	}
	snapshot, err := h.Versions.snapshot(ctx, h, service, account, league)
	if err != nil {
		return nil, err
	}
	var scope RawWorkflowScope
	if raw.Mode == "read" {
		if err := json.Unmarshal(raw.Arguments, &scope); err != nil {
			return nil, fmt.Errorf("read tag scope: %w", err)
		}
	}
	tags := &ReadTags{ServerEpoch: snapshot.ServerEpoch, League: make(map[string]string), Football: football, Live: snapshot.Live, Values: snapshot.Values}
	for _, topic := range snapshot.Topics {
		if topic.Kind == "league" || raw.Command == "versions.read" || matchesWorkflowDependency(topic, catalog.ReadDependencies(raw.Command, scope)) {
			tags.League[topic.Key] = topic.Version
		}
	}
	if err := tags.Validate(); err != nil {
		return nil, fmt.Errorf("published workflow tags: %w", err)
	}
	return tags, nil
}

func matchesWorkflowDependency(topic WorkflowTopic, dependencies []catalog.ReadDependency) bool {
	for _, dependency := range dependencies {
		if dependency.Kind == topic.Kind && (dependency.Franchise == "" || topic.Franchise == "" || dependency.Franchise == topic.Franchise) &&
			(dependency.Season == 0 || topic.Season == 0 || dependency.Season == topic.Season) &&
			(dependency.Week == 0 || topic.Week == 0 || dependency.Week == topic.Week) {
			return true
		}
	}
	return false
}

func sameReadTags(current, known *ReadTags) bool {
	return current != nil && known != nil && current.ServerEpoch != "" &&
		current.ServerEpoch == known.ServerEpoch && current.Football == known.Football &&
		current.Live == known.Live && current.Values == known.Values && maps.Equal(current.League, known.League)
}

func admitWorkflowRead(service *leagueservice.Service, raw RawWorkflow, league string) error {
	var scope RawWorkflowScope
	if err := json.Unmarshal(raw.Arguments, &scope); err != nil {
		return fmt.Errorf("read admission scope: %w", err)
	}
	franchise := scope.Franchise
	if scope.Viewer != "" {
		franchise = scope.Viewer
	}
	operation := raw.Command
	if raw.Command == "commish.market.read" {
		operation = scope.Command
	}
	if raw.Command == "versions.read" {
		operation = "native.directory.read"
	}

	if err := service.AdmitRead(league, operation, franchise); err != nil {
		return fmt.Errorf("admit workflow reading: %w", err)
	}
	return nil
}
