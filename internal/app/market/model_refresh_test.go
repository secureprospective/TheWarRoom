package market

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/labcompile"
)

func TestFitAssetModelVerifiedOnlySuccessPublishes(t *testing.T) {
	at := time.Now()
	for _, test := range []struct {
		sha    string
		failed bool
		want   []string
	}{{"", false, nil}, {"verified", true, []string{"build", "log"}}, {"verified", false, []string{"build", "publish", "log", "changed"}}} {
		calls := []string(nil)
		FitAssetModel(context.Background(), at, report.ArchiveSnapshot{SHA: test.sha, Through: at},
			func(_ context.Context, _ facts.Facts, evidence labcompile.Evidence) (*labcompile.Model, []byte, error) {
				calls = append(calls, "build")
				if !evidence.Built.Equal(at) || !evidence.ArchiveThrough.Equal(at) {
					t.Fatal(evidence)
				}
				if test.failed {
					return nil, nil, errors.New("compile failed")
				}
				return &labcompile.Model{}, nil, nil
			}, func(Model) { calls = append(calls, "publish") }, func(context.Context) { calls = append(calls, "changed") }, func(string) { calls = append(calls, "log") })
		if !reflect.DeepEqual(calls, test.want) {
			t.Fatal(test, calls)
		}
	}
}
