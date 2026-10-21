package offers

import (
	"fmt"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/lab"
)

func TestLandingOfferHeadlines(t *testing.T) {
	t.Parallel()
	d := &lab.Data{Franchises: map[string]lab.RawFranchise{"a": {Name: "Team A"}}}
	for _, tc := range []struct {
		known []int
		want  string
	}{
		{[]int{10, 10}, "No stretch's acceptance rate stands out from the others beyond chance."},
		{[]int{10, 9}, "Too few stretches have enough decided offers to compare."},
		{[]int{0, 0}, "Too few stretches have enough decided offers to compare."},
	} {
		t.Run(fmt.Sprint(tc.known), func(t *testing.T) {
			v := lab.OfferResult{Owner: "a", Offers: []lab.Offer{{}}}
			for _, n := range tc.known {
				v.Stretches = append(v.Stretches, lab.OfferStretch{Known: n})
			}
			if got := Summary(d, v, lab.OfferScope{}); !strings.HasSuffix(got, tc.want) {
				t.Fatal(got)
			}
		})
	}
}
