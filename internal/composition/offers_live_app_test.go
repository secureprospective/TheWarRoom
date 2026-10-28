package composition

import (
	"fmt"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/lab"
)

func TestOfferHistorySaveFailureLeavesDeskRead(t *testing.T) {
	t.Parallel()
	a, transport, _, _ := seasonTestApp(t)
	if err := a.core.KeyStore.Set(a.core.Ctx, "synthetic-stored-key"); err != nil {
		t.Fatal(err)
	}
	var logged []string
	a.core.FallbackLog = func(line string) { logged = append(logged, line) }
	a.core.LabData = func() (*lab.Data, error) { return nil, fmt.Errorf("lab unavailable") }
	transport.bodies["pendingTrades"] = `{"pendingTrades":{"pendingTrade":{
 "trade_id":"9001","offeringteam":"0025","offeredto":"0001","timestamp":"1791810000",
 "expires":"1791982800","will_give_up":"0042,","will_receive":"0816,"}}}`
	a.refreshPendingTrades(a.core.Ctx)
	if a.core.SeasonPendingTrades.RefreshError != "" || len(a.core.SeasonPendingTrades.Value) != 1 {
		t.Fatalf("a failed history save marked the desk's MFL read failed: %q %+v",
			a.core.SeasonPendingTrades.RefreshError, a.core.SeasonPendingTrades.Value)
	}
	if len(logged) != 1 || !strings.Contains(logged[0], "offers: archive owner") {
		t.Fatalf("history save failure not logged once: %q", logged)
	}
}
