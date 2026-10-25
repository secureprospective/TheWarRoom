package serverversion

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// A schema, command or protocol change without a new release row would let an older server
// open a file, or serve a client, it does not understand.
func TestCurrentContract(t *testing.T) {
	current := Current()
	if current.Schema != history.SchemaVersion() || current.Commands != commissioner.DurableCommandVersion {
		t.Fatalf("add a release row: contract %+v, history schema %d, commands %d", current, history.SchemaVersion(), commissioner.DurableCommandVersion)
	}
	for i, row := range releases() {
		if row.Release != i+1 || row.SecurityFloor > row.ProtocolMinor || row.Commands < OldestCommands {
			t.Fatalf("release row %d is inconsistent: %+v", i, row)
		}
		if i == 0 {
			continue
		}
		before := releases()[i-1]
		if row.Schema < before.Schema || row.ProtocolMinor < before.ProtocolMinor || row.SecurityFloor < before.SecurityFloor || row.Commands < before.Commands {
			t.Fatalf("release %d goes backwards from %+v", row.Release, before)
		}
		before.Release = row.Release
		if row == before {
			t.Fatalf("release %d changes nothing; reuse release %d", row.Release, row.Release-1)
		}
	}
}

func TestAdmitProtocolAndCommands(t *testing.T) {
	contract := Contract{Release: 1, Schema: 8, ProtocolMinor: 4, SecurityFloor: 0, Commands: 1}
	for _, test := range []struct {
		protocol, commands string
		want               Side
		message            string
	}{
		{protocol: "1.4", commands: "1"},
		{protocol: "1.3"}, // the previous minor
		{protocol: "1.2", want: App, message: appTooOld},
		{protocol: "", want: App, message: appTooOld}, // headerless clients are protocol 1.0
		{protocol: "0.9", want: App, message: appTooOld},
		{protocol: "1.5", want: Server, message: serverTooOld},
		{protocol: "2.0", want: Server, message: serverTooOld},
		{protocol: "1.4", commands: "2", want: Server, message: serverTooOld},
		{protocol: "1.4", commands: "0", want: App, message: appTooOld},
	} {
		err := admit(contract, test.protocol, test.commands)
		var update *UpdateRequired
		if test.want == "" {
			if err != nil {
				t.Errorf("%q/%q refused: %v", test.protocol, test.commands, err)
			}
			continue
		}
		if !errors.As(err, &update) || update.Update != test.want || update.Message != test.message {
			t.Errorf("%q/%q = %v, want %s update", test.protocol, test.commands, err, test.want)
		}
	}
	// A security fix raises the floor and cuts off the previous minor too.
	contract.SecurityFloor = 4
	var update *UpdateRequired
	if err := admit(contract, "1.3", "1"); !errors.As(err, &update) || update.Update != App || update.Message != appUnsafe {
		t.Errorf("previous minor below the security floor = %v", err)
	}
	for _, bad := range [][2]string{{"1", ""}, {"1.", ""}, {".1", ""}, {"1.4.0", ""}, {"+1.4", ""}, {"1.4", "-1"}, {"1.4", "x"}, {"1.44444", ""}} {
		if err := Admit(bad[0], bad[1]); !errors.Is(err, ErrMalformed) {
			t.Errorf("%q/%q = %v, want malformed", bad[0], bad[1], err)
		}
	}
}

func TestPreflightJudgesWithoutWriting(t *testing.T) {
	ctx := context.Background()
	current := Current()
	for _, test := range []struct {
		name   string
		change func(*testing.T, *db.Pools)
		state  State
		err    error
	}{
		{name: "unstamped at this schema", state: Ready},
		{name: "stamped by this release", change: stamp(current.Release), state: Ready},
		{name: "schema from before this release", change: setSchema(current.Schema - 1), state: Older},
		{name: "schema from a newer release", change: setSchema(current.Schema + 1), err: ErrNewerFile},
		{name: "binary-only downgrade", change: stamp(current.Release + 1), err: ErrNewerFile},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := leagueFile(ctx, t, test.change)
			before, beforeInfo := snapshot(t, path)
			plan, err := Preflight(ctx, path)
			if !errors.Is(err, test.err) || (test.err == nil && plan.State != test.state) {
				t.Fatalf("preflight = %+v, %v", plan, err)
			}
			after, afterInfo := snapshot(t, path)
			if !bytes.Equal(before, after) || !beforeInfo.ModTime().Equal(afterInfo.ModTime()) {
				t.Fatal("preflight changed the league file")
			}
			for _, side := range []string{"-wal", "-shm", "-journal"} {
				if _, err := os.Lstat(path + side); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("preflight left %s beside the league file", side)
				}
			}
		})
	}
	empty := filepath.Join(t.TempDir(), "server.db")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if plan, err := Preflight(ctx, empty); err != nil || plan.State != Fresh {
		t.Fatalf("new installation = %+v, %v", plan, err)
	}
	if err := os.WriteFile(empty, []byte("not a database at all, but long enough to have a header-sized prefix in it......................"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Preflight(ctx, empty); !errors.Is(err, ErrNotLeague) {
		t.Fatalf("foreign file = %v", err)
	}
}

// leagueFile is a closed server database at this build's schema, after change.
func leagueFile(ctx context.Context, t *testing.T, change func(*testing.T, *db.Pools)) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "server.db")
	pools, err := db.Open(ctx, path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	if err := history.New(pools, registry).Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	if change != nil {
		change(t, pools)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func stamp(release int) func(*testing.T, *db.Pools) {
	return func(t *testing.T, pools *db.Pools) {
		t.Helper()
		if err := pools.StampServerRelease(context.Background(), release, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
}

func setSchema(version int) func(*testing.T, *db.Pools) {
	return func(t *testing.T, pools *db.Pools) {
		t.Helper()
		if _, err := pools.Write().ExecContext(context.Background(), "PRAGMA user_version = "+strconv.Itoa(version)); err != nil {
			t.Fatal(err)
		}
	}
}

func snapshot(t *testing.T, path string) ([]byte, os.FileInfo) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return body, info
}
