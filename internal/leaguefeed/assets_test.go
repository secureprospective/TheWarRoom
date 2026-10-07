package leaguefeed

import (
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func TestParseAssets(t *testing.T) {
	id, err := playerid.New("16289")
	if err != nil {
		t.Fatal(err)
	}
	cents := int64(1050)
	cases := []struct {
		list string
		want []Asset
		bad  string
	}{
		{"16289", []Asset{{Player: &id}}, ""},
		{"DP_02_05,", []Asset{{CurrentPick: &CurrentPick{Round: 3, Pick: 6}}}, ""},
		{"DP_4_9", []Asset{{CurrentPick: &CurrentPick{Round: 5, Pick: 10}}}, ""},
		{"DP_0_0", []Asset{{CurrentPick: &CurrentPick{Round: 1, Pick: 1}}}, ""},
		{"FP_0019_2027_1,", []Asset{{FuturePick: &FuturePick{Franchise: "0019", Year: 2027, Round: 1}}}, ""},
		{"FP_0019_2027_01", []Asset{{FuturePick: &FuturePick{Franchise: "0019", Year: 2027, Round: 1}}}, ""},
		{"FP_19_2027_1", nil, "FP_19_2027_1"},
		{"DP_100_0", nil, "DP_100_0"},
		{"BB_10.50,", []Asset{{BlindBidCents: &cents}}, ""},
		{"16289,BB_10.5,", []Asset{{Player: &id}, {BlindBidCents: &cents}}, ""},
		{"", []Asset{}, ""},
		{"UNKNOWN", nil, "UNKNOWN"},
		{"DP_x_1", nil, "DP_x_1"},
		{"FP_0019_27_1", nil, "FP_0019_27_1"},
		{"FP_0019_2027_0", nil, "FP_0019_2027_0"},
		{"BB_10.501", nil, "BB_10.501"},
		{"BB_-1", nil, "BB_-1"},
		{"16289,,", nil, `asset ""`},
		{"DP_9223372036854775807_0", nil, "DP_9223372036854775807_0"},
	}
	for _, tc := range cases {
		t.Run(tc.list, func(t *testing.T) {
			got, err := ParseAssets(tc.list)
			if tc.bad != "" {
				if err == nil || !strings.Contains(err.Error(), tc.bad) {
					t.Fatalf("error = %v, want %q", err, tc.bad)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v, %v; want %#v", got, err, tc.want)
			}
		})
	}
}

func TestAssetTokenRoundTrip(t *testing.T) {
	assets, err := ParseAssets("99,DP_4_9,FP_0001_2027_1,BB_10.50")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"0099", "DP_04_09", "FP_0001_2027_1", "BB_10.50"}
	got := AssetTokens(assets)
	if !reflect.DeepEqual(got, want) {
		t.Fatal(got)
	}
	for i, token := range got {
		parsed, err := ParseAssets(token)
		if err != nil || !reflect.DeepEqual(parsed[0], assets[i]) {
			t.Fatal("round trip", token, err)
		}
	}
	if AssetToken(Asset{}) != "" || AssetTokens(nil) == nil {
		t.Fatal("empty asset contract")
	}
}
