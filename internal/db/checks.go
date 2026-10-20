package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Check returns the actual integrity and foreign-key check rows. A reported violation is
// evidence for the caller, not an error that hides the other findings.
func Check(ctx context.Context, pool *sql.DB) (CheckFindings, error) {
	var result CheckFindings
	rows, err := pool.QueryContext(ctx, "PRAGMA integrity_check")
	if err != nil {
		return result, fmt.Errorf("db: integrity check: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var finding string
		if err := rows.Scan(&finding); err != nil {
			return result, fmt.Errorf("db: scan integrity check: %w", err)
		}
		result.Integrity = append(result.Integrity, finding)
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("db: integrity check rows: %w", err)
	}
	if err := rows.Close(); err != nil {
		return result, fmt.Errorf("db: close integrity check: %w", err)
	}
	result.ForeignKeys, err = foreignKeys(ctx, pool)
	return result, err
}

type checkQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// CheckForeignKeys reads violations inside a migration's transaction.
func CheckForeignKeys(ctx context.Context, tx *sql.Tx) ([]ForeignKeyFinding, error) {
	return foreignKeys(ctx, tx)
}

func foreignKeys(ctx context.Context, query checkQuerier) ([]ForeignKeyFinding, error) {
	rows, err := query.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return nil, fmt.Errorf("db: foreign key check: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var result []ForeignKeyFinding
	for rows.Next() {
		var finding ForeignKeyFinding
		if err := rows.Scan(&finding.Table, &finding.RowID, &finding.Parent, &finding.Constraint); err != nil {
			return nil, fmt.Errorf("db: scan foreign key check: %w", err)
		}
		result = append(result, finding)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: foreign key check rows: %w", err)
	}
	return result, nil
}
