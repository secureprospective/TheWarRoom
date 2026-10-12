package labcompile

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/facts"
)

type columns map[string][]json.RawMessage

func appendRow[T tradeRow | legRow | draftRow | teamRow | playerRow | deliveryRow](out columns, row T) error {
	body, err := json.Marshal(row)
	if err != nil {
		return fmt.Errorf("encode lab row: %w", err)
	}
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(body, &fields); err != nil {
		return fmt.Errorf("transpose lab row: %w", err)
	}
	for k, v := range fields {
		out[k] = append(out[k], v)
	}
	return nil
}

type document struct {
	Schema         string         `json:"schema"`
	Built          string         `json:"built"`
	ArchiveThrough string         `json:"archiveThrough"`
	Eras           []Era          `json:"eras"`
	Seasons        []Season       `json:"seasons"`
	Franchises     []Franchise    `json:"franchises"`
	Checks         Checks         `json:"checks"`
	Trades         columns        `json:"trades"`
	Legs           columns        `json:"legs"`
	Draft          columns        `json:"draft"`
	Delivered      columns        `json:"delivered"`
	TeamSeasons    columns        `json:"teamSeasons"`
	Players        columns        `json:"players"`
	Offers         OfferColumns   `json:"offers"`
	Market         Market         `json:"market"`
	Checks2        AnalysisChecks `json:"checks2"`
	Ledger         Ledger         `json:"ledger"`
	Source         Source         `json:"source"`
}
type Checks struct {
	DraftOrder        SlotCheck `json:"draftOrderEstimate"`
	CurrentPickCodes  PickProof `json:"currentPickCodes"`
	BirthdateCoverage float64   `json:"birthdateCoverage"`
}

// Compile deliberately remains unwired until the value analysis port is available.
func Compile(ctx context.Context, _ facts.Facts, _ Evidence) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("compile lab: %w", err)
	}
	return nil, fmt.Errorf("value analysis is not ported yet")
}
func compile(ctx context.Context, f facts.Facts, in Evidence, analyse Analyse) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("compile lab: %w", err)
	}
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if analyse == nil {
		return nil, fmt.Errorf("lab analysis function is missing")
	}
	l, err := newLab(f, in.ArchiveThrough.Unix())
	if err != nil {
		return nil, err
	}
	draft, spans := readDraft(f)
	trades, err := readTrades(f)
	if err != nil {
		return nil, err
	}
	l.buildCalendar(trades, spans)
	known, estimated := knownSlots(trades, draft), l.estimatedSlots()
	a, err := analyse(ctx, l, trades, draft, known, estimated)
	if err != nil {
		return nil, fmt.Errorf("analyse lab: %w", err)
	}
	if err := RawAnalysis(a).Validate(); err != nil {
		return nil, fmt.Errorf("lab analysis: %w", err)
	}
	if err := a.validateIndices(trades); err != nil {
		return nil, err
	}
	doc, err := l.assemble(ctx, f, in, trades, draft, known, estimated, a)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("encode lab document: %w", err)
	}
	return body, nil
}
func (l *Lab) assemble(ctx context.Context, f facts.Facts, in Evidence, trades []Trade, draft []Pick, known, estimated map[teamYear]int64, a Analysis) (document, error) {
	source, err := sourceChecks(f, in.Pages)
	if err != nil {
		return document{}, err
	}
	seasons, err := l.seasons(f)
	if err != nil {
		return document{}, err
	}
	franchises, err := l.franchises(f)
	if err != nil {
		return document{}, err
	}
	slotCheck, pickProof := slotChecks(known, estimated, trades, draft)
	doc := document{Schema: "league-lab-2", Built: in.Built.UTC().Format("2006-01-02T15:04:05+00:00"), ArchiveThrough: in.ArchiveThrough.UTC().Format("2006-01-02T15:04:05+00:00"), Eras: eras(), Seasons: seasons, Franchises: franchises, Checks: Checks{DraftOrder: slotCheck, CurrentPickCodes: pickProof}, Trades: columns{}, Legs: columns{}, Draft: columns{}, Delivered: columns{}, TeamSeasons: columns{}, Players: columns{}, Offers: l.offerOutcomes(f.Views().Offer), Market: a.Market, Checks2: a.Checks2, Ledger: l.reconcile(trades, readMoves(f)), Source: source}
	index := indexAnalysis(a)
	seen, tradeIndex, err := l.outputTrades(&doc, trades, draft, known, estimated, index)
	if err != nil {
		return document{}, err
	}
	if err := l.outputDraft(&doc, draft, index, seen); err != nil {
		return document{}, err
	}
	if err := l.outputDirectories(&doc, index, seen); err != nil {
		return document{}, err
	}
	for _, r := range a.Delivered {
		if err := appendRow(doc.Delivered, deliveryRow{Trade: tradeIndex[r.Trade], Class: r.Class, Expected: pythonRound(r.Expected, 4), Delivered: pythonRound(r.Delivered, 4), Seasons: r.Seasons}); err != nil {
			return document{}, err
		}
	}
	if err := ctx.Err(); err != nil {
		return document{}, fmt.Errorf("assemble lab: %w", err)
	}
	return doc, nil
}
func iso(d *time.Time) *string {
	if d == nil {
		return nil
	}
	return ptr(d.Format("2006-01-02"))
}
