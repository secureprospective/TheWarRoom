package market

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

// Host supplies detached readings and serializes refreshes; no lock or transport enters the API.
type Host interface {
	Ready() error
	MarketContext() context.Context
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

// WireAssetBoard converts a board built by the service to its wire twin.
func WireAssetBoard(b Board) AssetBoard {
	values := make([]AssetValue, len(b.Values))
	for i, v := range b.Values {
		values[i] = AssetValue{ID: v.ID, Now: AssetCell(v.Now), Dynasty: AssetCell(v.Dynasty), Market: AssetCell(v.Market),
			NowValue: v.NowValue, DynastyValue: v.DynastyValue}
	}
	return AssetBoard{Values: wireSlice(b.Values, values), NowSource: b.NowSource, DynastySource: b.DynastySource, How: b.How}
}

// wireSlice keeps a nil slice nil, so it still encodes as null rather than [].
func wireSlice[S, T any](in []S, out []T) []T {
	if in == nil {
		return nil
	}
	return out
}

func Commands() []catalog.Command { return catalog.DesktopCommands("market", nil) }
