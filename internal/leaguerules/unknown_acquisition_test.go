package leaguerules

import (
	"encoding/json"
	"testing"
)

func TestUnknownAcquisitionHasNoDefaultSetting(t *testing.T) {
	t.Parallel()
	const path FieldPath = "contracts.unknown_acquisition"
	if _, ok := Field(path); ok {
		t.Fatal("unknown contracts must be filled in, not defaulted by a rule")
	}
	for _, body := range []string{`0`, `null`, `{"years":1,"salaryRule":"fixed","salary":0}`} {
		t.Run(body, func(t *testing.T) {
			if _, err := DecodeSetting(path, json.RawMessage(body)); err == nil {
				t.Fatal("removed default accepted as an ordinary setting")
			}
		})
	}
}
