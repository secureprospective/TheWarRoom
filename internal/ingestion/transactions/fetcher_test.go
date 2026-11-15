package transactions

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func TestSyntheticTransactions(t *testing.T) {
	// Synthetic: unknown kinds must not hide malformed known rows.
	unknown := `{"type":"NEW_KIND","franchise":"0001","timestamp":"100"}`
	raw, err := Parse([]byte(`{"transactions":{"transaction":` + unknown + `}}`))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := ToTransactions(raw)
	want := leaguefeed.Transaction{
		Kind: "NEW_KIND", Franchise: "0001", Time: time.Unix(100, 0).UTC(), Unparsed: true,
	}
	if err != nil || len(rows) != 1 || !reflect.DeepEqual(rows[0], want) {
		t.Fatalf("unknown: %+v, %v", rows, err)
	}
	for _, kind := range []string{"FREE_AGENT", "LOAD_ROSTERS", "TRADE", "IR", "TAXI"} {
		bad := `{"type":"` + kind + `","franchise":"0001","timestamp":"100"}`
		body := `{"transactions":{"transaction":[` + unknown + `,` + bad + `]}}`
		if _, err := Parse([]byte(body)); err == nil || !strings.Contains(err.Error(), "index 1") {
			t.Fatalf("%s: %v", kind, err)
		}
	}
	for _, body := range []string{`{}`, `{`, `{"error":{"$t":"denied"}}`,
		`{"transactions":{"transaction":{"type":"IR","franchise":"0001","timestamp":"x"}}}`} {
		if _, err := Parse([]byte(body)); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	raw.Transactions[0].Kind = "IR"
	if _, err := ToTransactions(raw); err == nil {
		t.Fatal("conversion skipped validation")
	}
}
