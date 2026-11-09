package market

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestMissingContractMonetaryAdmission(t *testing.T) {
	for _, operation := range []string{"contract quote", "trade cap", "roster cap", "cut"} {
		t.Run(operation, func(t *testing.T) {
			base := quoteBase()
			p := &base.Assets.Snapshot.Players[0]
			p.ContractMissing, p.Salary, p.ExpirationYear, p.ContractStatus = true, 0, 0, ""
			original := *p
			var err error
			switch operation {
			case "contract quote":
				var q ContractQuote
				q, err = checkContract(contractCheck{base: base, player: *p, action: nativecheck.ActionBuyout})
				if q.Allowed || q.SalaryCent != nil || q.Cap != nil {
					t.Fatal("missing contract acquired a price")
				}
			case "trade cap":
				_, err = nativeFranchiseCap(base.Assets.Snapshot, base.Rules, p.Franchise, time.UTC)
			case "roster cap":
				err = rosterCapAllows(base.Assets.Snapshot, base.Rules, p.Franchise, nil, time.UTC)
			case "cut":
				_, err = waiverDeadMoney(*p, base.Rules)
			}
			if err == nil || !strings.Contains(err.Error(), "missing contract") {
				t.Fatalf("unrecorded salary was judged: %v", err)
			}
			if !reflect.DeepEqual(*p, original) {
				t.Fatal("refusal changed the contract")
			}
			if err := base.Assets.Snapshot.CheckContracts("franchise.02"); err != nil {
				t.Fatal("unrelated franchise blocked")
			}
		})
	}
}
