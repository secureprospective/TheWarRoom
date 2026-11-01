package appvalue

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/playerrecord"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

type countedFacts struct {
	Facts
	reads *int
}

func (f countedFacts) Book(ctx context.Context, q playerrecord.Query) (playerrecord.Book, error) {
	*f.reads++
	return f.Facts.Book(ctx, q)
}

func checkSourceCache(t *testing.T, src Source, league string, want Reading) {
	t.Helper()
	at := src.Now().Add(time.Minute)
	now := at
	revision := int64(1)
	reads := 0
	src.Cache = &Cache{}
	src.Facts = countedFacts{src.Facts, &reads}
	src.Now = func() time.Time { return now }
	src.Revision = func(context.Context) (int64, error) { return revision, nil }
	src.NextFeatureAt = func(context.Context, time.Time) (time.Time, error) { return at.Add(time.Hour), nil }
	read := func(franchise string) Reading {
		t.Helper()
		out, err := src.Read(t.Context(), league, franchise, season)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	cold := read("")
	if !reflect.DeepEqual(cold, want) {
		t.Fatal("cached assembly changed values")
	}
	cold.Players[0].DynastyPP = -999
	cold.Notes = append(cold.Notes, "caller-owned")
	warm := read("")
	if !reflect.DeepEqual(warm, want) || reads != 1 {
		t.Fatalf("cache miss or shared mutable result: reads=%d", reads)
	}
	franchise := ""
	for _, p := range want.Players {
		if p.Franchise != "" && p.Available {
			franchise = p.Franchise
			break
		}
	}
	if franchise == "" {
		t.Fatal("no valued roster player in cache fixture")
	}
	scoped := read(franchise)
	if len(scoped.Players) == 0 || !scoped.Best.Available || scoped.Franchise != franchise || reads != 1 {
		t.Fatal("franchise read did not share the pass")
	}
	for _, p := range scoped.Players {
		if p.Franchise != franchise {
			t.Fatal("owner leaked into scoped reading")
		}
	}
	revision++
	read("")
	if reads != 2 {
		t.Fatal("database commit did not invalidate")
	}
	values := src.Params().Values()
	values[params.KeyDynastyDiscount] = 0.5
	src.Params = func() params.Set { return params.SetOf(values) }
	changed := read("")
	if reads != 3 || reflect.DeepEqual(changed, want) {
		t.Fatal("parameter change did not invalidate and change the values")
	}
	now = at.Add(time.Hour)
	read("")
	if reads != 4 {
		t.Fatal("future observation did not invalidate")
	}
	now = at.Add(-time.Second)
	read("")
	if reads != 5 {
		t.Fatal("clock rewind reused future knowledge")
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := src.Read(canceled, league, "", season); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	failure := errors.New("revision unavailable")
	src.Revision = func(context.Context) (int64, error) { return 0, failure }
	if _, err := src.Read(t.Context(), league, "", season); !errors.Is(err, failure) {
		t.Fatalf("revision failure hidden: %v", err)
	}
}
