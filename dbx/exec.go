package dbx

import (
	"context"
	"fmt"
)

// ExecOne runs a statement that targets an existing row, and returns
// notFound if it affected none.
func ExecOne(ctx context.Context, q Querier, notFound error, query string, args ...any) error {
	n, err := ExecCount(ctx, q, query, args...)
	if err != nil {
		return err
	}

	if n == 0 {
		return notFound
	}

	return nil
}

// ExecCount runs a statement and returns how many rows it affected.
func ExecCount(ctx context.Context, q Querier, query string, args ...any) (int64, error) {
	res, err := q.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count affected rows: %w", err)
	}

	return n, nil
}
