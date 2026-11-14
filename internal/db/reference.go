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
// connection (read connections attach it read-only, as their own file is). Both files must already exist;
// the league file is refused when newer than version, and the reference file is checked by its own Open.
func OpenWithReference(ctx context.Context, path string, version int, reference string) (*Pools, error) {
	if reference == "" {
		return nil, fmt.Errorf("db: name the reference file")
	}
	attachOnce.Do(func() { sqlite.RegisterConnectionHook(attachReference) })
	return open(ctx, path, reference, version)
}

// Reference is the attached reference file's path, or empty for a file opened on its own.
func (p *Pools) Reference() string { return p.reference }

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
	target := fileURI(reference)
	if values.Get("mode") == "ro" {
		target += "?mode=ro"
	}
	if _, err := conn.ExecContext(context.Background(), `ATTACH DATABASE '`+strings.ReplaceAll(target, "'", "''")+`' AS `+ReferenceSchema, nil); err != nil {
		return fmt.Errorf("db: attach the reference file: %w", err)
	}
	return nil
}
