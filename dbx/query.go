package dbx

import "context"

// Scanner reads the current row: a *sql.Row or *sql.Rows.
type Scanner interface {
	Scan(dest ...any) error
}

// ScanFunc reads one row into a value, choosing the columns itself.
type ScanFunc[T any] func(Scanner) (T, error)

// Get runs a query returning at most one row and reads it with scan. With no
// row it returns sql.ErrNoRows, for the caller to map to its own error.
func Get[T any](ctx context.Context, q Querier, scan ScanFunc[T], query string, args ...any) (T, error) {
	return scan(q.QueryRowContext(ctx, query, args...))
}

// Select runs a query and reads every row with scan, in order.
func Select[T any](ctx context.Context, q Querier, scan ScanFunc[T], query string, args ...any) ([]T, error) {
	var out []T

	err := Each(ctx, q, func(sc Scanner) error {
		v, err := scan(sc)
		if err != nil {
			return err
		}

		out = append(out, v)

		return nil
	}, query, args...)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// Each runs a query and calls fn for every row, in order, stopping at the
// first error. Use it when rows fold into something other than a slice, such
// as a parent with its children from a join.
func Each(ctx context.Context, q Querier, fn func(Scanner) error, query string, args ...any) error {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := fn(rows); err != nil {
			return err
		}
	}

	return rows.Err()
}
