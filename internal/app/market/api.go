package market

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// Host supplies detached readings and serializes refreshes; no lock or transport enters the API.
type Host interface {
	Ready() error
	MarketContext() context.Context
	// LabFacts is the shared Lab decode; LabStamp is this build's stamp on every Lab reading.
	LabFacts() (*lab.Data, error)
	LabStamp() string
	// AssetInput hydrates and reads Research's finished-week scores, and lists every rostered id.
	AssetInput(context.Context) (OnField, []string)
	MarketWeek() int
	// CaptureMarket runs snapshot under the shared lock the root writers publish under.
	CaptureMarket(snapshot func() Feeds) Feeds
	MarketLookup() normalize.Lookup
	MarketZone() *time.Location
	RefreshMarket(bool) error
}

type API struct {
	host Host
	held *Held
}

func New(host Host, held *Held) *API { return &API{host: host, held: held} }

// assetTimeout is the bound on a held asset-board read, the M4 read timeout.
const assetTimeout = 120 * time.Second

// TargetAssetBoard reads C-01 from held state: the weekly scores (from their archived copies when
// no refresh has run this launch) and, once compiled, the Lab's value model. Every rostered player
// has a row, scored or not. It never fetches.
func (a *API) TargetAssetBoard() (AssetBoard, error) {
	if err := a.host.Ready(); err != nil {
		return AssetBoard{}, fmt.Errorf("asset board: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.MarketContext(), assetTimeout)
	defer cancel()
	now, rostered := a.host.AssetInput(ctx)
	return WireAssetBoard(BuildBoard(now, a.held.SnapshotModel(), time.Now(), rostered)), nil
}

func (a *API) TargetMarket(raw RawMarketScope) report.Reading[lab.Market] {
	if err := raw.Validate(); err != nil {
		return FailedMarket(err)
	}
	d, err := a.host.LabFacts()
	if err != nil {
		return FailedMarket(err)
	}
	return a.lab(d).Market(lab.Scope{From: raw.From, To: raw.To, Phases: slices.Clone(raw.Phases), ByAge: raw.ByAge})
}

func (a *API) TargetMarketLists(raw RawMarketScope) report.Reading[MarketSurface] {
	l, err := a.input(raw)
	if err != nil {
		return failedSurface(err)
	}
	return WireReading(l.Lists(Scope(raw)), WireSurface)
}

func (a *API) TargetPickClock(raw RawMarketScope) report.Reading[MarketSurface] {
	l, err := a.input(raw)
	if err != nil {
		return failedSurface(err)
	}
	return WireReading(l.PickClock(Scope(raw)), WireSurface)
}

func (a *API) TargetMarketStretches(raw RawMarketScope) report.Reading[MarketSurface] {
	l, err := a.input(raw)
	if err != nil {
		return failedSurface(err)
	}
	return WireReading(l.Stretches(Scope(raw)), WireSurface)
}

func (a *API) TargetAssetProfile(raw RawMarketAsset) report.Reading[MarketSurface] {
	if err := raw.Validate(); err != nil {
		return failedSurface(err)
	}
	d, err := a.host.LabFacts()
	if err != nil {
		return failedSurface(err)
	}
	return WireReading(a.lab(d).Profile(Scope(raw.Scope), raw.Kind), WireSurface)
}

func (a *API) TargetMarketReportsRefresh(force bool) error {
	return a.host.RefreshMarket(force) //nolint:wrapcheck // The host preserves the existing refresh error.
}

func (a *API) TargetMarketReports() (report.Board, error) {
	if err := a.host.Ready(); err != nil {
		return report.Board{}, fmt.Errorf("market reports: startup: %w", err)
	}
	week := a.host.MarketWeek()
	held := HeldFeeds(ForWeek(a.host.CaptureMarket(a.held.SnapshotFeeds), week))
	lookup := a.host.MarketLookup()
	return Reports(held, Names(held, lookup), week, a.host.MarketZone()), nil
}

// input checks the scope, then reads the Lab the surface is built on.
func (a *API) input(raw RawMarketScope) (Lab, error) {
	if err := raw.Validate(); err != nil {
		return Lab{}, fmt.Errorf("read Market: %w", err)
	}
	d, err := a.host.LabFacts()
	return a.lab(d), err
}

// lab is the Lab decode with this build's stamp, as the Market service reads it.
func (a *API) lab(d *lab.Data) Lab { return Lab{Data: d, Stamp: a.host.LabStamp()} }

func failedSurface(err error) report.Reading[MarketSurface] {
	return WireReading(FailedSurface(err), WireSurface)
}

type RawMarketScope struct {
	From   int      `json:"from"`
	To     int      `json:"to"`
	Phases []string `json:"phases"`
	Assets []string `json:"assets"`
	ByAge  bool     `json:"byAge"`
}

// Validate checks a scope that came over IPC, in the service.
func (r RawMarketScope) Validate() error {
	return Scope(r).Validate()
}

type RawMarketAsset struct {
	Scope RawMarketScope `json:"scope"`
	Kind  string         `json:"kind"`
}

// Validate checks an asset profile request that came over IPC, in the service.
func (r RawMarketAsset) Validate() error {
	return ValidateAsset(Scope(r.Scope), r.Kind)
}

// AssetBoard is C-01: what each player is worth on the field now and as a dynasty asset. Go words
// every cell; a value the app cannot support says why in place of a number.
type AssetBoard struct {
	Values        []AssetValue `json:"values"`
	NowSource     string       `json:"nowSource"`
	DynastySource string       `json:"dynastySource"`
	How           string       `json:"how"`
}

// AssetValue is one player's three cells: on-field-now, dynasty value and market. NowValue and
// DynastyValue order the board (points per game, discounted wins); zero when not held.
type AssetValue struct {
	ID           string    `json:"id"`
	Now          AssetCell `json:"now"`
	Dynasty      AssetCell `json:"dynasty"`
	Market       AssetCell `json:"market"`
	NowValue     float64   `json:"nowValue"`
	DynastyValue float64   `json:"dynastyValue"`
}

// AssetCell is a worded value: a short headline, the qualifier beneath it and the detail on hover.
// Held is false when the words say why there is no value.
type AssetCell struct {
	Text   string `json:"text"`
	Sub    string `json:"sub"`
	Detail string `json:"detail"`
	Held   bool   `json:"held"`
}

// MarketLine, MarketBreakdown, MarketSection and MarketSurface are the wire twins of the
// service's surface types; WireSurface converts.
//
//nolint:revive // Preserve the existing exported wire name.
type MarketLine struct {
	Key    string        `json:"key"`
	Label  string        `json:"label"`
	Detail string        `json:"detail"`
	Count  int           `json:"count"`
	Range  *report.Range `json:"range"`
	Mark   string        `json:"mark"`
}

//nolint:revive // Preserve the existing exported wire name.
type MarketBreakdown struct {
	Title string       `json:"title"`
	Count int          `json:"count"`
	Rows  []MarketLine `json:"rows"`
}

//nolint:revive // Preserve the existing exported wire name.
type MarketSection struct {
	Breakdowns []MarketBreakdown `json:"breakdowns,omitempty"`
	ID         string            `json:"id"`
	Title      string            `json:"title"`
	Note       string            `json:"note"`
	Rows       []MarketLine      `json:"rows"`
}

//nolint:revive // Preserve the existing exported wire name.
type MarketSurface struct {
	Summary  string          `json:"summary"`
	Sections []MarketSection `json:"sections"`
}

// WireAssetBoard converts a board built by the service to its wire twin.
func WireAssetBoard(b Board) AssetBoard {
	values := make([]AssetValue, len(b.Values))
	for i, v := range b.Values {
		values[i] = AssetValue{ID: v.ID, Now: AssetCell(v.Now), Dynasty: AssetCell(v.Dynasty), Market: AssetCell(v.Market),
			NowValue: v.NowValue, DynastyValue: v.DynastyValue}
	}
	return AssetBoard{Values: wireSlice(b.Values, values), NowSource: b.NowSource, DynastySource: b.DynastySource, How: b.How}
}

// WireSurface converts a surface built by the service to its wire twin. The two type sets are
// kept identical field for field; TestMarketWireTwinsMatch proves it.
func WireSurface(s Surface) MarketSurface {
	sections := make([]MarketSection, len(s.Sections))
	for i, section := range s.Sections {
		sections[i] = MarketSection{ID: section.ID, Title: section.Title, Note: section.Note, Rows: wireLines(section.Rows)}
		if section.Breakdowns != nil {
			breakdowns := make([]MarketBreakdown, len(section.Breakdowns))
			for j, b := range section.Breakdowns {
				breakdowns[j] = MarketBreakdown{Title: b.Title, Count: b.Count, Rows: wireLines(b.Rows)}
			}
			sections[i].Breakdowns = breakdowns
		}
	}
	return MarketSurface{Summary: s.Summary, Sections: wireSlice(s.Sections, sections)}
}

func wireLines(lines []Line) []MarketLine {
	out := make([]MarketLine, len(lines))
	for i, l := range lines {
		out[i] = MarketLine(l)
	}
	return wireSlice(lines, out)
}

// WireReading converts a reading's value through value, keeping every disclosure field.
func WireReading[S, T any](r report.Reading[S], value func(S) T) report.Reading[T] {
	return report.Reading[T]{Sourced: snapshot.Sourced[T]{Value: value(r.Value), Provenance: r.Provenance},
		Status: r.Status, AbsenceReason: r.AbsenceReason, Count: r.Count, Threshold: r.Threshold,
		Baseline: r.Baseline, Cutoff: r.Cutoff, Method: r.Method, InputSHA: r.InputSHA, BuildStamp: r.BuildStamp,
		HowCounted: r.HowCounted}
}

// wireSlice keeps a nil slice nil, so it still encodes as null rather than [].
func wireSlice[S, T any](in []S, out []T) []T {
	if in == nil {
		return nil
	}
	return out
}

func Commands() []catalog.Command { return catalog.DesktopCommands("market", nil) }
