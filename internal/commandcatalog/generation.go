package catalog

import (
	"reflect"

	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

// GenerationRegistry builds the catalog registry for code generators without desktop wiring.
func GenerationRegistry() (*Registry, error) {
	return NewRegistry(RegistryInputs{
		Seed:    Shape(reflect.TypeFor[seedRequestShape](), "RawNativeSeedRequest.Validate/ValidatePreview"),
		MFL:     generationMFLShapes(),
		Service: new(leagueservice.Service),
	})
}

func generationMFLShapes() MFLShapes {
	return MFLShapes{
		Bid:   Shape(reflect.TypeFor[bidInputShape](), "RawBidInput.Validate"),
		Plan:  Shape(reflect.TypeFor[bidPlanScopeShape](), "RawBidPlanScope.Validate"),
		Pool:  Shape(reflect.TypeFor[poolScopeShape](), "RawPoolScope.Validate"),
		Owner: Shape(reflect.TypeFor[bidOwnerShape](), "RawBidOwner.Validate"),
	}
}

type seedRequestShape struct {
	League   string `json:"league"`
	Expected int64  `json:"expected"`
	Request  string `json:"request"`
	Reason   string `json:"reason"`
}

type bidInputShape struct {
	ConditionalDrop string `json:"conditionalDrop"`
	FranchiseID     string `json:"franchiseId"`
	Player          string `json:"player"`
	Salary          string `json:"salary"`
	Years           int    `json:"years"`
	Snipe           bool   `json:"snipe"`
	Beat            string `json:"beat"`
	Evidence        string `json:"evidence"`
	Closes          string `json:"closes"`
}

type poolScopeShape struct {
	ShowNoTeam  bool   `json:"showNoTeam"`
	FranchiseID string `json:"franchiseId"`
	Position    string `json:"position"`
	Sort        string `json:"sort"`
	Desc        bool   `json:"desc"`
}

type bidOwnerShape struct {
	FranchiseID string `json:"franchiseId"`
}

type bidPlanScopeShape struct {
	ID string `json:"id"`
}
