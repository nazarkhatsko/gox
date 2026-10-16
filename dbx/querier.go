// Package dbx makes database/sql less repetitive: one
// interface for a database, connection or transaction, transactions as
// closures, queries that return values instead of rows to loop over, and
// conversions for column types. It knows nothing about any schema; what a
// query reads is decided by the scan function its caller passes.
package dbx

import (
	"context"
	"database/sql"
)

// Querier runs statements: a *sql.DB, *sql.Conn or *sql.Tx.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

var (
	_ Querier = (*sql.DB)(nil)
	_ Querier = (*sql.Conn)(nil)
	_ Querier = (*sql.Tx)(nil)
)
