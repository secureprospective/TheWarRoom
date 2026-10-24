package history

import "github.com/secureprospective/TheWarRoom/internal/commissioner"

const (
	maxCachedLeagues = 4
	maxCachedRecords = 100_000
)

// cacheRecords runs under recordsMu. Bound both decoded records and their projections;
// oversized leagues still work, but replay without retaining another revision in memory.
func (s *Store) cacheRecords(league string, revision int64, records []commissioner.Record) []commissioner.Record {
	delete(s.records, league)
	if len(records) > maxCachedRecords {
		return records
	}
	retained := len(records)
	for _, hit := range s.records {
		retained += len(hit.records)
	}
	if retained > maxCachedRecords || len(s.records) >= maxCachedLeagues {
		clear(s.records)
	}
	if s.records == nil {
		s.records = map[string]recordRevision{}
	}
	records = commissioner.WithReplayCache(records)
	s.records[league] = recordRevision{revision: revision, records: records}
	return records
}
