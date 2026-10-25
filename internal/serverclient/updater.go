package serverclient

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/secureprospective/TheWarRoom/internal/releasetrust"
	"github.com/secureprospective/TheWarRoom/internal/serverversion"
)

// ConfigureUpdater uses only installer-provisioned public trust in the private update folder.
// The composition root holds the folder's instance lock until the window closes.
func (d *Desktop) ConfigureUpdater(directory, installed string, quit func()) error {
	config, err := os.ReadFile(filepath.Join(directory, "bootstrap-config.json"))
	if err != nil {
		return fmt.Errorf("read installed update addresses: %w", err)
	}
	raw, err := releasetrust.DecodeConfig(config)
	if err != nil {
		return fmt.Errorf("check installed update addresses: %w", err)
	}
	root, err := os.ReadFile(filepath.Join(directory, "bootstrap-root.json"))
	if err != nil {
		return fmt.Errorf("read installed update trust: %w", err)
	}
	client, err := releasetrust.New(releasetrust.Bootstrap{Config: raw, Root: root}, directory)
	if err != nil {
		return fmt.Errorf("open installed update trust: %w", err)
	}
	d.update = func() error {
		path, err := client.Stage(installed)
		if err != nil {
			return fmt.Errorf("stage desktop update: %w", err)
		}
		return d.launchUpdate(path, quit)
	}
	return nil
}

// PracticeUpdate exposes the blocking screen without inventing a release or trusting test keys.
// Only an unstamped development composition may enable this fixture.
func (d *Desktop) PracticeUpdate() { d.required = serverversion.App }

func (d *Desktop) installUpdate() error {
	if d.required != serverversion.App {
		return fmt.Errorf("no desktop update is required")
	}
	if d.update == nil {
		return fmt.Errorf("update setup is missing. Ask the person who installed The War Room to restore its update setup, then try again. Your league is unchanged")
	}
	if err := d.update(); err != nil {
		return fmt.Errorf("the update could not be checked or opened. Your league is unchanged. Try again; if it keeps failing, contact the person who installed The War Room")
	}
	return nil
}

func (d *Desktop) launchUpdate(path string, quit func()) error {
	command := exec.Command(path, "-native", d.client.base, d.league) //nolint:noctx,gosec // Only Stage supplies this verified private executable; the child must outlive the request.
	if err := command.Start(); err != nil {
		return fmt.Errorf("open updated desktop: %w", err)
	}
	go func() { _ = command.Wait() }()
	quit()
	return nil
}
