package transactions

import "context"

// WithDirectory returns a copy of c that resolves requests against dir, so a test can vary the
// players directory per call.
func (c *Coordinator) WithDirectory(dir Directory) *Coordinator {
	cp := *c
	cp.dirs = func(context.Context) (Directory, error) { return dir, nil }
	return &cp
}
