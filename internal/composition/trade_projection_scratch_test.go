package composition

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	tradesapp "github.com/secureprospective/TheWarRoom/internal/app/trades"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// whatIfRows is every table of whatif.db, each row rendered in rowid order.
func whatIfRows(t *testing.T, a *App) map[string][]string {
	t.Helper()
	ctx := a.core.Ctx
	read := a.core.WhatifPools.Read()
	names, err := read.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for names.Next() {
		var name string
		if err := names.Scan(&name); err != nil {
			t.Fatal(err)
		}
		tables = append(tables, name)
	}
	if err := names.Close(); err != nil {
		t.Fatal(err)
	}
	out := map[string][]string{}
	for _, table := range tables {
		rows, err := read.QueryContext(ctx, `SELECT * FROM "`+table+`"`) //nolint:gosec // table names come from sqlite_master
		if err != nil {
			t.Fatal(err)
		}
		cols, err := rows.Columns()
		if err != nil {
			t.Fatal(err)
		}
		out[table] = []string{}
		for rows.Next() {
			cells := make([]any, len(cols))
			for i := range cells {
				cells[i] = new(any)
			}
			if err := rows.Scan(cells...); err != nil {
				t.Fatal(err)
			}
			parts := make([]string, len(cells))
			for i, cell := range cells {
				parts[i] = fmt.Sprint(*cell.(*any))
			}
			out[table] = append(out[table], strings.Join(parts, "|"))
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func unchangedWhatIf(t *testing.T, label string, before, after map[string][]string) {
	t.Helper()
	for table, rows := range after {
		if len(before[table]) != len(rows) {
			t.Fatalf("%s: %s went from %d to %d rows", label, table, len(before[table]), len(rows))
		}
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("%s: whatif.db changed", label)
	}
}

// oldTradeProjection is the projection wiring before scratch pools: the owner's what-if opened
// (and on first use published) in whatif.db, and both scratch ledgers seeded there too.
func oldTradeProjection(a *App) func(context.Context, tradesapp.RawTradeBuilder, snapshot.Snapshot) (tradesapp.TradeBuilderPlan, error) {
	simulate := tradesapp.SimulationWorkflow{Scratch: db.OpenScratch,
		Open: func(ctx context.Context, _ *db.Pools, fid string) (state.WhatIfSession, *state.Store, error) {
			return contractsapp.OpenWhatIf(ctx, contractsapp.StoresOf(a.core), contractsapp.RawWhatIfRequest{Mine: fid, Action: "read"})
		},
		Mirror: func(ctx context.Context, fid string) (state.WhatIfSession, error) {
			return contractsapp.MirrorWhatIf(ctx, contractsapp.StoresOf(a.core), fid)
		},
		Load: func(ctx context.Context, _ *db.Pools, s state.WhatIfSession) (*state.Store, error) {
			return contractsapp.LoadWhatIf(ctx, a.core.WhatifPools, s)
		}, Now: time.Now}
	return (tradesapp.ProjectionWorkflow{Simulate: simulate.Run, Stamps: a.core.League.Stamps, Possessive: possessive,
		ClockReasons: a.tradeClockReasons, PickReason: a.tradePickReason, Teams: a.tradeBuilderTeams, Figure: tradeFigure}).Run
}

func TestTradeBuilderReadLeavesWhatIfFileUnchanged(t *testing.T) {
	t.Parallel()
	a, _ := realTradeBuilderApp(t)
	for _, give := range []string{"11675", "13448", "17034"} {
		deal := realDeal()
		deal.Action, deal.Give = "save", []string{give}
		builderCall(t, a, deal)
	}
	read := tradesapp.RawTradeBuilder{Mine: "0001", Action: "read"}
	before := whatIfRows(t, a)
	if len(before["trade_plans"]) != 3 {
		t.Fatalf("saved plans: %d", len(before["trade_plans"]))
	}
	if plans := builderCall(t, a, read).Plans; len(plans) != 3 {
		t.Fatalf("read plans: %d", len(plans))
	}
	unchangedWhatIf(t, "read before any what-if", before, whatIfRows(t, a))

	// With a saved what-if (one cut), a read replays it in scratch and still writes nothing.
	cut := a.contracts().TargetWhatIf(contractsapp.RawWhatIfRequest{Mine: "0001", Action: "cut", Player: "16190"})
	if cut.Status != report.ReadingOK || cut.Value.Notice != "" {
		t.Fatal(cut)
	}
	before = whatIfRows(t, a)
	got := builderCall(t, a, read).Plans
	unchangedWhatIf(t, "read with a saved what-if", before, whatIfRows(t, a))

	plans := a.tradePlans()
	plans.Project = oldTradeProjection(a)
	snap, _, err := a.tradeState("0001")
	if err != nil {
		t.Fatal(err)
	}
	want, err := plans.Saved(a.core.Ctx, "0001", snap)
	if err != nil {
		t.Fatal(err)
	}
	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(want)
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("scratch projection differs from the whatif.db path:\n got %s\nwant %s", gotJSON, wantJSON)
	}
	if len(whatIfRows(t, a)["rosters"]) == len(before["rosters"]) {
		t.Fatal("old path wrote no scratch rows; the equality above compared nothing different")
	}
}
