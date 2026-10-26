package composition

import (
	"fmt"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
)

// CommandRegistry binds the real ports without starting workers, opening stores or reading secrets.
// Generators inspect this same executable registry used by desktop startup.
func CommandRegistry() (*catalog.Registry, error) {
	app := NewApp()
	registry, err := catalog.NewRegistry(newCommandAPIs(app).registryInputs(app))
	if err != nil {
		return nil, fmt.Errorf("bind command registry for generation: %w", err)
	}
	return registry, nil
}
