// The War Room desktop app. The root holds only this file: it embeds the files that live at the repo
// root and hands them to internal/composition, which wires the features and runs the window.
package main

import (
	"embed"
	"os"

	"github.com/secureprospective/TheWarRoom/internal/composition"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed docs/league-rules/Official_Rulebook.md
var rulebook []byte

//go:embed docs/ui/endpoint-registry.csv
var endpointRegistry string

func main() {
	os.Exit(composition.Main(os.Args[1:], composition.Embedded{
		Assets: assets, Rulebook: rulebook, EndpointRegistry: endpointRegistry,
	}))
}
