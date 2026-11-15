package serverhttp

import (
	"strconv"
	"strings"
)

// PublisherVersionSource uses the same publications as the invalidation stream.
type PublisherVersionSource struct{ Publisher *Publisher }

func (s PublisherVersionSource) Observe(account, league string, revision int64) {
	s.Publisher.ObserveView(account, league, revision)
}

func (s PublisherVersionSource) Snapshot(account, league string) WorkflowVersionSnapshot {
	publication := s.Publisher.Versions(account, league)
	snapshot := WorkflowVersionSnapshot{ServerEpoch: publication.ServerEpoch}
	for _, topic := range publication.Topics {
		parts := []string{topic.Kind}
		if topic.Franchise != "" {
			parts = append(parts, topic.Franchise)
		}
		if topic.Season != 0 {
			parts = append(parts, strconv.Itoa(topic.Season))
		}
		if topic.Week != 0 {
			parts = append(parts, strconv.Itoa(topic.Week))
		}
		if topic.Run != "" {
			parts = append(parts, topic.Run)
		}
		snapshot.Topics = append(snapshot.Topics, WorkflowTopic{Key: strings.Join(parts, ":"), Kind: topic.Kind, Franchise: topic.Franchise, Season: topic.Season, Week: topic.Week, Version: topic.Version})
	}
	return snapshot
}
