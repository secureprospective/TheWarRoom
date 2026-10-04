// Package contracts reads NFL contracts (OverTheCap's, as nflverse publishes them) into facts: for
// each player, the terms of the contract he signed in a season. The model reads the latest one
// signed by a season's end as that season's tenure, so a later signing never leaks into the past.
package contracts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"

	"github.com/parquet-go/parquet-go"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/measures"
)

// SourceURL is nflverse's contracts release. It is published only as Parquet; the CSV beside it
// stopped updating in 2022.
const SourceURL = "https://github.com/nflverse/nflverse-data/releases/download/contracts/historical_contracts.parquet"

// Source is the registry source the facts are filed under; IDType the id they carry.
const (
	Source = "nflverse"
	IDType = "gsis"
)

// maxBytes caps the file; it was 11 MB on 2026-10-04.
const maxBytes = 64 << 20

// The registry fields a contract fills.
const (
	FieldCapPct     = "contracts.apy_cap_pct"
	FieldYears      = "contracts.years"
	FieldValue      = "contracts.value"
	FieldGuaranteed = "contracts.guaranteed"
)

// row is the part of a contract the model reads; the file's other columns are not decoded.
type row struct {
	GSIS       *string  `parquet:"gsis_id,optional"`
	YearSigned *int32   `parquet:"year_signed,optional"`
	Years      *int32   `parquet:"years,optional"`
	Value      *float64 `parquet:"value,optional"`
	Guaranteed *float64 `parquet:"guaranteed,optional"`
	CapPct     *float64 `parquet:"apy_cap_pct,optional"`
}

// Fetch reads the contracts file at url, keeping the players keep accepts: most contracts are
// older players' who never reach the directory, and would wait in the store forever.
func Fetch(ctx context.Context, client *http.Client, url string, keep func(gsis string) bool) (measures.Batch, error) {
	body, err := ingestion.Get(ctx, client, url, maxBytes)
	if err != nil {
		return measures.Batch{}, fmt.Errorf("contracts: %w", err)
	}
	return Map(body, keep)
}

// Map reads a contracts file as nflverse sent it, such as an archived copy. A player who signed
// twice in one season keeps the larger contract; a contract without a gsis id, a signing season
// (OTC writes 0 for an unknown one) or a length is left out.
func Map(body []byte, keep func(gsis string) bool) (measures.Batch, error) {
	rows, err := parquet.Read[row](bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return measures.Batch{}, fmt.Errorf("contracts: read parquet: %w", err)
	}
	type key struct {
		id     string
		season int
	}
	kept := map[key]row{}
	var order []key
	for _, r := range rows {
		if r.GSIS == nil || !keep(*r.GSIS) || r.YearSigned == nil || *r.YearSigned <= 0 || r.Years == nil {
			continue
		}
		k := key{*r.GSIS, int(*r.YearSigned)}
		prev, seen := kept[k]
		if !seen {
			order = append(order, k)
		}
		if !seen || value(r) > value(prev) {
			kept[k] = r
		}
	}
	var facts []measures.Fact
	for _, k := range order {
		r := kept[k]
		add := func(field string, v float64) {
			facts = append(facts, measures.Fact{IDType: IDType, ID: k.id, Season: k.season, Field: field,
				Raw: strconv.FormatFloat(v, 'g', -1, 64)})
		}
		add(FieldYears, float64(*r.Years))
		for _, f := range []struct {
			field string
			v     *float64
		}{{FieldCapPct, r.CapPct}, {FieldValue, r.Value}, {FieldGuaranteed, r.Guaranteed}} {
			if f.v != nil {
				add(f.field, *f.v)
			}
		}
	}
	sum := sha256.Sum256(body)
	return measures.Batch{Source: Source, BodySHA256: hex.EncodeToString(sum[:]), Facts: facts}, nil
}

func value(r row) float64 {
	if r.Value == nil {
		return 0
	}
	return *r.Value
}
