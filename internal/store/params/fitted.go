package params

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

// FittedFile is the fit tool's output (cmd/fit): every fitted value, with when and on which
// seasons it was fitted.
type FittedFile struct {
	Fitted  string        `json:"fitted"`
	First   int           `json:"first"`
	Holdout int           `json:"holdout"`
	Params  []FittedValue `json:"params"`
}

// FittedValue is one fitted parameter at one position.
type FittedValue struct {
	Key      string  `json:"key"`
	Position string  `json:"position"`
	Value    float64 `json:"value"`
}

//go:embed fitted.json
var fittedJSON []byte

// fittedParams are the shipped fitted values as calibrated defaults. The range is wide by kind:
// it catches a typo in the admin console, not a judgement of the value.
func fittedParams() []ParamDef {
	var f FittedFile
	if err := json.Unmarshal(fittedJSON, &f); err != nil {
		panic(fmt.Sprintf("params: fitted.json does not decode: %v", err)) // a build artifact, checked by test
	}
	out := make([]ParamDef, len(f.Params))
	for i, v := range f.Params {
		lo, hi, meaning := fittedKind(v.Key)
		out[i] = ParamDef{Key: v.Key, Position: v.Position, Type: TypeFloat, Default: v.Value, Min: lo, Max: hi,
			IsCalibrated: true, Description: fmt.Sprintf("%s (fitted %s on %d–%d)", meaning, f.Fitted, f.First, f.Holdout)}
	}
	return out
}

func fittedKind(key string) (lo, hi float64, meaning string) {
	switch {
	case key == "model.k_now":
		return 0.01, 10000, "Games of this season's evidence at which production and the prior weigh equally"
	case key == "model.k_dynasty":
		return 0.01, 10000, "Games of evidence at which production and the prior weigh equally for dynasty"
	case key == "model.z_exponential":
		return 0, 1, "1: production's weight is 1 − exp(−games/k); 0: games/(games + k)"
	case strings.HasPrefix(key, "model.recency."):
		return 0, 1, "Weight of an older season against the latest"
	case strings.HasPrefix(key, "model.prior."):
		return -1000, 1000, "Prior term: " + strings.TrimPrefix(key, "model.prior.")
	case strings.HasPrefix(key, "model.arc."):
		return -10, 10, "Talent arc term: " + strings.TrimPrefix(key, "model.arc.")
	case strings.HasPrefix(key, "model.survival."):
		return -100, 100, "Survival log-odds term: " + strings.TrimPrefix(key, "model.survival.")
	}
	return -1e6, 1e6, key
}
