package dbx

import (
	"context"
	"database/sql"
	"fmt"
)

// beginner starts transactions: a *sql.DB or *sql.Conn.
type beginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// InTx runs fn in a transaction on q, committing if fn returns nil and
// rolling back otherwise. If q is already a transaction, fn joins it and
// committing is left to whoever began it. Callers wrap the error with what
// the transaction does.
func InTx(ctx context.Context, q Querier, fn func(Querier) error) error {
	if tx, ok := q.(*sql.Tx); ok {
		return fn(tx)
	}

	b, ok := q.(beginner)
	if !ok {
		return fmt.Errorf("begin: %T cannot start a transaction", q)
	}

	tx, err := b.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	// A no-op returning sql.ErrTxDone once Commit has run.
	defer func() { _ = tx.Rollback() }()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}
