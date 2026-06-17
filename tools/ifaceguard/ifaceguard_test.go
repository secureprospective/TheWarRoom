package ifaceguard_test

import (
	"testing"

	"ifaceguard"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestIfaceguard(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), ifaceguard.Analyzer, "a")
}
