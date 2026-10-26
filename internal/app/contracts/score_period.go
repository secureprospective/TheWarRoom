package contracts

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

func ParseScorePeriod(body []byte, period string) ([]ingestion.RawPlayerScore, error) {
	rows, err := ingestion.ParsePlayerScores(body)
	if err != nil {
		return nil, fmt.Errorf("player scores: parse: %w", err)
	}
	out := []ingestion.RawPlayerScore{}
	for _, r := range rows {
		if r.Week == period {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("player scores: period %s not held", period)
	}
	return out, nil
}
