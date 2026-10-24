package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestInvalidInputWritesNoKey(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		args []string
		data string
	}{
		{"missing mode", nil, "secret"},
		{"unknown mode", []string{"other"}, "secret"},
		{"short key", []string{"seal"}, "secret"},
		{"empty binding", []string{"unseal"}, ""},
		{"oversize binding", []string{"unseal"}, strings.Repeat("x", 65537)},
		{"corrupt binding", []string{"unseal"}, "secret-not-a-clevis-message"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var output bytes.Buffer
			if err := run(test.args, strings.NewReader(test.data), &output); err == nil {
				t.Fatal("invalid input accepted")
			}
			if output.Len() != 0 {
				t.Fatal("failed unlock wrote key material")
			}
		})
	}
}
