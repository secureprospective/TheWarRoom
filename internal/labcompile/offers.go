package labcompile

import (
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/facts"
)

type OfferColumns struct {
	Ts      []int64   `json:"ts"`
	Year    []int64   `json:"year"`
	Season  []int64   `json:"season"`
	Phase   []*string `json:"phase"`
	By      []*string `json:"by"`
	To      []*string `json:"to"`
	Outcome []string  `json:"outcome"`
	Decided []*int64  `json:"decided"`
}

func outcome(kind string) string {
	switch kind {
	case "TRADE_ACCEPT":
		return "accepted"
	case "TRADE_REJECTION":
		return "rejected"
	case "TRADE_REVOKE":
		return "withdrawn"
	case "TRADE_OFFER_EXPIRED":
		return "expired"
	}
	return ""
}
func equalPtr[T ~int64 | ~string](a, b *T) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}
func (l *Lab) offerOutcomes(v facts.OfferColumns) OfferColumns {
	indices := []int{}
	for i, ts := range v.Ts {
		if value(ts) != 0 {
			indices = append(indices, i)
		}
	}
	slices.SortFunc(indices, func(a, b int) int {
		if *v.Ts[a] < *v.Ts[b] {
			return -1
		}
		if *v.Ts[a] > *v.Ts[b] {
			return 1
		}
		if v.TxnID[a] < v.TxnID[b] {
			return -1
		}
		if v.TxnID[a] > v.TxnID[b] {
			return 1
		}
		return 0
	})
	used := map[int]bool{}
	out := OfferColumns{}
	for _, i := range indices {
		if v.Type[i] != "TRADE_PROPOSAL" {
			continue
		}
		end := matchOffer(v, indices, used, i)
		year, ts := v.Year[i], *v.Ts[i]
		season := year
		if regularEnd := l.Marks[year].RegularEnd; regularEnd != nil && dayOf(ts).After(*regularEnd) {
			season++
		}
		var phase *string
		if _, ok := l.Marks[year]; ok {
			phase = ptr(l.phase(year, ts))
		}
		out.Ts = append(out.Ts, ts)
		out.Year = append(out.Year, year)
		out.Season = append(out.Season, season)
		out.Phase = append(out.Phase, phase)
		out.By = append(out.By, v.OfferedBy[i])
		out.To = append(out.To, v.OfferedTo[i])
		result := "unknown"
		var decided *int64
		if end >= 0 {
			used[end] = true
			result = outcome(v.Type[end])
			decided = v.Ts[end]
		}
		out.Outcome = append(out.Outcome, result)
		out.Decided = append(out.Decided, decided)
	}
	return out
}
func matchOffer(v facts.OfferColumns, indices []int, used map[int]bool, i int) int {
	first := -1
	for _, j := range indices {
		if used[j] || outcome(v.Type[j]) == "" || *v.Ts[j] < *v.Ts[i] {
			continue
		}
		if !equalPtr(v.OfferedBy[j], v.OfferedBy[i]) || !equalPtr(v.OfferedTo[j], v.OfferedTo[i]) || !equalPtr(v.Expires[j], v.Expires[i]) {
			continue
		}
		if first < 0 {
			first = j
		}
		if value(v.Comments[j]) == value(v.Comments[i]) {
			return j
		}
	}
	return first
}
