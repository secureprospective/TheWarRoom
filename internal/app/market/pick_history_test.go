package market

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/pickvalue"
	"github.com/secureprospective/TheWarRoom/internal/playerrecord"
)

func TestPickSlotsContinueAfter2026(t *testing.T) {
	t.Parallel()
	for _, season := range []int{2026, 2027, 2028} {
		t.Run(fmt.Sprint(season), func(t *testing.T) {
			records := []playerrecord.Record{
				{MFL: "0001", Draft: playerrecord.Draft{Year: 2027, Status: playerrecord.Drafted, Overall: 5}},
				{MFL: "0002", Draft: playerrecord.Draft{Year: 2027, Status: playerrecord.Undrafted}},
				{MFL: "0003", Draft: playerrecord.Draft{Year: 2027, Status: playerrecord.DraftUnknown}},
				{MFL: "0004", Draft: playerrecord.Draft{Year: 2028, Status: playerrecord.Drafted, Overall: 10}},
			}
			url := func(year int, league string) string {
				return fmt.Sprintf("https://example.test/%d/export?TYPE=draftResults&L=%s", year, league)
			}
			body := func(player string) []byte {
				return []byte(`{"draftResults":{"draftUnit":{"unit":"LEAGUE","draftPick":[` +
					`{"round":"2","pick":"1","player":"` + player + `","franchise":"0001"},` +
					`{"round":"2","pick":"2","player":"0002","franchise":"0002"},` +
					`{"round":"2","pick":"3","player":"0003","franchise":"0003"},` +
					`{"round":"2","pick":"4","player":"","franchise":"0004"}]}}}`)
			}
			bodies := func(_ context.Context, _ string, take func(string, []byte, time.Time) bool) (bool, error) {
				take(url(2027, "other"), body("0001"), time.Now())
				take(url(2027, "14432"), []byte(`bad`), time.Now())
				take(url(2028, "14432"), body("0004"), time.Now())
				take(url(2027, "14432"), body("0001"), time.Now())
				take(url(2027, "14432"), body("0003"), time.Now()) // older correction is not counted again
				return false, nil
			}
			h, err := HeldPickHistory(t.Context(), bodies, "14432", season, records)
			if err != nil {
				t.Fatal(err)
			}
			want := map[int]int{2026: 0, 2027: 2, 2028: 3}[season]
			if len(h) != want {
				t.Fatalf("slots: %+v, want %d", h, want)
			}
			if season >= 2027 && (h[0].Selection != 5 || h[0].NFLOverall != 5 || h[1].NFLOverall != 0) {
				t.Fatalf("wrong slots or undrafted handling: %+v", h)
			}
			base, err := pickvalue.History()
			if err != nil {
				t.Fatal(err)
			}
			if len(base) != 1689 {
				t.Fatal("frozen approved sample changed")
			}
			n := NewNative(NativeInput{History: h, Records: records, Season: season, Discount: 0.75, Teams: 4})
			if season >= 2027 && (n.picks == nil || !n.Pick(season+1, 1, "", time.Now()).Valued) {
				t.Fatal("new slots did not reach Pick Value v1", n.pickWhy)
			}
		})
	}
}

func TestPickHistoryFailureDoesNotRemovePlayerValues(t *testing.T) {
	failure := errors.New("archive unavailable")
	_, err := HeldPickHistory(t.Context(), func(context.Context, string, func(string, []byte, time.Time) bool) (bool, error) {
		return false, failure
	}, "14432", 2027, nil)
	if !errors.Is(err, failure) {
		t.Fatal("lost archive failure", err)
	}
	n := NewNative(NativeInput{Values: map[string]float64{"0042": 6}, PickWhy: err.Error()})
	if !n.Player("0042", time.Now()).Valued || n.Pick(2027, 1, "", time.Now()).Valued {
		t.Fatal("archive failure removed player values or fabricated pick history")
	}
}

func TestDraftSlotSelectionUsesThatDraftsWidth(t *testing.T) {
	for _, teams := range []int{2, 4, 16, 32} {
		t.Run(fmt.Sprint(teams), func(t *testing.T) {
			body := []byte(fmt.Sprintf(`{"draftResults":{"draftUnit":{"draftPick":[
 {"round":"1","pick":"%d","player":"","franchise":"0002"},
 {"round":"2","pick":"1","player":"0001","franchise":"0001"}]}}}`, teams))
			records := []playerrecord.Record{{MFL: "0001", Draft: playerrecord.Draft{Year: 2027,
				Status: playerrecord.Drafted, Overall: 10}}}
			bodies := func(_ context.Context, _ string, take func(string, []byte, time.Time) bool) (bool, error) {
				take("https://example.test/2027/export?TYPE=draftResults&L=14432", body, time.Now())
				return false, nil
			}
			h, err := HeldPickHistory(t.Context(), bodies, "14432", 2028, records)
			if err != nil || len(h) != 1 || h[0].Selection != teams+1 {
				t.Fatalf("historical width replaced by today's league: %+v %v", h, err)
			}
		})
	}
}
