package dbutil

import (
	"context"
	"database/sql"
	"strings"

	"github.com/cockroachdb/errors"
)

// EscapeLike escapes the LIKE wildcards so the keyword matches literally.
func EscapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// TryAdvisoryLock tries to acquire a session-level advisory lock on a dedicated connection, ok is false if it is held by others.
// The lock is released along with the connection when the process exits.
func TryAdvisoryLock(ctx context.Context, db *sql.DB, key int64) (unlock func(), ok bool, err error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, false, errors.Wrap(err, "get connection")
	}
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&ok); err != nil {
		_ = conn.Close()
		return nil, false, errors.Wrap(err, "try advisory lock")
	}
	if !ok {
		_ = conn.Close()
		return nil, false, nil
	}
	return func() {
		_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", key)
		_ = conn.Close()
	}, true, nil
}
