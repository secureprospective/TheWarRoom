package labcompile

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/facts"
)

type rawMethodSample struct {
	Args   []json.RawMessage `json:"args"`
	Result json.RawMessage   `json:"result"`
}
type rawMethods struct {
	Facts         facts.Facts     `json:"facts"`
	Trades        []Trade         `json:"trades"`
	Moves         []Move          `json:"moves"`
	OfferRows     []facts.Offer   `json:"offerRows"`
	OfferOutcomes json.RawMessage `json:"offer_outcomes"`
	Reconcile     json.RawMessage `json:"reconcile"`
}

func (r rawMethods) Validate() error {
	v := reflect.ValueOf(r.Facts)
	for i := 0; i < v.NumField(); i++ {
		cols := v.Field(i)
		n := 0
		for j := 0; j < cols.NumField(); j++ {
			if cols.Field(j).Len() > 0 {
				if n != 0 && n != cols.Field(j).Len() {
					return fmt.Errorf("method facts %s has unequal columns", v.Type().Field(i).Name)
				}
				n = cols.Field(j).Len()
			}
		}
	}
	return nil
}
func fillOmittedColumns(f *facts.Facts) {
	v := reflect.ValueOf(f).Elem()
	for i := 0; i < v.NumField(); i++ {
		cols := v.Field(i)
		n := 0
		for j := 0; j < cols.NumField(); j++ {
			n = max(n, cols.Field(j).Len())
		}
		for j := 0; j < cols.NumField(); j++ {
			if cols.Field(j).Len() == 0 {
				cols.Field(j).Set(reflect.MakeSlice(cols.Field(j).Type(), n, n))
			}
		}
	}
}
func TestReferenceLabMethods(t *testing.T) {
	body := readGzip(t, "testdata/lab-methods.json.gz")
	var raw rawMethods
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	if err := raw.Validate(); err != nil {
		t.Fatal(err)
	}
	fillOmittedColumns(&raw.Facts)
	l, err := newLab(raw.Facts, 0)
	if err != nil {
		t.Fatal(err)
	}
	spans := map[int64]draftSpan{}
	// Calendar inputs are original trades and original draft spans, not Go-produced markers.
	var extra struct {
		DraftSpans map[int64][2]int64 `json:"draftSpans"`
	}
	if err := json.Unmarshal(body, &extra); err != nil {
		t.Fatal(err)
	}
	for y, s := range extra.DraftSpans {
		spans[y] = draftSpan{s[0], s[1]}
	}
	l.buildCalendar(raw.Trades, spans)
	var methods map[string]json.RawMessage
	if err := json.Unmarshal(body, &methods); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"week_at", "phase", "situation", "year_before", "cap_room", "contract", "contract_after", "tier_from_rank", "known_tier", "live_rank", "after_use", "era_of", "norm_pos"} {
		t.Run(name, func(t *testing.T) {
			var samples []rawMethodSample
			if err := json.Unmarshal(methods[name], &samples); err != nil {
				t.Fatal(err)
			}
			for i, s := range samples {
				got, err := runMethod(l, name, s.Args)
				if err != nil {
					t.Fatal(err)
				}
				compareJSON(t, got, s.Result, fmt.Sprintf("%s[%d]", name, i))
			}
			t.Logf("samples=%d", len(samples))
		})
	}
	result, err := json.Marshal(l.reconcile(raw.Trades, raw.Moves))
	if err != nil {
		t.Fatal(err)
	}
	compareJSON(t, result, raw.Reconcile, "reconcile")
	offers := offerColumns(raw.OfferRows)
	result, err = json.Marshal(l.offerOutcomes(offers))
	if err != nil {
		t.Fatal(err)
	}
	compareJSON(t, result, raw.OfferOutcomes, "offer_outcomes")
}
func offerColumns(rows []facts.Offer) facts.OfferColumns {
	out := facts.OfferColumns{}
	for _, r := range rows {
		out.TxnID = append(out.TxnID, r.TxnID)
		out.Year = append(out.Year, r.Year)
		out.Ts = append(out.Ts, r.Ts)
		out.Type = append(out.Type, r.Type)
		out.OfferedBy = append(out.OfferedBy, r.OfferedBy)
		out.OfferedTo = append(out.OfferedTo, r.OfferedTo)
		out.Comments = append(out.Comments, r.Comments)
		out.Expires = append(out.Expires, r.Expires)
	}
	return out
}
func argument[T int64 | float64 | string | *string | *int64 | []int64](args []json.RawMessage, i int) T {
	var out T
	// Invalid oracle arguments are caught before dispatch, rather than silently becoming zero.
	if err := json.Unmarshal(args[i], &out); err != nil {
		panic(fmt.Sprintf("invalid reference argument: %v", err))
	}
	return out
}
func methodJSON[T *int64 | int64 | *float64 | *string | string | *Contract](v T) (json.RawMessage, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("method result: %w", err)
	}
	return body, nil
}
func runMethod(l *Lab, name string, a []json.RawMessage) (json.RawMessage, error) {
	switch name {
	case "week_at":
		return methodJSON(l.weekAt(argument[int64](a, 0), argument[int64](a, 1)))
	case "phase":
		return methodJSON(l.phase(argument[int64](a, 0), argument[int64](a, 1)))
	case "situation":
		return situationJSON(l.situation(argument[int64](a, 0), argument[int64](a, 1), argument[string](a, 2)))
	case "year_before":
		return methodJSON(l.yearBefore(argument[int64](a, 0), argument[*string](a, 1), argument[string](a, 2)))
	case "cap_room":
		return methodJSON(l.capRoom(argument[int64](a, 0), argument[int64](a, 1), argument[string](a, 2)))
	case "contract":
		return methodJSON(l.contract(argument[string](a, 0), argument[int64](a, 1), argument[int64](a, 2), argument[string](a, 3)))
	case "contract_after":
		return methodJSON(l.contractAfter(argument[string](a, 0), argument[int64](a, 1), argument[int64](a, 2), argument[string](a, 3)))
	case "tier_from_rank":
		return methodJSON(l.tierFromRank(argument[*int64](a, 0), argument[*string](a, 1), argument[int64](a, 2)))
	case "known_tier":
		return methodJSON(l.knownTier(argument[string](a, 0), argument[int64](a, 1), argument[int64](a, 2)))
	case "live_rank":
		return methodJSON(l.liveRank(argument[string](a, 0), argument[int64](a, 1), argument[[]int64](a, 2), argument[float64](a, 3)))
	case "after_use":
		return useJSON(l.afterUse(argument[string](a, 0), argument[int64](a, 1), argument[int64](a, 2), argument[string](a, 3)))
	case "era_of":
		v, err := eraOf(argument[int64](a, 0))
		if err != nil {
			return nil, err
		}
		return methodJSON(v)
	case "norm_pos":
		return methodJSON(normPos(argument[*string](a, 0)))
	}
	return nil, fmt.Errorf("unknown method %s", name)
}
func situationJSON(s Situation) (json.RawMessage, error) {
	rank, err := methodJSON(s.Rank)
	if err != nil {
		return nil, err
	}
	basis, err := methodJSON(s.Basis)
	if err != nil {
		return nil, err
	}
	played, err := methodJSON(s.Played)
	if err != nil {
		return nil, err
	}
	record, err := methodJSON(s.Record)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal([]json.RawMessage{rank, basis, played, record})
	if err != nil {
		return nil, fmt.Errorf("situation tuple: %w", err)
	}
	return body, nil
}
func useJSON(s Use) (json.RawMessage, error) {
	starts, err := methodJSON(s.Starts)
	if err != nil {
		return nil, err
	}
	points, err := methodJSON(s.Points)
	if err != nil {
		return nil, err
	}
	weeks, err := methodJSON(s.Weeks)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal([]json.RawMessage{starts, points, weeks})
	if err != nil {
		return nil, fmt.Errorf("use tuple: %w", err)
	}
	return body, nil
}
