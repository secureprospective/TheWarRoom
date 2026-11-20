package leagueservice

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

type newsTestHost struct {
	admissionHost
	assets commissioner.NativeAssets
}

func (h newsTestHost) NativeAssets(context.Context, string) (commissioner.NativeAssets, error) {
	return h.assets, nil
}

type newsSourceFunc func(context.Context, []NFLNewsPlayer) (NFLNewsReading, error)

func (f newsSourceFunc) ReadNFLNews(ctx context.Context, players []NFLNewsPlayer) (NFLNewsReading, error) {
	return f(ctx, players)
}

func TestTargetNFLNewsReadsOnlyRosteredPlayersAndRefusesOtherLeagues(t *testing.T) {
	asOf := time.Date(2026, 11, 20, 14, 0, 0, 0, time.UTC)
	assets := commissioner.NativeAssets{Identities: []commissioner.NativeAssetIdentity{
		{Player: "0099", Franchise: "F1", Name: "Rostered One", Position: "WR"},
		{Player: "0100", Franchise: "", Name: "Free Agent", Position: "RB"},
		{Player: "0101", Franchise: "F2", Name: "Rostered Two", Position: "TE"},
	}}
	var asked []string
	var admitted commissioner.Access
	source := newsSourceFunc(func(_ context.Context, players []NFLNewsPlayer) (NFLNewsReading, error) {
		for _, p := range players {
			asked = append(asked, p.Player)
		}
		return NFLNewsReading{AsOf: asOf, Items: []NFLNewsItem{{At: asOf, Player: "0099", Tag: "INJ", Text: "Questionable (Ankle)", Source: "sleeper"}}}, nil
	})
	service := NewHosted(newsTestHost{admissionHost: admissionHost{}, assets: assets}, admissionFunc(func(_ context.Context, access commissioner.Access) error {
		admitted = access
		if access.League != "native.test" {
			return errors.New("not a member of this league")
		}
		return nil
	})).WithNFLNews(source)

	if _, err := service.TargetNFLNews("native.other", ""); err == nil {
		t.Fatal("news read for a league the caller is not in succeeded")
	}
	if len(asked) != 0 {
		t.Fatalf("refused read still asked the feed for %v", asked)
	}

	news, err := service.TargetNFLNews("native.test", "")
	if err != nil {
		t.Fatal(err)
	}
	if admitted.Operation != "nfl.news.read" || admitted.Mode != "read" {
		t.Fatalf("admitted = %+v", admitted)
	}
	if !slices.Equal(asked, []string{"0099", "0101"}) {
		t.Fatalf("feed asked about %v, want the rostered players only", asked)
	}
	if news.AsOf == nil || !news.AsOf.Equal(asOf) || news.Stale || len(news.Items) != 1 || news.Items[0].Tag != "INJ" {
		t.Fatalf("news = %+v", news)
	}

	asked = nil
	if _, err := service.TargetNFLNews("native.test", "0101"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(asked, []string{"0101"}) {
		t.Fatalf("narrowed read asked about %v", asked)
	}
}

func TestTargetNFLNewsWithoutASourceSaysStale(t *testing.T) {
	service := NewHosted(newsTestHost{admissionHost: admissionHost{}}, admissionFunc(func(context.Context, commissioner.Access) error { return nil }))
	news, err := service.TargetNFLNews("native.test", "")
	if err != nil {
		t.Fatal(err)
	}
	if !news.Stale || news.AsOf != nil || news.Items == nil || len(news.Items) != 0 {
		t.Fatalf("no-source news = %+v", news)
	}
}
