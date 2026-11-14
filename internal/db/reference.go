package db

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	sqlite "modernc.org/sqlite"
)

// The hosted server keeps football reference data (raw archive, observations, models, live lines,
// player identity and records) in its own file beside the league file. A league file is opened with
// the reference file attached on every connection as schema "reference": a table the league file no
// longer holds resolves to the reference file by name, so the stores' queries read across the two
// files unchanged. SQLite commits each file on its own under WAL, so no transaction may write both:
// every reference write commits first and on its own (identity, blobs), then the league write runs.

// ReferenceSchema is the attached reference file's schema name on a league connection.
const ReferenceSchema = "reference"

const attachKey = "twr_reference"

//nolint:gochecknoglobals // the driver's connection hook is process-wide; registered once
var attachOnce sync.Once

// OpenWithReference opens a league file with the reference file at reference attached on every
// connection (all connections attach it read-only). The caller owns both pools;
// the league file is refused when newer than version, and the reference file is checked by its own Open.
func OpenWithReference(ctx context.Context, path string, version int, reference *Pools) (*Pools, error) {
	if reference == nil {
		return nil, fmt.Errorf("db: name the reference file")
	}
	attachOnce.Do(func() { sqlite.RegisterConnectionHook(attachReference) })
	league, err := open(ctx, path, reference.path, version)
	if err != nil {
		return nil, err
	}
	league.referencePools = reference
	return league, nil
}

// Reference is the attached reference file's path, or empty for a file opened on its own.
func (p *Pools) Reference() string { return p.reference }

// ReferencePools owns reference writes. A desktop file owns both kinds of data.
func (p *Pools) ReferencePools() *Pools {
	if p.referencePools != nil {
		return p.referencePools
	}
	return p
}

func attachParam(reference string) string {
	if reference == "" {
		return ""
	}
	return "&" + attachKey + "=" + url.QueryEscape(reference)
}

// attachReference runs on every new connection; only DSNs built by OpenWithReference carry the key.
func attachReference(conn sqlite.ExecQuerierContext, dsn string) error {
	_, query, found := strings.Cut(dsn, "?")
	if !found {
		return nil
	}
	values, err := url.ParseQuery(query)
	if err != nil {
		return nil //nolint:nilerr // not ours to judge: the driver parses its own DSN
	}
	reference := values.Get(attachKey)
	if reference == "" {
		return nil
	}
	target := fileURI(reference) + "?mode=ro"
	if _, err := conn.ExecContext(context.Background(), `ATTACH DATABASE '`+strings.ReplaceAll(target, "'", "''")+`' AS `+ReferenceSchema, nil); err != nil {
		return fmt.Errorf("db: attach the reference file: %w", err)
	}
	return nil
}
